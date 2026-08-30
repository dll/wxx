package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/temporal/workflows"
	"github.com/dll/wxx/server/internal/util"
	sdkclient "go.temporal.io/sdk/client"
)

// askViaTemporal 通过 Temporal 工作流引擎执行问答链路
// 工作流编排：验证会话 → 知识检索 → LLM 调用 → 构造 AnswerCard
func (s *ChatService) askViaTemporal(ctx context.Context, userCtx *model.UserContext, sessionID, question, agentID, traceID string) (*model.AnswerCard, string, error) {
	workflowOpts := sdkclient.StartWorkflowOptions{
		ID:                       "chat-" + traceID,
		TaskQueue:                s.temporalClient.TaskQueue(),
		WorkflowExecutionTimeout: 120 * time.Second,
	}

	input := workflows.ChatAskInput{
		UserID:     userCtx.UserID,
		Username:   userCtx.Username,
		Role:       userCtx.Role,
		OwnerScope: userCtx.OwnerScope,
		OwnerID:    userCtx.OwnerID,
		SessionID:  sessionID,
		Question:   question,
		AgentID:    agentID,
		TraceID:    traceID,
	}

	run, err := s.temporalClient.SDKClient().ExecuteWorkflow(ctx, workflowOpts, workflows.ChatAskWorkflow, input)
	if err != nil {
		log.Printf("启动问答工作流失败 [trace=%s]: %v", traceID, err)
		// 降级到同步直接调用
		return s.askDirect(ctx, userCtx, sessionID, question, agentID, traceID)
	}

	var output workflows.ChatAskOutput
	err = run.Get(ctx, &output)
	if err != nil {
		log.Printf("问答工作流执行失败 [trace=%s]: %v", traceID, err)
		return s.askDirect(ctx, userCtx, sessionID, question, agentID, traceID)
	}

	// 反序列化 AnswerCard
	var card model.AnswerCard
	if err := json.Unmarshal([]byte(output.AnswerCardJSON), &card); err != nil {
		log.Printf("反序列化 AnswerCard 失败 [trace=%s]: %v", traceID, err)
		return s.fallbackAnswer(traceID, question), sessionID, nil
	}

	log.Printf("问答工作流完成 [trace=%s] session=%s", traceID, output.SessionID)
	return &card, output.SessionID, nil
}

// askDirect 直接调用链路（Temporal 失败时的降级路径）
func (s *ChatService) askDirect(ctx context.Context, userCtx *model.UserContext, sessionID, question, agentID, traceID string) (*model.AnswerCard, string, error) {
	log.Printf("使用直接调用链路 [trace=%s]", traceID)
	// 复用原有同步逻辑（提取到 askDirect 中）
	return s.askDirectImpl(ctx, userCtx, sessionID, question, agentID, traceID)
}

// askDirectImpl 直接调用链路的实现（Temporal 降级路径，与 askSync 主链路共用会话/落库/过滤/检索逻辑）
func (s *ChatService) askDirectImpl(ctx context.Context, userCtx *model.UserContext, sessionID, question, agentID, traceID string) (*model.AnswerCard, string, error) {
	// ── 1. 会话管理（与 askSync 共用 ensureSession）──
	sessionID, err := s.ensureSession(userCtx, sessionID, question)
	if err != nil {
		return nil, "", err
	}

	// 保存用户消息（统一落库入口，失败记日志）
	s.saveUserMessage(sessionID, question, traceID)

	// │ 内容安全过滤 —— 用户输入检查
	if category, blocked := s.filterUserInput(question, traceID); blocked {
		return s.buildBlockedAnswer(traceID, category), sessionID, nil
	}

	// ── 2. Context Engine 统一检索（与主链路一致）──
	searchResults := s.retrieveWithContextEngine(ctx, userCtx, sessionID, question)

	// MED-KB2：无结果时跳过 LLM
	if len(searchResults) == 0 {
		log.Printf("检索结果为空，跳过 LLM [trace=%s]", traceID)
		return s.buildEmptyResultAnswer(traceID), sessionID, nil
	}

	// ── 3. 拼装 LLM 上下文 ──
	sanitizedQuestion := util.SanitizeForLLM(question, 2000)
	messages := s.buildMessages(ctx, sessionID, sanitizedQuestion, agentID, searchResults, nil)
	if s.llmClient == nil {
		card := s.fallbackAnswerWithSources(traceID, question, searchResults)
		s.saveAssistantMessage(sessionID, card.Conclusion, traceID)
		return card, sessionID, nil
	}

	// ── 4. 调 LLM（A1 网关统一出口）──
	req := &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2048,
	}
	llmResp, err := s.chatViaGateway(ctx, userCtx, sessionID, traceID, req)
	if err != nil {
		log.Printf("LLM 调用失败 [trace=%s]: %v", traceID, err)
		return s.fallbackAnswerWithSources(traceID, question, searchResults), sessionID, nil
	}

	// │ LLM 返回内容 PII 脱敏 + 内容安全过滤
	llmContent := util.SanitizeLLMResponse(llmResp.Content)

	if fr := util.CheckLLMOutput(llmContent); fr.Action == util.FilterBlock {
		log.Printf("内容过滤拦截 [trace=%s] category=%s reason=%s", traceID, fr.Category, fr.Reason)
		return s.buildBlockedAnswer(traceID, fr.Category), sessionID, nil
	}

	// ── 5. 构造 AnswerCard ──
	card := s.buildAnswerCard(llmContent, searchResults, traceID, nil)

	s.saveAssistantMessage(sessionID, llmResp.Content, traceID)

	// 记录词元使用
	if s.tokenStatsSvc != nil {
		s.tokenStatsSvc.RecordUsage(userCtx.UserID, sessionID, s.llmClient.Name(), llmResp.PromptTokens, llmResp.OutputTokens)
	}

	// │ 缓存写入 ── 入学/离校等固定流程问题缓存
	s.cacheSet(question, sessionID, card)

	log.Printf("问答完成 [trace=%s] prompt_tokens=%d output_tokens=%d sources=%d",
		traceID, llmResp.PromptTokens, llmResp.OutputTokens, len(card.Sources))

	return card, sessionID, nil
}
