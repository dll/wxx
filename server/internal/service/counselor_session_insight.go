package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateSessionInsight 生成辅导员会话洞察。
func (s *CounselorService) GenerateSessionInsight(ctx context.Context, studentName string, messages []string) *SessionInsight {
	insight := &SessionInsight{StudentName: studentName, MainTopics: []string{"学业咨询", "生活服务"}, EmotionTrend: "平稳→积极", KeyConcerns: []string{"对课程难度有担忧", "希望了解更多实习信息"}, Suggestions: []string{"推荐相关学习资源", "推送近期实习招聘信息"}, DataSource: "fallback"}
	if s.llmClient != nil && len(messages) > 0 {
		joined := strings.Join(messages, "\n")
		prompt := fmt.Sprintf("你是辅导员助理。分析学生%s的对话记录，提取关键信息（话题/情绪/诉求）。50字。\n对话：%s", studentName, joined[:min(len(joined), 500)])
		if resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.3, MaxTokens: 300}); err == nil && resp != nil && resp.Content != "" {
			insight.KeyConcerns = append(insight.KeyConcerns, "AI分析："+resp.Content)
			insight.DataSource = "ai"
		}
	}
	return insight
}
