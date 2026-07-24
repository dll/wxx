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

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/ports"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/temporal"
	"github.com/dll/wxx/server/internal/temporal/workflows"
	"github.com/dll/wxx/server/internal/util"
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
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		kbRepo:      kbRepo,
		agentRepo:   agentRepo,
		llmClient:   llmClient,
	}
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

// Ask 问答主链路
// 1. 创建/获取会话 → 2. 搜索知识库 → 3. 拼装上下文 → 4. 调 LLM → 5. 构造 AnswerCard
// 当 Temporal 已配置时，通过工作流引擎执行（获得重试/可观测性）
func (s *ChatService) Ask(ctx context.Context, userCtx *model.UserContext, sessionID string, question string, agentID string) (*model.AnswerCard, string, error) {
	traceID := uuid.New().String()

	// │ ❶ 缓存检查 ── 入学/离校等固定流程问题命中缓存即返回
	if agentID == "" && sessionID == "" && isProcessCacheableQuestion(question) {
		if card := s.cacheGet(question); card != nil {
			log.Printf("问答缓存命中 [trace=%s] question_hash=%s", traceID, cacheKeyForQuestion(question))
			return card, "", nil
		}
		// │ ❷ FAQ 持久化缓存 ── 在已有 FAQ 资源中按 BM25 搜索，命中即跳过 LLM
		if card := s.faqLookup(question, userCtx); card != nil {
			log.Printf("FAQ 命中 [trace=%s] question=%q", traceID, truncateContent(question, 60))
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
			Title:     defaultSessionTitle(question),
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

	// ── 2. 多智能体协同编排（agentID 为空时启用）──
	var multiAgentResult *agent.MergedResult
	if agentID == "" && s.orchestrator != nil {
		multiAgentResult, _ = s.orchestrator.Execute(ctx, question, userCtx)
	}

	// │ 内容安全过滤 —— 用户输入检查
	if fr := util.CheckUserInput(question); fr.Action == util.FilterBlock {
		log.Printf("用户输入过滤拦截 [trace=%s] category=%s reason=%s", traceID, fr.Category, fr.Reason)
		return s.buildBlockedAnswer(traceID, fr.Category), sessionID, nil
	}

	// ── 3. FTS5/BM25 知识检索 ──
	searchResults, err := s.kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, 5)
	if err != nil {
		log.Printf("知识检索失败 [trace=%s]: %v", traceID, err)
	}

	// ── 3.5 相关性预检 ──
	// 对检索结果做相关性打分，过滤掉低质量结果，避免误导 LLM
	searchResults = filterLowRelevanceResults(searchResults, question)

	// ── 4. 拼装 LLM 上下文 ──
	// 发送给 LLM 前对用户问题进行 PII 脱敏
	sanitizedQuestion := util.SanitizeForLLM(question, 2000)
	messages := s.buildMessages(ctx, sessionID, sanitizedQuestion, agentID, searchResults, multiAgentResult)
	if s.llmClient == nil {
		card := s.fallbackAnswerWithSources(traceID, question, searchResults)
		_ = s.messageRepo.Create(&model.Message{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   card.Conclusion,
			TraceID:   traceID,
		})
		return card, sessionID, nil
	}

	// ── 5. 调 LLM ──
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

	// │ LLM 返回内容 PII 脱敏 —— 防止模型幻觉输出真实 PII
	llmContent := util.SanitizeLLMResponse(llmResp.Content)

	// │ 内容安全过滤 ── LLM 返回内容检查
	if fr := util.CheckLLMOutput(llmContent); fr.Action == util.FilterBlock {
		log.Printf("内容过滤拦截 [trace=%s] category=%s reason=%s", traceID, fr.Category, fr.Reason)
		return s.buildBlockedAnswer(traceID, fr.Category), sessionID, nil
	}

	// ── 6. 构造 AnswerCard ──
	card := s.buildAnswerCard(llmContent, searchResults, traceID, multiAgentResult)

	// 保存助手回复
	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResp.Content,
		TraceID:   traceID,
	})

	// 记录词元使用
	if s.tokenStatsSvc != nil {
		s.tokenStatsSvc.RecordUsage(userCtx.UserID, sessionID, s.llmClient.Name(), llmResp.PromptTokens, llmResp.OutputTokens)
	}

	// │ 缓存写入 ── 入学/离校等固定流程问题缓存 24 小时
	s.cacheSet(question, sessionID, card)

	// │ FAQ 持久化缓存写入 ── 仅在有引用且非 agent/多智能体路径时入库
	// 排除流程类问题：流程类由结构化端点保证确定性，缓存会绕过最新 process_steps 数据
	hasProcessResult := false
	for _, r := range searchResults {
		if r.Resource.ResourceType == "Process" {
			hasProcessResult = true
			break
		}
	}
	if agentID == "" && multiAgentResult == nil && len(card.Sources) > 0 && !hasProcessResult {
		go s.faqStore(question, card, userCtx.Role)
	}

	log.Printf("问答完成 [trace=%s] prompt_tokens=%d output_tokens=%d sources=%d",
		traceID, llmResp.PromptTokens, llmResp.OutputTokens, len(card.Sources))

	return card, sessionID, nil
}
func (s *ChatService) buildMessages(ctx context.Context, sessionID string, question string, agentID string, results []*repository.SearchResult, multiAgentResult *agent.MergedResult) []llm.ChatMessage {
	var messages []llm.ChatMessage

	// 查找智能体的自定义系统提示词
	systemPrompt := s.getSystemPrompt(agentID)

	// 拼接多智能体协同结果
	if multiAgentResult != nil && multiAgentResult.AgentCount > 0 {
		systemPrompt += fmt.Sprintf("\n\n--- 多智能体协同分析（%d 个 Agent 参与）---\n%s",
			multiAgentResult.AgentCount, multiAgentResult.Content)
	}

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

	// 默认系统提示词（强精准约束版）
	return `你是"蔚小芯"，一个高校智慧学工 AI 助手。你必须严格基于知识库中与用户问题【直接相关】的内容回答。

【核心规则——违反任何一条都是严重错误】
1. 只回答知识库中【明确存在且直接相关】的内容。绝对不能根据不相关的资料进行推测、联想或编造。
2. 判断相关性的标准：知识库资料的标题、摘要或核心内容必须与用户问题的主题高度一致。
3. 如果检索到的资料与用户问题不相关（例如问"请假"但资料是"入党"），视为未找到相关信息。
4. 如果没有足够相关的知识库内容，必须明确说"知识库中暂未找到相关信息"，并建议联系辅导员或相关部门确认。
5. 绝不能因为某个字相同就把不相关的内容当作答案。例如问"请假流程"不能用"入党流程"回答。
6. 涉及政策、条件、数字、时间时必须原文引用，不能含糊。
7. 回答要简洁、准确、有条理；流程类按步骤列出。

【请记住】：回答错误比不回答更糟糕。不确定就说不知道。`
}

