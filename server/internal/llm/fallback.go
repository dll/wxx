package llm

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// ChainClient 按顺序尝试多个大模型客户端，前一个失败时自动切换下一个。
type ChainClient struct {
	clients []ChatClient
}

func NewChainClient(clients ...ChatClient) *ChainClient {
	filtered := make([]ChatClient, 0, len(clients))
	for _, c := range clients {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	return &ChainClient{clients: filtered}
}

func (c *ChainClient) Name() string {
	if len(c.clients) == 0 {
		return "none"
	}
	names := make([]string, 0, len(c.clients))
	for _, client := range c.clients {
		names = append(names, client.Name())
	}
	return strings.Join(names, "+")
}

func (c *ChainClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if len(c.clients) == 0 {
		return nil, fmt.Errorf("未配置可用的大模型客户端")
	}

	var lastErr error
	for i, client := range c.clients {
		resp, err := client.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if i < len(c.clients)-1 {
			log.Printf("LLM 客户端 %s 调用失败，切换备用模型: %v", client.Name(), err)
		}
	}

	return nil, fmt.Errorf("所有 LLM 客户端调用失败: %w", lastErr)
}
