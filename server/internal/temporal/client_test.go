package temporal

import (
	"testing"

	"github.com/dll/wxx/server/internal/config"
)

func TestNew_EmptyHostPort_ReturnsNil(t *testing.T) {
	cfg := &config.Config{
		TemporalHostPort:  "",
		TemporalNamespace: "wxx",
		TemporalTaskQueue: "wxx-critical",
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("空 HostPort 不应返回错误，得到: %v", err)
	}
	if client != nil {
		t.Error("空 HostPort 应返回 nil client（降级模式）")
	}
}

func TestNew_InvalidHostPort_ReturnsError(t *testing.T) {
	cfg := &config.Config{
		TemporalHostPort:  "localhost:19999", // 无服务监听
		TemporalNamespace: "wxx",
		TemporalTaskQueue: "wxx-critical",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("无效 HostPort 应返回连接错误")
	}
}
