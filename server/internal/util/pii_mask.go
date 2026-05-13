package util

import (
	"regexp"
	"strings"
)

// 预编译脱敏正则
var (
	// 学号：6-12 位数字（保留前 4 后 2，中间脱敏）
	studentIDRe = regexp.MustCompile(`\b(\d{4})\d{2,8}(\d{2})\b`)
	// 手机号：1 开头的 11 位数字（保留前 3 后 4）
	phoneRe = regexp.MustCompile(`\b(1\d{2})\d{4}(\d{4})\b`)
	// 身份证：18 位或 17 位+X（保留前 6 后 4）
	idCardRe = regexp.MustCompile(`\b(\d{6})\d{8,9}([\dXx]{4})\b`)
)

// MaskPII 对文本中的个人身份信息脱敏
// 返回脱敏后的文本和是否进行了脱敏
func MaskPII(text string) (string, bool) {
	masked := false
	original := text

	// 身份证（先处理，避免被其他规则误匹配）
	if idCardRe.MatchString(text) {
		text = idCardRe.ReplaceAllString(text, "$1****$2")
		masked = true
	}

	// 手机号
	if phoneRe.MatchString(text) {
		text = phoneRe.ReplaceAllString(text, "$1****$2")
		masked = true
	}

	// 学号
	if studentIDRe.MatchString(text) {
		text = studentIDRe.ReplaceAllString(text, "$1**$2")
		masked = true
	}

	// 未匹配到任何规则时返回原文
	if !masked {
		return original, false
	}
	return text, true
}

// MaskPIIStrict 严格模式：对未命中正则的数字串也做保守脱敏
// 将连续 6 位以上数字串中段替换为 ****
func MaskPIIStrict(text string) string {
	text, _ = MaskPII(text)
	// 连续 6 位以上纯数字且未被上述规则覆盖，保守脱敏
	strictRe := regexp.MustCompile(`\b(\d{2})\d{4,}(\d{2})\b`)
	return strictRe.ReplaceAllString(text, "$1****$2")
}

// MaskPhone 仅脱敏手机号
func MaskPhone(s string) string {
	return phoneRe.ReplaceAllString(s, "$1****$2")
}

// MaskStudentID 仅脱敏学号
func MaskStudentID(s string) string {
	return studentIDRe.ReplaceAllString(s, "$1**$2")
}

// TruncateForLog 截断文本用于日志记录（保留前 100 字符）
func TruncateForLog(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// SanitizeForLLM 发送 LLM 前的安全处理
// 1. 脱敏 PII  2. 去除首尾空白  3. 限制最大长度
func SanitizeForLLM(text string, maxLen int) string {
	text, _ = MaskPII(text)
	text = strings.TrimSpace(text)
	if maxLen > 0 {
		runes := []rune(text)
		if len(runes) > maxLen {
			text = string(runes[:maxLen])
		}
	}
	return text
}
