package agent

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// stubTool 可注入行为的工具桩
type stubTool struct {
	name   string
	panics bool
}

func (s *stubTool) Name() string        { return s.name }
func (s *stubTool) Description() string { return "测试工具" }
func (s *stubTool) Execute(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	if s.panics {
		panic("boom")
	}
	return &ToolResult{Content: "ok"}, nil
}

func TestToolRegistry_RegisterAndRun(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&stubTool{name: "query_x"})
	reg.Register(&stubTool{name: "query_a"})

	// List 按名排序，输出稳定
	tools := reg.List()
	if len(tools) != 2 || tools[0].Name() != "query_a" {
		t.Fatalf("List 排序异常: %v", tools)
	}

	out, err := reg.Run(context.Background(), "query_x", ToolArgs{})
	if err != nil || out.Content != "ok" {
		t.Fatalf("Run 失败: %v %v", out, err)
	}
	if _, err := reg.Run(context.Background(), "no_such", ToolArgs{}); err == nil {
		t.Fatal("未知工具应报错")
	}
}

// stubAgent 可注入 panic 的 Agent 桩
type stubAgent struct {
	key    string
	panics bool
}

func (s *stubAgent) Key() string  { return s.key }
func (s *stubAgent) Name() string { return s.key }
func (s *stubAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	if s.panics {
		panic("agent boom")
	}
	return &AgentResult{AgentName: s.key, Confidence: 0.8}, nil
}

// TestOrchestrator_PanicRecovery Agent panic 必须被恢复，不影响其它 Agent 与主链路
func TestOrchestrator_PanicRecovery(t *testing.T) {
	o := NewOrchestrator(nil, nil)
	o.Register(&stubAgent{key: "good"})
	o.Register(&stubAgent{key: "bad", panics: true})

	// 路由到两个 Agent：panic 的那个应返回空结果，good 正常
	results := o.executeParallel(context.Background(), "测试问题", &model.UserContext{Role: "student"}, []string{"good", "bad"})
	if len(results) != 2 {
		t.Fatalf("应返回 2 个结果（含 panic 兜底空结果），实际 %d", len(results))
	}
	for _, r := range results {
		if r.AgentName == "bad" && r.Confidence != 0 {
			t.Fatalf("panic Agent 应返回空结果: %+v", r)
		}
	}
}
