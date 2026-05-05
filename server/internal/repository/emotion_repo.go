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

// GetByAlertID 根据告警 ID 查询
func (r *EmotionRepo) GetByAlertID(alertID string) (*model.EmotionLog, error) {
	a := &model.EmotionLog{}
	err := r.db.QueryRow(
		`SELECT e.id, e.user_id, e.username, e.session_id, e.alert_id,
		        e.message_text, e.score, e.risk_level, e.analysis_json,
		        e.notified, e.status, e.acknowledged_by,
		        COALESCE(e.acknowledged_at,''), e.created_at
		 FROM emotion_logs e
		 WHERE e.alert_id = ?`, alertID,
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

// UpdateStatus 更新告警状态
func (r *EmotionRepo) UpdateStatus(alertID string, status string, acknowledgedBy string) error {
	_, err := r.db.Exec(
		`UPDATE emotion_logs SET
		 status = ?,
		 acknowledged_by = CASE WHEN ? != '' THEN ? ELSE acknowledged_by END,
		 acknowledged_at = CASE WHEN ? != '' THEN datetime('now') ELSE acknowledged_at END
		 WHERE alert_id = ?`,
		status, acknowledgedBy, acknowledgedBy, acknowledgedBy, alertID,
	)
	if err != nil {
		return fmt.Errorf("更新告警状态失败: %w", err)
	}
	return nil
}
