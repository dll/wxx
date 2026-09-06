package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateMentalHealthReport 依据近期情感记录生成心理健康评估，数据不足时使用安全兜底。
func (s *StudentService) GenerateMentalHealthReport(ctx context.Context, userID int64) *MentalHealthReport {
	userName := "同学"
	if s.userRepo != nil {
		user, err := s.userRepo.GetByID(userID)
		if err == nil && user != nil {
			userName = user.DisplayName
		}
	}
	report := &MentalHealthReport{
		Date: time.Now().Format("2006-01-02"), StressLevel: "中等", EmotionState: "总体平稳", SocialLevel: "良好", Resilience: "较强",
		Suggestions: []string{"建议每天保持30分钟运动", "尝试正念冥想来缓解压力", "与朋友保持定期社交联系", "遇到困难及时向辅导员或心理咨询中心求助"}, DataSource: "fallback",
	}
	if s.emotionRepo != nil {
		logs, err := s.emotionRepo.ListRecentByUser(userID, 20)
		if err == nil && len(logs) > 0 {
			var sum float64
			var high, urgent int
			for _, l := range logs {
				sum += l.Score
				switch l.RiskLevel {
				case "high":
					high++
				case "urgent":
					urgent++
				}
			}
			avg := sum / float64(len(logs))
			switch {
			case urgent > 0 || avg <= -0.5:
				report.StressLevel, report.EmotionState, report.Resilience = "偏高", "近期情绪波动较大", "需关注"
			case high > 0 || avg <= -0.2:
				report.StressLevel, report.EmotionState, report.Resilience = "中等偏上", "存在一定压力", "尚可"
			case avg >= 0.3:
				report.StressLevel, report.EmotionState, report.Resilience = "较低", "情绪积极稳定", "较强"
			}
			report.DataSource = "real"
			if urgent > 0 || high > 0 {
				report.Suggestions = append([]string{"近期检测到情绪压力信号，建议尽快联系学校心理咨询中心或辅导员当面沟通"}, report.Suggestions...)
			}
			if s.llmClient != nil {
				prompt := fmt.Sprintf("你是心理健康顾问。学生%s近期%d条情绪记录平均情感分%.2f（-1~1，越低压力越大），高风险%d条、紧急%d条。请用50字内给出温暖、可执行的建议，勿诊断、勿夸大。", userName, len(logs), avg, high, urgent)
				if resp, e := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.5, MaxTokens: 200}); e == nil && resp != nil && resp.Content != "" {
					report.Suggestions = append(report.Suggestions, "AI个性化建议："+resp.Content)
				}
			}
			return report
		}
	}
	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是心理健康顾问。为%s生成简短的心理健康建议（50字）：保持规律作息，适当运动，积极社交。", userName)
		if resp, e := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.5, MaxTokens: 200}); e == nil && resp != nil && resp.Content != "" {
			report.Suggestions = append(report.Suggestions, "AI个性化建议："+resp.Content)
			report.DataSource = "ai"
		}
	}
	return report
}
