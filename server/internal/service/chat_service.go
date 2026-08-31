package service

import (
	"context"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/context_engine"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/ports"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/temporal"
	"github.com/google/uuid"
)

// ChatService 问答业务服务（Context Engine 主链路）
// 依赖 Outbound Port 接口，不直接依赖 SQLite / 具体实现。
type ChatService struct {
	sessionRepo    ports.SessionRepository
	messageRepo    ports.MessageRepository
	kbRepo         ports.KBRepository
	agentRepo      ports.AgentRepository
	llmClient      llm.ChatClient
	temporalClient *temporal.Client        // 可选：Temporal 工作流客户端
	orchestrator   ports.AgentOrchestrator // 多智能体编排器（agentID 为空时启用，可选注入）
	tokenStatsSvc  *TokenStatsService      // 可选：词元统计服务
	modelConfigSvc *ModelConfigService     // 可选：用户 AI 模型配置（默认 provider/Key/模型名覆盖）
	llmGateway     *LLMGateway             // A1：统一 LLM 出口（路由 + 调用审计）
	toolRegistry   *agent.ToolRegistry     // A3：function calling 工具注册表（可为 nil）
	contextEngine  *context_engine.Engine  // 统一知识检索管道
}

// NewChatService 创建问答服务（依赖通过 Outbound Port 接口注入）
func NewChatService(
	sessionRepo ports.SessionRepository,
	messageRepo ports.MessageRepository,
	kbRepo ports.KBRepository,
	agentRepo ports.AgentRepository,
	llmClient llm.ChatClient,
) *ChatService {
	return &ChatService{
		sessionRepo:   sessionRepo,
		messageRepo:   messageRepo,
		kbRepo:        kbRepo,
		agentRepo:     agentRepo,
		llmClient:     llmClient,
		llmGateway:    NewLLMGateway(llmClient),
		contextEngine: newProductionContextEngine(kbRepo, messageRepo),
	}
}

// SetModelConfigService 注入用户 AI 模型配置服务（可选）
func (s *ChatService) SetModelConfigService(svc *ModelConfigService) {
	s.modelConfigSvc = svc
	if s.llmGateway != nil {
		s.llmGateway.SetModelConfigService(svc)
	}
}

// SetLLMCallLogRepo 注入 LLM 调用审计仓库（A1；nil = 仅控制台日志）
func (s *ChatService) SetLLMCallLogRepo(repo *repository.LLMCallLogRepo) {
	if s.llmGateway != nil {
		s.llmGateway.SetCallLogRepo(repo)
	}
}

// chatViaGateway 同步对话统一出口（A1）：路由 + 调用 + 审计。
// 网关未初始化时（极端场景）退回直连语义，保证行为不劣化。
func (s *ChatService) chatViaGateway(ctx context.Context, userCtx *model.UserContext, sessionID, traceID string, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	if s.llmGateway != nil {
		return s.llmGateway.Chat(ctx, userCtx.UserID, sessionID, traceID, req)
	}
	if override := s.resolveUserLLMOverrides(userCtx.UserID); override != nil {
		req.APIKey = override.APIKey
		req.Model = override.Model
	}
	return s.llmClient.Chat(ctx, req)
}

// streamViaGateway 流式对话统一出口（A1）：路由 + 调用 + 首包审计。
func (s *ChatService) streamViaGateway(ctx context.Context, userCtx *model.UserContext, sessionID, traceID string, req *llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	if s.llmGateway != nil {
		return s.llmGateway.Stream(ctx, userCtx.UserID, sessionID, traceID, req)
	}
	if override := s.resolveUserLLMOverrides(userCtx.UserID); override != nil {
		req.APIKey = override.APIKey
		req.Model = override.Model
	}
	return s.llmClient.Stream(ctx, req)
}

// resolveUserLLMOverrides 解析用户自定义模型配置（default_provider + Key + 模型名 + 参数）。
// 返回 nil 表示使用服务器默认配置。用户配置的模型/Key 会覆盖服务器默认。
func (s *ChatService) resolveUserLLMOverrides(userID int64) *llm.ChatRequest {
	if s.llmGateway != nil {
		return s.llmGateway.resolveOverride(userID)
	}
	if s.modelConfigSvc == nil {
		return nil
	}
	cfg, err := s.modelConfigSvc.Get(userID)
	if err != nil || cfg == nil {
		return nil
	}
	// 仅当用户绑定了对应 provider 的 Key 时才生效（否则退回服务器额度与默认模型）
	switch cfg.DefaultProvider {
	case "deepseek":
		if cfg.DeepseekKey != "" {
			return &llm.ChatRequest{APIKey: cfg.DeepseekKey, Model: cfg.DeepseekModel}
		}
	case "zhipu":
		if cfg.ZhipuKey != "" {
			return &llm.ChatRequest{APIKey: cfg.ZhipuKey, Model: cfg.ZhipuModel}
		}
	}
	return nil
}

// SetOrchestrator 注入多智能体编排器（可选，nil = 不启用多 Agent 协同）
func (s *ChatService) SetOrchestrator(o ports.AgentOrchestrator) {
	s.orchestrator = o
}

// SetTemporalClient 设置 Temporal 客户端（nil = 走直接调用路径）
func (s *ChatService) SetTemporalClient(tc *temporal.Client) {
	s.temporalClient = tc
}

// SetTokenStatsService 设置词元统计服务（可选）
func (s *ChatService) SetTokenStatsService(svc *TokenStatsService) {
	s.tokenStatsSvc = svc
}

// Ask 问答主链路（编排层）
// 仅负责按固定顺序调用各阶段私有步骤函数：
// askCheckCaches → askCheckQuota → (Temporal 分支) → askSync（会话/过滤/编排/检索/拼装/LLM/卡片）
// 行为与被拆分前的单一 Ask 完全一致：执行顺序、缓存命中语义、配额计数、Temporal 分支、兜底分支均不变。
func (s *ChatService) Ask(ctx context.Context, userCtx *model.UserContext, sessionID string, question string, agentID string) (*model.AnswerCard, string, error) {
	traceID := uuid.New().String()

	// │ ❶ 缓存检查 ── 入学/离校等固定流程问题命中缓存即返回（内存缓存 + FAQ 持久化缓存）
	if card := s.askCheckCaches(agentID, sessionID, question, userCtx, traceID); card != nil {
		return card, "", nil
	}

	// │ ❸ 配额检查 ── 真正调用 LLM 前检查用户日/月配额
	if card, sid, done := s.askCheckQuota(userCtx, sessionID, traceID); done {
		return card, sid, nil
	}

	// │ ⓿ Temporal 分支 ── 若已启用则走工作流引擎（失败时降级到直接调用链路，行为不变）
	if s.temporalClient != nil {
		return s.askViaTemporal(ctx, userCtx, sessionID, question, agentID, traceID)
	}

	// ── 原有同步链路（行为不变，拆为 askSync 编排）──
	return s.askSync(ctx, userCtx, sessionID, question, agentID, traceID)
}
