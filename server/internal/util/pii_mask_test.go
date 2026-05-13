package util

import "testing"

func TestMaskPII_StudentID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		masked   bool
	}{
		{"学号 2024123456 请查询", "学号 2024**56 请查询", true},
		{"工号 20240001", "工号 2024**01", true},
		{"普通数字 123", "普通数字 123", false},
	}

	for _, tt := range tests {
		result, masked := MaskPII(tt.input)
		if result != tt.expected || masked != tt.masked {
			t.Errorf("MaskPII(%q) = (%q, %v), want (%q, %v)",
				tt.input, result, masked, tt.expected, tt.masked)
		}
	}
}

func TestMaskPII_Phone(t *testing.T) {
	input := "联系我 13812345678 或者 15900001111"
	result, masked := MaskPII(input)
	if !masked {
		t.Error("phone number should be masked")
	}
	if result == input {
		t.Error("phone number was not masked")
	}
	// 检查第一个手机号被脱敏
	expectedPart := "138****5678"
	if !contains(result, expectedPart) {
		t.Errorf("expected masked phone %q in result %q", expectedPart, result)
	}
}

func TestMaskPII_IDCard(t *testing.T) {
	input := "身份证号 320102199001011234 请核对"
	result, masked := MaskPII(input)
	if !masked {
		t.Error("ID card should be masked")
	}
	expectedPart := "320102****1234"
	if !contains(result, expectedPart) {
		t.Errorf("expected masked ID %q in result %q", expectedPart, result)
	}
}

func TestMaskPII_NoPII(t *testing.T) {
	input := "奖学金什么时候发放？需要什么条件？"
	result, masked := MaskPII(input)
	if masked {
		t.Error("should not mask non-PII text")
	}
	if result != input {
		t.Errorf("result should match input: %q != %q", result, input)
	}
}

func TestSanitizeForLLM(t *testing.T) {
	input := "  学号 2024123456 申请奖学金，手机 13812345678  "
	result := SanitizeForLLM(input, 500)
	if result == input {
		t.Error("sanitization should have modified input")
	}
	// 不应含原始学号
	if contains(result, "2024123456") {
		t.Error("sanitized text should not contain raw student ID")
	}
	// 不应含原始手机号
	if contains(result, "13812345678") {
		t.Error("sanitized text should not contain raw phone")
	}
	// 不应有首尾空白
	if result != SanitizeForLLM(input, 500) {
		// 仅验证无空白
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
