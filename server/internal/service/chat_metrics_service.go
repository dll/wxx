package service

import "github.com/dll/wxx/server/internal/repository"

// ChatMetricsService 问答质量指标服务
type ChatMetricsService struct {
	repo *repository.ChatMetricsRepo
}

// NewChatMetricsService 创建问答质量指标服务
func NewChatMetricsService(repo *repository.ChatMetricsRepo) *ChatMetricsService {
	return &ChatMetricsService{repo: repo}
}

// Insert 写入一条问答质量指标
func (s *ChatMetricsService) Insert(sessionID string, userID int64, question, intent string, confidence float64, fallback bool, sourcesCount int, durationMs int64, traceID string) error {
	return s.repo.Insert(&repository.ChatMetric{
		SessionID:    sessionID,
		UserID:       userID,
		Question:     question,
		Intent:       intent,
		Confidence:   confidence,
		Fallback:     fallback,
		SourcesCount: sourcesCount,
		DurationMs:   durationMs,
		TraceID:      traceID,
	})
}

// Aggregate 聚合最近 sinceDays 天的质量指标
func (s *ChatMetricsService) Aggregate(sinceDays int) (*repository.AggregatedMetrics, error) {
	return s.repo.Aggregate(sinceDays)
}

// CountByIntent 统计最近 sinceDays 天各意图的调用次数
func (s *ChatMetricsService) CountByIntent(sinceDays int) ([]repository.IntentCount, error) {
	return s.repo.CountByIntent(sinceDays)
}
