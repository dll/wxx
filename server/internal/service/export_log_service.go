package service

import (
	"context"
	"fmt"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// ExportLogService 负责写入导出审计日志，满足 3.5.4 要求。
type ExportLogService struct {
	repo *repository.ExportLogRepo
}

func NewExportLogService(repo *repository.ExportLogRepo) *ExportLogService {
	return &ExportLogService{repo: repo}
}

func (s *ExportLogService) Log(ctx context.Context, userID int64, role, format, answerID, traceID string, hasSensitive bool) error {
	if s == nil || s.repo == nil {
		return nil
	}
	hasSensitiveInt := 0
	if hasSensitive {
		hasSensitiveInt = 1
	}
	err := s.repo.Insert(&model.ExportLog{
		UserID:       userID,
		Role:         role,
		Format:       format,
		AnswerID:     answerID,
		HasSensitive: hasSensitiveInt,
		TraceID:      traceID,
	})
	if err != nil {
		return fmt.Errorf("写入导出审计日志失败: %w", err)
	}
	return nil
}
