package activities

import (
	"testing"
)

func TestParseEmotionResponse_Valid(t *testing.T) {
	response := `{"score": -0.8, "risk_level": "high", "emotions": ["焦虑","抑郁"], "keywords": ["压力","失眠"], "reasoning": "学生表现出明显的负面情绪", "need_follow_up": true}`

	result, err := parseEmotionResponse(response)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.Score != -0.8 {
		t.Errorf("期望 Score=-0.8，得到 %f", result.Score)
	}
	if result.RiskLevel != "high" {
		t.Errorf("期望 RiskLevel=high，得到 %s", result.RiskLevel)
	}
	if len(result.Emotions) != 2 {
		t.Errorf("期望 2 个情绪，得到 %d", len(result.Emotions))
	}
	if !result.NeedFollowUp {
		t.Error("期望 need_follow_up=true")
	}
}

func TestParseEmotionResponse_WithMarkdownWrapper(t *testing.T) {
	// LLM 有时会在 JSON 外包裹 markdown 代码块
	response := "```json\n{\"score\": 0.3, \"risk_level\": \"low\", \"emotions\": [\"平静\"], \"keywords\": [], \"reasoning\": \"正常交流\", \"need_follow_up\": false}\n```"

	result, err := parseEmotionResponse(response)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.RiskLevel != "low" {
		t.Errorf("期望 RiskLevel=low，得到 %s", result.RiskLevel)
	}
	if result.Score != 0.3 {
		t.Errorf("期望 Score=0.3，得到 %f", result.Score)
	}
}

func TestParseEmotionResponse_InvalidRiskLevel(t *testing.T) {
	response := `{"score": 0, "risk_level": "unknown", "emotions": [], "keywords": [], "reasoning": "", "need_follow_up": false}`

	result, err := parseEmotionResponse(response)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.RiskLevel != "low" {
		t.Errorf("无效 risk_level 应默认为 low，得到 %s", result.RiskLevel)
	}
}

func TestParseEmotionResponse_OutOfRangeScore(t *testing.T) {
	response := `{"score": 10.0, "risk_level": "high", "emotions": [], "keywords": [], "reasoning": "", "need_follow_up": true}`

	result, err := parseEmotionResponse(response)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.Score > 1.0 {
		t.Errorf("Score 应限制在 1.0 以内，得到 %f", result.Score)
	}

	// 测试低于下限
	response2 := `{"score": -5.0, "risk_level": "urgent", "emotions": [], "keywords": [], "reasoning": "", "need_follow_up": true}`
	result2, err := parseEmotionResponse(response2)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result2.Score < -1.0 {
		t.Errorf("Score 应限制在 -1.0 以上，得到 %f", result2.Score)
	}
}

func TestParseEmotionResponse_InvalidJSON(t *testing.T) {
	_, err := parseEmotionResponse("这不是 JSON")
	if err == nil {
		t.Error("无效 JSON 应返回错误")
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("true 应转为 1")
	}
	if boolToInt(false) != 0 {
		t.Error("false 应转为 0")
	}
}
