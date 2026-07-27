package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// TokenStatsService 词元统计业务服务
// 同时负责 LLM 调用配额管理，防止滥用产生高额费用
type TokenStatsService struct {
	tokenRepo *repository.TokenUsageRepo
	userRepo  *repository.UserRepo

	// 配额管理（内存计数）
	quotaMu     sync.RWMutex
	dailyCounts map[int64]map[string]int // userID -> YYYY-MM-DD -> count
	monthlyCounts map[int64]map[string]int // userID -> YYYY-MM -> count
	dailyQuota   int // 每日上限，0 表示不限
	monthlyQuota int // 每月上限，0 表示不限
}

// NewTokenStatsService 创建词元统计服务
func NewTokenStatsService(tokenRepo *repository.TokenUsageRepo, userRepo *repository.UserRepo, dailyQuota, monthlyQuota int) *TokenStatsService {
	return &TokenStatsService{
		tokenRepo:     tokenRepo,
		userRepo:      userRepo,
		dailyCounts:   make(map[int64]map[string]int),
		monthlyCounts: make(map[int64]map[string]int),
		dailyQuota:    dailyQuota,
		monthlyQuota:  monthlyQuota,
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

// CheckAndIncrementQuota 检查调用配额并递增计数（真正调用 LLM 前调用）
// 返回: ok(是否通过), msg(超限提示信息)
// 缓存命中和 FAQ 命中不消耗配额，只有真正调用 LLM 时才检查
func (s *TokenStatsService) CheckAndIncrementQuota(userID int64) (bool, string) {
	if s == nil {
		return true, ""
	}
	if s.dailyQuota == 0 && s.monthlyQuota == 0 {
		return true, ""
	}

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	s.quotaMu.Lock()
	defer s.quotaMu.Unlock()

	// 日配额检查
	if s.dailyQuota > 0 {
		if s.dailyCounts[userID] == nil {
			s.dailyCounts[userID] = make(map[string]int)
		}
		if s.dailyCounts[userID][today] >= s.dailyQuota {
			return false, fmt.Sprintf("今日对话次数已达上限(%d次)，请明天再用或联系管理员提升配额", s.dailyQuota)
		}
	}

	// 月配额检查
	if s.monthlyQuota > 0 {
		if s.monthlyCounts[userID] == nil {
			s.monthlyCounts[userID] = make(map[string]int)
		}
		if s.monthlyCounts[userID][thisMonth] >= s.monthlyQuota {
			return false, fmt.Sprintf("本月对话次数已达上限(%d次)，请下月再用或联系管理员提升配额", s.monthlyQuota)
		}
	}

	// 递增计数
	if s.dailyQuota > 0 {
		s.dailyCounts[userID][today]++
	}
	if s.monthlyQuota > 0 {
		s.monthlyCounts[userID][thisMonth]++
	}

	return true, ""
}

// GetQuotaStats 获取用户当前配额使用情况
func (s *TokenStatsService) GetQuotaStats(userID int64) (dailyUsed, monthlyUsed, dailyQuota, monthlyQuota int) {
	if s == nil {
		return 0, 0, 0, 0
	}

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	s.quotaMu.RLock()
	defer s.quotaMu.RUnlock()

	if d, ok := s.dailyCounts[userID]; ok {
		dailyUsed = d[today]
	}
	if m, ok := s.monthlyCounts[userID]; ok {
		monthlyUsed = m[thisMonth]
	}

	return dailyUsed, monthlyUsed, s.dailyQuota, s.monthlyQuota
}

// CleanupQuotaCache 清理过期配额缓存数据（每天调用一次）
func (s *TokenStatsService) CleanupQuotaCache() {
	if s == nil {
		return
	}

	s.quotaMu.Lock()
	defer s.quotaMu.Unlock()

	cutoffDay := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	cutoffMonth := time.Now().AddDate(0, -2, 0).Format("2006-01")

	for userID, days := range s.dailyCounts {
		for day := range days {
			if day < cutoffDay {
				delete(days, day)
			}
		}
		if len(days) == 0 {
			delete(s.dailyCounts, userID)
		}
	}

	for userID, months := range s.monthlyCounts {
		for month := range months {
			if month < cutoffMonth {
				delete(months, month)
			}
		}
		if len(months) == 0 {
			delete(s.monthlyCounts, userID)
		}
	}
}
