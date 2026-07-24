package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
)

// TestEmotionAgent_CrisisMode 危机关键词必须命中兜底安抚回复，不依赖 LLM
func TestEmotionAgent_CrisisMode(t *testing.T) {
	a := NewEmotionAgent(nil) // 不依赖 LLM
	userCtx := &model.UserContext{UserID: 1, Role: "student"}

	result, err := a.Execute(context.Background(), "我活不下去了", userCtx, nil)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}
	if !strings.Contains(result.Content, "心理援助") {
		t.Errorf("危机模式应返回求助渠道，实际：%s", result.Content)
	}
	if result.Confidence < 0.99 {
		t.Errorf("危机模式置信度应为 1.0，实际 %.2f", result.Confidence)
	}
}

// TestEmotionAgent_FallbackWithoutLLM LLM 不可用时应返回温和兜底语
func TestEmotionAgent_FallbackWithoutLLM(t *testing.T) {
	a := NewEmotionAgent(nil)
	userCtx := &model.UserContext{UserID: 1, Role: "student"}

	result, err := a.Execute(context.Background(), "最近压力很大睡不好", userCtx, nil)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}
	if result.Content == "" {
		t.Error("LLM 不可用时仍应返回兜底回复")
	}
	if result.Confidence > 0.5 {
		t.Errorf("LLM 兜底置信度应较低，实际 %.2f", result.Confidence)
	}
}

// TestEmotionAgent_KeyAndName Key/Name 与 Router 路由对齐
func TestEmotionAgent_KeyAndName(t *testing.T) {
	a := NewEmotionAgent(nil)
	if a.Key() != "emotion-counselor" {
		t.Errorf("Key 应为 emotion-counselor，实际 %s", a.Key())
	}
	if a.Name() == "" {
		t.Error("Name 不应为空")
	}
}

// TestRouterMatchesAgentKey 验证路由返回的 Key 与所有 Agent 的 Key 对齐（防止再次出现 key 不匹配 bug）
func TestRouterMatchesAgentKey(t *testing.T) {
	registered := map[string]bool{
		(&QAAgent{}).Key():           true,
		(&PolicyAgent{}).Key():       true,
		(&ProcessAgent{}).Key():      true,
		(NewEmotionAgent(nil)).Key(): true,
	}

	router := NewRouter()
	for _, q := range []string{
		"奖学金的申请政策是什么", // policy
		"如何办理休学",      // process
		"焦虑睡不好",       // emotion
		"你好",          // qa-default
	} {
		names := router.Route(q)
		for _, n := range names {
			if !registered[n] {
				t.Errorf("路由返回的 Key=%s 没有对应已注册的 Agent，问题：%s", n, q)
			}
		}
	}
}
