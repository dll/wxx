package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/temporal"
	"github.com/dll/wxx/server/internal/temporal/workflows"
	"github.com/google/uuid"
	sdkclient "go.temporal.io/sdk/client"
)

// answerCache 问答结果缓存（用于入学/离校等固定流程问题，避免重复调用 LLM）
// 缓存 24 小时，后台每 30 分钟清理过期条目
var (
	answerCache   = make(map[string]*answerCacheEntry)
	answerCacheMu sync.RWMutex
)

const answerCacheTTL = 24 * time.Hour

func init() {
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			answerCacheMu.Lock()
			now := time.Now()
			for k, v := range answerCache {
				if now.Sub(v.CachedAt) > answerCacheTTL {
					delete(answerCache, k)
				}
			}
			answerCacheMu.Unlock()
		}
	}()
}

type answerCacheEntry struct {
	Card      *model.AnswerCard
	SessionID string
	CachedAt  time.Time
}

// ChatService 问答业务服务（Context Engine 主链路）
type ChatService struct {
	sessionRepo    *repository.SessionRepo
	messageRepo    *repository.MessageRepo
	kbRepo         *repository.KBRepo
	agentRepo      *repository.AgentRepo
	llmClient      llm.ChatClient
	temporalClient *temporal.Client // 可选：Temporal 工作流客户端
}

// NewChatService 创建问答服务
func NewChatService(
	sessionRepo *repository.SessionRepo,
	messageRepo *repository.MessageRepo,
	kbRepo *repository.KBRepo,
	agentRepo *repository.AgentRepo,
	llmClient llm.ChatClient,
) *ChatService {
	return &ChatService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		kbRepo:      kbRepo,
		agentRepo:   agentRepo,
		llmClient:   llmClient,
	}
}

// SetTemporalClient 设置 Temporal 客户端（nil = 走直接调用路径）
func (s *ChatService) SetTemporalClient(tc *temporal.Client) {
	s.temporalClient = tc
}

// Ask 问答主链路
// 1. 创建/获取会话 → 2. 搜索知识库 → 3. 拼装上下文 → 4. 调 LLM → 5. 构造 AnswerCard
// 当 Temporal 已配置时，通过工作流引擎执行（获得重试/可观测性）
func (s *ChatService) Ask(ctx context.Context, userCtx *model.UserContext, sessionID string, question string, agentID string) (*model.AnswerCard, string, error) {
	traceID := uuid.New().String()

	// │ ❶ 缓存检查 ── 入学/离校等固定流程问题命中缓存即返回
	if agentID == "" && sessionID == "" {
		if card := s.cacheGet(question); card != nil {
			log.Printf("问答缓存命中 [trace=%s] question_hash=%s", traceID, cacheKeyForQuestion(question))
			return card, "", nil
		}
	}

	// 如果 Temporal 已启用，走工作流
	if s.temporalClient != nil {
		return s.askViaTemporal(ctx, userCtx, sessionID, question, agentID, traceID)
	}

	// ── 原有同步链路（不变）──

	// ── 1. 会话管理 ──
	if sessionID == "" {
		sessionID = uuid.New().String()
		err := s.sessionRepo.Create(&model.Session{
			SessionID: sessionID,
			UserID:    userCtx.UserID,
		})
		if err != nil {
			return nil, "", fmt.Errorf("创建会话失败: %w", err)
		}
	} else {
		// 验证会话属于当前用户
		session, err := s.sessionRepo.GetBySessionID(sessionID)
		if err != nil {
			return nil, "", fmt.Errorf("查询会话失败: %w", err)
		}
		if session == nil || session.UserID != userCtx.UserID {
			return nil, "", fmt.Errorf("会话不存在或无权访问")
		}
		_ = s.sessionRepo.Touch(sessionID)
	}

	// 保存用户消息
	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
		TraceID:   traceID,
	})

	// ── 2. FTS5/BM25 知识检索 ──
	searchResults, err := s.kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, 5)
	if err != nil {
		log.Printf("知识检索失败 [trace=%s]: %v", traceID, err)
		// 检索失败不中断链路，走兜底
	}

	// ── 3. 拼装 LLM 上下文 ──
	messages := s.buildMessages(ctx, sessionID, question, agentID, searchResults)

	// ── 4. 调 LLM ──
	llmResp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.3, // 问答场景用低温度，减少编造
		MaxTokens:   2048,
	})
	if err != nil {
		log.Printf("LLM 调用失败 [trace=%s]: %v", traceID, err)
		// 返回兜底回答，但保留搜索到的 sources
		return s.fallbackAnswerWithSources(traceID, question, searchResults), sessionID, nil
	}

	// ── 5. 构造 AnswerCard ──
	card := s.buildAnswerCard(llmResp.Content, searchResults, traceID)

	// 保存助手回复
	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResp.Content,
		TraceID:   traceID,
	})

	// │ 缓存写入 ── 入学/离校等固定流程问题缓存 24 小时
	s.cacheSet(question, sessionID, card)

	log.Printf("问答完成 [trace=%s] prompt_tokens=%d output_tokens=%d sources=%d",
		traceID, llmResp.PromptTokens, llmResp.OutputTokens, len(card.Sources))

	return card, sessionID, nil
}

