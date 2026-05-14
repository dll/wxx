package agent

import (
	"testing"
)

func TestRouter_Route_PolicyIntent(t *testing.T) {
	r := NewRouter()
	agents := r.Route("奖学金申请条件是什么")
	if !contains(agents, "policy-expert") {
		t.Errorf("期望路由到 policy-expert，实际: %v", agents)
	}
}

func TestRouter_Route_ProcessIntent(t *testing.T) {
	r := NewRouter()
	agents := r.Route("休学怎么办理，需要什么材料")
	if !contains(agents, "process-guide") {
		t.Errorf("期望路由到 process-guide，实际: %v", agents)
	}
}

func TestRouter_Route_EmotionIntent(t *testing.T) {
	r := NewRouter()
	agents := r.Route("最近很焦虑失眠怎么办")
	if !contains(agents, "emotion-counselor") {
		t.Errorf("期望路由到 emotion-counselor，实际: %v", agents)
	}
}

func TestRouter_Route_FAQIntent(t *testing.T) {
	r := NewRouter()
	agents := r.Route("教务处电话是什么")
	if !contains(agents, "qa-default") {
		t.Errorf("期望路由到 qa-default，实际: %v", agents)
	}
}

func TestRouter_Route_UnknownFallback(t *testing.T) {
	r := NewRouter()
	agents := r.Route("你好")
	if len(agents) != 1 || agents[0] != "qa-default" {
		t.Errorf("未知意图应回退到 qa-default，实际: %v", agents)
	}
}

func TestRouter_Route_MultiIntent(t *testing.T) {
	r := NewRouter()
	agents := r.Route("转专业的政策规定和办理流程")
	if len(agents) < 2 {
		t.Errorf("期望多意图路由到多个 Agent，实际: %v", agents)
	}
	if !contains(agents, "policy-expert") || !contains(agents, "process-guide") {
		t.Errorf("期望同时路由到 policy-expert 和 process-guide，实际: %v", agents)
	}
}

func TestRouter_Route_NoDuplicates(t *testing.T) {
	r := NewRouter()
	agents := r.Route("活动报名参加社团")
	seen := make(map[string]bool)
	for _, a := range agents {
		if seen[a] {
			t.Errorf("路由结果存在重复 Agent: %s", a)
		}
		seen[a] = true
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
