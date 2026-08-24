package handler

// vOPC 全链路流程打通验证：创建项目「测试」（全字段占位）→ 提交立项 → 进入 S1/pending_review。
// 用于验证用户反馈「项目名 测试，提交立项 422 需补齐产品形态」已修复：带完整字段后流程贯通。

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

func vopcFullFlowDB(t *testing.T) *sql.DB {
	t.Helper()
	db := vopcTestDB(t) // 097-104
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

func vopcFullFlowRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	g.POST("/projects", h.CreateProject)
	g.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitProject)
	g.GET("/projects/:id", h.GetProject)
	return r
}

// validFullProject 返回含全部必填字段的项目创建体（占位值，字段待定）。
func validFullProject(name string) map[string]any {
	return map[string]any{
		"name": name, "summary": "项目摘要", "problem_statement": "待解决的问题",
		"target_users": "目标用户", "expected_outcome": "预期成果", "validation_plan": "验证计划",
		"project_type": "自由探索项目", "project_source": "self_proposed", "product_form": "Web 应用",
		"project_cycle": "8 周", "acceptance_criteria": "验收标准", "mentor_needs": "导师需求",
		"resource_needs": "资源需求", "risk_level": "R0", "data_type": "公开数据",
		"real_user_trial": false, "external_publish": false, "funds_involved": false,
	}
}

// TestVOPCFullFlowCreateAndSubmit 创建「测试」→ 提交立项 → 进入 S1/pending_review。
func TestVOPCFullFlowCreateAndSubmit(t *testing.T) {
	db := vopcFullFlowDB(t)
	r := vopcFullFlowRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")

	// 1. 创建项目（全字段）
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validFullProject("测试"))
	if w.Code != 201 {
		t.Fatalf("create got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)

	// 2. 详情应返回 product_form 已存（Web 应用）
	var pf, nm string
	if e := db.QueryRow(`SELECT product_form,name FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&pf, &nm); e != nil {
		t.Fatal(e)
	}
	if pf != "Web 应用" || nm != "测试" {
		t.Fatalf("stored name=%q product_form=%q", nm, pf)
	}

	// 3. 提交立项（应 200，不再 422）
	sw := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), owner, nil)
	if sw.Code != 200 {
		t.Fatalf("submit got %d %s（若为 422 说明必填字段仍缺失）", sw.Code, sw.Body.String())
	}
	// 4. 项目进入 G1/pending_review
	var stage, status string
	if e := db.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&stage, &status); e != nil {
		t.Fatal(e)
	}
	if stage != "G1" || status != "pending_review" {
		t.Fatalf("post-submit stage=%s status=%s want G1/pending_review", stage, status)
	}
	t.Logf("全链路打通：项目「%s」(id=%d) 已从 G0/draft 提交立项进入 %s/%s", nm, out.Data.ID, stage, status)
}

// TestVOPCFullFlowMissingFieldShowsMessage 验证缺字段时 SubmitProject 明确报缺失（诊断用户原 422）。
func TestVOPCFullFlowMissingFieldShowsMessage(t *testing.T) {
	db := vopcFullFlowDB(t)
	r := vopcFullFlowRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	// 缺 product_form 创建（模拟用户之前保存草稿时没填产品形态）
	body := validFullProject("缺字段项目")
	body["product_form"] = ""
	w := request(r, "POST", "/api/v1/vopc/projects", owner, body)
	if w.Code != 201 {
		t.Fatalf("create got %d", w.Code)
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	// 提交立项 → 422 明确提示产品形态
	sw := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), owner, nil)
	if sw.Code != 422 {
		t.Fatalf("missing product_form submit got %d, want 422", sw.Code)
	}
	if !jsonContains(sw.Body.String(), "产品形态") {
		t.Fatalf("422 message should mention 产品形态, got: %s", sw.Body.String())
	}
	t.Logf("缺字段时后端明确提示：%s", sw.Body.String())
}

func jsonContains(body, sub string) bool {
	return strings.Contains(body, sub)
}
