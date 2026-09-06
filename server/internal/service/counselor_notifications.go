package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateSmartNotification 生成面向不同受众的通知版本。
func (s *CounselorService) GenerateSmartNotification(ctx context.Context, content string, audienceTypes []string) *SmartNotification {
	sn := &SmartNotification{OriginalContent: content, Variants: []map[string]string{{"audience": "全体学生", "tone": "正式", "text": content}, {"audience": "学生干部", "tone": "简要+行动导向", "text": "【通知】" + content + "\n请各班班长落实并反馈。"}, {"audience": "重点关注学生", "tone": "温和关怀", "text": content + "\n如有困难可随时联系辅导员。"}}, DataSource: "fallback"}
	if s.llmClient != nil && content != "" {
		prompt := fmt.Sprintf("你是辅导员助理。请将以下通知改写为3个版本：1)正式通知 2)学生干部版(简要) 3)关怀版(温和)。各不超过40字。\n原文：%s", content)
		if resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.5, MaxTokens: 400}); err == nil && resp != nil && resp.Content != "" {
			sn.Variants = append(sn.Variants, map[string]string{"audience": "AI定制版", "tone": "智能适配", "text": strings.TrimSpace(resp.Content)})
			sn.DataSource = "ai"
		}
	}
	return sn
}

// GenerateCheckinStats 返回诚实的班级打卡统计，没有真实数据时不生成样例指标。
func (s *CounselorService) GenerateCheckinStats(ctx context.Context, className string) *CheckinStats {
	_ = ctx
	if className == "" {
		className = "全部班级"
	}
	return &CheckinStats{ClassName: className, TotalStudents: 0, TodayRate: 0, StreakDistribution: map[string]int{}, DeclineStudents: []map[string]interface{}{}, AIAnalysis: "暂无真实打卡统计数据。学生启用每日打卡后，这里会自动汇聚班级打卡率与中断提醒，不展示示例数据。", DataSource: "real"}
}
