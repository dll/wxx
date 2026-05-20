package service

import (
	"log"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// TokenStatsService 词元统计业务服务
type TokenStatsService struct {
	tokenRepo *repository.TokenUsageRepo
	userRepo  *repository.UserRepo
}

// NewTokenStatsService 创建词元统计服务
func NewTokenStatsService(tokenRepo *repository.TokenUsageRepo, userRepo *repository.UserRepo) *TokenStatsService {
	return &TokenStatsService{
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
	}
}

// RecordUsage 记录一次词元使用
func (s *TokenStatsService) RecordUsage(userID int64, sessionID, modelProvider string, promptTokens, outputTokens int) {
	if promptTokens == 0 && outputTokens == 0 {
		return
	}
	err := s.tokenRepo.Create(&model.TokenUsage{
		UserID:        userID,
		SessionID:     sessionID,
		PromptTokens:  promptTokens,
		OutputTokens:  outputTokens,
		ModelProvider: modelProvider,
	})
	if err != nil {
		log.Printf("[TokenStats] 记录词元使用失败: user_id=%d err=%v", userID, err)
	}
}

// GetMyStats 获取当前用户的词元统计
func (s *TokenStatsService) GetMyStats(userID int64, days int) (*model.TokenStatsData, error) {
	if days <= 0 {
		days = 30
	}
	return s.tokenRepo.GetStatsByUserID(userID, days)
}

// GetSubordinateStats 获取下级用户的词元统计
func (s *TokenStatsService) GetSubordinateStats(userCtx *model.UserContext, days int) ([]model.SubordinateTokenStats, error) {
	if days <= 0 {
		days = 30
	}

	var userIDs []int64

	switch userCtx.Role {
	case "counselor":
		users, err := s.userRepo.List("student", userCtx.OwnerScope, userCtx.OwnerID, 0, 1000)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
	case "teacher":
		users, err := s.userRepo.List("student", userCtx.OwnerScope, userCtx.OwnerID, 0, 1000)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
	case "college_admin":
		users, err := s.userRepo.List("", userCtx.OwnerScope, userCtx.OwnerID, 0, 1000)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			if u.ID != userCtx.UserID {
				userIDs = append(userIDs, u.ID)
			}
		}
	case "school_admin", "sys_admin":
		users, err := s.userRepo.List("", "", "", 0, 1000)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			if u.ID != userCtx.UserID {
				userIDs = append(userIDs, u.ID)
			}
		}
	default:
		return nil, nil
	}

	return s.tokenRepo.GetSubordinateStats(userIDs, days)
}
