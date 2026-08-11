package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// FeedbackRepairRepo 反馈 AI 自动修复工单数据访问
type FeedbackRepairRepo struct {
	db *sql.DB
}

// NewFeedbackRepairRepo 创建反馈修复工单 repo
func NewFeedbackRepairRepo(db *sql.DB) *FeedbackRepairRepo {
	return &FeedbackRepairRepo{db: db}
}

const repairJobCols = `id, run_id, feedback_id, operator, status, stage, log_text,
 edited_files, summary, detail, created_at, updated_at`

func scanRepairJob(row interface{ Scan(...interface{}) error }, j *model.FeedbackRepairJob) error {
	return row.Scan(&j.ID, &j.RunID, &j.FeedbackID, &j.Operator, &j.Status, &j.Stage,
		&j.LogText, &j.EditedFiles, &j.Summary, &j.Detail, &j.CreatedAt, &j.UpdatedAt)
}

// Create 新建修复工单，返回自增 ID
func (r *FeedbackRepairRepo) Create(j *model.FeedbackRepairJob) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO feedback_repair_jobs
		 (run_id, feedback_id, operator, status, stage, log_text, edited_files, summary, detail)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.RunID, j.FeedbackID, j.Operator, j.Status, j.Stage, j.LogText, j.EditedFiles, j.Summary, j.Detail,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AppendLog 追加一行日志
func (r *FeedbackRepairRepo) AppendLog(jobID int64, line string) error {
	_, err := r.db.Exec(
		`UPDATE feedback_repair_jobs
		 SET log_text = log_text || ?, updated_at = datetime('now','localtime')
		 WHERE id = ?`,
		line+"\n", jobID,
	)
	return err
}

// UpdateStage 更新阶段
func (r *FeedbackRepairRepo) UpdateStage(jobID int64, stage string) error {
	_, err := r.db.Exec(
		`UPDATE feedback_repair_jobs
		 SET stage = ?, updated_at = datetime('now','localtime')
		 WHERE id = ?`,
		stage, jobID,
	)
	return err
}

// Finalize 结束工单（status + detail）
func (r *FeedbackRepairRepo) Finalize(jobID int64, status, detail string) error {
	_, err := r.db.Exec(
		`UPDATE feedback_repair_jobs
		 SET status = ?, detail = ?, updated_at = datetime('now','localtime')
		 WHERE id = ?`,
		status, detail, jobID,
	)
	return err
}

// SetEditedFiles 记录被修改文件（JSON 数组）
func (r *FeedbackRepairRepo) SetEditedFiles(jobID int64, filesJSON string) error {
	_, err := r.db.Exec(
		`UPDATE feedback_repair_jobs
		 SET edited_files = ?, updated_at = datetime('now','localtime')
		 WHERE id = ?`,
		filesJSON, jobID,
	)
	return err
}

// GetByRunID 按 run_id 查询工单
func (r *FeedbackRepairRepo) GetByRunID(runID string) (*model.FeedbackRepairJob, error) {
	row := r.db.QueryRow(`SELECT `+repairJobCols+` FROM feedback_repair_jobs WHERE run_id = ?`, runID)
	j := &model.FeedbackRepairJob{}
	if err := scanRepairJob(row, j); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}

// LatestByFeedback 查询反馈的最新工单
func (r *FeedbackRepairRepo) LatestByFeedback(feedbackID string) (*model.FeedbackRepairJob, error) {
	row := r.db.QueryRow(
		`SELECT `+repairJobCols+` FROM feedback_repair_jobs WHERE feedback_id = ? ORDER BY id DESC LIMIT 1`,
		feedbackID,
	)
	j := &model.FeedbackRepairJob{}
	if err := scanRepairJob(row, j); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}
