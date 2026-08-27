package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// createRepairTaskTable 在内存 SQLite 中创建 feedback_repair_tasks 表（迁移 109）。
// 说明：testutil.NewTestDB 的迁移清单未包含 109，此处按迁移脚本内容手工建表，
// 保证测试环境与生产表结构一致。
func createRepairTaskTable(t *testing.T, db *sql.DB) {
	t.Helper()
	stmt := `CREATE TABLE IF NOT EXISTS feedback_repair_tasks (
		id                  INTEGER PRIMARY KEY AUTOINCREMENT,
		task_no             TEXT NOT NULL UNIQUE,
		creator             TEXT NOT NULL,
		feedback_ids        TEXT NOT NULL,
		title               TEXT NOT NULL DEFAULT '',
		diagnosis           TEXT NOT NULL DEFAULT '',
		status              TEXT NOT NULL DEFAULT 'approved',
		worker_host         TEXT NOT NULL DEFAULT '',
		worker_token_note   TEXT NOT NULL DEFAULT '',
		base_commit         TEXT NOT NULL DEFAULT '',
		branch              TEXT NOT NULL DEFAULT '',
		verify_result       TEXT NOT NULL DEFAULT '',
		diff_stat           TEXT NOT NULL DEFAULT '',
		log_text            TEXT DEFAULT '',
		accept_note         TEXT NOT NULL DEFAULT '',
		accepted_by         TEXT NOT NULL DEFAULT '',
		reject_reason       TEXT NOT NULL DEFAULT '',
		rejected_by         TEXT NOT NULL DEFAULT '',
		deploy_confirmed_by TEXT NOT NULL DEFAULT '',
		deploy_ref          TEXT NOT NULL DEFAULT '',
		created_at          TEXT NOT NULL DEFAULT (datetime('now','localtime')),
		updated_at          TEXT NOT NULL DEFAULT (datetime('now','localtime'))
	); CREATE INDEX IF NOT EXISTS idx_frt_status ON feedback_repair_tasks(status);`
	if _, err := db.Exec(stmt); err != nil {
		t.Fatalf("创建 feedback_repair_tasks 表失败: %v", err)
	}
}

// setupRepairTaskSvc 构造含真实内存库的 FeedbackRepairTaskService。
// 返回 svc 与底层 db，供测试直接插入任务记录。
func setupRepairTaskSvc(t *testing.T) (*FeedbackRepairTaskService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDBFull(t)
	createRepairTaskTable(t, db)

	userRepo := repository.NewUserRepo(db)
	screenshotRepo := repository.NewFeedbackScreenshotRepo(db)
	fbRepo := repository.NewFeedbackRepo(db)
	feedbackSvc := NewFeedbackService(fbRepo, userRepo, screenshotRepo)
	feedbackSvc.SetDB(db)

	taskRepo := repository.NewFeedbackRepairTaskRepo(db)
	svc := NewFeedbackRepairTaskService(taskRepo, feedbackSvc)
	return svc, db
}

// insertTask 直接向表中插入一条指定状态的任务，返回其 ID。
func insertTask(t *testing.T, db *sql.DB, taskNo, status string) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO feedback_repair_tasks
		 (task_no, creator, feedback_ids, title, diagnosis, status)
		 VALUES (?, 'admin', '["fb-1"]', 't', '{}', ?)`,
		taskNo, status,
	)
	if err != nil {
		t.Fatalf("插入任务失败: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// statusOf 读取任务当前状态。
func statusOf(t *testing.T, db *sql.DB, taskNo string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT status FROM feedback_repair_tasks WHERE task_no = ?`, taskNo).Scan(&s); err != nil {
		t.Fatalf("读取状态失败 (taskNo=%s): %v", taskNo, err)
	}
	return s
}

