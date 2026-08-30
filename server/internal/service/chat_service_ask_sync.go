package service

import (
	"context"
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
)

// askCheckCaches 缓存检查（步 ❶）：仅对固定流程类问题检查内存问答缓存，未命中再查 FAQ 持久化缓存。
// 返回非 nil 的 AnswerCard 表示命中，调用方应直接返回该结果。
func (s *ChatService) askCheckCaches(agentID, sessionID, question string, userCtx *model.UserContext, traceID string) *model.AnswerCard {
	if agentID == "" && sessionID == "" && isProcessCacheableQuestion(question) {
		if card := s.cacheGet(question); card != nil {
			log.Printf("问答缓存命中 [trace=%s] question_hash=%s", traceID, cacheKeyForQuestion(question))
			return card
		}
		// ❷ FAQ 持久化缓存 ── 在已有 FAQ 资源中按 BM25 搜索，命中即跳过 LLM
		if card := s.faqLookup(question, userCtx); card != nil {
			log.Printf("FAQ 命中 [trace=%s] question=%q", traceID, truncateContent(question, 60))
			return card
		}
	}
	return nil
}

// askCheckQuota 配额检查（步 ❸）：超限时返回兜底回答并标记 done；未超限或未启用配额服务时不做额外处理。
// 注意：配额自增（CheckAndIncrementQuota）语义与拆分前一致，仅在 tokenStatsSvc 非 nil 时触发。
func (s *ChatService) askCheckQuota(userCtx *model.UserContext, sessionID, traceID string) (*model.AnswerCard, string, bool) {
	if s.tokenStatsSvc == nil {
		return nil, sessionID, false
	}
	if ok, msg := s.tokenStatsSvc.CheckAndIncrementQuota(userCtx.UserID); !ok {
		log.Printf("配额超限 [user=%d] %s", userCtx.UserID, msg)
		return s.buildQuotaExceededAnswer(traceID, msg), sessionID, true
	}
	return nil, sessionID, false
}

// askSync 同步主链路编排：会话管理 → 保存用户消息 → 内容过滤 → 多智能体 → 检索 → 相关性预检
// → 空结果兜底 → 拼装/调 LLM/构造卡片。执行顺序与拆分前 Ask 内核完全一致。
func (s *ChatService) askSync(ctx context.Context, userCtx *model.UserContext, sessionID, question, agentID, traceID string) (*model.AnswerCard, string, error) {
	// ── 1. 会话管理（创建或校验归属 + 刷新）──
	sessionID, err := s.ensureSession(userCtx, sessionID, question)
	if err != nil {
		return nil, "", err
	}

	// 保存用户消息
	s.saveUserMessage(sessionID, question, traceID)

	// │ 内容安全过滤 —— 用户输入检查（必须在 LLM 调用和多智能体编排之前）
	if category, blocked := s.filterUserInput(question, traceID); blocked {
		return s.buildBlockedAnswer(traceID, category), sessionID, nil
	}

	// ── 2. 多智能体协同编排（agentID 为空时启用）──
	multiAgentResult := s.runAgents(ctx, userCtx, question, agentID, traceID)

	// ── 3. 结构化优先检索（MED-KB1）＋ 3.5 相关性预检 ──
	searchResults := s.retrieveWithContextEngine(ctx, userCtx, sessionID, question)

	// ── MED-KB2：检索 + 多智能体均无结果时，跳过 LLM 调用 ──
	hasAgentResult := multiAgentResult != nil && multiAgentResult.AgentCount > 0 && len(multiAgentResult.Sources) > 0
	if len(searchResults) == 0 && !hasAgentResult {
		log.Printf("检索结果为空且无多智能体结果，跳过 LLM [trace=%s]", traceID)
		return s.buildEmptyResultAnswer(traceID), sessionID, nil
	}

	// ── 4~6. 拼装上下文 → 调 LLM → 构造 AnswerCard（含兜底与缓存/FAQ 写入）──
	return s.askGenerateAndAssemble(ctx, userCtx, sessionID, question, agentID, searchResults, multiAgentResult, traceID)
}

