package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// RetentionResult 记录一次数据保留清理的结果。
type RetentionResult struct {
	AuditLogsDeleted   int64 `json:"audit_logs_deleted"`
	SessionsDeleted    int64 `json:"sessions_deleted"`
	MessagesDeleted    int64 `json:"messages_deleted"`
	EmotionLogsDeleted int64 `json:"emotion_logs_deleted"`
	ExportLogsDeleted  int64 `json:"export_logs_deleted"`
}

// RetentionService 负责按《蔚小芯智能体.md》9.2 的数据保留要求清理过期数据。
// 默认保留策略：审计/导出日志 180 天，会话/情感记录 1 学年（365 天）。
type RetentionService struct {
	db *sql.DB
}

func NewRetentionService(db *sql.DB) *RetentionService {
	return &RetentionService{db: db}
}

func (s *RetentionService) RunOnce(ctx context.Context, auditDays, sessionDays, emotionDays, exportDays int) (RetentionResult, error) {
	var result RetentionResult
	if s.db == nil {
		return result, fmt.Errorf("数据库未初始化")
	}
	if auditDays <= 0 {
		auditDays = 180
	}
	if sessionDays <= 0 {
		sessionDays = 365
	}
	if emotionDays <= 0 {
		emotionDays = 365
	}
	if exportDays <= 0 {
		exportDays = 180
	}

	// 会话过期时先删消息，再删会话。
	if err := s.deleteBefore(ctx, `DELETE FROM messages WHERE session_id IN (
		SELECT session_id FROM sessions WHERE updated_at < datetime('now', ?)
	)`, sessionDays); err != nil {
		return result, err
	}
	if n, err := s.deleteBeforeReturnCount(ctx, `DELETE FROM sessions WHERE updated_at < datetime('now', ?)`, sessionDays); err != nil {
		return result, err
	} else {
		result.SessionsDeleted = n
	}

	// 其余表按各自保留期清理。
	if n, err := s.deleteBeforeReturnCount(ctx, `DELETE FROM audit_logs WHERE created_at < datetime('now', ?)`, auditDays); err != nil {
		return result, err
	} else {
		result.AuditLogsDeleted = n
	}
	if n, err := s.deleteBeforeReturnCount(ctx, `DELETE FROM emotion_logs WHERE created_at < datetime('now', ?)`, emotionDays); err != nil {
		return result, err
	} else {
		result.EmotionLogsDeleted = n
	}
	if n, err := s.deleteBeforeReturnCount(ctx, `DELETE FROM export_logs WHERE created_at < datetime('now', ?)`, exportDays); err != nil {
		return result, err
	} else {
		result.ExportLogsDeleted = n
	}

	log.Printf("数据保留清理完成 audit=%d sessions=%d emotion=%d export=%d",
		result.AuditLogsDeleted, result.SessionsDeleted, result.EmotionLogsDeleted, result.ExportLogsDeleted)
	return result, nil
}

func (s *RetentionService) RunLoop(ctx context.Context, interval time.Duration, auditDays, sessionDays, emotionDays, exportDays int) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.RunOnce(ctx, auditDays, sessionDays, emotionDays, exportDays); err != nil {
				log.Printf("数据保留清理失败: %v", err)
			}
		}
	}
}

func (s *RetentionService) deleteBefore(ctx context.Context, query string, days int) error {
	_, err := s.db.ExecContext(ctx, query, fmt.Sprintf("-%d days", days))
	return err
}

func (s *RetentionService) deleteBeforeReturnCount(ctx context.Context, query string, days int) (int64, error) {
	res, err := s.db.ExecContext(ctx, query, fmt.Sprintf("-%d days", days))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