// buildAnswerCard 从 LLM 回复和检索结果构造 AnswerCard
func (s *ChatService) buildAnswerCard(content string, results []*repository.SearchResult, traceID string, multiAgentResult *agent.MergedResult) *model.AnswerCard {
	card := &model.AnswerCard{
		Conclusion: content,
		TraceID:    traceID,
		Confidence: 0.8, // 默认置信度
		Fallback:   false,
	}

	// 附加来源引用（含多智能体来源）
	sourceSet := make(map[string]bool)
	for _, r := range results {
		key := r.Resource.ResourceID + r.Resource.Version
		if sourceSet[key] {
			continue
		}
		sourceSet[key] = true
		card.Sources = append(card.Sources, model.Source{
			ResourceID:     r.Resource.ResourceID,
			Title:          r.Resource.Title,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: -r.Score,
			EffectiveAt:    r.Resource.EffectiveAt,
			Snippet:        r.Resource.Summary,
		})
	}
	// 合并多智能体来源（去重）
	if multiAgentResult != nil {
		for _, s := range multiAgentResult.Sources {
			key := s.ResourceID + s.Version
			if sourceSet[key] {
				continue
			}
			sourceSet[key] = true
			card.Sources = append(card.Sources, s)
		}
	}

	// 无知识命中时降低置信度
	if len(results) == 0 {
		card.Confidence = 0.3
		card.Fallback = true
		// 如果多智能体也没有来源，替换结论为兜底引导文案（避免 LLM 无依据编造）
		if multiAgentResult == nil || len(multiAgentResult.Sources) == 0 {
			card.Conclusion = `知识库中暂未找到足够信息。建议联系辅导员、学院学工办公室或相关职能部门确认最新要求。`
		}
	}

	// 生成追问建议
	card.FollowUps = generateFollowUps(content)

	return card
}

