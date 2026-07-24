package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// ChatActivities 问答相关活动集合
// 持有 repo/llm 引用，每个活动方法对应 ChatAskWorkflow 中的一个步骤
type ChatActivities struct {
	SessionRepo *repository.SessionRepo
	MessageRepo *repository.MessageRepo
	KBRepo      *repository.KBRepo
	AgentRepo   *repository.AgentRepo
	LLMClient   llm.ChatClient
}

// ── 步骤 1：验证或创建会话，保存用户消息 ──

// ValidateSessionActivity 验证或创建会话，保存用户问题消息
func (a *ChatActivities) ValidateSessionActivity(ctx context.Context, input ValidateSessionInput) (*ValidateSessionResult, error) {
	sessionID := input.SessionID

	if sessionID == "" {
		// 创建新会话
		sessionID = uuid.New().String()
		err := a.SessionRepo.Create(&model.Session{
			SessionID: sessionID,
			UserID:    input.UserID,
		})
		if err != nil {
			return nil, fmt.Errorf("创建会话失败: %w", err)
		}
	} else {
		// 验证会话属于当前用户
		session, err := a.SessionRepo.GetBySessionID(sessionID)
		if err != nil {
			return nil, fmt.Errorf("查询会话失败: %w", err)
		}
		if session == nil || session.UserID != input.UserID {
			return nil, fmt.Errorf("会话不存在或无权访问")
		}
		_ = a.SessionRepo.Touch(sessionID)
	}

	// 保存用户消息
	_ = a.MessageRepo.Create(&model.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   input.Question,
		TraceID:   input.TraceID,
	})

	return &ValidateSessionResult{SessionID: sessionID}, nil
}

// ── 步骤 2：知识检索（只读，失败不中断工作流）──

// SearchKnowledgeActivity 执行 FTS5 知识检索
func (a *ChatActivities) SearchKnowledgeActivity(ctx context.Context, input SearchKnowledgeInput) (*SearchKnowledgeResult, error) {
	results, err := a.KBRepo.Search(input.Question, input.OwnerScope, input.OwnerID, input.Role, 5)
	if err != nil {
		return nil, fmt.Errorf("知识检索失败: %w", err)
	}

	resultsJSON, _ := json.Marshal(results)
	return &SearchKnowledgeResult{ResultsJSON: string(resultsJSON)}, nil
}

// ── 步骤 3：LLM 调用（高风险操作，独立重试策略）──

