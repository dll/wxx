package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// MessageRepo 消息数据访问
type MessageRepo struct {
	db *sql.DB
}

// NewMessageRepo 创建消息 repo
func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// Create 保存一条消息
func (r *MessageRepo) Create(msg *model.Message) error {
	_, err := r.db.Exec(
		`INSERT INTO messages (session_id, role, content, trace_id)
		 VALUES (?, ?, ?, ?)`,
		msg.SessionID, msg.Role, msg.Content, msg.TraceID,
	)
	return err
}

// ListBySessionID 查询会话的消息列表（按时间正序）
func (r *MessageRepo) ListBySessionID(sessionID string, limit int) ([]*model.Message, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, trace_id, created_at
		 FROM messages WHERE session_id = ?
		 ORDER BY id ASC LIMIT ?`, sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		m := &model.Message{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TraceID, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// GetRecentContext 获取会话的最近 N 条消息（用于构造 LLM 上下文）
func (r *MessageRepo) GetRecentContext(sessionID string, n int) ([]*model.Message, error) {
	// 先取最新 n 条（倒序），再正序返回
	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, trace_id, created_at
		 FROM messages WHERE session_id = ?
		 ORDER BY id DESC LIMIT ?`, sessionID, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		m := &model.Message{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TraceID, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 反转为正序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}
