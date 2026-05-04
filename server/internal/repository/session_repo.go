package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// SessionRepo 会话数据访问
type SessionRepo struct {
	db *sql.DB
}

// NewSessionRepo 创建会话 repo
func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create 创建新会话
func (r *SessionRepo) Create(session *model.Session) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (session_id, user_id) VALUES (?, ?)`,
		session.SessionID, session.UserID,
	)
	return err
}

// GetBySessionID 根据会话 ID 查询
func (r *SessionRepo) GetBySessionID(sessionID string) (*model.Session, error) {
	s := &model.Session{}
	err := r.db.QueryRow(
		`SELECT id, session_id, user_id, created_at, updated_at
		 FROM sessions WHERE session_id = ?`, sessionID,
	).Scan(&s.ID, &s.SessionID, &s.UserID, &s.CreatedAt, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListByUserID 查询用户的会话列表（按更新时间倒序）
func (r *SessionRepo) ListByUserID(userID int64, limit int) ([]*model.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, user_id, created_at, updated_at
		 FROM sessions WHERE user_id = ?
		 ORDER BY updated_at DESC LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*model.Session
	for rows.Next() {
		s := &model.Session{}
		if err := rows.Scan(&s.ID, &s.SessionID, &s.UserID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Touch 更新会话的 updated_at 时间戳
func (r *SessionRepo) Touch(sessionID string) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET updated_at = datetime('now') WHERE session_id = ?`, sessionID,
	)
	return err
}
