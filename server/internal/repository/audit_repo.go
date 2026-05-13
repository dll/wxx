package repository

import (
	"database/sql"

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
