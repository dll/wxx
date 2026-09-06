package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateIntervention 生成学生风险干预方案。
func (s *CounselorService) GenerateIntervention(ctx context.Context, studentName, riskLevel, reason string) (*InterventionPlan, error) {
	if s.llmClient == nil {
		return fallbackIntervention(studentName, riskLevel), nil
	}
	prompt := fmt.Sprintf("你是辅导员的专业顾问。请为以下预警学生制定个性化干预方案。\n\n学生：%s\n风险等级：%s\n预警原因：%s\n\n输出格式：\n紧急措施：xxx（用/分隔）\n长期方案：xxx（用/分隔）\n类似案例：xxx", studentName, riskLevel, reason)
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.3, MaxTokens: 600})
	if err != nil || resp == nil || resp.Content == "" {
		return fallbackIntervention(studentName, riskLevel), nil
	}
	return parseIntervention(resp.Content, studentName, riskLevel), nil
}

func parseIntervention(text, name, risk string) *InterventionPlan {
	plan := fallbackIntervention(name, risk)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "紧急措施："):
			plan.UrgentActions = strings.Split(strings.TrimPrefix(line, "紧急措施："), "/")
		case strings.HasPrefix(line, "长期方案："):
			plan.LongTermPlan = strings.Split(strings.TrimPrefix(line, "长期方案："), "/")
		case strings.HasPrefix(line, "类似案例："):
			plan.SimilarCases = strings.TrimPrefix(line, "类似案例：")
		}
	}
	return plan
}

func fallbackIntervention(name, risk string) *InterventionPlan {
	return &InterventionPlan{TargetStudent: name, RiskLevel: risk, UrgentActions: []string{"立即与学生本人联系", "告知家长关注学生状态", "联系心理健康中心评估"}, LongTermPlan: []string{"建立定期沟通机制", "推荐参加校园活动", "安排学业帮扶"}, SimilarCases: "同类案例处理经验：早期介入是关键，多部门联动效果更好。"}
}
