package app

// 回归测试（qa-regression-wxx）
//
// 本文件为新增验证性测试，不修改任何既有生产代码。
// 聚焦点（对应 dev-refactor-wxx refactor-notes.md「qa 重点验证点」）：
//   1. 路由注册完整性 / 关键路径 / 静态-API 顺序
//   2. /health 返回结构
//   3. 迁移幂等（重复执行 _migrations 去重）
//   4. SQL 分割 / duplicate column|index 容错
//   5. placeholder 501 占位响应

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/gin-gonic/gin"

	_ "modernc.org/sqlite" // 内存 SQLite 驱动（含 FTS5）
)

// ─────────────────────────────────────────────────────────────
// 1. 路由注册完整性（与拆分前备份 app.go.bak.orig 比对）
// ─────────────────────────────────────────────────────────────

// TestRouteRegistrationCount 校验拆分后路由注册计数为 478 处
//（465 = 454 + 督办工单 D5-3 新增 11 处；+2 = 教师成绩录入 P0-1 新增 2 处；
//  +5 = 教师授课关系申报+审核 R3 新增 5 处：teacher 2 + assistant 3；
//  +6 = 教师作业信息发布+成绩统计 P2 新增 6 处：teacher POST/PUT/DELETE/mine/courses/grade-stats）。
// 含全套 GET/POST/PUT/DELETE/PATCH 路由方法。
// 注：拆分前备份文件 app.go.bak.orig 已删除，故不再与备份比对（路由一致性在
// qa 阶段已通过 数量/集合/顺序 三维对比验证，且拆分的 9 个函数逐字节保留）。
func TestRouteRegistrationCount(t *testing.T) {
	// 直接读取本目录源码做静态断言，避免依赖解析器的脆弱性。
	cur, err := os.ReadFile(filepath.Join("routes.go"))
	if err != nil {
		t.Fatalf("读取 routes.go 失败: %v", err)
	}
	curText := string(cur)

	// 统计所有 .METHOD("..." 形式的路由注册（GROUP 用 .Group( 不匹配）
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "ANY"}
	total := 0
	for _, m := range methods {
		total += strings.Count(curText, "."+m+`("`)
	}

	t.Logf("当前 routes.go 路由注册调用数: %d", total)

	if total != 478 {
		t.Errorf("routes.go 路由注册数 = %d, 期望 478（472 + 教师作业 P2 新增 6 处）", total)
	}
}

// TestKeyRoutesReachable 校验关键路由路径在源码中的注册存在性，
// 覆盖 refactor-notes 抽样要求：auth/chat/kb/student/counselor/admin/college/union。
func TestKeyRoutesReachable(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("routes.go"))
	if err != nil {
		t.Fatalf("读取 routes.go 失败: %v", err)
	}
	s := string(src)

	// 路由组前缀存在性。Gin 采用 router.Group() 拼装路径，源码中呈现为组前缀而非完整字面量。
	requiredGroups := []string{
		`"/api/v1"`,    // API v1 总组
		`"/auth"`,      // 认证
		`"/student"`,   // 学生
		`"/counselor"`, // 辅导员
		`"/admin"`,     // 管理员
		`"/union"`,     // 学生会
		`"/college"`,   // 学院
		`"/sys-admin"`, // 系统管理员
		`"/kb"`,        // 知识库
		`"/teacher"`,   // 教师
		`/health`,      // 健康检查
	}
	for _, p := range requiredGroups {
		if !strings.Contains(s, p) {
			t.Errorf("路由组/路径缺失: %s", p)
		} else {
			t.Logf("checked route group present: %s", p)
		}
	}

	// 关键业务端点（已落实的具体注册字面量）
	requiredEndpointFragments := []string{
		`"/login"`,            // auth.login
		`"/chat"`,             // 问答主链路
		`"/knowledge/public"`, // 知识大厅（公开）
		`"/campus/steps"`,     // 校园报到步骤（公开）
	}
	for _, p := range requiredEndpointFragments {
		if !strings.Contains(s, p) {
			t.Errorf("关键端点注册缺失: %s", p)
		}
	}

	// 书记/教务秘书（SecretaryOutcome）无独立路由组，端点布置在 student/teacher/assistant 组内。
	// 校验其底层能力门控端点在 routes.go 已注册。
	secretaryAnchors := []string{
		`"/outcome/self-report"`, // student 组：学生成效自评
		`"/outcome/records"`,     // assistant 组：成效记录列表
		`"/party/register"`,      // teacher 组：党建登记
	}
	for _, p := range secretaryAnchors {
		if !strings.Contains(s, p) {
			t.Errorf("秘书/书记相关端点注册缺失: %s", p)
		}
	}

	// 能力门控锚点：确认 RequireCapability 仍被用于受保护路由
	if !strings.Contains(s, "RequireCapability(") {
		t.Error("routes.go 中未找到 RequireCapability 能力门控调用")
	}
	if !strings.Contains(s, "JWTAuth(") {
		t.Error("routes.go 中未找到 JWTAuth 中间件绑定")
	}
}

