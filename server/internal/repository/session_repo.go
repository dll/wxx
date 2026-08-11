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

// Create 创建新会话（含可选标题）
func (r *SessionRepo) Create(session *model.Session) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (session_id, user_id, title) VALUES (?, ?, ?)`,
		session.SessionID, session.UserID, session.Title,
	)
	return err
}

// GetBySessionID 根据会话 ID 查询
func (r *SessionRepo) GetBySessionID(sessionID string) (*model.Session, error) {
	s := &model.Session{}
	err := r.db.QueryRow(
		`SELECT id, session_id, user_id, title, created_at, updated_at
		 FROM sessions WHERE session_id = ?`, sessionID,
	).Scan(&s.ID, &s.SessionID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt)

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
		`SELECT id, session_id, user_id, title, created_at, updated_at
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
		if err := rows.Scan(&s.ID, &s.SessionID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Delete 删除会话（同时级联删除关联消息）
func (r *SessionRepo) Delete(sessionID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM messages WHERE session_id = ?`, sessionID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE session_id = ?`, sessionID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Touch 更新会话的 updated_at 时间戳
func (r *SessionRepo) Touch(sessionID string) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE session_id = ?`, sessionID,
	)
	return err
}

// UpdateTitle 修改会话标题（用户重命名 / 首条问题自动设置）
func (r *SessionRepo) UpdateTitle(sessionID, title string) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE session_id = ?`,
		title, sessionID,
	)
	return err
}
