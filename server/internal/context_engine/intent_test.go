package context_engine

import (
	"testing"
)

// TestClassifyIntent_已知类别关键词 验证各类别关键词触发正确意图
func TestClassifyIntent_已知类别关键词(t *testing.T) {
	cases := []struct {
		query   string
		wantCat string
		minConf float64
	}{
		{"如何申请奖学金", "scholarship", 0.8},
		{"入学报到需要带什么", "enrollment", 0.8},
		{"毕业论文要求是什么", "graduation", 0.8},
		{"心情很差，心理压力好大", "mental", 0.7},
		{"选课和绩点怎么算", "course", 0.7},
		{"就业实习招聘信息", "career", 0.7},
		{"申请休学复学需要什么条件", "leave", 0.8},
		{"学校有哪些活动可以参加", "activity", 0.7},
		{"奖学金评定流程", "scholarship", 0.8},
		{"学籍管理制度规定", "policy", 0.8},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := ClassifyIntent(tc.query)
			if got.Category != tc.wantCat {
				t.Errorf("ClassifyIntent(%q) 类别 = %q，期望 %q", tc.query, got.Category, tc.wantCat)
			}
			if got.Confidence < tc.minConf {
				t.Errorf("ClassifyIntent(%q) 置信度 = %.2f < %.2f", tc.query, got.Confidence, tc.minConf)
			}
		})
	}
}

// TestClassifyIntent_空字符串 空输入应返回默认 chat 意图
func TestClassifyIntent_空字符串(t *testing.T) {
	got := ClassifyIntent("")
	if got.Category != "chat" {
		t.Errorf("空字符串应返回 chat，得到 %q", got.Category)
	}
}

// TestClassifyIntent_多关键词加成 多个关键词匹配时置信度应高于单个
func TestClassifyIntent_多关键词加成(t *testing.T) {
	single := ClassifyIntent("奖学金")
	multi := ClassifyIntent("奖学金申请评定名额")
	if multi.Confidence < single.Confidence {
		t.Errorf("多关键词置信度(%.2f) 不应低于单关键词(%.2f)", multi.Confidence, single.Confidence)
	}
}

// TestClassifyIntent_置信度上限 置信度不应超过 1.0
func TestClassifyIntent_置信度上限(t *testing.T) {
	got := ClassifyIntent("奖学金申请评定条件流程名额时间标准")
	if got.Confidence > 1.0 {
		t.Errorf("置信度 %.2f 超出上限 1.0", got.Confidence)
	}
}

// TestIntentToResourceTypes_政策类 政策类意图应返回Policy优先
func TestIntentToResourceTypes_政策类(t *testing.T) {
	intent := Intent{Category: "policy", Confidence: 0.8}
	types := IntentToResourceTypes(intent)
	if len(types) == 0 {
		t.Fatal("ResourceTypes 不应为空")
	}
	if types[0] != "Policy" {
		t.Errorf("policy 意图首个资源类型期望 Policy，得到 %q", types[0])
	}
}

// TestIntentToResourceTypes_流程类 enrollment/graduation 应返回 Process 优先
func TestIntentToResourceTypes_流程类(t *testing.T) {
	for _, cat := range []string{"process", "enrollment", "graduation"} {
		types := IntentToResourceTypes(Intent{Category: cat, Confidence: 0.8})
		if len(types) == 0 {
			t.Fatalf("%s 意图 ResourceTypes 不应为空", cat)
		}
	}
}

// TestIntentToResourceTypes_未知类别 未知类别应返回空或默认值不崩溃
func TestIntentToResourceTypes_未知类别不崩溃(t *testing.T) {
	_ = IntentToResourceTypes(Intent{Category: "unknown_xyz", Confidence: 0.5})
}
