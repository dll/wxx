package repository

import (
	"database/sql"
	"encoding/json"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// FeedbackRepairTaskRepo 反馈修复任务（闭环 MVP）数据访问。
// 支持 MySQL/SQLite 双方言。
type FeedbackRepairTaskRepo struct {
	db *sql.DB
}

// NewFeedbackRepairTaskRepo 创建修复任务 repo
func NewFeedbackRepairTaskRepo(db *sql.DB) *FeedbackRepairTaskRepo {
	return &FeedbackRepairTaskRepo{db: db}
}

const repairTaskCols = `id, task_no, creator, feedback_ids, title, diagnosis, status,
 worker_host, worker_token_note, base_commit, branch, verify_result, diff_stat, log_text,
 accept_note, accepted_by, reject_reason, rejected_by, deploy_confirmed_by, deploy_ref, created_at, updated_at`

func scanRepairTask(row interface{ Scan(...interface{}) error }, t *model.FeedbackRepairTask) error {
	return row.Scan(&t.ID, &t.TaskNo, &t.Creator, &t.FeedbackIDs, &t.Title, &t.Diagnosis,
		&t.Status, &t.WorkerHost, &t.WorkerTokenNote, &t.BaseCommit, &t.Branch,
		&t.VerifyResult, &t.DiffStat, &t.LogText,
		&t.AcceptNote, &t.AcceptedBy, &t.RejectReason, &t.RejectedBy,
		&t.DeployConfirmedBy, &t.DeployRef, &t.CreatedAt, &t.UpdatedAt)
}

// Create 创建修复任务，返回自增 ID。
// 显式写入全部非自增列：MySQL 下长文本列（feedback_ids/diagnosis 等）无 DEFAULT，
// 若不显式赋值会得到 NULL，导致后续 scanRepairTask 把 NULL 扫入 string 报错。
func (r *FeedbackRepairTaskRepo) Create(t *model.FeedbackRepairTask) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO feedback_repair_tasks
		 (task_no, creator, feedback_ids, title, diagnosis, status, worker_host,
		  worker_token_note, base_commit, branch, verify_result, diff_stat, log_text,
		  accept_note, accepted_by, reject_reason, rejected_by, deploy_confirmed_by, deploy_ref)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.TaskNo, t.Creator, t.FeedbackIDs, t.Title, t.Diagnosis, t.Status, t.WorkerHost,
		t.WorkerTokenNote, t.BaseCommit, t.Branch, t.VerifyResult, t.DiffStat, t.LogText,
		t.AcceptNote, t.AcceptedBy, t.RejectReason, t.RejectedBy, t.DeployConfirmedBy, t.DeployRef,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetByTaskNo 按任务号查询
func (r *FeedbackRepairTaskRepo) GetByTaskNo(taskNo string) (*model.FeedbackRepairTask, error) {
	row := r.db.QueryRow(`SELECT `+repairTaskCols+` FROM feedback_repair_tasks WHERE task_no = ?`, taskNo)
	t := &model.FeedbackRepairTask{}
	if err := scanRepairTask(row, t); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// List 分页查询任务（可选状态过滤）
func (r *FeedbackRepairTaskRepo) List(status string, offset, limit int) ([]*model.FeedbackRepairTask, int, error) {
	where := ""
	args := []interface{}{}
	if status != "" {
		where = " WHERE status = ?"
		args = append(args, status)
	}

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM feedback_repair_tasks`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + repairTaskCols + ` FROM feedback_repair_tasks` + where +
		` ORDER BY id DESC LIMIT ? OFFSET ?`
	qargs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := r.db.Query(query, qargs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*model.FeedbackRepairTask
	for rows.Next() {
		t := &model.FeedbackRepairTask{}
		if err := scanRepairTask(rows, t); err != nil {
			return nil, 0, err
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}

// NextClaimable 认领最老的等待执行任务（approved 或 verify_failed）。
// 说明：MVP 为保证并发改码安全，全局同时最多 1 个 running 任务；
// running 唯一性由服务层再加一层校验。
func (r *FeedbackRepairTaskRepo) NextClaimable() (*model.FeedbackRepairTask, error) {
	row := r.db.QueryRow(
		`SELECT `+repairTaskCols+` FROM feedback_repair_tasks
		 WHERE status IN (?, ?) ORDER BY id ASC LIMIT 1`,
		model.RepairTaskApproved, model.RepairTaskVerifyFailed,
	)
	t := &model.FeedbackRepairTask{}
	if err := scanRepairTask(row, t); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// CountActiveRunning 统计当前 running（含 awaiting_acceptance）任务数，用于并发闸门。
func (r *FeedbackRepairTaskRepo) CountActiveRunning() (int, error) {
	var n int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM feedback_repair_tasks WHERE status IN (?, ?)`,
		model.RepairTaskRunning, model.RepairTaskAwaitingAcceptance,
	).Scan(&n)
	return n, err
}

// AppendLog 追加任务日志（保留既有历史）
func (r *FeedbackRepairTaskRepo) AppendLog(taskID int64, line string) error {
	stmt := `UPDATE feedback_repair_tasks
		 SET log_text = CASE WHEN log_text = '' THEN ? ELSE log_text || ? END,
		     updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)), line, "\n"+line, taskID)
	return err
}

// UpdateStatus 更新状态
func (r *FeedbackRepairTaskRepo) UpdateStatus(taskID int64, status string) error {
	stmt := `UPDATE feedback_repair_tasks SET status = ?, updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)), status, taskID)
	return err
}

