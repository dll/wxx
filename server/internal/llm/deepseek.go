package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
)

// DeepSeekClient DeepSeek API 客户端
// 兼容 OpenAI Chat Completions API 格式
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewDeepSeekClient 创建 DeepSeek 客户端实例
func NewDeepSeekClient(cfg *config.Config) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey:  cfg.DeepSeekAPIKey,
		baseURL: cfg.DeepSeekBaseURL,
		model:   cfg.DeepSeekModel,
		client: &http.Client{
			Timeout: 60 * time.Second, // LLM 响应可能较慢
		},
	}
}

// Name 返回客户端名称
func (c *DeepSeekClient) Name() string {
	return "deepseek"
}

// Model 返回当前使用的模型名（如 deepseek-v4-flash）
func (c *DeepSeekClient) Model() string {
	return c.model
}

// Chat 发起对话请求
func (c *DeepSeekClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 用户自配模型/Key 覆盖（默认用客户端构造时的服务器配置）
	model := c.model
	if req.Model != "" {
		model = req.Model
	}
	apiKey := c.apiKey
	if req.APIKey != "" {
		apiKey = req.APIKey
	}

	// 构造 OpenAI 兼容的请求体
	body := openAIRequest{
		Model:       model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Tools:       req.Tools,
	}

	// 设置默认值
	if body.Temperature == 0 {
		body.Temperature = 0.7
	}
	if body.MaxTokens == 0 {
		body.MaxTokens = 2048
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 传播 TraceID
	if tid := middleware.GetTraceIDFromContext(ctx); tid != "" {
		httpReq.Header.Set("X-Trace-ID", tid)
	}

	// 发送请求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 非 200 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DeepSeek API 错误 (HTTP %d): %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	// 解析响应
	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("DeepSeek 返回空结果")
	}

	return &ChatResponse{
		Content:      openAIResp.Choices[0].Message.Content,
		FinishReason: openAIResp.Choices[0].FinishReason,
		PromptTokens: openAIResp.Usage.PromptTokens,
		OutputTokens: openAIResp.Usage.CompletionTokens,
		ToolCalls:    openAIResp.Choices[0].Message.ToolCalls,
	}, nil
}

// Stream 发起流式对话请求（OpenAI 兼容 SSE）
func (c *DeepSeekClient) Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	// 用户自配模型/Key 覆盖（默认用客户端构造时的服务器配置）
	model := c.model
	if req.Model != "" {
		model = req.Model
	}
	apiKey := c.apiKey
	if req.APIKey != "" {
		apiKey = req.APIKey
	}

	body := openAIRequest{
		Model:       model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	}
	if body.Temperature == 0 {
		body.Temperature = 0.7
	}
	if body.MaxTokens == 0 {
		body.MaxTokens = 2048
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	if tid := middleware.GetTraceIDFromContext(ctx); tid != "" {
		httpReq.Header.Set("X-Trace-ID", tid)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek 流式请求失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("DeepSeek API 错误 (HTTP %d): %s", resp.StatusCode, truncate(string(body), 500))
	}

	return parseOpenAIStream(ctx, resp.Body), nil
}

// parseOpenAIStream 逐行解析 OpenAI 兼容 SSE（data: {...} / data: [DONE]），
// 每收到一个 delta 就推送到 channel；上下文取消或流结束时关闭 channel。
func parseOpenAIStream(ctx context.Context, r io.ReadCloser) <-chan StreamChunk {
	ch := make(chan StreamChunk, 8)
	go func() {
		defer close(ch)
		defer r.Close()

		// 发送辅助：消费方停止读取或 ctx 取消时不再阻塞发送，避免 goroutine/连接泄漏
		trySend := func(v StreamChunk) bool {
			select {
			case ch <- v:
				return true
			case <-ctx.Done():
				return false
			}
		}

		var full strings.Builder
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				trySend(StreamChunk{Done: true, Content: full.String()})
				return
			}
			var chunk openAIStreamResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					full.WriteString(delta)
					if !trySend(StreamChunk{Delta: delta}) {
						return
					}
				}
			}
		}
		trySend(StreamChunk{Done: true, Content: full.String()})
	}()
	return ch
}

// ── OpenAI 兼容的请求/响应结构 ──

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []ToolFunction  `json:"tools,omitempty"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// truncate 截断字符串用于错误日志
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