// TestRepairTask_StateMachine_FullChain 全链路流转：
// approved → running → awaiting_acceptance → deploy_pending → deploying → deployed → closed。
func TestRepairTask_StateMachine_FullChain(t *testing.T) {
	svc, db := setupRepairTaskSvc(t)
	insertTask(t, db, "rt-full", model.RepairTaskApproved)

	// 1. 认领：approved → running
	if _, err := svc.Claim(context.Background(), "dev-host", "abc123", "repair/rt-full"); err != nil {
		t.Fatalf("Claim 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-full"); got != model.RepairTaskRunning {
		t.Fatalf("认领后状态应为 running，得到 %s", got)
	}

	// 2. 验证通过上报：running → awaiting_acceptance
	if _, err := svc.SubmitVerify("rt-full", &model.RepairTaskVerifyRequest{
		Passed: true, GoVet: "pass", GoTest: "pass", FlutterAnalyze: "pass", FlutterTest: "skip",
	}, "repair-agent"); err != nil {
		t.Fatalf("SubmitVerify(passed) 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-full"); got != model.RepairTaskAwaitingAcceptance {
		t.Fatalf("验证通过后状态应为 awaiting_acceptance，得到 %s", got)
	}

	// 3. 验收：awaiting_acceptance → deploy_pending
	if _, err := svc.Accept("rt-full", "admin", "验收通过"); err != nil {
		t.Fatalf("Accept 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-full"); got != model.RepairTaskDeployPending {
		t.Fatalf("验收后状态应为 deploy_pending，得到 %s", got)
	}

	// 4. 部署确认：deploy_pending → deploying
	if _, err := svc.DeployConfirm("rt-full", "admin", "gh-run-999"); err != nil {
		t.Fatalf("DeployConfirm 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-full"); got != model.RepairTaskDeploying {
		t.Fatalf("部署确认后状态应为 deploying，得到 %s", got)
	}

	// 5. 部署完成（不联动解决反馈）：deploying → deployed → closed
	if _, err := svc.DeployDone("rt-full", "admin", "已上线", false); err != nil {
		t.Fatalf("DeployDone 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-full"); got != model.RepairTaskClosed {
		t.Fatalf("部署完成后状态应为 closed，得到 %s", got)
	}
}

// TestRepairTask_StateMachine_VerifyFailedLoop running → verify_failed → running → awaiting_acceptance。
func TestRepairTask_StateMachine_VerifyFailedLoop(t *testing.T) {
	svc, db := setupRepairTaskSvc(t)
	insertTask(t, db, "rt-vf", model.RepairTaskApproved)

	if _, err := svc.Claim(context.Background(), "dev-host", "abc", "branch"); err != nil {
		t.Fatalf("Claim 失败: %v", err)
	}

	// 验证失败上报：running → verify_failed
	if _, err := svc.SubmitVerify("rt-vf", &model.RepairTaskVerifyRequest{
		Passed: false, GoVet: "fail", GoTest: "fail",
	}, "repair-agent"); err != nil {
		t.Fatalf("SubmitVerify(failed) 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-vf"); got != model.RepairTaskVerifyFailed {
		t.Fatalf("验证失败后状态应为 verify_failed，得到 %s", got)
	}

	// 重新认领：verify_failed → running（NextClaimable 会选中 verify_failed）
	if _, err := svc.Claim(context.Background(), "dev-host", "abc", "branch2"); err != nil {
		t.Fatalf("重新认领失败: %v", err)
	}
	if got := statusOf(t, db, "rt-vf"); got != model.RepairTaskRunning {
		t.Fatalf("重新认领后状态应为 running，得到 %s", got)
	}

	// 再次验证通过
	if _, err := svc.SubmitVerify("rt-vf", &model.RepairTaskVerifyRequest{Passed: true}, "repair-agent"); err != nil {
		t.Fatalf("再次验证上报失败: %v", err)
	}
	if got := statusOf(t, db, "rt-vf"); got != model.RepairTaskAwaitingAcceptance {
		t.Fatalf("再次验证通过后状态应为 awaiting_acceptance，得到 %s", got)
	}
}

// TestRepairTask_StateMachine_Cancel approved → cancelled（终态）。
func TestRepairTask_StateMachine_Cancel(t *testing.T) {
	svc, db := setupRepairTaskSvc(t)
	insertTask(t, db, "rt-cancel", model.RepairTaskApproved)

	dto, err := svc.Cancel("rt-cancel", "admin")
	if err != nil {
		t.Fatalf("Cancel 失败: %v", err)
	}
	if dto.Status != model.RepairTaskCancelled {
		t.Fatalf("取消后状态应为 cancelled，得到 %s", dto.Status)
	}
	// cancelled 是终态，再次取消应报错
	if _, err := svc.Cancel("rt-cancel", "admin"); err == nil {
		t.Fatal("cancelled 终态再次取消应报错，但未报错")
	}
}

// TestRepairTask_StateMachine_Reject 验收/部署阶段驳回 → verify_failed。
func TestRepairTask_StateMachine_Reject(t *testing.T) {
	svc, db := setupRepairTaskSvc(t)
	insertTask(t, db, "rt-reject", model.RepairTaskApproved)

	if _, err := svc.Claim(context.Background(), "dev", "c", "b"); err != nil {
		t.Fatalf("Claim 失败: %v", err)
	}
	if _, err := svc.SubmitVerify("rt-reject", &model.RepairTaskVerifyRequest{Passed: true}, "agent"); err != nil {
		t.Fatalf("SubmitVerify 失败: %v", err)
	}
	// awaiting_acceptance → reject → verify_failed
	if _, err := svc.Reject("rt-reject", "admin", "验收不通过，需整改"); err != nil {
		t.Fatalf("Reject 失败: %v", err)
	}
	if got := statusOf(t, db, "rt-reject"); got != model.RepairTaskVerifyFailed {
		t.Fatalf("驳回后状态应为 verify_failed，得到 %s", got)
	}
}

// TestRepairTask_IllegalTransition_ReturnsError 非法流转应返回错误（Sentinel 为 ErrRepairTaskBadState）。
// 例：approved 状态直接 Accept（验收）属非法流转。
func TestRepairTask_IllegalTransition_ReturnsError(t *testing.T) {
	svc, db := setupRepairTaskSvc(t)
	insertTask(t, db, "rt-illegal", model.RepairTaskApproved)

	// 直接验收一个仍在 approved 的任务 → 应报错
	if _, err := svc.Accept("rt-illegal", "admin", "非法验收"); err == nil {
		t.Fatal("approved 状态直接验收应报错，但未报错")
	} else if !errorsIsBadState(err) {
		t.Fatalf("期望 ErrRepairTaskBadState 类错误，得到 %v", err)
	}

	// 直接部署确认一个 approved 任务 → 应报错
	if _, err := svc.DeployConfirm("rt-illegal", "admin", "gh"); err == nil {
		t.Fatal("approved 状态直接部署确认应报错，但未报错")
	}

	// 直接验证上报一个 approved（未领取）任务 → 应报错（仅 running 可上报）
	if _, err := svc.SubmitVerify("rt-illegal", &model.RepairTaskVerifyRequest{Passed: true}, "agent"); err == nil {
		t.Fatal("approved 状态直接验证上报应报错，但未报错")
	}
}

// errorsIsBadState 判断错误是否为状态机非法流转（ErrRepairTaskBadState，含 %w 包装）。
func errorsIsBadState(err error) bool {
	for e := err; e != nil; {
		if e == ErrRepairTaskBadState {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// TestRepairTask_ConcurrencyGate 全局单 running 闸门：已存在 running 时新任务不可领取。
func TestRepairTask_ConcurrencyGate(t *testing.T) {
	svc, db := setupRepairTaskSvc(t)
	insertTask(t, db, "rt-a", model.RepairTaskApproved)
	insertTask(t, db, "rt-b", model.RepairTaskApproved)

	// 领取第一个任务
	if _, err := svc.Claim(context.Background(), "dev", "c", "b"); err != nil {
		t.Fatalf("第一次认领失败: %v", err)
	}

	// 第二个任务在存在 running 时应被并发闸门拒绝
	_, err := svc.Claim(context.Background(), "dev", "c", "b2")
	if err != ErrRepairTaskConcurrency {
		t.Fatalf("期望 ErrRepairTaskConcurrency，得到 %v", err)
	}
}