// CallLLMActivity 调用 LLM 生成回答
func (a *ChatActivities) CallLLMActivity(ctx context.Context, input CallLLMInput) (*CallLLMResult, error) {
	// 解析检索结果
	var searchResults []*repository.SearchResult
	if input.SearchResults != "" {
		_ = json.Unmarshal([]byte(input.SearchResults), &searchResults)
	}

	// 构造消息列表
	messages := buildChatMessages(a.AgentRepo, input.SessionID, input.Question, input.AgentID, searchResults)

	// 调用 LLM
	llmResp, err := a.LLMClient.Chat(ctx, &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	tokens, _ := json.Marshal(map[string]int{
		"prompt": llmResp.PromptTokens,
		"output": llmResp.OutputTokens,
	})

	log.Printf("LLM 调用成功 prompt_tokens=%d output_tokens=%d", llmResp.PromptTokens, llmResp.OutputTokens)
	return &CallLLMResult{
		Content: llmResp.Content,
		Tokens:  string(tokens),
	}, nil
}

// ── 步骤 4：构造 AnswerCard 并保存助手消息 ──

// BuildAnswerActivity 构造 AnswerCard 并保存助手回复
func (a *ChatActivities) BuildAnswerActivity(ctx context.Context, input BuildAnswerInput) (*BuildAnswerResult, error) {
	// 解析检索结果
	var searchResults []*repository.SearchResult
	if input.SearchResults != "" {
		_ = json.Unmarshal([]byte(input.SearchResults), &searchResults)
	}

	// 构造 AnswerCard
	card := buildAnswerCard(input.LLMContent, searchResults, input.TraceID)

	// 保存助手消息
	content := input.LLMContent
	if content == "" {
		content = card.Conclusion // 兜底回答
	}
	_ = a.MessageRepo.Create(&model.Message{
		SessionID: input.SessionID,
		Role:      "assistant",
		Content:   content,
		TraceID:   input.TraceID,
	})

	cardJSON, err := json.Marshal(card)
	if err != nil {
		return nil, fmt.Errorf("序列化 AnswerCard 失败: %w", err)
	}

	return &BuildAnswerResult{AnswerCardJSON: string(cardJSON)}, nil
}

// ── 辅助函数 ──

// buildChatMessages 构造 LLM 消息列表（复用 chat_service 中同等逻辑）
func buildChatMessages(agentRepo *repository.AgentRepo, sessionID, question, agentID string, results []*repository.SearchResult) []llm.ChatMessage {
	var messages []llm.ChatMessage

	// 获取系统提示词
	systemPrompt := getSystemPrompt(agentRepo, agentID)

	// 拼接知识库内容
	if len(results) > 0 {
		var kb strings.Builder
		kb.WriteString("\n\n--- 知识库参考资料 ---\n")
		for i, r := range results {
			kb.WriteString(fmt.Sprintf("\n【资料 %d】%s（%s）\n", i+1, r.Resource.Title, r.Resource.ResourceType))
			if r.Resource.Summary != "" {
				kb.WriteString("摘要：" + r.Resource.Summary + "\n")
			}
			kb.WriteString("内容：" + truncateContent(r.Resource.Content, 1500) + "\n")
		}
		systemPrompt += kb.String()
	}

	messages = append(messages, llm.ChatMessage{Role: "system", Content: systemPrompt})
	messages = append(messages, llm.ChatMessage{Role: "user", Content: question})

	return messages
}

// getSystemPrompt 获取智能体系统提示词，未指定或查找失败返回默认值
func getSystemPrompt(agentRepo *repository.AgentRepo, agentID string) string {
	if agentID != "" && agentRepo != nil {
		agent, err := agentRepo.GetByAgentID(agentID)
		if err == nil && agent != nil && agent.Status == "active" && agent.SystemPrompt != "" {
			return agent.SystemPrompt
		}
	}

	return `你是"蔚小芯"，一个高校智慧学工 AI 助手。请严格基于以下知识库内容回答用户问题。

规则：
1. 只使用知识库中的信息回答，不要编造内容
2. 涉及政策、条件、数字时必须原文引用
3. 如果知识库中没有相关内容，明确告知用户你无法回答并建议联系辅导员
4. 回答要简洁、准确、有条理
5. 如果涉及流程，按步骤列出`
}

// buildAnswerCard 从 LLM 回复和检索结果构造 AnswerCard
func buildAnswerCard(content string, results []*repository.SearchResult, traceID string) *model.AnswerCard {
	if content == "" {
		// LLM 调用失败时，基于已召回资料生成本地结构化回答，不向用户暴露模型服务状态。
		conclusion := "知识库中暂未找到足够信息。建议联系辅导员或学工办公室确认。"
		confidence := 0.5
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
		for _, r := range results {
			card.Sources = append(card.Sources, model.Source{
				ResourceID:     r.Resource.ResourceID,
				Title:          r.Resource.Title,
				ResourceType:   r.Resource.ResourceType,
				Version:        r.Resource.Version,
				SourceLink:     r.Resource.SourceLink,
				RelevanceScore: -r.Score,
				EffectiveAt:    r.Resource.EffectiveAt,
				Snippet:        r.Resource.Summary,
			})
		}
		// 按相关度降序排序
		sort.Slice(card.Sources, func(i, j int) bool {
			return card.Sources[i].RelevanceScore > card.Sources[j].RelevanceScore
		})
		return card
	}

	card := &model.AnswerCard{
		Conclusion: content,
		TraceID:    traceID,
		Confidence: 0.8,
		Fallback:   false,
	}

	for _, r := range results {
		card.Sources = append(card.Sources, model.Source{
			ResourceID:     r.Resource.ResourceID,
			Title:          r.Resource.Title,
			ResourceType:   r.Resource.ResourceType,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: -r.Score,
			EffectiveAt:    r.Resource.EffectiveAt,
			Snippet:        r.Resource.Summary,
		})
	}

	// 按相关度降序排序
	sort.Slice(card.Sources, func(i, j int) bool {
		return card.Sources[i].RelevanceScore > card.Sources[j].RelevanceScore
	})

	if len(results) == 0 {
		card.Confidence = 0.3
		card.Fallback = true
	}

	// 简单追问生成
	card.FollowUps = generateFollowUps(content)
	return card
}

// generateFollowUps 根据回答内容生成追问建议
func generateFollowUps(content string) []string {
	var followUps []string
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

// ── 类型定义（避免循环依赖 workflow 包）──

// ValidateSessionInput 会话验证活动输入
type ValidateSessionInput struct {
	UserID    int64  `json:"user_id"`
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
	TraceID   string `json:"trace_id"`
}

// ValidateSessionResult 会话验证活动输出
type ValidateSessionResult struct {
	SessionID string `json:"session_id"`
}

// SearchKnowledgeInput 知识检索活动输入
type SearchKnowledgeInput struct {
	Question   string `json:"question"`
	OwnerScope string `json:"owner_scope"`
	OwnerID    string `json:"owner_id"`
	Role       string `json:"role"`
}

// SearchKnowledgeResult 知识检索活动输出
type SearchKnowledgeResult struct {
	ResultsJSON string `json:"results_json"`
}

// CallLLMInput LLM 调用活动输入
type CallLLMInput struct {
	SessionID     string `json:"session_id"`
	Question      string `json:"question"`
	AgentID       string `json:"agent_id"`
	SearchResults string `json:"search_results"`
}

// CallLLMResult LLM 调用活动输出
type CallLLMResult struct {
	Content string `json:"content"`
	Tokens  string `json:"tokens"`
}

// BuildAnswerInput 构造回答活动输入
type BuildAnswerInput struct {
	SessionID     string `json:"session_id"`
	Question      string `json:"question"`
	LLMContent    string `json:"llm_content"`
	LLMTokens     string `json:"llm_tokens"`
	SearchResults string `json:"search_results"`
	TraceID       string `json:"trace_id"`
}

// BuildAnswerResult 构造回答活动输出
type BuildAnswerResult struct {
	AnswerCardJSON string `json:"answer_card_json"`
}