// buildBlockedAnswer 内容过滤拦截时返回的兜底回答
func (s *ChatService) buildBlockedAnswer(traceID string, category string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: util.GetFallbackResponse(category),
		TraceID:    traceID,
		Confidence: 0.0,
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
		},
	}
}

// fallbackAnswer 构造兜底回答
func (s *ChatService) fallbackAnswer(traceID string, question string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: "知识库中暂未找到足够信息。建议联系辅导员、学院学工办公室或相关职能部门确认。",
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
	conclusion := "知识库中暂未找到足够信息。建议联系辅导员或学工办公室确认。"
	confidence := 0.3
	if len(results) > 0 {
		var b strings.Builder
		b.WriteString("我已根据知识库资料为您整理如下：\n\n")
		for i, r := range results {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Resource.Title))
			if r.Resource.Summary != "" {
				b.WriteString(r.Resource.Summary)
			} else {
				b.WriteString(truncateContent(r.Resource.Content, 260))
			}
			b.WriteString("\n\n")
		}
		conclusion = strings.TrimSpace(b.String())
		confidence = 0.5
	}
	card := &model.AnswerCard{
		Conclusion: conclusion,
		TraceID:    traceID,
		Confidence: confidence,
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

func isProcessCacheableQuestion(question string) bool {
	q := strings.TrimSpace(question)
	if q == "" {
		return false
	}
	keywords := []string{"入学", "迎新", "报到", "离校", "毕业", "转专业", "助学贷款", "流程"}
	for _, keyword := range keywords {
		if strings.Contains(q, keyword) {
			return true
		}
	}
	return false
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
			Title:     defaultSessionTitle(question),
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
	sanitizedQuestion := util.SanitizeForLLM(question, 2000)
	messages := s.buildMessages(ctx, sessionID, sanitizedQuestion, agentID, searchResults, nil)
	if s.llmClient == nil {
		card := s.fallbackAnswerWithSources(traceID, question, searchResults)
		_ = s.messageRepo.Create(&model.Message{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   card.Conclusion,
			TraceID:   traceID,
		})
		return card, sessionID, nil
	}

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

	// │ LLM 返回内容 PII 脱敏 + 内容安全过滤
	llmContent := util.SanitizeLLMResponse(llmResp.Content)

	if fr := util.CheckLLMOutput(llmContent); fr.Action == util.FilterBlock {
		log.Printf("内容过滤拦截 [trace=%s] category=%s reason=%s", traceID, fr.Category, fr.Reason)
		return s.buildBlockedAnswer(traceID, fr.Category), sessionID, nil
	}

	// ── 5. 构造 AnswerCard ──
	card := s.buildAnswerCard(llmContent, searchResults, traceID, nil)

	_ = s.messageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResp.Content,
		TraceID:   traceID,
	})

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

// defaultSessionTitle 用首条问题前 30 个字符作为会话默认标题
// 用户可通过 PATCH /sessions/:id 重命名
func defaultSessionTitle(question string) string {
	q := strings.TrimSpace(question)
	if q == "" {
		return ""
	}
	runes := []rune(q)
	if len(runes) > 30 {
		return string(runes[:30]) + "…"
	}
	return string(runes)
}

// ── FAQ 持久化缓存 ──
//
// 设计要点：
// 1. 命中：在 resource_type='FAQ' 中按 BM25 搜索 question；得分阈值更严（≤ -8.0）
//    且字符级 Jaccard 相似度 > 0.6，避免假阳性。
// 2. 写入：仅在「单 LLM 调用 + 有引用 sources」时入库；纯生成无来源不写。
// 3. 失效：用户提交 category=answer_error 反馈时，由 FeedbackService 调
//    s.kbRepo.SetStatus(faqResourceIDFor(question), 'retired') 立刻失效。

