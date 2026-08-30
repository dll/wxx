// Package service LLM 网关（A1 AI 运行基座）。
//
// 职责：所有面向大模型的对话调用统一经此出口——
//  1. 模型路由：用户自备 Key/模型（ModelConfigService）优先，覆盖服务器默认；
//  2. 调用审计：每次调用落 llm_call_logs（trace_id 贯穿、延迟、用量、成败），
//     与 TokenStatsService 的配额计量互补；
//  3. 失败留痕：调用失败记录日志后原样返回错误，由上层走既有兜底链路。
package service

import (
	"context"
	"log"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// LLMGateway 统一 LLM 出口。
type LLMGateway struct {
	client   llm.ChatClient           // 服务器默认客户端（可含 FailoverClient 容灾）
	modelCfg *ModelConfigService      // 可选：用户级模型路由
	callLog  *repository.LLMCallLogRepo // 可选：调用审计落库
}

// NewLLMGateway 创建网关。client 为底层默认客户端；其余依赖可后续注入。
func NewLLMGateway(client llm.ChatClient) *LLMGateway {
	return &LLMGateway{client: client}
}

// SetModelConfigService 注入用户模型配置（路由依据）。
func (g *LLMGateway) SetModelConfigService(svc *ModelConfigService) { g.modelCfg = svc }

// SetCallLogRepo 注入调用审计仓库（nil = 仅控制台日志）。
func (g *LLMGateway) SetCallLogRepo(repo *repository.LLMCallLogRepo) { g.callLog = repo }

// resolveOverride 解析用户自定义模型配置（default_provider + Key + 模型名）。
// 返回 nil 表示使用服务器默认配置。与原 ChatService.resolveUserLLMOverrides 语义一致。
func (g *LLMGateway) resolveOverride(userID int64) *llm.ChatRequest {
	if g.modelCfg == nil {
		return nil
	}
	cfg, err := g.modelCfg.Get(userID)
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

// applyOverride 将用户覆盖写入请求，返回生效的模型名（用于审计标注）。
func (g *LLMGateway) applyOverride(userID int64, req *llm.ChatRequest) string {
	if override := g.resolveOverride(userID); override != nil {
		req.APIKey = override.APIKey
		req.Model = override.Model
		return override.Model
	}
	return g.modelName()
}

func (g *LLMGateway) modelName() string {
	if g.client == nil {
		return ""
	}
	return g.client.Model()
}

// Chat 同步对话统一出口：路由 → 调用 → 审计落库。
func (g *LLMGateway) Chat(ctx context.Context, userID int64, sessionID, traceID string, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	modelUsed := g.applyOverride(userID, req)
	start := time.Now()
	resp, err := g.client.Chat(ctx, req)
	g.record(userID, sessionID, traceID, modelUsed, resp, time.Since(start).Milliseconds(), err)
	return resp, err
}

// Stream 流式对话统一出口：路由 → 启动流 → 首包审计落库（用量由消费方统计）。
func (g *LLMGateway) Stream(ctx context.Context, userID int64, sessionID, traceID string, req *llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	modelUsed := g.applyOverride(userID, req)
	start := time.Now()
	ch, err := g.client.Stream(ctx, req)
	g.record(userID, sessionID, traceID, modelUsed, nil, time.Since(start).Milliseconds(), err)
	return ch, err
}

// record 调用审计落库（best-effort，失败记 [WARN] 不影响对话）。
func (g *LLMGateway) record(userID int64, sessionID, traceID, modelUsed string, resp *llm.ChatResponse, latencyMS int64, callErr error) {
	entry := &model.LLMCallLog{
		TraceID:   traceID,
		UserID:    userID,
		SessionID: sessionID,
		Provider:  g.providerName(),
		Model:     modelUsed,
		LatencyMS: latencyMS,
	}
	if callErr != nil {
		entry.Status = "error"
		entry.ErrorMsg = callErr.Error()
	} else {
		entry.Status = "ok"
		if resp != nil {
			entry.PromptTokens = resp.PromptTokens
			entry.OutputTokens = resp.OutputTokens
		}
	}
	if g.callLog != nil {
		if err := g.callLog.Insert(entry); err != nil {
			log.Printf("[WARN] LLM 调用日志落库失败 [trace=%s]: %v", traceID, err)
		}
	}
}

func (g *LLMGateway) providerName() string {
	if g.client == nil {
		return "none"
	}
	return g.client.Name()
}
