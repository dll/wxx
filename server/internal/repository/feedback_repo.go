package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// FeedbackRepo 用户反馈数据访问
type FeedbackRepo struct {
	db *sql.DB
}

// NewFeedbackRepo 创建反馈 repo
func NewFeedbackRepo(db *sql.DB) *FeedbackRepo {
	return &FeedbackRepo{db: db}
}

// Create 创建反馈
func (r *FeedbackRepo) Create(fb *model.Feedback) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO feedback (feedback_id, user_id, username, message_id, resource_id, category, content)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fb.FeedbackID, fb.UserID, fb.Username, fb.MessageID, fb.ResourceID, fb.Category, fb.Content,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// List 分页查询反馈列表
func (r *FeedbackRepo) List(status string, offset, limit int) ([]*model.Feedback, error) {
	query := `SELECT id, feedback_id, user_id, username, message_id, resource_id,
		category, content, status, resolved_by, resolved_at, created_at, updated_at
		FROM feedback WHERE 1=1`
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Feedback
	for rows.Next() {
		fb := &model.Feedback{}
		if err := rows.Scan(&fb.ID, &fb.FeedbackID, &fb.UserID, &fb.Username,
			&fb.MessageID, &fb.ResourceID, &fb.Category, &fb.Content,
			&fb.Status, &fb.ResolvedBy, &fb.ResolvedAt, &fb.CreatedAt, &fb.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, fb)
	}
	return items, rows.Err()
}

// Count 统计反馈总数
func (r *FeedbackRepo) Count(status string) (int, error) {
	query := `SELECT COUNT(*) FROM feedback WHERE 1=1`
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	var count int
	if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetByFeedbackID 按反馈 ID 查询
func (r *FeedbackRepo) GetByFeedbackID(feedbackID string) (*model.Feedback, error) {
	fb := &model.Feedback{}
	err := r.db.QueryRow(
		`SELECT id, feedback_id, user_id, username, message_id, resource_id,
		 category, content, status, resolved_by, resolved_at, created_at, updated_at
		 FROM feedback WHERE feedback_id = ?`, feedbackID,
	).Scan(&fb.ID, &fb.FeedbackID, &fb.UserID, &fb.Username,
		&fb.MessageID, &fb.ResourceID, &fb.Category, &fb.Content,
		&fb.Status, &fb.ResolvedBy, &fb.ResolvedAt, &fb.CreatedAt, &fb.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return fb, nil
}

// Update 更新反馈状态
func (r *FeedbackRepo) Update(fb *model.Feedback) error {
	_, err := r.db.Exec(
		`UPDATE feedback SET status=?, resolved_by=?, resolved_at=?, updated_at=datetime('now')
		 WHERE feedback_id=?`,
		fb.Status, fb.ResolvedBy, fb.ResolvedAt, fb.FeedbackID,
	)
	return err
}
