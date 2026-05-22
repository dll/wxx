package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// ProcessRecordRepo 办事流程记录数据访问
type ProcessRecordRepo struct {
	db *sql.DB
}

// NewProcessRecordRepo 创建仓储
func NewProcessRecordRepo(db *sql.DB) *ProcessRecordRepo {
	return &ProcessRecordRepo{db: db}
}

const processRecordCols = `id, record_id, user_id, flow_type, flow_label, current_step,
completed_steps, total_steps, status, notes, created_at, updated_at`

func (r *ProcessRecordRepo) scan(row interface {
	Scan(dest ...interface{}) error
}) (*model.ProcessRecord, error) {
	rec := &model.ProcessRecord{}
	err := row.Scan(
		&rec.ID, &rec.RecordID, &rec.UserID, &rec.FlowType, &rec.FlowLabel,
		&rec.CurrentStep, &rec.CompletedSteps, &rec.TotalSteps,
		&rec.Status, &rec.Notes, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// GetActiveByUserFlow 查询用户的某流程进行中记录（每用户每流程仅一条）
func (r *ProcessRecordRepo) GetActiveByUserFlow(userID int64, flowType string) (*model.ProcessRecord, error) {
	row := r.db.QueryRow(
		`SELECT `+processRecordCols+` FROM process_records
		 WHERE user_id = ? AND flow_type = ? AND status != 'abandoned'
		 ORDER BY id DESC LIMIT 1`,
		userID, flowType,
	)
	return r.scan(row)
}

// GetByRecordID 按 record_id 查询
func (r *ProcessRecordRepo) GetByRecordID(recordID string) (*model.ProcessRecord, error) {
	row := r.db.QueryRow(
		`SELECT `+processRecordCols+` FROM process_records WHERE record_id = ?`,
		recordID,
	)
	return r.scan(row)
}

// Create 新建记录
func (r *ProcessRecordRepo) Create(rec *model.ProcessRecord) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO process_records
		 (record_id, user_id, flow_type, flow_label, current_step, completed_steps, total_steps, status, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.RecordID, rec.UserID, rec.FlowType, rec.FlowLabel, rec.CurrentStep,
		rec.CompletedSteps, rec.TotalSteps, rec.Status, rec.Notes,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Update 更新记录的进度/状态字段
func (r *ProcessRecordRepo) Update(rec *model.ProcessRecord) error {
	_, err := r.db.Exec(
		`UPDATE process_records
		 SET current_step = ?, completed_steps = ?, total_steps = ?,
		     status = ?, notes = ?, flow_label = ?, updated_at = datetime('now')
		 WHERE record_id = ?`,
		rec.CurrentStep, rec.CompletedSteps, rec.TotalSteps,
		rec.Status, rec.Notes, rec.FlowLabel, rec.RecordID,
	)
	return err
}

// ListByUser 按用户列出全部办事记录（最新优先）
func (r *ProcessRecordRepo) ListByUser(userID int64, limit int) ([]*model.ProcessRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT `+processRecordCols+` FROM process_records
		 WHERE user_id = ?
		 ORDER BY updated_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.ProcessRecord
	for rows.Next() {
		rec := &model.ProcessRecord{}
		if err := rows.Scan(
			&rec.ID, &rec.RecordID, &rec.UserID, &rec.FlowType, &rec.FlowLabel,
			&rec.CurrentStep, &rec.CompletedSteps, &rec.TotalSteps,
			&rec.Status, &rec.Notes, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}
