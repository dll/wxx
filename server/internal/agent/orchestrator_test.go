package agent

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// mockAgent 用于测试的模拟 Agent
type mockAgent struct {
	name       string
	content    string
	confidence float64
}

func (m *mockAgent) Name() string { return m.name }

func (m *mockAgent) Execute(_ context.Context, _ string, _ *model.UserContext, _ *repository.KBRepo) (*AgentResult, error) {
	return &AgentResult{
		AgentName:  m.name,
		Content:    m.content,
		Confidence: m.confidence,
	}, nil
}

func TestOrchestrator_Register(t *testing.T) {
	o := &Orchestrator{
		router: NewRouter(),
		merger: NewMerger(),
		agents: make(map[string]Agent),
	}
	mock := &mockAgent{name: "test-agent", content: "测试", confidence: 0.9}
	o.Register(mock)
	if _, ok := o.agents["test-agent"]; !ok {
		t.Error("注册 Agent 后应能通过名称查找")
	}
}

func TestOrchestrator_ExecuteParallel(t *testing.T) {
	o := &Orchestrator{
		router: NewRouter(),
		merger: NewMerger(),
		agents: make(map[string]Agent),
	}
	o.Register(&mockAgent{name: "qa-default", content: "QA回答", confidence: 0.7})
	o.Register(&mockAgent{name: "policy-expert", content: "政策回答", confidence: 0.9})

	userCtx := &model.UserContext{
		UserID:     1,
		Role:       "student",
		OwnerScope: "college",
		OwnerID:    "cs",
	}

	result, err := o.Execute(context.Background(), "奖学金政策是什么", userCtx)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}
	if result == nil {
		t.Fatal("结果不应为 nil")
	}
	if result.AgentCount == 0 {
		t.Error("应至少有一个 Agent 参与")
	}
}

func TestOrchestrator_UnknownAgentSkipped(t *testing.T) {
	o := &Orchestrator{
		router: NewRouter(),
		merger: NewMerger(),
		agents: make(map[string]Agent),
	}
	userCtx := &model.UserContext{UserID: 1, Role: "student"}
	result, err := o.Execute(context.Background(), "你好", userCtx)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}
	if result.Content == "" {
		t.Error("无 Agent 可用时应返回兜底回复")
	}
}

func TestTruncateForLog(t *testing.T) {
	short := "短文本"
	if truncateForLog(short) != short {
		t.Error("短文本不应被截断")
	}
	long := "这是一段超过五十个字符的长文本用于测试截断功能是否正常工作这里需要足够多的字符来触发截断逻辑"
	result := truncateForLog(long)
	runes := []rune(result)
	if len(runes) > 54 {
		t.Errorf("长文本应被截断，实际长度: %d", len(runes))
	}
}
