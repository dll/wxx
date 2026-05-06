package temporal

import (
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/config"
	"go.temporal.io/sdk/client"
)

// Client Temporal 客户端单例包装
type Client struct {
	cli client.Client
	cfg *config.Config
}

// New 创建 Temporal 客户端。hostPort 为空时返回 nil（优雅降级）。
func New(cfg *config.Config) (*Client, error) {
	if cfg.TemporalHostPort == "" {
		log.Println("Temporal 未配置（TEMPORAL_HOST_PORT 为空），工作流引擎已禁用")
		return nil, nil
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalHostPort,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 Temporal 失败: %w", err)
	}

	log.Printf("Temporal 已连接: %s (namespace=%s)", cfg.TemporalHostPort, cfg.TemporalNamespace)
	return &Client{cli: c, cfg: cfg}, nil
}

// SDKClient 返回底层 Temporal SDK client（供 workflow execution 使用）
func (c *Client) SDKClient() client.Client { return c.cli }

// TaskQueue 返回配置的任务队列名
func (c *Client) TaskQueue() string { return c.cfg.TemporalTaskQueue }

// Close 关闭连接
func (c *Client) Close() {
	if c.cli != nil {
		c.cli.Close()
	}
}
