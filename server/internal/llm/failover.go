package llm

import (
	"context"
	"log"
	"time"
)

// FailoverClient 双模型容灾客户端：主模型超时/失败自动切换到备选模型。
// 实现 ChatClient 接口，可无缝替换单模型客户端。
type FailoverClient struct {
	primary  ChatClient
	backup   ChatClient
	timeout  time.Duration // 单次调用超时
	maxRetry int           // 主模型失败后切换备选的最大尝试
}

// NewFailoverClient 创建容灾客户端。
// primary 主模型；backup 备选模型（可为 nil，此时退化为单模型+超时）。
// timeout 单次调用超时（默认 8s）；maxRetry 主模型失败切换次数（默认 1）。
func NewFailoverClient(primary, backup ChatClient, timeout time.Duration) *FailoverClient {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &FailoverClient{
		primary:  primary,
		backup:   backup,
		timeout:  timeout,
		maxRetry: 1,
	}
}

func (c *FailoverClient) Name() string {
	return "failover(" + c.primary.Name() + "→" + c.backupName() + ")"
}

func (c *FailoverClient) backupName() string {
	if c.backup == nil {
		return "none"
	}
	return c.backup.Name()
}

// Chat 带超时 + 失败切换
func (c *FailoverClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 第一次尝试主模型
	resp, err := c.try(c.primary, ctx, req)
	if err == nil {
		return resp, nil
	}
	log.Printf("主模型 %s 失败: %v，尝试备选 %s", c.primary.Name(), err, c.backupName())

	// 主模型失败 → 切备选
	if c.backup != nil {
		if resp2, err2 := c.try(c.backup, ctx, req); err2 == nil {
			return resp2, nil
		} else {
			log.Printf("备选模型 %s 也失败: %v", c.backup.Name(), err2)
		}
	}
	return nil, err
}

// Stream 带超时 + 失败切换（SSE 场景）
func (c *FailoverClient) Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	// 主模型流式
	ch, err := c.tryStream(c.primary, ctx, req)
	if err == nil {
		return ch, nil
	}
	log.Printf("主模型 %s 流式失败: %v，尝试备选 %s", c.primary.Name(), err, c.backupName())
	if c.backup != nil {
		if ch2, err2 := c.tryStream(c.backup, ctx, req); err2 == nil {
			return ch2, nil
		}
	}
	return nil, err
}

// try 带超时调用
func (c *FailoverClient) try(client ChatClient, ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return client.Chat(ctx, req)
}

// tryStream 带超时启动流式
func (c *FailoverClient) tryStream(client ChatClient, ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	ch, err := client.Stream(ctx, req)
	if err != nil {
		cancel()
		return nil, err
	}
	// 流式已启动后由上层消费，超时由调用方 ctx 控制
	_ = cancel
	return ch, nil
}
