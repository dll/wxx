package llm

import "context"

// ChatMessage LLM 对话消息
type ChatMessage struct {
	Role    string `json:"role"`    // system/user/assistant
	Content string `json:"content"` // 消息内容
}

// ChatRequest LLM 请求参数
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"` // 温度参数，0-2
	MaxTokens   int           `json:"max_tokens,omitempty"`  // 最大生成 token 数
}

// ChatResponse LLM 响应
type ChatResponse struct {
	Content      string // 生成的文本内容
	FinishReason string // 结束原因：stop/length
	PromptTokens int    // 输入 token 数
	OutputTokens int    // 输出 token 数
}

// ChatClient LLM 客户端接口
// 所有 LLM 提供商（DeepSeek、智谱、讯飞）均实现此接口
type ChatClient interface {
	// Chat 发起对话请求
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// Name 返回客户端名称（用于日志和审计）
	Name() string
}
