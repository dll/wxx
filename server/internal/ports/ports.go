// Package ports 定义六边形架构的 Outbound Port 接口。
// Service 层依赖这些接口而非具体实现（repository 结构体），
// 使业务逻辑可脱离 SQLite / LLM 进行单元测试。
package ports

import (
	"context"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// ── 知识库端口 ──

// KBRepository 知识库访问端口（FTS5/BM25 全文检索 + CRUD）
type KBRepository interface {
	// Search 全文检索，返回 BM25 相关性排序结果
	Search(query string, ownerScope string, ownerID string, role string, limit int) ([]*repository.SearchResult, error)
	// SearchFAQ 仅在 FAQ 资源中检索（用于持久化问答缓存命中）
	SearchFAQ(query string, ownerScope string, ownerID string, role string, limit int) ([]*repository.SearchResult, error)
	// Upsert 幂等导入资源（按 resource_id + version）
	Upsert(kb *model.KBResource) (int64, string, error)
	// SetStatus 修改资源状态（如把 FAQ 标为 retired）
	SetStatus(resourceID string, status string) error
}

// ── 会话端口 ──

// SessionRepository 会话持久化端口
type SessionRepository interface {
	// Create 创建新会话
	Create(session *model.Session) error
	// GetBySessionID 按 sessionID 查询会话（返回 nil 表示不存在）
	GetBySessionID(sessionID string) (*model.Session, error)
	// Touch 更新会话活跃时间
	Touch(sessionID string) error
}

// ── 消息端口 ──

// MessageRepository 消息持久化端口
type MessageRepository interface {
	// Create 保存消息
	Create(message *model.Message) error
	// GetRecentContext 获取会话最近 N 条消息（用于 LLM 上下文）
	GetRecentContext(sessionID string, limit int) ([]*model.Message, error)
}

// ── 智能体端口 ──

// AgentRepository 智能体配置访问端口
type AgentRepository interface {
	// GetByAgentID 按 agentID 查询智能体配置
	GetByAgentID(agentID string) (*model.Agent, error)
}

// ── 多智能体编排端口 ──

// AgentOrchestrator 多智能体编排器端口
type AgentOrchestrator interface {
	// Execute 执行多智能体协同问答
	Execute(ctx context.Context, question string, userCtx *model.UserContext) (*agent.MergedResult, error)
}
