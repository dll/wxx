package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
)

// askGenerateAndAssemble 尾部阶段（步 4~6）：拼装 LLM 上下文 → 调 LLM → 构造 AnswerCard，
// 并完成助手消息落库、词元统计、缓存写入与 FAQ 持久化写入。所有兜底分支与拆分前完全一致。
func (s *ChatService) askGenerateAndAssemble(ctx context.Context, userCtx *model.UserContext, sessionID, question, agentID string, searchResults []*repository.SearchResult, multiAgentResult *agent.MergedResult, traceID string) (*model.AnswerCard, string, error) {
	// ── 4. 拼装 LLM 上下文 ──
	// 发送给 LLM 前对用户问题进行 PII 脱敏
	sanitizedQuestion := util.SanitizeForLLM(question, 2000)
	messages := s.buildMessages(ctx, sessionID, sanitizedQuestion, agentID, searchResults, multiAgentResult)
	if s.llmClient == nil {
		card := s.fallbackAnswerWithSources(traceID, question, searchResults)
		s.saveAssistantMessage(sessionID, card.Conclusion, traceID)
		return card, sessionID, nil
	}

	// ── 5. 调 LLM ──
	// 用户自定义模型配置（default_provider + Key + 模型名）优先覆盖服务器默认
	req := &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.3, // 问答场景用低温度，减少编造
		MaxTokens:   2048,
	}
	if override := s.resolveUserLLMOverrides(userCtx.UserID); override != nil {
		req.APIKey = override.APIKey
		req.Model = override.Model
	}
	llmResp, err := s.llmClient.Chat(ctx, req)
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

	// 保存助手回复（原文落库，脱敏仅用于模型上下文）
	s.saveAssistantMessage(sessionID, llmResp.Content, traceID)

	// 记录词元使用
	if s.tokenStatsSvc != nil {
		s.tokenStatsSvc.RecordUsage(userCtx.UserID, sessionID, s.llmClient.Name(), llmResp.PromptTokens, llmResp.OutputTokens)
	}

	// │ 缓存写入 ── 入学/离校等固定流程问题缓存 24 小时
	s.cacheSet(question, sessionID, card)

	// │ FAQ 持久化缓存写入 ── 仅在有引用且非 agent/多智能体路径时入库
	// 排除流程类问题：流程类由结构化端点保证确定性，缓存会绕过最新 process_steps 数据
	if agentID == "" && multiAgentResult == nil && len(card.Sources) > 0 && !hasProcessResult(searchResults) {
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
		// 判断是否全部为低置信（仅强留的弱相关结果），若是则提示 LLM 谨慎作答
		allLow := true
		for _, r := range results {
			if !r.LowConfidence {
				allLow = false
				break
			}
		}

		var knowledgeBuilder strings.Builder
		knowledgeBuilder.WriteString("\n\n--- 知识库参考资料 ---\n")
		if allLow {
			knowledgeBuilder.WriteString("注意：以下资料与问题的相关性较低，可能并不匹配。若资料未明确覆盖用户问题，请勿臆测或编造条款、数字、期限等关键信息，应明确告知信息不足并建议咨询相关部门。\n")
		}
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
	// 安全修复 SEC-03/SEC-04：历史消息按原文落库，回放给 LLM 前必须执行二次 PII 脱敏。
	// 第一次过滤（一次过滤）：用户当轮输入进来时，在 Ask() 调用 util.SanitizeForLLM(question, ...) 完成。
	// 第二次过滤（二次过滤，本处 SEC-04）：历史轮次消息拼装进 LLM 请求前重新调用 PII 脱敏，
	//   防止早前轮次的 PII（如学号、手机号、身份证号）绕过当前轮的入参检查直接进入模型上下文。
	history, _ := s.messageRepo.GetRecentContext(sessionID, 6)
	for _, h := range history {
		content := h.Content
		if h.Role == "assistant" {
			// 助手消息：脱敏 + trim（防止模型幻觉输出泄漏到历史上下文）
			content = util.SanitizeLLMResponse(content)
		} else {
			// 用户消息：脱敏（调用 util.MaskPII）+ trim + 截断至 2000 字符（二次过滤）
			content = util.SanitizeForLLM(content, 2000)
		}
		messages = append(messages, llm.ChatMessage{
			Role:    h.Role,
			Content: content,
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

【引用标注规则】
1. 回答中涉及具体政策、流程、数据时，必须标注来源编号，格式为 [资料N]，其中 N 为资料序号（1、2、3...），对应上下文提供的资料顺序。
2. 引用编号标注在句子末尾的句号前，例如："新生报到需携带身份证[资料1][资料3]。"
3. 每句话引用的资料最多 3 个，选择最相关的。
4. 如果引用的资料之间有冲突，以最新版本或官方发布的为准，并在回答中说明差异。
5. 回答末尾可列出参考来源摘要，便于用户溯源。

【版本冲突处理规则】
- 当多个资料对同一问题有不同表述时，优先采用版本号更新、发布时间更近的资料。
- 若资料有明确的生效时间，以生效时间最新的为准。
- 学校级政策优先于学院级，学院级优先于班级级。
- 如无法判断版本先后，应同时列出不同说法并注明存在差异，建议咨询相关部门确认。

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
	// 标注参与本次回答的智能体（D4-3：透明化多智能体参与）。数据来源为多 Agent 编排的
	// 汇聚结果（merger.Merge 收集的实际参与者名），无 Agent 信息路径保持为空，不硬编。
	if multiAgentResult != nil && len(multiAgentResult.Agents) > 0 {
		card.Agents = multiAgentResult.Agents
	}
	// 标注回答所用大模型（如 deepseek-v4-flash），供前端展示来源可信度
	if s.llmClient != nil {
		card.Model = s.llmClient.Model()
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
			ResourceType:   r.Resource.ResourceType,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: normalizeRelevanceScore(-r.Score),
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

	// 按相关度降序排序（RelevanceScore 越大越相关）
	sort.Slice(card.Sources, func(i, j int) bool {
		return card.Sources[i].RelevanceScore > card.Sources[j].RelevanceScore
	})

	// 判定是否所有命中都是「仅为避免空结果而强留的低置信结果」
	allLowConfidence := len(results) > 0
	for _, r := range results {
		if !r.LowConfidence {
			allLowConfidence = false
			break
		}
	}

	// 无知识命中、或命中全部为低置信时降低置信度并走兜底
	if len(results) == 0 || allLowConfidence {
		card.Confidence = 0.3
		card.Fallback = true
		// CE-02：低置信时清空可能误导的来源，避免弱相关资料被当作权威依据
		if allLowConfidence {
			card.Sources = nil
		}
		// 如果多智能体也没有来源，替换结论为兜底引导文案（避免 LLM 无依据编造关键数字）
		if multiAgentResult == nil || len(multiAgentResult.Sources) == 0 {
			card.Conclusion = `知识库中暂未找到足够匹配的信息，为避免提供不准确的条款或数字，建议联系辅导员、学院学工办公室或相关职能部门确认最新要求。`
		}
	}

	// 生成追问建议
	card.FollowUps = generateFollowUps(content)

	return card
}
