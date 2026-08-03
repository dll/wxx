package llm

import "context"

// MockClient 模拟 LLM 客户端，用于测试
type MockClient struct {
	ChatFunc func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	name     string
}

// NewMockClient 创建模拟 LLM 客户端
func NewMockClient(name string) *MockClient {
	return &MockClient{name: name}
}

// Chat 调用模拟响应函数，若未设置则返回默认内容
func (m *MockClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, req)
	}
	// 默认行为：返回简单回答
	return &ChatResponse{
		Content:      "这是模拟的回答内容。",
		FinishReason: "stop",
		PromptTokens: 100,
		OutputTokens: 50,
	}, nil
}

// Name 返回客户端名称
func (m *MockClient) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

// Stream 模拟流式：一次性返回全部内容
func (m *MockClient) Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 2)
	go func() {
		defer close(ch)
		resp, _ := m.Chat(ctx, req)
		content := "这是模拟的回答内容。"
		if resp != nil && resp.Content != "" {
			content = resp.Content
		}
		// 分块发送，模拟增量
		for _, r := range content {
			ch <- StreamChunk{Delta: string(r)}
		}
		ch <- StreamChunk{Done: true, Content: content}
	}()
	return ch, nil
}

// Reset 重置 ChatFunc（用于不同测试用例）
func (m *MockClient) Reset() {
	m.ChatFunc = nil
}