// buildMessages 构造 LLM 消息列表
func (s *ChatService) buildMessages(ctx context.Context, sessionID string, question string, agentID string, results []*repository.SearchResult) []llm.ChatMessage {
	var messages []llm.ChatMessage

	// 查找智能体的自定义系统提示词
	systemPrompt := s.getSystemPrompt(agentID)

	// 拼接检索到的知识库内容
	if len(results) > 0 {
		var knowledgeBuilder strings.Builder
		knowledgeBuilder.WriteString("\n\n--- 知识库参考资料 ---\n")
		for i, r := range results {
			knowledgeBuilder.WriteString(fmt.Sprintf("\n【资料 %d】%s（%s）\n", i+1, r.Resource.Title, r.Resource.ResourceType))
			if r.Resource.Summary != "" {
				knowledgeBuilder.WriteString("摘要：" + r.Resource.Summary + "\n")
			}
			knowledgeBuilder.WriteString("内容：" + truncateContent(r.Resource.Content, 1500) + "\n")
		}
		systemPrompt += knowledgeBuilder.String()
	}

	messages = append(messages, llm.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// 历史对话上下文（最近 6 条）
	history, _ := s.messageRepo.GetRecentContext(sessionID, 6)
	for _, h := range history {
		messages = append(messages, llm.ChatMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	// 当前用户问题
	messages = append(messages, llm.ChatMessage{
		Role:    "user",
		Content: question,
	})

	return messages
}

// getSystemPrompt 获取智能体的系统提示词，未指定或查找失败时返回默认提示词
func (s *ChatService) getSystemPrompt(agentID string) string {
	// 如果指定了智能体，尝试查找
	if agentID != "" && s.agentRepo != nil {
		agent, err := s.agentRepo.GetByAgentID(agentID)
		if err == nil && agent != nil && agent.Status == "active" && agent.SystemPrompt != "" {
			return agent.SystemPrompt
		}
	}

	// 默认系统提示词
	return `你是"蔚小芯"，一个高校智慧学工 AI 助手。请严格基于以下知识库内容回答用户问题。

规则：
1. 只使用知识库中的信息回答，不要编造内容
2. 涉及政策、条件、数字时必须原文引用
3. 如果知识库中没有相关内容，明确告知用户你无法回答并建议联系辅导员
4. 回答要简洁、准确、有条理
5. 如果涉及流程，按步骤列出`
}

// buildAnswerCard 从 LLM 回复和检索结果构造 AnswerCard
func (s *ChatService) buildAnswerCard(content string, results []*repository.SearchResult, traceID string) *model.AnswerCard {
	card := &model.AnswerCard{
		Conclusion: content,
		TraceID:    traceID,
		Confidence: 0.8, // 默认置信度
		Fallback:   false,
	}

	// 附加来源引用
	for _, r := range results {
		card.Sources = append(card.Sources, model.Source{
			ResourceID:     r.Resource.ResourceID,
			Title:          r.Resource.Title,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: -r.Score, // BM25 分数取反（原始为负值）
		})
	}

	// 无知识命中时降低置信度
	if len(results) == 0 {
		card.Confidence = 0.3
		card.Fallback = true
	}

	// 生成追问建议
	card.FollowUps = generateFollowUps(content)

	return card
}

// fallbackAnswer 构造兜底回答
func (s *ChatService) fallbackAnswer(traceID string, question string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: "抱歉，我暂时无法回答您的问题。建议您联系辅导员或学工办公室获取帮助。",
		TraceID:    traceID,
		Confidence: 0.0,
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
		},
	}
}

// fallbackAnswerWithSources 构造兜底回答（保留搜索到的 sources）
func (s *ChatService) fallbackAnswerWithSources(traceID string, question string, results []*repository.SearchResult) *model.AnswerCard {
	card := &model.AnswerCard{
		Conclusion: "抱歉，AI 服务暂时不可用，但我为您找到了以下相关资料，请查阅：",
		TraceID:    traceID,
		Confidence: 0.5, // 有搜索结果，置信度提高
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
		},
	}

	// 附加搜索到的来源
	for _, r := range results {
		card.Sources = append(card.Sources, model.Source{
			ResourceID:     r.Resource.ResourceID,
			Title:          r.Resource.Title,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: -r.Score,
		})
	}

	return card
}

