package service

import (
	"fmt"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// SessionService 会话业务服务
type SessionService struct {
	sessionRepo *repository.SessionRepo
	messageRepo *repository.MessageRepo
}

// NewSessionService 创建会话服务
func NewSessionService(sessionRepo *repository.SessionRepo, messageRepo *repository.MessageRepo) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

// ListSessions 查询用户的会话列表
func (s *SessionService) ListSessions(userID int64, limit int) ([]*model.Session, error) {
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认返回 20 条
	}
	return s.sessionRepo.ListByUserID(userID, limit)
}

// GetSessionMessages 查询会话的消息历史
func (s *SessionService) GetSessionMessages(userID int64, sessionID string, limit int) ([]*model.Message, error) {
	// 验证会话归属
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("无权访问该会话")
	}

	if limit <= 0 || limit > 500 {
		limit = 100 // 默认返回 100 条
	}

	return s.messageRepo.ListBySessionID(sessionID, limit)
}

// DeleteSession 删除会话（同时级联删除消息）
func (s *SessionService) DeleteSession(userID int64, sessionID string) error {
	// 验证会话归属
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("查询会话失败: %w", err)
	}
	if session == nil {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权操作该会话")
	}

	return s.sessionRepo.Delete(sessionID)
}

// RenameSession 重命名会话标题（仅本人可改）
func (s *SessionService) RenameSession(userID int64, sessionID, title string) error {
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("查询会话失败: %w", err)
	}
	if session == nil {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权操作该会话")
	}

	// 标题长度上限：60 字符（避免滥用）
	if l := len([]rune(title)); l > 60 {
		runes := []rune(title)
		title = string(runes[:60])
	}

	return s.sessionRepo.UpdateTitle(sessionID, title)
}
