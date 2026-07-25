// Package logger 结构化日志封装（A-03）
// 基于标准库 log/slog，支持 trace_id、user_id、path 等结构化字段。
// 生产环境输出 JSON 格式，开发环境输出可读文本。
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/dll/wxx/server/internal/middleware"
)

var defaultLogger *slog.Logger

func init() {
	// 根据环境选择 Handler
	env := os.Getenv("APP_ENV")
	var handler slog.Handler
	if env == "production" || env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// L 返回默认 logger
func L() *slog.Logger {
	return defaultLogger
}

// WithContext 从 context 中提取 trace_id 等字段，返回带上下文的 logger
func WithContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}
	traceID := middleware.GetTraceIDFromContext(ctx)
	if traceID != "" {
		return defaultLogger.With("trace_id", traceID)
	}
	return defaultLogger
}

// WithTraceID 使用指定 traceID 创建 logger
func WithTraceID(traceID string) *slog.Logger {
	if traceID == "" {
		return defaultLogger
	}
	return defaultLogger.With("trace_id", traceID)
}

// WithFields 创建带自定义字段的 logger
func WithFields(fields ...any) *slog.Logger {
	return defaultLogger.With(fields...)
}

// Info 信息日志
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn 警告日志
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error 错误日志
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Debug 调试日志
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}