// generateFollowUps 根据回答内容生成追问建议
func generateFollowUps(content string) []string {
	var followUps []string

	// 简单的关键词匹配生成追问（后续可用 LLM 增强）
	if strings.Contains(content, "申请") {
		followUps = append(followUps, "申请需要哪些材料？")
	}
	if strings.Contains(content, "流程") || strings.Contains(content, "步骤") {
		followUps = append(followUps, "具体的办理地点在哪里？")
	}
	if strings.Contains(content, "截止") || strings.Contains(content, "日期") {
		followUps = append(followUps, "如果错过截止日期怎么办？")
	}

	return followUps
}

// truncateContent 截断内容，保留前 maxLen 个字符
func truncateContent(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// MarshalAnswerCard 将 AnswerCard 序列化为 JSON 字符串（用于审计日志等）
func MarshalAnswerCard(card *model.AnswerCard) string {
	b, err := json.Marshal(card)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// cacheKeyForQuestion 为问题生成缓存键（去空格 + 小写后取 FNV-1a 64-bit 哈希）
func cacheKeyForQuestion(q string) string {
	normalized := strings.ToLower(strings.TrimSpace(q))
	h := fnv.New64a()
	h.Write([]byte(normalized))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// cacheGet 从缓存读取（仅限无 agentID 且无 sessionID 的固定流程问题）
func (s *ChatService) cacheGet(question string) *model.AnswerCard {
	cacheKey := cacheKeyForQuestion(question)
	answerCacheMu.RLock()
	entry, ok := answerCache[cacheKey]
	answerCacheMu.RUnlock()
	if !ok {
		return nil
	}
	if time.Since(entry.CachedAt) > answerCacheTTL {
		return nil
	}
	return entry.Card
}

// cacheSet 写入缓存（仅限无 agentID 的固定流程问题）
func (s *ChatService) cacheSet(question, sessionID string, card *model.AnswerCard) {
	if sessionID != "" {
		return
	}
	cacheKey := cacheKeyForQuestion(question)
	answerCacheMu.Lock()
	if _, exists := answerCache[cacheKey]; !exists {
		answerCache[cacheKey] = &answerCacheEntry{
			Card:      card,
			SessionID: sessionID,
			CachedAt:  time.Now(),
		}
	}
	answerCacheMu.Unlock()
}

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

// askDirectImpl 直接调用链路的实现（原 Ask() 方法的核心逻辑）
func (s *ChatService) askDirectImpl(ctx context.Context, userCtx *model.UserContext, sessionID, question, agentID, traceID string) (*model.AnswerCard, string, error) {
	// ── 1. 会话管理 ──
	if sessionID == "" {
		sessionID = uuid.New().String()
		err := s.sessionRepo.Create(&model.Session{
			SessionID: sessionID,
			UserID:    userCtx.UserID,
		})
		if err != nil {
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
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
		TraceID:   traceID,
	})

	// ── 2. FTS5 知识检索 ──
	searchResults, err := s.kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, 5)
	if err != nil {
		log.Printf("知识检索失败 [trace=%s]: %v", traceID, err)
	}

	// ── 3. 拼装 LLM 上下文 ──
	messages := s.buildMessages(ctx, sessionID, question, agentID, searchResults)

	// ── 4. 调 LLM ──
	llmResp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2048,
	})
	if err != nil {
		log.Printf("LLM 调用失败 [trace=%s]: %v", traceID, err)
		return s.fallbackAnswerWithSources(traceID, question, searchResults), sessionID, nil
	}

	// ── 5. 构造 AnswerCard ──
	card := s.buildAnswerCard(llmResp.Content, searchResults, traceID)

	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResp.Content,
		TraceID:   traceID,
	})

	// │ 缓存写入 ── 入学/离校等固定流程问题缓存
	s.cacheSet(question, sessionID, card)

	log.Printf("问答完成 [trace=%s] prompt_tokens=%d output_tokens=%d sources=%d",
		traceID, llmResp.PromptTokens, llmResp.OutputTokens, len(card.Sources))

	return card, sessionID, nil
}
