package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
)

// EmotionRepo 情感预警数据访问
type EmotionRepo struct {
	db *sql.DB
}

// NewEmotionRepo 创建情感预警 repo
func NewEmotionRepo(db *sql.DB) *EmotionRepo {
	return &EmotionRepo{db: db}
}

// Create 创建情感评估记录
func (r *EmotionRepo) Create(log *model.EmotionLog) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO emotion_logs
		 (alert_id, user_id, username, session_id, message_text,
		  score, risk_level, analysis_json, notified, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.AlertID, log.UserID, log.Username, log.SessionID, log.MessageText,
		log.Score, log.RiskLevel, log.AnalysisJSON, log.Notified, log.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("创建情感记录失败: %w", err)
	}
	return result.LastInsertId()
}

// ListAlerts 分页查询告警（按角色过滤范围）
func (r *EmotionRepo) ListAlerts(riskLevel string, status string, ownerScope string, ownerID string, role string, page int, pageSize int) ([]*model.EmotionLog, int, error) {
	offset := (page - 1) * pageSize

	where := []string{"1=1"}
	args := []interface{}{}

	if riskLevel != "" {
		where = append(where, "e.risk_level = ?")
		args = append(args, riskLevel)
	}
	if status != "" {
		where = append(where, "e.status = ?")
		args = append(args, status)
	}

	if role == "counselor" || role == "college_admin" {
		where = append(where, "u.owner_scope = ?")
		args = append(args, ownerScope)
		if role == "counselor" && ownerID != "" {
			where = append(where, "u.owner_id = ?")
			args = append(args, ownerID)
		}
	}

	whereClause := strings.Join(where, " AND ")

	// 计数
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM emotion_logs e
		 JOIN users u ON e.user_id = u.id
		 WHERE %s`, whereClause)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计告警数量失败: %w", err)
	}

	// 查询列表
	query := fmt.Sprintf(
		`SELECT e.id, e.user_id, e.username, e.session_id, e.alert_id,
		        e.message_text, e.score, e.risk_level, e.analysis_json,
		        e.notified, e.status, e.acknowledged_by,
		        COALESCE(e.acknowledged_at,''), e.created_at
		 FROM emotion_logs e
		 JOIN users u ON e.user_id = u.id
		 WHERE %s
		 ORDER BY
		   CASE e.risk_level WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
		   e.created_at DESC
		 LIMIT ? OFFSET ?`, whereClause)
	queryArgs := append(args, pageSize, offset)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询告警列表失败: %w", err)
	}
	defer rows.Close()

	var alerts []*model.EmotionLog
	for rows.Next() {
		a := &model.EmotionLog{}
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Username, &a.SessionID, &a.AlertID,
			&a.MessageText, &a.Score, &a.RiskLevel, &a.AnalysisJSON,
			&a.Notified, &a.Status, &a.AcknowledgedBy, &a.AcknowledgedAt, &a.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, a)
	}

	return alerts, total, rows.Err()
}

// GetStats 获取告警统计（按角色过滤范围）
func (r *EmotionRepo) GetStats(ownerScope string, ownerID string, role string) (*model.EmotionStats, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if role == "counselor" || role == "college_admin" {
		where += " AND u.owner_scope = ?"
		args = append(args, ownerScope)
		if role == "counselor" && ownerID != "" {
			where += " AND u.owner_id = ?"
			args = append(args, ownerID)
		}
	}

	query := fmt.Sprintf(
		`SELECT
		  COALESCE(SUM(CASE WHEN e.status = 'pending' THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN e.risk_level = 'urgent' THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN e.risk_level = 'high' THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN e.risk_level = 'medium' THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN e.risk_level = 'low' THEN 1 ELSE 0 END), 0)
		FROM emotion_logs e
		JOIN users u ON e.user_id = u.id
		%s`, where)

	stats := &model.EmotionStats{}
	err := r.db.QueryRow(query, args...).Scan(
		&stats.Pending, &stats.Urgent, &stats.High, &stats.Medium, &stats.Low,
	)
	if err != nil {
		return nil, fmt.Errorf("统计告警失败: %w", err)
	}
	return stats, nil
}

// GetByAlertID 根据告警 ID 查询。
// 修复 GPT56SOL v3 P0-05：读路径增加资源范围过滤（JOIN users 按 ownerScope/ownerID/role），
// 防止跨学院越权读取原始敏感消息文本。role 为 school_admin/sys_admin 时不过滤。
func (r *EmotionRepo) GetByAlertID(alertID string, ownerScope, ownerID, role string) (*model.EmotionLog, error) {
	where := "e.alert_id = ?"
	args := []interface{}{alertID}
	if role == "counselor" || role == "college_admin" {
		where += " AND u.owner_scope = ?"
		args = append(args, ownerScope)
		if role == "counselor" && ownerID != "" {
			where += " AND u.owner_id = ?"
			args = append(args, ownerID)
		}
	}

	a := &model.EmotionLog{}
	err := r.db.QueryRow(
		`SELECT e.id, e.user_id, e.username, e.session_id, e.alert_id,
		        e.message_text, e.score, e.risk_level, e.analysis_json,
		        e.notified, e.status, e.acknowledged_by,
		        COALESCE(e.acknowledged_at,''), e.created_at
		 FROM emotion_logs e
		 JOIN users u ON e.user_id = u.id
		 WHERE `+where, args...,
	).Scan(
		&a.ID, &a.UserID, &a.Username, &a.SessionID, &a.AlertID,
		&a.MessageText, &a.Score, &a.RiskLevel, &a.AnalysisJSON,
		&a.Notified, &a.Status, &a.AcknowledgedBy, &a.AcknowledgedAt, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// GetTrends 获取情感趋势数据（按天聚合，含角色范围过滤）
func (r *EmotionRepo) GetTrends(days int, ownerScope, ownerID, role string) ([]*model.EmotionTrendPoint, error) {
	where := "WHERE e.created_at >= datetime('now', ?)"
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if role == "counselor" || role == "college_admin" {
		where += " AND u.owner_scope = ?"
		args = append(args, ownerScope)
		if role == "counselor" && ownerID != "" {
			where += " AND u.owner_id = ?"
			args = append(args, ownerID)
		}
	}

	query := fmt.Sprintf(
		`SELECT date(e.created_at) as date,
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN e.risk_level = 'urgent' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN e.risk_level = 'high' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN e.risk_level = 'medium' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN e.risk_level = 'low' THEN 1 ELSE 0 END), 0)
		FROM emotion_logs e
		JOIN users u ON e.user_id = u.id
		%s
		GROUP BY date(e.created_at)
		ORDER BY date(e.created_at) ASC`, where)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询趋势数据失败: %w", err)
	}
	defer rows.Close()

	var points []*model.EmotionTrendPoint
	for rows.Next() {
		p := &model.EmotionTrendPoint{}
		if err := rows.Scan(&p.Date, &p.Total, &p.Urgent, &p.High, &p.Medium, &p.Low); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

// UpdateStatus 更新告警状态。
// 修复 GPT56SOL v3 P0-05：写路径增加资源范围谓词（JOIN users 按 ownerScope/ownerID/role），
// 与读路径 ListAlerts/GetStats/GetTrends 对齐，防止跨学院越权修改他人学院告警。
// 返回受影响行数：0 表示无匹配（越权或不存在）。
func (r *EmotionRepo) UpdateStatus(alertID string, status string, acknowledgedBy string, ownerScope, ownerID, role string) (int64, error) {
	// 范围谓词：仅 counselor/college_admin 需按归属过滤；school_admin/sys_admin 不过滤
	whereScope := "1=1"
	scopeArgs := []interface{}{}
	if role == "counselor" || role == "college_admin" {
		whereScope += " AND u.owner_scope = ?"
		scopeArgs = append(scopeArgs, ownerScope)
		if role == "counselor" && ownerID != "" {
			whereScope += " AND u.owner_id = ?"
			scopeArgs = append(scopeArgs, ownerID)
		}
	}

	// 占位符顺序：status, ack×3, 外层 alert_id, 子查询 alert_id, 范围参数
	allArgs := append([]interface{}{status, acknowledgedBy, acknowledgedBy, acknowledgedBy, alertID, alertID}, scopeArgs...)
	result, err := r.db.Exec(
		`UPDATE emotion_logs SET
		 status = ?,
		 acknowledged_by = CASE WHEN ? != '' THEN ? ELSE acknowledged_by END,
		 acknowledged_at = CASE WHEN ? != '' THEN CURRENT_TIMESTAMP ELSE acknowledged_at END
		 WHERE alert_id = ? AND id IN (
			SELECT e.id FROM emotion_logs e JOIN users u ON e.user_id = u.id
			WHERE e.alert_id = ? AND `+whereScope+`
		 )`,
		allArgs...,
	)
	if err != nil {
		return 0, fmt.Errorf("更新告警状态失败: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// ListRecentByUser 读取某用户最近 N 条情感分析记录（按时间倒序），供个人心理健康报告使用。
// 仅返回该用户本人的数据，天然限定在 user_id，无越权风险。
func (r *EmotionRepo) ListRecentByUser(userID int64, limit int) ([]*model.EmotionLog, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, user_id, username, session_id, alert_id, message_text,
		        score, risk_level, analysis_json, notified, status,
		        acknowledged_by, acknowledged_at, created_at
		 FROM emotion_logs WHERE user_id = ?
		 ORDER BY created_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户情感记录失败: %w", err)
	}
	defer rows.Close()

	var logs []*model.EmotionLog
	for rows.Next() {
		l := &model.EmotionLog{}
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.SessionID, &l.AlertID,
			&l.MessageText, &l.Score, &l.RiskLevel, &l.AnalysisJSON, &l.Notified,
			&l.Status, &l.AcknowledgedBy, &l.AcknowledgedAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
