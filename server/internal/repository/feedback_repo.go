package repository

import (
	"database/sql"
	"time"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// FeedbackRepo 用户反馈数据访问
type FeedbackRepo struct {
	db    *sql.DB
	mysql bool
}

// NewFeedbackRepo 创建反馈 repo
func NewFeedbackRepo(db *sql.DB) *FeedbackRepo {
	return &FeedbackRepo{db: db, mysql: dbutil.IsMySQL(db)}
}

// listFeedbackCols 统一 SELECT 列名
const listFeedbackCols = `id, feedback_id, user_id, username, message_id, resource_id,
 category, module, content, screenshot_url, status, resolved_by, resolved_at, reply,
 rating, rating_comment, rated_at, linked_resource_note, linked_at, linked_by,
 created_at, updated_at`

// scanFeedback 扫描一行反馈数据
func scanFeedback(rows *sql.Rows, fb *model.Feedback) error {
	return rows.Scan(&fb.ID, &fb.FeedbackID, &fb.UserID, &fb.Username,
		&fb.MessageID, &fb.ResourceID, &fb.Category, &fb.Module, &fb.Content, &fb.ScreenshotURL,
		&fb.Status, &fb.ResolvedBy, &fb.ResolvedAt, &fb.Reply,
		&fb.Rating, &fb.RatingComment, &fb.RatedAt,
		&fb.LinkedResourceNote, &fb.LinkedAt, &fb.LinkedBy,
		&fb.CreatedAt, &fb.UpdatedAt)
}

// scanFeedbackRow 扫描单行反馈
func scanFeedbackRow(row *sql.Row, fb *model.Feedback) error {
	return row.Scan(&fb.ID, &fb.FeedbackID, &fb.UserID, &fb.Username,
		&fb.MessageID, &fb.ResourceID, &fb.Category, &fb.Module, &fb.Content, &fb.ScreenshotURL,
		&fb.Status, &fb.ResolvedBy, &fb.ResolvedAt, &fb.Reply,
		&fb.Rating, &fb.RatingComment, &fb.RatedAt,
		&fb.LinkedResourceNote, &fb.LinkedAt, &fb.LinkedBy,
		&fb.CreatedAt, &fb.UpdatedAt)
}

// Create 创建反馈（含截图链接）
func (r *FeedbackRepo) Create(fb *model.Feedback) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO feedback (feedback_id, user_id, username, message_id, resource_id, category, module, content, screenshot_url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fb.FeedbackID, fb.UserID, fb.Username, fb.MessageID, fb.ResourceID, fb.Category, fb.Module, fb.Content, fb.ScreenshotURL,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// List 分页查询反馈列表
func (r *FeedbackRepo) List(status string, offset, limit int) ([]*model.Feedback, error) {
	query := `SELECT ` + listFeedbackCols + ` FROM feedback WHERE 1=1`
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
		if err := scanFeedback(rows, fb); err != nil {
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

// ListByUser 按用户 ID 分页查询反馈列表（用于"我的反馈"）
func (r *FeedbackRepo) ListByUser(userID int64, status string, offset, limit int) ([]*model.Feedback, error) {
	query := `SELECT ` + listFeedbackCols + ` FROM feedback WHERE user_id = ?`
	args := []interface{}{userID}

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
		if err := scanFeedback(rows, fb); err != nil {
			return nil, err
		}
		items = append(items, fb)
	}
	return items, rows.Err()
}

// CountByUser 统计指定用户的反馈总数
func (r *FeedbackRepo) CountByUser(userID int64, status string) (int, error) {
	query := `SELECT COUNT(*) FROM feedback WHERE user_id = ?`
	args := []interface{}{userID}

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
	err := scanFeedbackRow(
		r.db.QueryRow(`SELECT `+listFeedbackCols+` FROM feedback WHERE feedback_id = ?`, feedbackID),
		fb,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return fb, nil
}

// Update 更新反馈状态和回复
func (r *FeedbackRepo) Update(fb *model.Feedback) error {
	_, err := r.db.Exec(
		`UPDATE feedback SET status=?, resolved_by=?, resolved_at=?, reply=?, updated_at=CURRENT_TIMESTAMP
		 WHERE feedback_id=?`,
		fb.Status, fb.ResolvedBy, fb.ResolvedAt, fb.Reply, fb.FeedbackID,
	)
	return err
}

// UpdateRating 更新满意度评分
func (r *FeedbackRepo) UpdateRating(feedbackID string, rating int, comment string) error {
	_, err := r.db.Exec(
		`UPDATE feedback SET rating=?, rating_comment=?, rated_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP
		 WHERE feedback_id=?`,
		rating, comment, feedbackID,
	)
	return err
}

// LinkResource 关联知识库资源
func (r *FeedbackRepo) LinkResource(feedbackID, resourceID, note, linkedBy string) error {
	_, err := r.db.Exec(
		`UPDATE feedback SET resource_id=?, linked_resource_note=?, linked_by=?, linked_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP
		 WHERE feedback_id=?`,
		resourceID, note, linkedBy, feedbackID,
	)
	return err
}

// CountByStatus 按状态统计
func (r *FeedbackRepo) CountByStatus() (map[string]int, error) {
	rows, err := r.db.Query(`SELECT status, COUNT(*) FROM feedback GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]int{
		"pending":    0,
		"processing": 0,
		"resolved":   0,
		"dismissed":  0,
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			result[status] = count
		}
	}
	return result, rows.Err()
}

// CountByCategory 按分类统计
func (r *FeedbackRepo) CountByCategory() (map[string]int, error) {
	rows, err := r.db.Query(`SELECT category, COUNT(*) FROM feedback GROUP BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]int{
		"answer_error":    0,
		"feature_request": 0,
		"bug":             0,
		"other":           0,
	}
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err == nil {
			result[category] = count
		}
	}
	return result, rows.Err()
}

// WeekTrend 近 7 天趋势
func (r *FeedbackRepo) WeekTrend() ([]model.WeekTrendItem, error) {
	now := time.Now()
	var items []model.WeekTrendItem

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		var count int
		err := r.db.QueryRow(
			`SELECT COUNT(*) FROM feedback WHERE date(created_at) = ?`,
			date,
		).Scan(&count)
		if err != nil {
			return nil, err
		}
		items = append(items, model.WeekTrendItem{Date: date, Count: count})
	}
	return items, nil
}

// TopIssues 热门问题关键词（简单按内容中出现的高频词，先用 category 代替，后续可用 FTS）
func (r *FeedbackRepo) TopIssues(limit int) ([]model.TopIssueItem, error) {
	rows, err := r.db.Query(
		`SELECT category, COUNT(*) as cnt FROM feedback 
		 WHERE category != '' GROUP BY category 
		 ORDER BY cnt DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.TopIssueItem
	categoryLabels := map[string]string{
		"answer_error":    "回答有误",
		"feature_request": "功能建议",
		"bug":             "系统问题",
		"other":           "其他反馈",
	}
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err == nil {
			label, ok := categoryLabels[category]
			if !ok {
				label = category
			}
			items = append(items, model.TopIssueItem{Keyword: label, Count: count})
		}
	}
	return items, rows.Err()
}

// AvgResolveHours 平均解决时长（小时）
func (r *FeedbackRepo) AvgResolveHours() (float64, error) {
	var avg sql.NullFloat64
	var err error
	if r.mysql {
		// MySQL 无 julianday()，用 TIMESTAMPDIFF 计算小时差
		err = r.db.QueryRow(`
			SELECT AVG(TIMESTAMPDIFF(HOUR, created_at, resolved_at))
			FROM feedback
			WHERE status = 'resolved' AND resolved_at IS NOT NULL
		`).Scan(&avg)
	} else {
		err = r.db.QueryRow(`
			SELECT AVG(
				(julianday(resolved_at) - julianday(created_at)) * 24
			) FROM feedback
			WHERE status = 'resolved' AND resolved_at IS NOT NULL
		`).Scan(&avg)
	}
	if err != nil {
		return 0, err
	}
	if avg.Valid {
		return avg.Float64, nil
	}
	return 0, nil
}

// AddLog 添加反馈处理记录
func (r *FeedbackRepo) AddLog(feedbackID, action, operator, detail string) error {
	_, err := r.db.Exec(
		`INSERT INTO feedback_logs (feedback_id, action, operator, detail) VALUES (?, ?, ?, ?)`,
		feedbackID, action, operator, detail,
	)
	return err
}

// ListLogs 获取反馈处理记录列表
func (r *FeedbackRepo) ListLogs(feedbackID string) ([]*model.FeedbackLog, error) {
	rows, err := r.db.Query(
		`SELECT id, feedback_id, action, operator, detail, created_at 
		 FROM feedback_logs WHERE feedback_id = ? ORDER BY id DESC`,
		feedbackID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.FeedbackLog
	for rows.Next() {
		log := &model.FeedbackLog{}
		if err := rows.Scan(&log.ID, &log.FeedbackID, &log.Action,
			&log.Operator, &log.Detail, &log.CreatedAt); err == nil {
			items = append(items, log)
		}
	}
	return items, rows.Err()
}
