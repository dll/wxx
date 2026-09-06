package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateTalkTips 根据学生画像推荐谈话话术。
func (s *CounselorService) GenerateTalkTips(ctx context.Context, studentProfile string) (*TalkTip, error) {
	if s.llmClient == nil {
		return fallbackTalkTip(), nil
	}
	prompt := fmt.Sprintf("你是一位经验丰富的辅导员。请为以下学生画像推荐谈话切入话术。\n\n学生情况：%s\n\n输出格式：\n场景：xxx\n开场白：xxx\n提问建议：xxx（用/分隔）\n注意事项：xxx（用/分隔）", studentProfile)
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.5, MaxTokens: 500})
	if err != nil || resp == nil || resp.Content == "" {
		return fallbackTalkTip(), nil
	}
	return parseTalkTip(resp.Content), nil
}

func parseTalkTip(text string) *TalkTip {
	tip := fallbackTalkTip()
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "场景："):
			tip.Scenario = strings.TrimPrefix(line, "场景：")
		case strings.HasPrefix(line, "开场白："):
			tip.OpeningLine = strings.TrimPrefix(line, "开场白：")
		case strings.HasPrefix(line, "提问建议："):
			tip.Questions = strings.Split(strings.TrimPrefix(line, "提问建议："), "/")
		case strings.HasPrefix(line, "注意事项："):
			tip.Cautions = strings.Split(strings.TrimPrefix(line, "注意事项："), "/")
		}
	}
	return tip
}

func fallbackTalkTip() *TalkTip {
	return &TalkTip{Scenario: "一般关心谈话", OpeningLine: "最近怎么样？学习和生活上有什么需要帮助的吗？", Questions: []string{"最近睡眠质量如何？", "学习上有没有遇到困难？", "和同学相处得怎么样？"}, Cautions: []string{"保持温和语气", "多倾听少说教", "注意观察对方情绪变化"}}
}
