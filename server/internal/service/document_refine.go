package service

import (
	"context"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// RefineMetadata 使用 LLM 精修文档元数据；失败、超时或校验不通过时稳定回退原值。
func (s *DocumentService) RefineMetadata(ctx context.Context, title, summary string, keywords []string, content string) *DocumentRefineResult {
	fallback := &DocumentRefineResult{Title: title, Summary: summary, Keywords: keywords, Fallback: true}
	if s.llmClient == nil {
		return fallback
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return fallback
	}
	prompt := buildRefinePrompt(truncateDocForRefine(content, refineMaxInputRunes))
	ctx, cancel := context.WithTimeout(ctx, refineTimeout)
	defer cancel()
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "system", Content: refineSystemPrompt}, {Role: "user", Content: prompt}}, Temperature: 0.3, MaxTokens: refineMaxTokens})
	if err != nil {
		log.Printf("文档精修 LLM 调用失败，回退规则: %v", err)
		return fallback
	}
	refined, err := parseRefinedMetadata(resp.Content)
	if err != nil {
		log.Printf("文档精修响应解析失败，回退规则: %v", err)
		return fallback
	}
	normalizeRefinedMetadata(refined, title, summary, keywords)
	if !validateRefinedMetadata(refined) {
		log.Printf("文档精修结果校验不通过，回退规则: %q / %q / %v", refined.Title, refined.Summary, refined.Keywords)
		return fallback
	}
	return refined
}