// TestStaticServiceOrder 校验静态文件服务位于 /health 与 API 路由之后注册，
// 且 NoRoute SPA 回退不拦截 /api/ 与 /health。
func TestStaticServiceOrder(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("routes.go"))
	if err != nil {
		t.Fatalf("读取 routes.go 失败: %v", err)
	}
	s := string(src)

	// NoRoute SPA 回退的关键：/api/ 与 /health 前缀必须被豁免（否则会拦截 API/健康检查）
	if !strings.Contains(s, `!strings.HasPrefix(c.Request.URL.Path, "/api/")`) {
		t.Error("NoRoute 未对 /api/ 前缀做好豁免")
	}
	if !strings.Contains(s, `!strings.HasPrefix(c.Request.URL.Path, "/health")`) {
		t.Error("NoRoute 未对 /health 前缀做好豁免")
	}
	// gin 中 NoRoute 仅在所有精确/通配路由不匹配时触发；静态与 API 路由均在 NoRoute 之前注册即可。
	// 校验静态服务与 API 均存在。
	if !strings.Contains(s, `router.Static("/assets"`) {
		t.Error("未找到 /assets 静态挂载")
	}
	if !strings.Contains(s, `router.Static("/canvaskit"`) {
		t.Error("未找到 /canvaskit 静态挂载")
	}
	// 静态 NoCache 入口文件
	for _, f := range []string{"/main.dart.js", "/index.html", "/flutter_bootstrap.js", "/flutter_service_worker.js"} {
		if !strings.Contains(s, f) {
			t.Errorf("静态 NoCache 入口缺失: %s", f)
		}
	}
}

// ─────────────────────────────────────────────────────────────
// 2. /health 返回结构
// ─────────────────────────────────────────────────────────────

func newHealthTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestHealthHandlerStructure 校验 /health 响应 JSON 结构（status/dependencies.*）
// 与 refactor-notes「qa 重点验证点 1」一致。
func TestHealthHandlerStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newHealthTestDB(t)

	// 注入 appRedis，确保 redis 分支进入
	origRedis := appRedis
	appRedis = nil
	defer func() { appRedis = origRedis }()

	router := gin.New()
	router.GET("/health", healthHandler(db))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("/health 状态码 = %d, 期望 200", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析 /health JSON 失败: %v", err)
	}

	// 顶层字段
	for _, k := range []string{"status", "service", "version", "uptime", "dependencies", "time"} {
		if _, ok := body[k]; !ok {
			t.Errorf("/health 缺少顶层字段: %s", k)
		}
	}
	if body["service"] != "蔚小芯" {
		t.Errorf("service = %v, 期望 蔚小芯", body["service"])
	}
	if body["version"] != "0.0.1" {
		t.Errorf("version = %v, 期望 0.0.1", body["version"])
	}

	// dependencies 子字段
	deps, ok := body["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatalf("dependencies 解析失败: %v", body["dependencies"])
	}
	for _, k := range []string{"database", "redis", "fts5", "llm_api"} {
		if _, ok := deps[k]; !ok {
			t.Errorf("dependencies 缺少字段: %s", k)
		}
	}

	// database 子结构
	ddb, ok := deps["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("database 解析失败: %v", deps["database"])
	}
	for _, k := range []string{"status", "latency", "driver"} {
		if _, ok := ddb[k]; !ok {
			t.Errorf("database 缺少字段: %s", k)
		}
	}
	if ddb["status"] != "ok" {
		t.Errorf("database.status = %v, 期望 ok", ddb["status"])
	}
	// 内存 SQLite → 无 FTS5 kb_fts 表，fts5 应为 unavailable
	if fts, ok := deps["fts5"].(map[string]interface{}); ok {
		if fts["status"] != "unavailable" && fts["status"] != "ok" {
			t.Errorf("fts5.status = %v, 期望 ok 或 unavailable", fts["status"])
		}
	}
	// 无 API Key 时应报告 no_api_key
	if llm, ok := deps["llm_api"].(map[string]interface{}); ok {
		if llm["status"] != "no_api_key" && llm["status"] != "configured" {
			t.Errorf("llm_api.status = %v", llm["status"])
		}
	}
	// redis 未初始化时为 disabled
	if rd, ok := deps["redis"].(map[string]interface{}); ok {
		if rd["status"] != "disabled" {
			t.Errorf("redis.status = %v, 期望 disabled", rd["status"])
		}
	}
}

// TestHealthHandlerDBDown 校验 DB ping 失败时 status=degraded。
func TestHealthHandlerDBDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	// 立即关闭，使 Ping 失败
	db.Close()

	router := gin.New()
	router.GET("/health", healthHandler(db))
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析 /health JSON 失败: %v", err)
	}
	if body["status"] != "degraded" {
		t.Errorf("status = %v, 期望 degraded (DB 不可用时)", body["status"])
	}
	if deps, ok := body["dependencies"].(map[string]interface{}); ok {
		if ddb, ok := deps["database"].(map[string]interface{}); ok {
			if ddb["status"] == "ok" {
				t.Error("database.status 应为非 ok（DB 已关闭）")
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────
// 3. placeholder 501 占位
// ─────────────────────────────────────────────────────────────

func TestPlaceholderHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/placeholder", placeholderHandler("测试功能"))

	req := httptest.NewRequest("GET", "/placeholder", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("placeholder 状态码 = %d, 期望 501", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析 placeholder JSON 失败: %v", err)
	}
	if body["code"] != float64(501) {
		t.Errorf("code = %v, 期望 501", body["code"])
	}
	if body["message"] != "测试功能 待实现" {
		t.Errorf("message = %v, 期望 '测试功能 待实现'", body["message"])
	}
}

// ─────────────────────────────────────────────────────────────
// 4. SQL 分割 / duplicate column|index 容错
// ─────────────────────────────────────────────────────────────

func TestSplitSQL_Basic(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"单条", "CREATE TABLE a (id INTEGER);", 1},
		{"多条", "CREATE TABLE a (id INTEGER);\nCREATE TABLE b (id INTEGER);", 2},
		{"空行", "CREATE TABLE a (id INTEGER);\n\n", 1},
		{"触发器复合", "CREATE TABLE a (id INTEGER);\nCREATE TRIGGER t AFTER INSERT ON a BEGIN INSERT INTO b VALUES (new.id); END;", 2},
		{"行尾注释分号", "ALTER TABLE a ADD COLUMN b TEXT DEFAULT '[]';  -- JSON 数组", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := splitSQL(c.in)
			if len(got) != c.want {
				t.Errorf("splitSQL(%q) 语句数 = %d, 期望 %d; got=%v", c.in, len(got), c.want, got)
			}
		})
	}
}