const (
	faqScoreThreshold       = -8.0 // BM25 分数（越小越相关）
	faqMinJaccardSimilarity = 0.6  // 中文逐字 Jaccard 相似度
)

// faqResourceIDFor 由问题原文生成确定性的 FAQ 资源 ID（与内存哈希同源，便于失效）
func faqResourceIDFor(question string) string {
	return "faq-cached-" + cacheKeyForQuestion(question)
}

// faqLookup 在 FAQ 资源中检索，命中阈值与相似度后还原 AnswerCard
func (s *ChatService) faqLookup(question string, userCtx *model.UserContext) *model.AnswerCard {
	if s.kbRepo == nil {
		return nil
	}
	results, err := s.kbRepo.SearchFAQ(question, userCtx.Role, 1)
	if err != nil || len(results) == 0 {
		return nil
	}
	hit := results[0]
	if hit.Score > faqScoreThreshold {
		return nil
	}
	if jaccardSimilarity(question, hit.Resource.Summary) < faqMinJaccardSimilarity {
		return nil
	}

	// content 字段存的是 AnswerCard JSON
	var card model.AnswerCard
	if err := json.Unmarshal([]byte(hit.Resource.Content), &card); err != nil {
		log.Printf("FAQ 反序列化失败 resource_id=%s err=%v", hit.Resource.ResourceID, err)
		return nil
	}
	// 标记为来自历史问答缓存，便于前端展示
	card.Sources = append([]model.Source{{
		ResourceID: hit.Resource.ResourceID,
		Title:      "历史问答缓存",
		Version:    hit.Resource.Version,
		SourceLink: "",
	}}, card.Sources...)
	return &card
}

// faqStore 把生成成功且有引用的回答持久化到知识库
func (s *ChatService) faqStore(question string, card *model.AnswerCard, role string) {
	if s.kbRepo == nil || card == nil {
		return
	}
	body, err := json.Marshal(card)
	if err != nil {
		return
	}
	q := strings.TrimSpace(question)
	titleRunes := []rune(q)
	if len(titleRunes) > 60 {
		titleRunes = titleRunes[:60]
	}
	res := &model.KBResource{
		ResourceID:   faqResourceIDFor(q),
		ResourceType: "FAQ",
		OwnerScope:   "school",
		OwnerID:      "all",
		RoleScope:    "[\"" + role + "\"]",
		Version:      time.Now().Format("20060102.150405"),
		Status:       "published",
		Title:        string(titleRunes),
		Summary:      q,
		Content:      string(body),
		SourceLink:   "",
		Tags:         "[\"faq-cached\"]",
		UpdatedBy:    "auto",
	}
	if _, action, err := s.kbRepo.Upsert(res); err != nil {
		log.Printf("FAQ 入库失败 resource_id=%s err=%v", res.ResourceID, err)
	} else {
		log.Printf("FAQ 入库成功 resource_id=%s action=%s", res.ResourceID, action)
	}
}

// RetireFAQ 把指定问题对应的 FAQ 资源标为 retired（用户反馈"回答有误"时调用）
func (s *ChatService) RetireFAQ(question string) error {
	if s.kbRepo == nil {
		return nil
	}
	resourceID := faqResourceIDFor(question)
	if err := s.kbRepo.SetStatus(resourceID, "retired"); err != nil {
		return err
	}
	log.Printf("FAQ 已失效 resource_id=%s", resourceID)
	return nil
}

// jaccardSimilarity 中文友好的字符级 Jaccard 相似度（去空白后按 rune 集合）
func jaccardSimilarity(a, b string) float64 {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return 0
	}
	setA := runeSet(a)
	setB := runeSet(b)
	inter := 0
	for r := range setA {
		if _, ok := setB[r]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func runeSet(s string) map[rune]struct{} {
	m := make(map[rune]struct{})
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		m[r] = struct{}{}
	}
	return m
}

