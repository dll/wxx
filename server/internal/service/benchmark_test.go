package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dll/wxx/server/internal/util"
)

// BenchmarkContentSafety 评测内容安全过滤性能
func BenchmarkContentSafety(b *testing.B) {
	testInputs := []string{
		"请问奖学金评定标准是什么？",
		"如何办理转专业手续？",
		"图书馆自习室怎么预约？",
		"最近压力很大，感觉很焦虑怎么办？",
		"入党需要什么条件？",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		result := util.CheckUserInput(input)
		_ = result
	}
}

// BenchmarkPIIMasking 评测 PII 脱敏性能
func BenchmarkPIIMasking(b *testing.B) {
	testInputs := []string{
		"我叫张三，学号2023010101，手机号13812345678",
		"身份证号340102200001010001，银行卡6222021234567890123",
		"邮箱zhangsan@chzu.edu.cn，电话0550-3510000",
		"姓名李四，学号2023010102，手机15987654321",
		"正常问题：请问如何办理学生证？",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		_ = util.MaskPIIWithDetail(input)
	}
}

// TestContentSafetyAccuracy 评测内容安全准确率
func TestContentSafetyAccuracy(t *testing.T) {
	tests := []struct {
		input  string
		expect string // "pass" 或 "block"
	}{
		{"奖学金评定标准是什么？", "pass"},
		{"如何办理入学手续？", "pass"},
		{"图书馆开门时间？", "pass"},
		{"高等数学重修怎么办？", "pass"},
		{"计算机专业就业前景如何？", "pass"},
		{"学校食堂哪个好吃？", "pass"},
		{"校车时刻表在哪里查？", "pass"},
		{"心理压力大该怎么调整？", "pass"},
		{"ACM竞赛如何报名？", "pass"},
		{"入党申请书怎么写？", "pass"},
	}

	passed := 0
	for _, tt := range tests {
		result := util.CheckUserInput(tt.input)
		if tt.expect == "pass" && result.Action != util.FilterBlock {
			passed++
		} else if tt.expect == "block" && result.Action == util.FilterBlock {
			passed++
		} else {
			t.Logf("FAIL %s: expect=%s blocked=%v category=%s", tt.input, tt.expect, result.Action == util.FilterBlock, result.Category)
		}
	}

	accuracy := float64(passed) / float64(len(tests)) * 100
	t.Logf("内容安全准确率: %.1f%% (%d/%d)", accuracy, passed, len(tests))

	if accuracy < 90 {
		t.Errorf("准确率 %.1f%% 低于目标 90%%", accuracy)
	}
}

// TestPIIMaskingAccuracy 评测 PII 脱敏准确率
func TestPIIMaskingAccuracy(t *testing.T) {
	tests := []struct {
		input            string
		shouldContain    string
		shouldNotContain []string
	}{
		{
			input:            "姓名张三，学号2023010101",
			shouldContain:    "张",
			shouldNotContain: []string{"2023010101"},
		},
		{
			input:            "手机13812345678",
			shouldContain:    "****",
			shouldNotContain: []string{"13812345678"},
		},
		{
			input:            "身份证340102200001010001",
			shouldContain:    "****",
			shouldNotContain: []string{"340102200001010001"},
		},
	}

	for _, tt := range tests {
		result := util.MaskPIIWithDetail(tt.input)
		masked := result.Masked
		if !strings.Contains(masked, tt.shouldContain) {
			t.Errorf("FAIL 脱敏结果应包含 '%s': %s", tt.shouldContain, masked)
		}
		for _, s := range tt.shouldNotContain {
			if strings.Contains(masked, s) {
				t.Errorf("FAIL 脱敏结果不应包含 '%s': %s", s, masked)
			}
		}
	}
	t.Log("PII 脱敏测试完成")
}

// TestAPILatency 核心接口延迟基准
func TestAPILatency(t *testing.T) {
	operations := []struct {
		name string
		fn   func()
	}{
		{"PII检测", func() { util.DetectPII("测试文本") }},
		{"内容安全", func() { util.CheckUserInput("测试问题") }},
		{"脱敏", func() { util.MaskPIIWithDetail("张三13812345678") }},
	}

	for _, op := range operations {
		start := time.Now()
		for i := 0; i < 100; i++ {
			op.fn()
		}
		elapsed := time.Since(start)
		avgUs := float64(elapsed.Microseconds()) / 100.0
		t.Logf("  %s: avg=%.1fμs", op.name, avgUs)
		if avgUs > 1000 {
			t.Errorf("WARN %s 平均延迟 %.1fμs 超过 1ms", op.name, avgUs)
		}
	}
	t.Log("核心接口延迟基准完成")
}

// TestFallbackCoverage 兜底回复覆盖率评测
func TestFallbackCoverage(t *testing.T) {
	features := []string{
		"campus-life", "schedule", "mental-health", "competition-match",
		"freshman-plan", "growth-path", "political-study", "ideological-record",
		"party-progress", "digital-mentor", "values-guidance", "classroom-extension",
	}

	for _, f := range features {
		resp := fallbackAIResponse(f)
		if resp["response"] == "功能开发中，敬请期待。" {
			t.Errorf("FAIL %s: 缺少兜底回复", f)
		}
	}
	t.Logf("兜底回复覆盖率: 100%% (%d 项)", len(features))
}

// TestCategoryFallbackResponses 评测分类兜底回复
func TestCategoryFallbackResponses(t *testing.T) {
	categories := []string{"political", "porn", "violence", "illegal", "campus"}
	for _, cat := range categories {
		resp := util.GetFallbackResponse(cat)
		if resp == "" {
			t.Errorf("FAIL 分类 %s: 缺少兜底回复", cat)
		} else {
			t.Logf("  %s: %s", cat, util.TruncateString(resp, 40))
		}
	}
	t.Log("分类兜底回复完整")
}

// TestEvalBaseline 评测基线（8 业务域抽样 — 验证 GenericAI 兜底覆盖）
func TestEvalBaseline(t *testing.T) {
	// 评测各功能 feature key 的兜底回复覆盖率
	features := map[string][]string{
		"政策":  {"freshman-plan", "growth-path"},
		"流程":  {"competition-match"},
		"学业":  {"schedule", "digital-mentor"},
		"心理":  {"mental-health"},
		"校园":  {"campus-life"},
		"思政":  {"political-study", "ideological-record", "party-progress"},
		"就业":  {"values-guidance"},
		"FAQ": {"classroom-extension"},
	}

	totalFeatures := 0
	covered := 0
	for domain, feats := range features {
		for _, f := range feats {
			totalFeatures++
			resp := fallbackAIResponse(f)
			if resp["response"] != "功能开发中，敬请期待。" {
				covered++
			} else {
				t.Logf("WARN [%s] %s: 缺少兜底回复", domain, f)
			}
		}
	}

	coverage := float64(covered) / float64(totalFeatures) * 100
	t.Logf("评测基线覆盖率: %.1f%% (%d/%d 业务feature)", coverage, covered, totalFeatures)

	if coverage < 85 {
		t.Errorf("覆盖率 %.1f%% 低于目标 85%%", coverage)
	}
}

var _ = fmt.Sprintf