// UpdateClaim 记录执行端认领信息（状态+running，写 worker/base_commit/branch）
func (r *FeedbackRepairTaskRepo) UpdateClaim(taskID int64, status, workerHost, baseCommit, branch string) error {
	stmt := `UPDATE feedback_repair_tasks
		 SET status = ?, worker_host = ?, base_commit = ?, branch = ?,
		     updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		status, workerHost, baseCommit, branch, taskID)
	return err
}

// UpdateVerifyReport 写入验证结果上报（状态 + verify_result + diff_stat + 日志）
func (r *FeedbackRepairTaskRepo) UpdateVerifyReport(taskID int64, status, verifyResultJSON, diffStat, logLine string) error {
	stmt := `UPDATE feedback_repair_tasks
		 SET status = ?, verify_result = ?, diff_stat = ?,
		 log_text = CASE WHEN log_text = '' THEN ? ELSE log_text || ? END,
		 updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		status, verifyResultJSON, diffStat, logLine, "\n"+logLine, taskID)
	return err
}

// UpdateAccept 管理员验收（status -> deploy_pending）
func (r *FeedbackRepairTaskRepo) UpdateAccept(taskID int64, status, acceptedBy, note string) error {
	stmt := `UPDATE feedback_repair_tasks
		 SET status = ?, accepted_by = ?, accept_note = ?,
		     updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		status, acceptedBy, note, taskID)
	return err
}

// UpdateReject 管理员驳回（回 verify_failed，记录原因与操作人）
func (r *FeedbackRepairTaskRepo) UpdateReject(taskID int64, status, rejectedBy, reason string) error {
	stmt := `UPDATE feedback_repair_tasks
		 SET status = ?, reject_reason = ?, rejected_by = ?, accepted_by = '',
		     updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		status, reason, rejectedBy, taskID)
	return err
}

// UpdateDeployConfirm 管理员部署确认（仅标记，不触发服务器动作）
func (r *FeedbackRepairTaskRepo) UpdateDeployConfirm(taskID int64, status, confirmedBy, deployRef string) error {
	stmt := `UPDATE feedback_repair_tasks
		 SET status = ?, deploy_confirmed_by = ?, deploy_ref = ?,
		     updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		status, confirmedBy, deployRef, taskID)
	return err
}

// UpdateDeployDone 部署完成（final status）
func (r *FeedbackRepairTaskRepo) UpdateDeployDone(taskID int64, status string) error {
	stmt := `UPDATE feedback_repair_tasks SET status = ?, updated_at = datetime('now','localtime') WHERE id = ?`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)), status, taskID)
	return err
}

// FeedbackIDsToJSON 序列化反馈 ID 切片
func FeedbackIDsToJSON(ids []string) string {
	b, _ := json.Marshal(ids)
	return string(b)
}
