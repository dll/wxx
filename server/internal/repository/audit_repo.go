package repository

import (
	"database/sql"
	"fmt"

	"github.com/dll/wxx/server/internal/model"
)

// AuditRepo 审计日志数据访问
type AuditRepo struct {
	db *sql.DB
}

// NewAuditRepo 创建审计日志 repo
func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// List 分页查询审计日志，支持多条件过滤
func (r *AuditRepo) List(username, action, resource, startDate, endDate string, offset, limit int) ([]*model.AuditLog, error) {
	query := `SELECT id, user_id, username, role, action, resource, detail, trace_id, ip, duration_ms, result_code, created_at
		FROM audit_logs WHERE 1=1`
	var args []interface{}

	if username != "" {
		query += " AND username LIKE ?"
		args = append(args, "%"+username+"%")
	}
	if action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}
	if resource != "" {
		query += " AND resource LIKE ?"
		args = append(args, "%"+resource+"%")
	}
	if startDate != "" {
		query += " AND created_at >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND created_at <= ?"
		args = append(args, endDate)
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		l := &model.AuditLog{}
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Role, &l.Action,
			&l.Resource, &l.Detail, &l.TraceID, &l.IP, &l.DurationMs, &l.ResultCode, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// Count 统计审计日志总数
func (r *AuditRepo) Count(username, action, resource, startDate, endDate string) (int, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	var args []interface{}

	if username != "" {
		query += " AND username LIKE ?"
		args = append(args, "%"+username+"%")
	}
	if action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}
	if resource != "" {
		query += " AND resource LIKE ?"
		args = append(args, "%"+resource+"%")
	}
	if startDate != "" {
		query += " AND created_at >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND created_at <= ?"
		args = append(args, endDate)
	}

	var count int
	if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountDistinctActiveUsers 统计最近 sinceDays 天内有操作记录的去重用户数（真实活跃用户）
func (r *AuditRepo) CountDistinctActiveUsers(sinceDays int) (int, error) {
	if sinceDays <= 0 {
		sinceDays = 1
	}
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(DISTINCT username) FROM audit_logs
		 WHERE username != '' AND created_at >= datetime('now', ?)`,
		fmt.Sprintf("-%d days", sinceDays),
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ActionCount 操作类型计数项
type ActionCount struct {
	Action string
	Count  int
}

// TopActions 统计最近 sinceDays 天内最高频的操作类型（真实功能使用分布）
func (r *AuditRepo) TopActions(sinceDays, limit int) ([]ActionCount, error) {
	if sinceDays <= 0 {
		sinceDays = 30
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := r.db.Query(
		`SELECT action, COUNT(*) AS cnt FROM audit_logs
		 WHERE action != '' AND created_at >= datetime('now', ?)
		 GROUP BY action ORDER BY cnt DESC LIMIT ?`,
		fmt.Sprintf("-%d days", sinceDays), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ActionCount
	for rows.Next() {
		var a ActionCount
		if err := rows.Scan(&a.Action, &a.Count); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
