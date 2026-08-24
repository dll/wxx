package handler

// vOPC 项目生命周期修复测试：删除（仅草稿，级联清理）+ 归档扩展 + 越权。
// 删除的级联验证需要一个启用 foreign_keys 的 SQLite 连接（生产 migration.go 已启用）。

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

// vopcLifecycleDB 复用 vopcTestDB 的完整 schema（097-105）并启用 foreign_keys，用于验证删除级联。
func vopcLifecycleDB(t *testing.T) *sql.DB {
	t.Helper()
	db := vopcTestDB(t) // 已含 097-104
	for _, n := range []string{"105_vopc_milestone_rubric_waiver_evidence.sql", "106_vopc_ai_tasks.sql"} {
		if _, err := db.Exec(string(mustReadMigration(t, n))); err != nil {
			t.Fatal(err)
		}
	}
	// 同一连接上开启 foreign_keys（对 :memory: 生效需在连接首次使用前；vopcTestDB 用 SetMaxOpenConns(1)）
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustReadMigration(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../migrations/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// vopcLifecycleRouter 注册创建/删除/close 路由（含 delete）。
func vopcLifecycleRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	g.POST("/projects", h.CreateProject)
	g.GET("/projects/:id", h.GetProject)
	g.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitProject)
	g.POST("/projects/:id/close", auth.RequireCapability(auth.VOPCProjectManage), h.CloseProject)
	g.POST("/projects/:id/delete", auth.RequireCapability(auth.VOPCProjectManage), h.DeleteProject)
	g.GET("/projects/:id/tasks", h.ListTasks)
	g.GET("/projects/:id/decisions", h.ListDecisions)
	g.GET("/projects/:id/members", h.ListMembers)
	return r
}

func lifecycleCreateProject(t *testing.T, r *gin.Engine, tok string) int64 {
	t.Helper()
	w := request(r, "POST", "/api/v1/vopc/projects", tok, validProject())
	if w.Code != 201 {
		t.Fatalf("create got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out.Data.ID
}

// TestVOPCLifecycleDeleteDraft verifies deleting a draft project clears cascade rows.
func TestVOPCLifecycleDeleteDraft(t *testing.T) {
	db := vopcLifecycleDB(t)
	r := vopcLifecycleRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := lifecycleCreateProject(t, r, owner)

	// 创建项目时应已插入 members + 4 ai_roles + 5 milestones
	var mCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_project_members WHERE project_id=?`, id).Scan(&mCount)
	if mCount == 0 {
		t.Fatal("project members should exist")
	}
	// 删除草稿
	base := fmtPathA4("/api/v1/vopc/projects/%d/delete", id)
	if got := request(r, "POST", base, owner, nil).Code; got != 200 {
		t.Fatalf("delete got %d", got)
	}
	// 项目行删除
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_projects WHERE id=?`, id).Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("project row still exists, count=%d", cnt)
	}
	// 子表级联清理
	for _, tbl := range []string{"vopc_project_members", "vopc_ai_roles", "vopc_milestones", "vopc_decisions", "vopc_events", "vopc_tasks"} {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM ` + tbl + ` WHERE project_id=?`, id).Scan(&n)
		if n != 0 {
			t.Fatalf("cascade cleanup failed for %s: count=%d", tbl, n)
		}
	}
}

// TestVOPCLifecycleDeleteNonDraft verifies non-draft project delete returns 409.
func TestVOPCLifecycleDeleteNonDraft(t *testing.T) {
	db := vopcLifecycleDB(t)
	r := vopcLifecycleRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := lifecycleCreateProject(t, r, owner)
	// 提交立项（进入 G1/pending_review，非 draft）
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", id), owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/delete", id), owner, nil).Code; got != 409 {
		t.Fatalf("non-draft delete got %d want 409", got)
	}
	// 项目仍存在
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_projects WHERE id=?`, id).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("project should remain, count=%d", cnt)
	}
}

// TestVOPCLifecycleDeleteNonManager verifies non-manager cannot delete.
func TestVOPCLifecycleDeleteNonManager(t *testing.T) {
	db := vopcLifecycleDB(t)
	r := vopcLifecycleRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	other := token(t, 2, "student", "college", "cs", "active")
	id := lifecycleCreateProject(t, r, owner)
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/delete", id), other, nil).Code; got != 404 {
		t.Fatalf("non-manager delete got %d want 404", got)
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_projects WHERE id=?`, id).Scan(&cnt)
	if cnt != 1 {
		t.Fatal("project should not be deleted by non-manager")
	}
}

// TestVOPCLifecycleArchiveExpand verifies archive works from a normal non-draft state, and draft too.
func TestVOPCLifecycleArchiveExpand(t *testing.T) {
	db := vopcLifecycleDB(t)
	r := vopcLifecycleRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	// 场景1：draft 直接归档（草稿也可收尾）
	draftID := lifecycleCreateProject(t, r, owner)
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/close", draftID), owner, map[string]any{"action": "archive", "reason": "草稿废弃"}).Code; got != 200 {
		t.Fatalf("draft archive got %d", got)
	}
	var st string
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, draftID).Scan(&st); err != nil || st != "archived" {
		t.Fatalf("draft post-archive status=%s err=%v", st, err)
	}
	// 场景2：提交后（常规状态）归档
	pid := lifecycleCreateProject(t, r, owner)
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", pid), owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/close", pid), owner, map[string]any{"action": "archive", "reason": "不再推进"}).Code; got != 200 {
		t.Fatalf("submitted archive got %d", got)
	}
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, pid).Scan(&st); err != nil || st != "archived" {
		t.Fatalf("submitted post-archive status=%s err=%v", st, err)
	}
	// 场景3：archived 不可再 archive（409）
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/close", pid), owner, map[string]any{"action": "archive", "reason": "重复"}).Code; got != 409 {
		t.Fatalf("re-archive got %d want 409", got)
	}
}
