// Package util 提供跨包共享的通用工具函数
package util

// TruncateString 按 rune 截断字符串，超出 maxLen 时附加 "..."
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
