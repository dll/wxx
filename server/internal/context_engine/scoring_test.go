package context_engine

import (
	"testing"
)

// TestComputeTrustScore_正常分数范围
func TestComputeTrustScore_正常分数范围(t *testing.T) {
	r := &SearchResult{Score: -10.0, ResourceType: "Policy"}
	score := computeTrustScore(r)
	if score < 0 || score > 1 {
		t.Errorf("TrustScore=%.3f 超出 [0,1] 范围", score)
	}
}

// TestComputeTrustScore_零分
func TestComputeTrustScore_零分(t *testing.T) {
	r := &SearchResult{Score: 0.0, ResourceType: "Policy"}
	score := computeTrustScore(r)
	if score != 0 {
		t.Errorf("Score=0 时 TrustScore 期望 0，得到 %.3f", score)
	}
}

// TestComputeTrustScore_类型权重 Policy > FAQ
func TestComputeTrustScore_类型权重(t *testing.T) {
	policy := &SearchResult{Score: -5.0, ResourceType: "Policy"}
	faq := &SearchResult{Score: -5.0, ResourceType: "FAQ"}
	if computeTrustScore(policy) <= computeTrustScore(faq) {
		t.Errorf("Policy 权重应大于 FAQ")
	}
}

// TestComputeTrustScore_未知类型不崩溃
func TestComputeTrustScore_未知类型不崩溃(t *testing.T) {
	r := &SearchResult{Score: -5.0, ResourceType: "Unknown"}
	score := computeTrustScore(r)
	if score < 0 {
		t.Errorf("未知类型 TrustScore 不应为负数，得到 %.3f", score)
	}
}

// TestSortByTrust_排序正确性
func TestSortByTrust_排序正确性(t *testing.T) {
	results := []*SearchResult{
		{Score: -2.0, TrustScore: 0.3, ResourceType: "FAQ"},
		{Score: -8.0, TrustScore: 0.8, ResourceType: "Policy"},
		{Score: -5.0, TrustScore: 0.5, ResourceType: "Process"},
	}
	sortByTrust(results)
	if results[0].TrustScore < results[1].TrustScore {
		t.Errorf("排序后第一项 TrustScore(%.2f) 应 >= 第二项(%.2f)", results[0].TrustScore, results[1].TrustScore)
	}
}

// TestSortByTrust_空列表不崩溃
func TestSortByTrust_空列表不崩溃(t *testing.T) {
	sortByTrust([]*SearchResult{})
}

// TestExtractSnippet_包含关键词
func TestExtractSnippet_包含关键词(t *testing.T) {
	content := "根据学校规定，奖学金评定需要满足以下条件：GPA不低于3.5，无违纪记录，积极参与校园活动。"
	snippet := extractSnippet(content, "奖学金条件")
	if snippet == "" {
		t.Error("含关键词的内容应返回非空片段")
	}
}

// TestExtractSnippet_空内容
func TestExtractSnippet_空内容(t *testing.T) {
	if extractSnippet("", "奖学金") != "" {
		t.Error("空内容应返回空字符串")
	}
}

// TestExtractKeywords_正常提取
func TestExtractKeywords_正常提取(t *testing.T) {
	kws := extractKeywords("奖学金申请条件和流程")
	if len(kws) == 0 {
		t.Error("应提取到至少一个关键词")
	}
}

// TestExtractKeywords_空字符串
func TestExtractKeywords_空字符串(t *testing.T) {
	kws := extractKeywords("")
	_ = kws // 不应崩溃，返回空 slice 即可
}
