package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

// setupRepairTaskHandlerRouter 构造最小路由，仅挂载 AcceptTask 用于验证
// 非法状态流转的 HTTP 状态码映射。
func setupRepairTaskHandlerRouter(t *testing.T) (*gin.Engine, *repository.FeedbackRepairTaskRepo, *sql.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDBFull(t)
	t.Cleanup(func() { db.Close() })

	// 手工建 feedback_repair_tasks 表（迁移 109 未在 testutil 清单内）
	createStmt := `CREATE TABLE IF NOT EXISTS feedback_repair_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT, task_no TEXT NOT NULL UNIQUE,
		creator TEXT NOT NULL, feedback_ids TEXT NOT NULL, title TEXT NOT NULL DEFAULT '',
		diagnosis TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'approved',
		worker_host TEXT NOT NULL DEFAULT '', worker_token_note TEXT NOT NULL DEFAULT '',
		base_commit TEXT NOT NULL DEFAULT '', branch TEXT NOT NULL DEFAULT '',
		verify_result TEXT NOT NULL DEFAULT '', diff_stat TEXT NOT NULL DEFAULT '',
		log_text TEXT DEFAULT '', accept_note TEXT NOT NULL DEFAULT '',
		accepted_by TEXT NOT NULL DEFAULT '', reject_reason TEXT NOT NULL DEFAULT '',
		rejected_by TEXT NOT NULL DEFAULT '', deploy_confirmed_by TEXT NOT NULL DEFAULT '',
		deploy_ref TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')));`
	if _, err := db.Exec(createStmt); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	taskRepo := repository.NewFeedbackRepairTaskRepo(db)
	userRepo := repository.NewUserRepo(db)
	ssRepo := repository.NewFeedbackScreenshotRepo(db)
	fbRepo := repository.NewFeedbackRepo(db)
	feedbackSvc := service.NewFeedbackService(fbRepo, userRepo, ssRepo)
	taskSvc := service.NewFeedbackRepairTaskService(taskRepo, feedbackSvc)
	taskH := NewFeedbackRepairTaskHandler(taskSvc)

	r := gin.New()
	r.POST("/api/v1/admin/feedback/repair-tasks/:no/accept", taskH.AcceptTask)

	return r, taskRepo, db
}

// TestRepairTaskHandler_AcceptIllegalState_StatusCode 验收一个非 awaiting_acceptance 状态的任务，
// 期望返回 400（非法流转）。当前实现 badStateOrErr 使用 switch err 直接比较，
// 而 service 返回的是 %w 包装错误，会导致落入 default 分支返回 500——本用例断言真实行为。
func TestRepairTaskHandler_AcceptIllegalState_StatusCode(t *testing.T) {
	r, taskRepo, _ := setupRepairTaskHandlerRouter(t)

	// 插入一个仍处于 approved 状态的任务（不可验收）
	if _, err := taskRepo.Create(&model.FeedbackRepairTask{
		TaskNo: "rt-x", Creator: "admin", FeedbackIDs: `["fb-1"]`,
		Title: "t", Diagnosis: "{}", Status: model.RepairTaskApproved,
	}); err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/feedback/repair-tasks/rt-x/accept",
		strings.NewReader(`{"note":"非法验收"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	t.Logf("Accept illegal state -> status=%d body=%s", w.Code, w.Body.String())

	// 依据需求：非法流转应返回 400。
	if w.Code == http.StatusInternalServerError {
		t.Errorf("非法状态流转应返回 400，实际返回 500（badStateOrErr 的 switch err 未识别 %%w 包装错误）")
	} else if w.Code != http.StatusBadRequest {
		t.Errorf("非法状态流转期望 400，得到 %d", w.Code)
	}
}
