package agent

import (
	"context"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// AgentResult 单个子智能体的执行结果
type AgentResult struct {
	Content    string         `json:"content"`
	Sources    []model.Source `json:"sources"`
	AgentName  string         `json:"agent_name"`
	Confidence float64        `json:"confidence"`
}

// Agent 子智能体接口
// 每个子 Agent 负责特定领域的问答，可独立演进
type Agent interface {
	// Key 返回 Agent 路由 Key（与 router.intentToAgent 返回值一致，如 "qa-default"）
	Key() string
	// Name 返回人类可读的智能体名称（用于日志与来源标注）
	Name() string
	// Execute 执行问答，返回 AgentResult
	Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error)
}
