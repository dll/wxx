package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
)

// AskStream 流式问答：检索/编排/提示词拼装与 Ask 语义一致，但通过 LLM 流式接口
// 逐块回调 onDelta，最终返回完整 AnswerCard。
// 说明：为避免改动核心同步链路，此处独立编排（与 Ask 共享 buildMessages/buildAnswerCard 等私有方法）。
func (s *ChatService) AskStream(ctx context.Context, userCtx *model.UserContext, sessionID string, question string, agentID string, onDelta func(string)) (*model.AnswerCard, string, error) {
	traceID := uuid.New().String()

	// ── 配额检查 ──
	if s.tokenStatsSvc != nil {
		if ok, msg := s.tokenStatsSvc.CheckAndIncrementQuota(userCtx.UserID); !ok {
			return s.buildQuotaExceededAnswer(traceID, msg), sessionID, nil
		}
	}

	// ── 会话管理 ──
	if sessionID == "" {
		sessionID = uuid.New().String()
		if err := s.sessionRepo.Create(&model.Session{
			SessionID: sessionID, UserID: userCtx.UserID, Title: defaultSessionTitle(question),
		}); err != nil {
			return nil, "", fmt.Errorf("创建会话失败: %w", err)
		}
	} else {
		session, err := s.sessionRepo.GetBySessionID(sessionID)
		if err != nil {
			return nil, "", fmt.Errorf("查询会话失败: %w", err)
		}
		if session == nil || session.UserID != userCtx.UserID {
			return nil, "", fmt.Errorf("会话不存在或无权访问")
		}
		_ = s.sessionRepo.Touch(sessionID)
	}

	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID, Role: "user", Content: question, TraceID: traceID,
	})

	// ── 内容安全过滤（用户输入） ──
	if fr := util.CheckUserInput(question); fr.Action == util.FilterBlock {
		log.Printf("用户输入过滤拦截 [trace=%s] category=%s", traceID, fr.Category)
		return s.buildBlockedAnswer(traceID, fr.Category), sessionID, nil
	}

	// ── 多智能体协同 ──
	var multiAgentResult *agent.MergedResult
	if agentID == "" && s.orchestrator != nil {
		multiAgentResult, _ = s.orchestrator.Execute(ctx, question, userCtx)
	}

	// ── Context Engine 统一检索：结构化优先 → FTS/BM25 兜底 → 过期/低相关过滤 ──
	searchResults := s.retrieveWithContextEngine(ctx, userCtx, question)

	hasAgentResult := multiAgentResult != nil && multiAgentResult.AgentCount > 0 && len(multiAgentResult.Sources) > 0
	if len(searchResults) == 0 && !hasAgentResult {
		log.Printf("检索结果为空，跳过 LLM [trace=%s]", traceID)
		return s.buildEmptyResultAnswer(traceID), sessionID, nil
	}

	// ── 提示词拼装 ──
	sanitizedQuestion := util.SanitizeForLLM(question, 2000)
	messages := s.buildMessages(ctx, sessionID, sanitizedQuestion, agentID, searchResults, multiAgentResult)
	if s.llmClient == nil {
		card := s.fallbackAnswerWithSources(traceID, question, searchResults)
		_ = s.messageRepo.Create(&model.Message{
			SessionID: sessionID, Role: "assistant", Content: card.Conclusion, TraceID: traceID,
		})
		return card, sessionID, nil
	}

	// ── 流式调用 LLM ──
	req := &llm.ChatRequest{
		Messages: messages, Temperature: 0.3, MaxTokens: 2048,
	}
	if override := s.resolveUserLLMOverrides(userCtx.UserID); override != nil {
		req.APIKey = override.APIKey
		req.Model = override.Model
	}
	chunks, err := s.llmClient.Stream(ctx, req)
	if err != nil {
		log.Printf("LLM 流式启动失败 [trace=%s]: %v", traceID, err)
		card := s.fallbackAnswerWithSources(traceID, question, searchResults)
		_ = s.messageRepo.Create(&model.Message{
			SessionID: sessionID, Role: "assistant", Content: card.Conclusion, TraceID: traceID,
		})
		return card, sessionID, nil
	}

	var full strings.Builder
	for chunk := range chunks {
		if chunk.Delta != "" {
			full.WriteString(chunk.Delta)
			if onDelta != nil {
				onDelta(chunk.Delta)
			}
		}
		if chunk.Done {
			if full.Len() == 0 && chunk.Content != "" {
				full.WriteString(chunk.Content)
				if onDelta != nil {
					onDelta(chunk.Content)
				}
			}
			break
		}
	}

	// LLM 返回内容脱敏 + 安全过滤
	llmContent := util.SanitizeLLMResponse(full.String())
	if fr := util.CheckLLMOutput(llmContent); fr.Action == util.FilterBlock {
		log.Printf("内容过滤拦截 [trace=%s] category=%s", traceID, fr.Category)
		return s.buildBlockedAnswer(traceID, fr.Category), sessionID, nil
	}

	card := s.buildAnswerCard(llmContent, searchResults, traceID, multiAgentResult)

	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID, Role: "assistant", Content: llmContent, TraceID: traceID,
	})
	if s.tokenStatsSvc != nil {
		s.tokenStatsSvc.RecordUsage(userCtx.UserID, sessionID, s.llmClient.Name(), 0, 0)
	}
	s.cacheSet(question, sessionID, card)

	log.Printf("流式问答完成 [trace=%s] chars=%d sources=%d", traceID, len(llmContent), len(card.Sources))
	return card, sessionID, nil
}
