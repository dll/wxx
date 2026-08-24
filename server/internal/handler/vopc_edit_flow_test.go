package handler

// 验证 UpdateProject（编辑草稿补全产品形态）对 G0 draft 项目是否生效，以及补全后能否提交立项。
// 用于诊断用户反馈「编辑草稿后修改没效果」。

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

func vopcEditDB(t *testing.T) *sql.DB {
	t.Helper()
	db := vopcTestDB(t)
	for _, n := range []string{"105_vopc_milestone_rubric_waiver_evidence.sql", "106_vopc_ai_tasks.sql"} {
		b, err := os.ReadFile("../../migrations/" + n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(b)); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func vopcEditRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	g.POST("/projects", h.CreateProject)
	g.GET("/projects/:id", h.GetProject)
	g.PUT("/projects/:id", auth.RequireCapability(auth.VOPCProjectManage), h.UpdateProject)
	g.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitProject)
	return r
}

// TestVOPCEditDraftFillsProductFormAndSubmits 核心：草稿缺 product_form → 编辑补全 → 提交立项 200。
func TestVOPCEditDraftFillsProductFormAndSubmits(t *testing.T) {
	db := vopcEditDB(t)
	r := vopcEditRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")

	// 1. 创建草稿（product_form 为空，模拟用户最初的状态）
	body := validFullProject("测试")
	body["product_form"] = ""
	w := request(r, "POST", "/api/v1/vopc/projects", owner, body)
	if w.Code != 201 {
		t.Fatalf("create got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	pid := out.Data.ID

	// 2. 确认项目仍是 G0/draft
	var stage, status string
	if e := db.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, pid).Scan(&stage, &status); e != nil {
		t.Fatal(e)
	}
	if stage != "G0" || status != "draft" {
		t.Fatalf("precondition: stage=%s status=%s want G0/draft", stage, status)
	}

	// 3. 编辑草稿：补全 product_form（PUT 带全字段）
	edit := validFullProject("测试")
	edit["product_form"] = "Web 应用" // 补齐
	uw := request(r, "PUT", fmtPath("/api/v1/vopc/projects/%d", pid), owner, edit)
	if uw.Code != 200 {
		t.Fatalf("update got %d %s", uw.Code, uw.Body.String())
	}

	// 4. 验证 product_form 已落库
	var pf string
	if e := db.QueryRow(`SELECT product_form FROM vopc_projects WHERE id=?`, pid).Scan(&pf); e != nil || pf != "Web 应用" {
		t.Fatalf("product_form after edit=%q err=%v want 'Web 应用'", pf, e)
	}

	// 5. 提交立项 → 应 200（不再 422）
	sw := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", pid), owner, nil)
	if sw.Code != 200 {
		t.Fatalf("submit after edit got %d %s", sw.Code, sw.Body.String())
	}
	var s2, st2 string
	if e := db.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, pid).Scan(&s2, &st2); e != nil || s2 != "G1" {
		t.Fatalf("post-submit stage=%s err=%v want G1", s2, e)
	}
	t.Logf("编辑草稿补全 product_form 后提交立项成功：项目→%s/%s", s2, st2)
}
