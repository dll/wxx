package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// EmotionAgent 心理疏导子智能体
// 用于处理学生情感/心理类问题：检索情感咨询知识 + LLM 共情式回复
// 输出强调倾听、共情、不诊断、危机识别后转介心理咨询
type EmotionAgent struct {
	llmClient  llm.ChatClient
	searchTopK int
}

// NewEmotionAgent 创建情感疏导 Agent
// llmClient 可为 nil，此时 Agent 仅返回检索的情感支持资源 + 兜底安抚语
func NewEmotionAgent(llmClient llm.ChatClient) *EmotionAgent {
	return &EmotionAgent{
		llmClient:  llmClient,
		searchTopK: 3,
	}
}

// Key 返回 Agent 路由 Key（与 router.go intentToAgent 保持一致）
func (a *EmotionAgent) Key() string { return "emotion-counselor" }

// Name 返回人类可读名称
func (a *EmotionAgent) Name() string { return "心理疏导" }

// 危机关键词 — 命中后直接进入危机模式，返回明确的求助渠道
var crisisKeywords = []string{
	"自杀", "自残", "自伤", "想死", "活不下去", "结束生命",
	"不想活", "了结", "跳楼", "割腕",
}

// emotionSystemPrompt 心理疏导提示词 — 强调共情、不诊断、危机转介
const emotionSystemPrompt = `你是一名温和、有同理心的校园心理疏导助手，名叫"蔚小芯"。

回复原则：
1. 先承接情绪：用 1-2 句共情学生当下的感受
2. 不进行诊断：不使用"抑郁症""焦虑症"等专业病名贴标签
3. 提供可操作的小建议（如呼吸练习、记录心情、与信任的人聊聊）
4. 鼓励寻求专业帮助：给出校心理咨询中心、辅导员等联系方式（如知识库提供）
5. 避免说教式语气；用学生听得懂的、平等的口吻

请基于知识库内容生成不超过 200 字的简短回复。`

// crisisResponse 危机模式兜底回复（不依赖 LLM，必须可用）
const crisisResponse = `我注意到你正在经历非常难熬的时刻，我很担心你。

请记住：你不是孤单一个人，现在请立刻联系：

📞 全国心理援助热线 400-161-9995（24 小时）
📞 北京心理危机研究与干预中心 010-82951332（24 小时）

如果在校园内，也可以直接拨打：
- 学校心理咨询中心：以学校公布电话为准
- 信任的辅导员或老师

你愿意先告诉我此刻的感受吗？我会一直在这里听你说。`

// Execute 执行情感疏导问答
func (a *EmotionAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	// 危机模式：命中关键词直接返回兜底安抚 + 求助渠道，绕过 LLM
	if isCrisisSignal(question) {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    crisisResponse,
			Confidence: 1.0, // 危机模式置信度最高
		}, nil
	}

	// 检索情感支持类知识（若有）
	var sources []model.Source
	var knowledgeSnippet string
	if kbRepo != nil {
		results, err := kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, a.searchTopK)
		if err == nil && len(results) > 0 {
			sources = kbResultsToSources(results)
			var parts []string
			for i, r := range results {
				parts = append(parts, fmt.Sprintf("【资料%d】%s\n%s", i+1, r.Resource.Title, truncate(r.Resource.Content, 500)))
			}
			knowledgeSnippet = strings.Join(parts, "\n\n")
		}
	}

	// LLM 生成共情回复
	if a.llmClient != nil {
		systemContent := emotionSystemPrompt
		if knowledgeSnippet != "" {
			systemContent += "\n\n--- 可参考的知识库内容 ---\n" + knowledgeSnippet
		}
		resp, err := a.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages: []llm.ChatMessage{
				{Role: "system", Content: systemContent},
				{Role: "user", Content: question},
			},
			Temperature: 0.7, // 共情场景需要更自然，温度略高
			MaxTokens:   400,
		})
		if err == nil && resp.Content != "" {
			return &AgentResult{
				AgentName:  a.Name(),
				Content:    resp.Content,
				Sources:    sources,
				Confidence: 0.8,
			}, nil
		}
	}

	// LLM 失败兜底：温和的安抚语 + 引导求助
	fallback := "我听到了你的烦恼，这种感受是真实而重要的。\n\n如果你愿意，可以试试以下方法：\n• 深呼吸 3 次，允许自己暂停一下\n• 找一个信任的人简短聊聊\n• 写下此刻的心情，让它流动起来\n\n如果情绪持续困扰你，请记得校心理咨询中心和辅导员都在你身边。"
	return &AgentResult{
		AgentName:  a.Name(),
		Content:    fallback,
		Sources:    sources,
		Confidence: 0.4,
	}, nil
}

// isCrisisSignal 检测高风险危机信号
func isCrisisSignal(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range crisisKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
