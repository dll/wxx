package context_engine

import (
	"strings"
	"testing"
)

func TestRewriteQuery_StripsPrefix(t *testing.T) {
	got := RewriteQuery("请问，帮忙告诉我想了解一下奖学金怎么申请？", "")
	if strings.Contains(got, "请问") || strings.Contains(got, "帮忙") || strings.Contains(got, "想了解") {
		t.Fatalf("装饰词未剥离: %q", got)
	}
	if !strings.Contains(got, "奖学金") {
		t.Fatalf("核心词丢失: %q", got)
	}
}

func TestRewriteQuery_Coreference(t *testing.T) {
	got := RewriteQuery("它需要哪些材料", "奖学金评定流程是什么")
	if !strings.Contains(got, "奖学金") {
		t.Fatalf("指代消解未拼接历史话题: %q", got)
	}
	if !strings.Contains(got, "材料") {
		t.Fatalf("当前问题丢失: %q", got)
	}
}

func TestRewriteQuery_ShortQuery(t *testing.T) {
	got := RewriteQuery("在哪办", "学生证补办流程")
	if !strings.Contains(got, "学生证") {
		t.Fatalf("短问题未补全话题: %q", got)
	}
}

func TestRewriteQuery_NoHistory(t *testing.T) {
	got := RewriteQuery("请假流程是什么", "")
	if got != "请假流程是什么" {
		t.Fatalf("无历史时应保持原样: %q", got)
	}
}

func TestApplyIntentBoost(t *testing.T) {
	intent := Intent{Category: "process", Confidence: 0.9}
	results := []*SearchResult{
		{ResourceType: "Process", TrustScore: 0.5},
		{ResourceType: "Activity", TrustScore: 0.5},
	}
	applyIntentBoost(results, intent)
	if results[0].TrustScore <= 0.5 {
		t.Fatalf("偏好类型未加权: %v", results[0].TrustScore)
	}
	if results[1].TrustScore != 0.5 {
		t.Fatalf("非偏好类型不应变化: %v", results[1].TrustScore)
	}
}
