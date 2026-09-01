package llm

import "context"

// ChatMessage LLM 对话消息
type ChatMessage struct {
	Role       string     `json:"role"`                   // system/user/assistant/tool
	Content    string     `json:"content"`                // 消息内容
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // assistant 发起的工具调用（回放用）
	ToolCallID string     `json:"tool_call_id,omitempty"` // role=tool 时对应调用 ID
}

// ToolDefinition 工具定义（OpenAI function calling 格式）
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"` // JSON Schema
}

// ToolFunction 请求体中的工具包装
type ToolFunction struct {
	Type     string         `json:"type"` // 固定 "function"
	Function ToolDefinition `json:"function"`
}

// ToolCall 模型发起的单次工具调用
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串（如 {"question":"..."}）
}

// ChatRequest LLM 请求参数
type ChatRequest struct {
	Messages    []ChatMessage  `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"` // 温度参数，0-2
	MaxTokens   int            `json:"max_tokens,omitempty"`  // 最大生成 token 数
	APIKey      string         `json:"-"`                     // 可选：用户自备 Key 覆盖（额度耗尽/自配场景）
	Model       string         `json:"-"`                     // 可选：用户配置的模型名覆盖（如 deepseek-v4-flash）
	Tools       []ToolFunction `json:"-"`                     // 可选：function calling 工具清单（A3）
}

// ChatResponse LLM 响应
type ChatResponse struct {
	Content      string     // 生成的文本内容
	FinishReason string     // 结束原因：stop/length/tool_calls
	PromptTokens int        // 输入 token 数
	OutputTokens int        // 输出 token 数
	ToolCalls    []ToolCall // 模型请求的工具调用（FinishReason=tool_calls 时非空）
}

// ChatClient LLM 客户端接口
// 所有 LLM 提供商（DeepSeek、智谱、讯飞）均实现此接口
type ChatClient interface {
	// Chat 发起对话请求
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// Stream 发起流式对话请求，逐块返回增量文本
	Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)

	// Name 返回客户端名称（用于日志和审计）
	Name() string

	// Model 返回当前使用的模型名（如 deepseek-v4-flash / glm-4），用于回答标注
	Model() string
}

// StreamChunk 流式响应增量
type StreamChunk struct {
	Delta   string // 增量文本
	Done    bool   // 是否为结束标记
	Content string // Done=true 时携带完整内容（含 usage 兜底场景）
}

// openAIStreamResponse OpenAI 兼容的流式响应块（data: {...}）
type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
