package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dll/wxx/server/internal/llm"
)

// GradingResult AI 作业批改结果
type GradingResult struct {
	TotalSubmissions int            `json:"total_submissions"`
	Graded           int            `json:"graded"`
	AverageScore     float64        `json:"average_score"`
	Distribution     map[string]int `json:"distribution"`
	CommonIssues     []string       `json:"common_issues"`
	ExcellentWorks   []string       `json:"excellent_works"`
	DataSource       string         `json:"data_source"`
}

func (s *TeacherService) GradeAssignments(ctx context.Context, courseName string) (*GradingResult, error) {
	if s.llmClient != nil {
		result, err := s.generateGradingWithLLM(ctx, courseName)
		if err == nil && result != nil {
			return result, nil
		}
	}
	return s.fallbackGrading(courseName), nil
}

func (s *TeacherService) generateGradingWithLLM(ctx context.Context, courseName string) (*GradingResult, error) {
	prompt := fmt.Sprintf("你是一位高校教师。请分析「%s」课程最近一次作业的批改情况。\n严格按以下 JSON 结构输出：{\"total_submissions\":45,\"graded\":45,\"average_score\":78.5,\"distribution\":{\"90-100\":8,\"80-89\":15,\"70-79\":12,\"60-69\":7,\"below_60\":3},\"common_issues\":[\"...\"],\"excellent_works\":[\"张三 - 代码简洁，注释清晰\"]}", courseName)
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.4, MaxTokens: 600})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}
	var parsed struct {
		TotalSubmissions int            `json:"total_submissions"`
		Graded           int            `json:"graded"`
		AverageScore     float64        `json:"average_score"`
		Distribution     map[string]int `json:"distribution"`
		CommonIssues     []string       `json:"common_issues"`
		ExcellentWorks   []string       `json:"excellent_works"`
	}
	if err := json.Unmarshal([]byte(extractJSON(resp.Content)), &parsed); err != nil || len(parsed.CommonIssues) == 0 {
		return nil, fmt.Errorf("LLM 批改结果解析失败")
	}
	return &GradingResult{TotalSubmissions: parsed.TotalSubmissions, Graded: parsed.Graded, AverageScore: parsed.AverageScore, Distribution: parsed.Distribution, CommonIssues: parsed.CommonIssues, ExcellentWorks: parsed.ExcellentWorks, DataSource: "ai"}, nil
}

func (s *TeacherService) fallbackGrading(courseName string) *GradingResult {
	return &GradingResult{TotalSubmissions: 45, Graded: 45, AverageScore: 78.5, Distribution: map[string]int{"90-100": 8, "80-89": 15, "70-79": 12, "60-69": 7, "below_60": 3}, CommonIssues: []string{"递归终止条件遗漏", "空指针未判断", "时间复杂度分析不准确"}, ExcellentWorks: []string{"张三 - 代码简洁，注释清晰", "李四 - 额外实现了迭代版本"}, DataSource: "fallback"}
}