func TestSplitSQL_CommentOnly(t *testing.T) {
	got := splitSQL("-- 纯注释行\n-- 另一行")
	if len(got) != 0 {
		t.Errorf("纯注释应返回 0 条, got %d: %v", len(got), got)
	}
}

func TestIsDuplicateColumnError(t *testing.T) {
	if !isDuplicateColumnError(sqlError("SQLite: duplicate column name: b")) {
		t.Error("应识别 SQLite duplicate column name")
	}
	if !isDuplicateColumnError(sqlError("Error 1060: Duplicate column name 'b'")) {
		t.Error("应识别 MySQL duplicate column name")
	}
	if isDuplicateColumnError(nil) {
		t.Error("nil 不应为 duplicate column")
	}
	if isDuplicateColumnError(sqlError("other error")) {
		t.Error("无关错误不应误判")
	}
}

func TestIsDuplicateIndexError(t *testing.T) {
	if !isDuplicateIndexError(sqlError("Error 1061: Duplicate key name 'idx_a'")) {
		t.Error("应识别 MySQL Duplicate key name")
	}
	if !isDuplicateIndexError(sqlError("index idx_a already exists")) {
		t.Error("应识别 SQLite already exists")
	}
	if isDuplicateIndexError(nil) {
		t.Error("nil 不应为 duplicate index")
	}
}

type fakeErr struct{ msg string }

func (e fakeErr) Error() string { return e.msg }

func sqlError(msg string) error { return fakeErr{msg} }

// ─────────────────────────────────────────────────────────────
// 5. 迁移幂等（重复执行 runMigrations _migrations 去重）
// ─────────────────────────────────────────────────────────────

// TestRunMigrationsIdempotent 对内存 SQLite 执行两次 runMigrations，
// 验证第二次不重复执行（_migrations 去重）。
func TestRunMigrationsIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full-migration idempotency in -short mode")
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	// 首次执行全部迁移
	if err := runMigrations(db, dbutil.DriverSQLite); err != nil {
		t.Fatalf("首次 runMigrations 失败: %v", err)
	}

	var countAfterFirst int
	if err := db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&countAfterFirst); err != nil {
		t.Fatalf("读取 _migrations 失败: %v", err)
	}
	t.Logf("首次迁移后 _migrations 记录数: %d", countAfterFirst)
	if countAfterFirst == 0 {
		t.Fatal("首次迁移后 _migrations 为空，异常")
	}

	// 第二次执行：应全部跳过（不重新 INSERT）
	if err := runMigrations(db, dbutil.DriverSQLite); err != nil {
		t.Fatalf("第二次 runMigrations 失败: %v", err)
	}
	var countAfterSecond int
	if err := db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&countAfterSecond); err != nil {
		t.Fatalf("读取 _migrations 失败: %v", err)
	}
	t.Logf("第二次迁移后 _migrations 记录数: %d", countAfterSecond)
	if countAfterSecond != countAfterFirst {
		t.Errorf("迁移幂等失败: 第一次=%d 第二次=%d, 期望相等", countAfterFirst, countAfterSecond)
	}
}

// TestRunMigrationsCreatesSchema 校验迁移确实建出关键表（确保幂等测试不是空跑）。
func TestRunMigrationsCreatesSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in -short mode")
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	if err := runMigrations(db, dbutil.DriverSQLite); err != nil {
		t.Fatalf("runMigrations 失败: %v", err)
	}
	// 校验关键表存在
	var n int
	for _, tbl := range []string{"users", "kb_resources", "sessions", "messages", "_migrations"} {
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&n)
		if err != nil {
			t.Fatalf("查询表 %s 失败: %v", tbl, err)
		}
		if n == 0 {
			t.Errorf("迁移后缺少关键表: %s", tbl)
		}
	}
}
