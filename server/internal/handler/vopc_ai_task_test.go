package handler

// vOPC 虚拟向导测试（v2.0）：验证模板化草稿生成（不再调真实 LLM）、岗位启用门禁、审阅四态、
// 修改率统计与重试语义。复用 vopcTestDB（097-104 已含 107）+ token/request/validProject/mustA4Project helper。

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// vopcB1DB 复用 vopcTestDB 并补 106 迁移（vopcTestDB 已含 107）。
func vopcB1DB(t *testing.T) *sql.DB {
	t.Helper()
	db := vopcTestDB(t)
	m, err := os.ReadFile("../../migrations/106_vopc_ai_tasks.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(m)); err != nil {
		t.Fatal(err)
	}
	return db
}

// vopcB1Router 注册 vOPC 组 + 虚拟向导 ai-tasks 路由。v2.0 不再注入 LLM 客户端。
func vopcB1Router(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	g.POST("/projects", h.CreateProject)
	g.POST("/projects/:id/ai-tasks", h.CreateAITask)
	g.GET("/projects/:id/ai-tasks", h.ListAITasks)
	g.GET("/projects/:id/ai-tasks/:taskId", h.GetAITask)
	g.POST("/projects/:id/ai-tasks/:taskId/review", h.ReviewAITask)
	g.POST("/projects/:id/ai-tasks/:taskId/retry", h.RetryAITask)
	return r
}

// TestVOPCB1CreateExecutesAndReturns 虚拟向导草稿生成：不调 LLM，模板渲染即 succeeded。
func TestVOPCB1CreateExecutesAndReturns(t *testing.T) {
	db := vopcB1DB(t)
	r := vopcB1Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)

	w := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/ai-tasks", id), owner, map[string]any{"role_key": "project_manager", "instruction": "为项目生成需求基线"})
	if w.Code != 201 {
		t.Fatalf("create virtual guide got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
			Output string `json:"output_content"`
			Model  string `json:"model"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Data.Status != "succeeded" {
		t.Fatalf("status=%s want succeeded", out.Data.Status)
	}
	if out.Data.Model != "virtual_guide" {
		t.Fatalf("model=%s want virtual_guide", out.Data.Model)
	}
	if out.Data.Output == "" {
		t.Fatalf("template draft must not be empty")
	}
	// 模板化草稿：不产生真实 token 消耗，额度保持默认。
	var pt, ot int64
	if err := db.QueryRow(`SELECT prompt_tokens,output_tokens FROM vopc_ai_tasks WHERE id=?`, out.Data.ID).Scan(&pt, &ot); err != nil || pt != 0 || ot != 0 {
		t.Fatalf("tokens pt=%d ot=%d err=%v want 0/0 (模板渲染不累计 token)", pt, ot, err)
	}
	var provider string
	if err := db.QueryRow(`SELECT provider FROM vopc_ai_tasks WHERE id=?`, out.Data.ID).Scan(&provider); err != nil || provider != "template" {
		t.Fatalf("provider=%s err=%v want template", provider, err)
	}
}

// TestVOPCB1RoleNotEnabled 虚拟向导岗位未启用门禁。
func TestVOPCB1RoleNotEnabled(t *testing.T) {
	db := vopcB1DB(t)
	r := vopcB1Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)
	if _, err := db.Exec(`UPDATE vopc_ai_roles SET enabled=0 WHERE project_id=? AND role_key='project_manager'`, id); err != nil {
		t.Fatal(err)
	}
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/ai-tasks", id), owner, map[string]any{"role_key": "project_manager", "instruction": "任务"}).Code; got != 422 {
		t.Fatalf("disabled role got %d want 422", got)
	}
}

// TestVOPCB1ReviewActions 审阅四态：accept/revise/reject/overrule + 修改率统计。
func TestVOPCB1ReviewActions(t *testing.T) {
	db := vopcB1DB(t)
	r := vopcB1Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	other := token(t, 2, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)
	w := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/ai-tasks", id), owner, map[string]any{"role_key": "project_manager", "instruction": "任务"})
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPathA4("/api/v1/vopc/projects/%d/ai-tasks/%d", id, out.Data.ID)

	// 非主理人不能审阅（普通成员无 manage）→ 404/403
	if got := request(r, "POST", base+"/review", other, map[string]any{"decision": "accept"}).Code; got != 404 && got != 403 {
		t.Fatalf("non-manager review got %d", got)
	}
	// 非法决策 → 422
	if got := request(r, "POST", base+"/review", owner, map[string]any{"decision": "maybe"}).Code; got != 422 {
		t.Fatalf("invalid decision got %d", got)
	}
	// revise 无指示 → 422
	if got := request(r, "POST", base+"/review", owner, map[string]any{"decision": "revise"}).Code; got != 422 {
		t.Fatalf("revise without revision got %d", got)
	}
	// revise + 修订指示 → 200 && final_decision=revise && 修改率 > 0
	if got := request(r, "POST", base+"/review", owner, map[string]any{"decision": "revise", "revision": "目标改为B\n增加两条验收"}).Code; got != 200 {
		t.Fatalf("revise got %d", got)
	}
	var fd sql.NullString
	var rev string
	if err := db.QueryRow(`SELECT final_decision,revision FROM vopc_ai_tasks WHERE id=?`, out.Data.ID).Scan(&fd, &rev); err != nil || !fd.Valid || fd.String != "revise" {
		t.Fatalf("final_decision=%v err=%v", fd, err)
	}
	if modificationRate(rev) <= 0 {
		t.Fatalf("modification rate should be > 0 for revise with revision=%q", rev)
	}
	// 二次审阅 → 409
	if got := request(r, "POST", base+"/review", owner, map[string]any{"decision": "reject"}).Code; got != 409 {
		t.Fatalf("double review got %d", got)
	}
}

// TestVOPCB1NoLLMClient 兼容验证：即使未注入 LLM（历史装配残留），虚拟向导仍可用（不依赖 LLM）。
func TestVOPCB1NoLLMClient(t *testing.T) {
	db := vopcB1DB(t)
	r := vopcB1Router(db) // 不注入任何 LLM 客户端
	owner := token(t, 1, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/ai-tasks", id), owner, map[string]any{"role_key": "project_manager", "instruction": "任务"}).Code; got != 201 {
		t.Fatalf("virtual guide without LLM got %d want 201", got)
	}
}
