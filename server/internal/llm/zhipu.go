package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
)

// ZhipuClient 智谱清言 API 客户端
// 兼容 OpenAI Chat Completions API 格式
type ZhipuClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewZhipuClient 创建智谱清言客户端实例
func NewZhipuClient(cfg *config.Config) *ZhipuClient {
	return &ZhipuClient{
		apiKey:  cfg.ZhipuAPIKey,
		baseURL: cfg.ZhipuBaseURL,
		model:   cfg.ZhipuModel,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Name 返回客户端名称
func (c *ZhipuClient) Name() string {
	return "zhipu"
}

// Chat 发起对话请求
func (c *ZhipuClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body := openAIRequest{
		Model:       c.model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
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
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 传播 TraceID
	if tid := middleware.GetTraceIDFromContext(ctx); tid != "" {
		httpReq.Header.Set("X-Trace-ID", tid)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("智谱 API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("智谱 API 错误 (HTTP %d): %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("智谱返回空结果")
	}

	return &ChatResponse{
		Content:      openAIResp.Choices[0].Message.Content,
		FinishReason: openAIResp.Choices[0].FinishReason,
		PromptTokens: openAIResp.Usage.PromptTokens,
		OutputTokens: openAIResp.Usage.CompletionTokens,
	}, nil
}
