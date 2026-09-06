package service

import (
	"context"
	"fmt"

	"github.com/dll/wxx/server/internal/llm"
)

// MockInterview AI 模拟面试。
type MockInterview struct {
	Position   string                   `json:"position"`
	Questions  []map[string]interface{} `json:"questions"`
	Tips       []string                 `json:"tips"`
	Score      float64                  `json:"score"`
	DataSource string                   `json:"data_source"`
}

func (s *StudentService) GenerateMockInterview(ctx context.Context, position string) *MockInterview {
	if position == "" {
		position = "Java后端开发工程师"
	}
	interview := &MockInterview{Position: position, Questions: []map[string]interface{}{{"type": "自我介绍", "question": "请做一个简短的自我介绍（1分钟）", "tips": "突出技术栈和项目经验，保持自信"}, {"type": "技术基础", "question": "请解释TCP三次握手和四次挥手的过程", "tips": "画出时序图，解释每个状态转换"}, {"type": "项目经验", "question": "介绍一个你最熟悉的项目，遇到了什么技术挑战？", "tips": "使用STAR法则（情境/任务/行动/结果）"}, {"type": "算法", "question": "实现一个LRU缓存，包含get和put操作", "tips": "使用HashMap+双向链表，O(1)时间复杂度"}, {"type": "反问", "question": "你对我们团队有什么想问的吗？", "tips": "建议问技术栈/团队规模/成长空间"}}, Tips: []string{"提前了解公司业务和技术栈", "准备2-3个有亮点的项目经历", "练习白板编程，注意代码规范"}, Score: 85, DataSource: "fallback"}
	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是大厂面试官。请为「%s」岗位生成5道模拟面试题(含简短答题提示)。输出JSON格式。", position)
		if resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.5, MaxTokens: 600}); err == nil && resp != nil && resp.Content != "" {
			interview.Tips = append(interview.Tips, "AI补充："+resp.Content[:min(len(resp.Content), 100)])
			interview.DataSource = "ai"
		}
	}
	return interview
}
