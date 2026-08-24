package service

import (
	"encoding/json"
	"strings"

	"github.com/dll/wxx/server/internal/model"
)

// generateFollowUps 根据回答内容生成追问建议
func generateFollowUps(content string) []string {
	var followUps []string

	// 简单的关键词匹配生成追问（后续可用 LLM 增强）
	if strings.Contains(content, "申请") {
		followUps = append(followUps, "申请需要哪些材料？")
	}
	if strings.Contains(content, "流程") || strings.Contains(content, "步骤") {
		followUps = append(followUps, "具体的办理地点在哪里？")
	}
	if strings.Contains(content, "截止") || strings.Contains(content, "日期") {
		followUps = append(followUps, "如果错过截止日期怎么办？")
	}

	return followUps
}

// truncateContent 截断内容，保留前 maxLen 个字符
func truncateContent(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// MarshalAnswerCard 将 AnswerCard 序列化为 JSON 字符串（用于审计日志等）
func MarshalAnswerCard(card *model.AnswerCard) string {
	b, err := json.Marshal(card)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// defaultSessionTitle 用首条问题前 30 个字符作为会话默认标题
// 用户可通过 PATCH /sessions/:id 重命名
func defaultSessionTitle(question string) string {
	q := strings.TrimSpace(question)
	if q == "" {
		return ""
	}
	runes := []rune(q)
	if len(runes) > 30 {
		return string(runes[:30]) + "…"
	}
	return string(runes)
}
