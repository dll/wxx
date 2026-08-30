// Package repository LLM 调用审计日志仓库（A1 网关）。
package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// LLMCallLogRepo llm_call_logs 表读写。
type LLMCallLogRepo struct {
	db *sql.DB
}

// NewLLMCallLogRepo 创建 LLM 调用日志仓库。
func NewLLMCallLogRepo(db *sql.DB) *LLMCallLogRepo {
	return &LLMCallLogRepo{db: db}
}

// Insert 写入一条调用日志。失败由调用方（网关）记 [WARN]，不影响对话主链路。
func (r *LLMCallLogRepo) Insert(log *model.LLMCallLog) error {
	_, err := r.db.Exec(
		`INSERT INTO llm_call_logs
		 (trace_id, user_id, session_id, provider, model, prompt_tokens, output_tokens, latency_ms, status, error_msg)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.TraceID, log.UserID, log.SessionID, log.Provider, log.Model,
		log.PromptTokens, log.OutputTokens, log.LatencyMS, log.Status, log.ErrorMsg,
	)
	return err
}