// filterLowRelevanceResults 过滤低相关性检索结果
// 综合使用：标题二元词组匹配、Jaccard 相似度、关键词覆盖率
// 目的：在送入 LLM 前就把明显不相关的内容过滤掉，避免误导
func filterLowRelevanceResults(results []*repository.SearchResult, question string) []*repository.SearchResult {
	if len(results) == 0 {
		return results
	}

	q := strings.TrimSpace(question)
	if q == "" {
		return results
	}

	// 提取问题中的中文二元词组（核心语义单元）
	qBigrams := extractChineseBigramsFromQuestion(q)
	if len(qBigrams) == 0 {
		// 问题太短，不做过滤
		return results
	}

	var filtered []*repository.SearchResult
	for _, r := range results {
		// 计算相关性得分
		score := calcRelevanceScore(r.Resource.Title, r.Resource.Summary, r.Resource.Content, q, qBigrams)

		// 阈值：至少 0.15 分才认为相关
		if score >= 0.15 {
			filtered = append(filtered, r)
		} else {
			log.Printf("过滤低相关性结果: title=%q score=%.3f question=%q",
				truncateContent(r.Resource.Title, 30), score, truncateContent(q, 30))
		}
	}

	// 如果全部被过滤了，保留分数最高的1条（避免完全无结果，但 LLM 会根据提示词判断是否使用）
	if len(filtered) == 0 && len(results) > 0 {
		bestIdx := 0
		bestScore := -1.0
		for i, r := range results {
			s := calcRelevanceScore(r.Resource.Title, r.Resource.Summary, r.Resource.Content, q, qBigrams)
			if s > bestScore {
				bestScore = s
				bestIdx = i
			}
		}
		filtered = append(filtered, results[bestIdx])
		log.Printf("所有结果相关性均较低，保留最佳: title=%q score=%.3f",
			truncateContent(results[bestIdx].Resource.Title, 30), bestScore)
	}

	return filtered
}

// calcRelevanceScore 计算文档与问题的相关性得分（0-1）
// 权重：标题60% + 摘要25% + 全文15%
func calcRelevanceScore(title, summary, content, question string, qBigrams []string) float64 {
	titleScore := bigramMatchRatio(title, qBigrams)
	summaryScore := bigramMatchRatio(summary, qBigrams)
	contentScore := bigramMatchRatio(content, qBigrams)

	// 标题中精确匹配整个问题，额外加分
	if strings.Contains(title, question) {
		titleScore = 1.0
	}

	return titleScore*0.6 + summaryScore*0.25 + contentScore*0.15
}

// bigramMatchRatio 计算文本中匹配的二元词组比例
func bigramMatchRatio(text string, bigrams []string) float64 {
	if len(bigrams) == 0 {
		return 0
	}
	matched := 0
	for _, bg := range bigrams {
		if strings.Contains(text, bg) {
			matched++
		}
	}
	return float64(matched) / float64(len(bigrams))
}

// extractChineseBigramsFromQuestion 从问题中提取中文二元词组（去停用词后）
func extractChineseBigramsFromQuestion(q string) []string {
	// 先去除常见停用词和疑问词
	stopWords := []string{"什么", "怎么", "如何", "为什么", "哪", "哪里", "哪个",
		"吗", "呢", "啊", "吧", "了", "的", "是", "有", "在", "我", "你", "他",
		"要", "需要", "可以", "能", "能够", "请问", "麻烦", "一下"}
	cleaned := q
	for _, sw := range stopWords {
		cleaned = strings.ReplaceAll(cleaned, sw, "")
	}
	cleaned = strings.TrimSpace(cleaned)

	runes := []rune(cleaned)
	var bigrams []string
	seen := make(map[string]bool)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] >= 0x4E00 && runes[i] <= 0x9FFF &&
			runes[i+1] >= 0x4E00 && runes[i+1] <= 0x9FFF {
			bg := string(runes[i : i+2])
			if !seen[bg] {
				seen[bg] = true
				bigrams = append(bigrams, bg)
			}
		}
	}
	return bigrams
}