// ensureSession 会话管理（步 1）：无会话则创建（默认标题取问题前 30 字符），有则校验归属并刷新。
func (s *ChatService) ensureSession(userCtx *model.UserContext, sessionID, question string) (string, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
		if err := s.sessionRepo.Create(&model.Session{
			SessionID: sessionID,
			UserID:    userCtx.UserID,
			Title:     defaultSessionTitle(question),
		}); err != nil {
			return "", fmt.Errorf("创建会话失败: %w", err)
		}
		return sessionID, nil
	}
	// 验证会话属于当前用户
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return "", fmt.Errorf("查询会话失败: %w", err)
	}
	if session == nil || session.UserID != userCtx.UserID {
		return "", fmt.Errorf("会话不存在或无权访问")
	}
	_ = s.sessionRepo.Touch(sessionID)
	return sessionID, nil
}

// saveUserMessage 将用户本轮问题落库为 user 角色消息（chat 链路统一入口）。
// 落库失败必须记日志：聊天记录属于用户数据，静默丢失会破坏会话完整性且无法排障。
func (s *ChatService) saveUserMessage(sessionID, question, traceID string) {
	if err := s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
		TraceID:   traceID,
	}); err != nil {
		log.Printf("[WARN] 保存用户消息失败 [trace=%s] session=%s: %v", traceID, sessionID, err)
	}
}

// saveAssistantMessage 将助手回复落库为 assistant 角色消息（chat 链路统一入口）。
func (s *ChatService) saveAssistantMessage(sessionID, content, traceID string) {
	if err := s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		TraceID:   traceID,
	}); err != nil {
		log.Printf("[WARN] 保存助手消息失败 [trace=%s] session=%s: %v", traceID, sessionID, err)
	}
}

// filterUserInput 用户输入内容安全过滤（必须在 LLM 调用与多智能体编排之前）。返回 true+分类表示需拦截。
func (s *ChatService) filterUserInput(question, traceID string) (string, bool) {
	if fr := util.CheckUserInput(question); fr.Action == util.FilterBlock {
		log.Printf("用户输入过滤拦截 [trace=%s] category=%s reason=%s", traceID, fr.Category, fr.Reason)
		return fr.Category, true
	}
	return "", false
}

// runAgents 多智能体协同编排（步 2）：agentID 为空且注入编排器时执行，否则返回 nil（行为一致）。
// 编排失败记录日志后返回 nil：主链路退化为纯检索问答，不中断对话，但错误可排障。
func (s *ChatService) runAgents(ctx context.Context, userCtx *model.UserContext, question, agentID, traceID string) *agent.MergedResult {
	if agentID == "" && s.orchestrator != nil {
		multiAgentResult, err := s.orchestrator.Execute(ctx, question, userCtx)
		if err != nil {
			log.Printf("[WARN] 多智能体编排失败 [trace=%s] user=%d: %v", traceID, userCtx.UserID, err)
		}
		return multiAgentResult
	}
	return nil
}

// retrieveWithRelevance 检索（步 3）＋ 相关性预检（步 3.5）：结构化优先（≥3 条跳过 FTS），否则并入 FTS
// 结果并去重，最后过滤低相关性结果（MED-KB1 + CE-02）。
func (s *ChatService) retrieveWithRelevance(userCtx *model.UserContext, question, traceID string) []*repository.SearchResult {
	// 结构化优先检索（MED-KB1：先查结构化字段，再回退 FTS）
	structuredResults, err := s.kbRepo.SearchStructured(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, 5)
	if err != nil {
		log.Printf("结构化检索失败 [trace=%s]: %v", traceID, err)
	}
	log.Printf("结构化检索 [trace=%s] count=%d", traceID, len(structuredResults))

	var searchResults []*repository.SearchResult
	// 阈值：结构化命中 ≥ 3 条时跳过 FTS
	if len(structuredResults) >= 3 {
		searchResults = structuredResults
		log.Printf("结构化结果充足，跳过 FTS/BM25 [trace=%s]", traceID)
	} else {
		ftsResults, ftsErr := s.kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, 5)
		if ftsErr != nil {
			log.Printf("FTS/BM25 检索失败 [trace=%s]: %v", traceID, ftsErr)
		}
		// 合并：结构化在前，FTS 在后，按 ResourceID 去重
		searchResults = mergeStructuredAndFTS(structuredResults, ftsResults)
	}

	// ── 3.5 相关性预检 ──
	// 对检索结果做相关性打分，过滤掉低质量结果，避免误导 LLM
	return filterLowRelevanceResults(searchResults, question)
}

// hasProcessResult 判断检索结果中是否存在流程类（Process）资源，用于决定是否跳过 FAQ 持久化缓存写入。
func hasProcessResult(results []*repository.SearchResult) bool {
	for _, r := range results {
		if r.Resource.ResourceType == "Process" {
			return true
		}
	}
	return false
}
