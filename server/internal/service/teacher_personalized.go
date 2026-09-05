package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

type PersonalizedTeaching struct {
	StudentName   string   `json:"student_name"`
	LearningStyle string   `json:"learning_style"`
	WeakPoints    []string `json:"weak_points"`
	Strategy      string   `json:"strategy"`
	Resources     []string `json:"resources"`
	DataSource    string   `json:"data_source"`
}

func (s *TeacherService) GeneratePersonalizedTeaching(ctx context.Context, studentName string) *PersonalizedTeaching {
	if studentName == "" {
		studentName = "张明"
	}
	data := &PersonalizedTeaching{StudentName: studentName, LearningStyle: "动手实践型", WeakPoints: []string{"递归思想", "动态规划", "图论算法"}, Strategy: "建议增加编程练习量，用可视化工具辅助理解抽象概念。每学完一个算法立即用代码实现，对比不同解法的复杂度。", Resources: []string{"LeetCode 热题100", "《算法导论》动态规划章节", "VisuAlgo 可视化学习平台"}, DataSource: "fallback"}
	if s.llmClient != nil {
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: fmt.Sprintf("你是教学专家。学生%s，学习风格动手实践型，薄弱点在递归和动态规划。请给出50字个性化教学建议。", studentName)}}, Temperature: 0.4, MaxTokens: 300})
		if err == nil && resp != nil && resp.Content != "" {
			if refined := strings.TrimSpace(resp.Content); refined != "" {
				data.Strategy = refined
				data.DataSource = "ai"
			}
		}
	}
	return data
}
