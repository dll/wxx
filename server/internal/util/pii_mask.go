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
	// 邮箱：保留首字符和域名
	emailRe = regexp.MustCompile(`\b(.)[^@]*(@[^\s]+\.[a-zA-Z]{2,})\b`)
	// 家庭住址：匹配常见住址模式（XX省XX市XX区/县XX路XX号）
	addressRe = regexp.MustCompile(
		`(?:省|市|区|县|镇|乡|村|路|街|巷|弄|号|栋|幢|单元|室|楼|层)\s*\d*\s*[号栋幢单元室楼层]?`,
	)
	// 银行卡号：16-19 位数字（保留前 4 后 4）
	bankCardRe = regexp.MustCompile(`\b(\d{4})\d{8,11}(\d{4})\b`)
)

// MaskPII 对文本中的个人身份信息脱敏
// 返回脱敏后的文本和是否进行了脱敏
func MaskPII(text string) (string, bool) {
	maskedCount := 0
	result := text

	// 身份证（先处理，避免被其他规则误匹配）
	if idCardRe.MatchString(result) {
		result = idCardRe.ReplaceAllString(result, "$1****$2")
		maskedCount++
	}

	// 银行卡号
	if bankCardRe.MatchString(result) {
		result = bankCardRe.ReplaceAllString(result, "$1****$2")
		maskedCount++
	}

	// 手机号
	if phoneRe.MatchString(result) {
		result = phoneRe.ReplaceAllString(result, "$1****$2")
		maskedCount++
	}

	// 学号
	if studentIDRe.MatchString(result) {
		result = studentIDRe.ReplaceAllString(result, "$1**$2")
		maskedCount++
	}

	// 邮箱
	result = emailRe.ReplaceAllString(result, "$1***$2")

	if maskedCount == 0 {
		return text, false
	}
	return result, true
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

// MaskEmail 仅脱敏邮箱
func MaskEmail(s string) string {
	return emailRe.ReplaceAllString(s, "$1***$2")
}

// MaskIDCard 仅脱敏身份证号
func MaskIDCard(s string) string {
	return idCardRe.ReplaceAllString(s, "$1****$2")
}

// DetectPII 检测文本中是否包含 PII（不修改文本）
func DetectPII(text string) bool {
	return idCardRe.MatchString(text) ||
		phoneRe.MatchString(text) ||
		studentIDRe.MatchString(text) ||
		emailRe.MatchString(text) ||
		bankCardRe.MatchString(text)
}

// PIIMaskResult 脱敏结果详情
type PIIMaskResult struct {
	Original      string
	Masked        string
	WasMasked     bool
	PIITypesFound []string // 检测到的 PII 类型
}

// MaskPIIWithDetail 脱敏并返回详细信息
func MaskPIIWithDetail(text string) PIIMaskResult {
	result := PIIMaskResult{Original: text}
	working := text

	if idCardRe.MatchString(working) {
		working = idCardRe.ReplaceAllString(working, "$1****$2")
		result.PIITypesFound = append(result.PIITypesFound, "身份证")
	}

	if bankCardRe.MatchString(working) {
		working = bankCardRe.ReplaceAllString(working, "$1****$2")
		result.PIITypesFound = append(result.PIITypesFound, "银行卡")
	}

	if phoneRe.MatchString(working) {
		working = phoneRe.ReplaceAllString(working, "$1****$2")
		result.PIITypesFound = append(result.PIITypesFound, "手机号")
	}

	if studentIDRe.MatchString(working) {
		working = studentIDRe.ReplaceAllString(working, "$1**$2")
		result.PIITypesFound = append(result.PIITypesFound, "学号")
	}

	if emailRe.MatchString(working) {
		working = emailRe.ReplaceAllString(working, "$1***$2")
		result.PIITypesFound = append(result.PIITypesFound, "邮箱")
	}

	result.WasMasked = len(result.PIITypesFound) > 0
	if result.WasMasked {
		result.Masked = working
	} else {
		result.Masked = text
	}
	return result
}

// TruncateForLog 截断文本用于日志记录（保留前 maxLen 字符）
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

// SanitizeLLMResponse 对 LLM 返回内容进行安全处理
// 脱敏 PII（防止模型幻觉输出真实 PII）+ TrimSpace
func SanitizeLLMResponse(text string) string {
	text, _ = MaskPII(text)
	return strings.TrimSpace(text)
}
