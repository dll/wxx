package util

import "testing"

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"短于限制", "hello", 10, "hello"},
		{"等于限制", "hello", 5, "hello"},
		{"超过限制ASCII", "hello world", 8, "hello wo..."},
		{"中文短于限制", "你好世界", 10, "你好世界"},
		{"中文超过限制", "你好世界，这是测试文本", 4, "你好世界..."},
		{"空字符串", "", 5, ""},
		{"混合中英文超过限制", "hello你好world世界", 8, "hello你好w..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
