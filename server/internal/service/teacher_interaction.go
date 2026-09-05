package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dll/wxx/server/internal/llm"
)

// ClassInteraction 课堂互动问题
type ClassInteraction struct {
	Question     string   `json:"question"`
	Difficulty   string   `json:"difficulty"`
	ExpectedTime int      `json:"expected_time"`
	Hints        []string `json:"hints"`
	FollowUp     string   `json:"follow_up"`
	DataSource   string   `json:"data_source"`
}

func (s *TeacherService) GenerateInteraction(ctx context.Context, topic string) (*ClassInteraction, error) {
	if topic == "" {
		topic = "二叉树"
	}
	if s.llmClient != nil {
		interaction, err := s.generateInteractionWithLLM(ctx, topic)
		if err == nil && interaction != nil {
			return interaction, nil
		}
	}
	return s.fallbackInteraction(topic), nil
}

func (s *TeacherService) generateInteractionWithLLM(ctx context.Context, topic string) (*ClassInteraction, error) {
	prompt := fmt.Sprintf("你是一位高校教师，正在教授「%s」。请设计一个课堂互动问题。\n要求：中等难度、3分钟回答时间、提供2个提示、1个追问。\n严格按以下 JSON 结构输出：{\"question\":\"...\",\"difficulty\":\"medium\",\"expected_time\":3,\"hints\":[\"...\",\"...\"],\"follow_up\":\"...\"}", topic)
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.5, MaxTokens: 400})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}
	var parsed struct {
		Question     string   `json:"question"`
		Difficulty   string   `json:"difficulty"`
		ExpectedTime int      `json:"expected_time"`
		Hints        []string `json:"hints"`
		FollowUp     string   `json:"follow_up"`
	}
	if err := json.Unmarshal([]byte(extractJSON(resp.Content)), &parsed); err != nil || parsed.Question == "" {
		return nil, fmt.Errorf("LLM 互动问题解析失败")
	}
	return &ClassInteraction{Question: parsed.Question, Difficulty: parsed.Difficulty, ExpectedTime: parsed.ExpectedTime, Hints: parsed.Hints, FollowUp: parsed.FollowUp, DataSource: "ai"}, nil
}

func (s *TeacherService) fallbackInteraction(topic string) *ClassInteraction {
	return &ClassInteraction{Question: fmt.Sprintf("请解释%s的核心原理及其应用场景", topic), Difficulty: "medium", ExpectedTime: 3, Hints: []string{"从基本定义出发", "联系实际应用场景"}, FollowUp: fmt.Sprintf("如果要优化%s的算法复杂度，你会从哪些方面入手？", topic), DataSource: "fallback"}
}
