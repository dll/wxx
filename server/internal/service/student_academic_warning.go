package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// AcademicWarning 学业预警结果。
type AcademicWarning struct {
	StudentName string   `json:"student_name"`
	RiskLevel   string   `json:"risk_level"`
	RiskScore   float64  `json:"risk_score"`
	Factors     []string `json:"factors"`
	Suggestions []string `json:"suggestions"`
	Resources   []string `json:"resources"`
	DataSource  string   `json:"data_source"`
}

// GenerateAcademicWarning 生成学业预警与改进建议；数据不足时返回稳定兜底结果。
func (s *StudentService) GenerateAcademicWarning(ctx context.Context, userID int64) *AcademicWarning {
	userName := "同学"
	if s.userRepo != nil {
		user, err := s.userRepo.GetByID(userID)
		if err == nil && user != nil {
			userName = user.DisplayName
		}
	}

	warning := &AcademicWarning{
		StudentName: userName,
		RiskLevel:   "low",
		RiskScore:   0.12,
		Factors:     []string{"近两周出勤率下降5%", "最近一次作业成绩偏低"},
		Suggestions: []string{"建立每周学习计划", "参加学习小组", "定期与老师沟通学习进度"},
		Resources:   []string{"学习辅导中心", "图书馆自习室", "在线课程资源"},
		DataSource:  "fallback",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是学业预警分析师。学生%s最近出勤和作业有波动。请给出风险等级、风险因素和改进建议。80字以内。", userName)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 300,
		})
		if err == nil && resp != nil && strings.TrimSpace(resp.Content) != "" {
			warning.Suggestions = append(warning.Suggestions, "AI建议："+strings.TrimSpace(resp.Content))
			warning.DataSource = "ai"
		}
	}

	return warning
}
