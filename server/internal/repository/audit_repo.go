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

// Delete 按条件删除审计日志（支持按时间范围/用户名/动作/资源过滤），返回删除条数。
func (r *AuditRepo) Delete(username, action, resource, startDate, endDate string) (int64, error) {
	query := `DELETE FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	if username != "" {
		query += ` AND username = ?`
		args = append(args, username)
	}
	if action != "" {
		query += ` AND action = ?`
		args = append(args, action)
	}
	if resource != "" {
		query += ` AND resource = ?`
		args = append(args, resource)
	}
	if startDate != "" {
		query += ` AND created_at >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND created_at <= ?`
		args = append(args, endDate)
	}
	res, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ClearAll 清空全部审计日志。
func (r *AuditRepo) ClearAll() error {
	_, err := r.db.Exec(`DELETE FROM audit_logs`)
	return err
}

// ListByUser 查询指定用户自己的操作日志（分页）
// actionType: "" 仅写操作(用户操作)；"all" 全部；其他 按指定 action
func (r *AuditRepo) ListByUser(userID int64, actionType, startDate, endDate string, offset, limit int) ([]*model.AuditLog, error) {
	query := `SELECT id, user_id, username, role, action, resource, detail, trace_id, ip, duration_ms, result_code, created_at
		FROM audit_logs WHERE user_id = ?`
	args := []interface{}{userID}
	switch actionType {
	case "all":
		// 全部
	case "", "user":
		// 仅用户操作（写操作，排除 GET 浏览）
		query += " AND action IN ('POST','PUT','PATCH','DELETE')"
	default:
		query += " AND action = ?"
		args = append(args, actionType)
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

// CountByUser 统计用户自己的日志总数
func (r *AuditRepo) CountByUser(userID int64, actionType, startDate, endDate string) (int, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE user_id = ?`
	args := []interface{}{userID}
	switch actionType {
	case "all":
		// 全部
	case "", "user":
		query += " AND action IN ('POST','PUT','PATCH','DELETE')"
	default:
		query += " AND action = ?"
		args = append(args, actionType)
	}
	if startDate != "" {
		query += " AND created_at >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND created_at <= ?"
		args = append(args, endDate)
	}
	var n int
	err := r.db.QueryRow(query, args...).Scan(&n)
	return n, err
}

// DeleteByUser 删除用户自己的日志（按 id 或清空）
func (r *AuditRepo) DeleteByUser(userID int64, id int64) (int64, error) {
	if id > 0 {
		res, err := r.db.Exec("DELETE FROM audit_logs WHERE user_id = ? AND id = ?", userID, id)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}
	res, err := r.db.Exec("DELETE FROM audit_logs WHERE user_id = ? AND action IN ('POST','PUT','PATCH','DELETE')", userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── 审计恢复快照 ──

const auditSnapshotCols = `id, audit_id, op_table, record_id, operation, before_json, after_json,
	restored, restored_at, restored_by, created_at`

// CreateSnapshot 记录可恢复操作快照
func (r *AuditRepo) CreateSnapshot(s *model.AuditSnapshot) error {
	_, err := r.db.Exec(`
		INSERT INTO audit_snapshots (audit_id, op_table, record_id, operation, before_json, after_json)
		VALUES (?, ?, ?, ?, ?, ?)`,
		s.AuditID, s.OpTable, s.RecordID, s.Operation, s.BeforeJSON, s.AfterJSON)
	return err
}

// GetSnapshotByAuditID 按审计 ID 查询快照（最新一条）
func (r *AuditRepo) GetSnapshotByAuditID(auditID int64) (*model.AuditSnapshot, error) {
	var s model.AuditSnapshot
	var restoredAt, restoredBy sql.NullString
	err := r.db.QueryRow(
		"SELECT "+auditSnapshotCols+" FROM audit_snapshots WHERE audit_id = ? ORDER BY id DESC LIMIT 1",
		auditID,
	).Scan(&s.ID, &s.AuditID, &s.OpTable, &s.RecordID, &s.Operation, &s.BeforeJSON,
		&s.AfterJSON, &s.Restored, &restoredAt, &restoredBy, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.RestoredAt = restoredAt.String
	s.RestoredBy = restoredBy.String
	return &s, nil
}

// GetSnapshotByID 按快照 ID 查询
func (r *AuditRepo) GetSnapshotByID(id int64) (*model.AuditSnapshot, error) {
	var s model.AuditSnapshot
	var restoredAt, restoredBy sql.NullString
	err := r.db.QueryRow(
		"SELECT "+auditSnapshotCols+" FROM audit_snapshots WHERE id = ?", id,
	).Scan(&s.ID, &s.AuditID, &s.OpTable, &s.RecordID, &s.Operation, &s.BeforeJSON,
		&s.AfterJSON, &s.Restored, &restoredAt, &restoredBy, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.RestoredAt = restoredAt.String
	s.RestoredBy = restoredBy.String
	return &s, nil
}

// ListSnapshots 列出未恢复的快照（管理端恢复列表）
func (r *AuditRepo) ListSnapshots(limit int) ([]*model.AuditSnapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(
		"SELECT "+auditSnapshotCols+" FROM audit_snapshots WHERE restored = 0 ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.AuditSnapshot
	for rows.Next() {
		s := &model.AuditSnapshot{}
		var restoredAt, restoredBy sql.NullString
		if err := rows.Scan(&s.ID, &s.AuditID, &s.OpTable, &s.RecordID, &s.Operation, &s.BeforeJSON,
			&s.AfterJSON, &s.Restored, &restoredAt, &restoredBy, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.RestoredAt = restoredAt.String
		s.RestoredBy = restoredBy.String
		list = append(list, s)
	}
	return list, rows.Err()
}

// MarkSnapshotRestored 标记快照已恢复
func (r *AuditRepo) MarkSnapshotRestored(snapshotID int64, by string) error {
	_, err := r.db.Exec(
		"UPDATE audit_snapshots SET restored = 1, restored_at = CURRENT_TIMESTAMP, restored_by = ? WHERE id = ?",
		by, snapshotID)
	return err
}
