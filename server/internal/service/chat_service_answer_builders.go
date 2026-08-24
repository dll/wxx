package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
)

// buildEmptyResultAnswer 检索结果为空时的直接返回（MED-KB2：不调 LLM）
func (s *ChatService) buildEmptyResultAnswer(traceID string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: "我暂时没有找到相关信息，请换个问题试试",
		TraceID:    traceID,
		Confidence: 0.1,
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
		},
	}
}

// buildBlockedAnswer 内容过滤拦截时返回的兜底回答
func (s *ChatService) buildBlockedAnswer(traceID string, category string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: util.GetFallbackResponse(category),
		TraceID:    traceID,
		Confidence: 0.0,
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
		},
	}
}

// buildQuotaExceededAnswer 配额超限时返回的回答
func (s *ChatService) buildQuotaExceededAnswer(traceID string, reason string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: reason,
		TraceID:    traceID,
		Confidence: 0.0,
		Fallback:   true,
		FollowUps:  []string{},
	}
}

// fallbackAnswer 构造兜底回答
func (s *ChatService) fallbackAnswer(traceID string, question string) *model.AnswerCard {
	return &model.AnswerCard{
		Conclusion: "知识库中暂未找到足够信息，我不能凭空猜测（会误导你）。\n\n你可以这样做，很快就能问到答案：\n1. 联系你的辅导员（班级群/企业微信/电话）；\n2. 到学院学工办公室现场咨询；\n3. 在下方“问题反馈”里提交这个问题，我们修复后会通知你。",
		TraceID:    traceID,
		Confidence: 0.0,
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
			"怎么提交问题反馈？",
		},
	}
}

// fallbackAnswerWithSources 构造兜底回答（保留搜索到的 sources）
func (s *ChatService) fallbackAnswerWithSources(traceID string, question string, results []*repository.SearchResult) *model.AnswerCard {
	conclusion := "知识库中暂未找到足够信息，我不能凭空猜测（会误导你）。我可以给你整理出相关的资料，如需准确答案建议：\n1. 联系你的辅导员；\n2. 到学院学工办公室现场咨询；\n3. 在“问题反馈”提交，修复后通知你。"
	confidence := 0.3
	if len(results) > 0 {
		var b strings.Builder
		b.WriteString("我已根据知识库资料为您整理如下：\n\n")
		for i, r := range results {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Resource.Title))
			if r.Resource.Summary != "" {
				b.WriteString(r.Resource.Summary)
			} else {
				b.WriteString(truncateContent(r.Resource.Content, 260))
			}
			b.WriteString("\n\n")
		}
		conclusion = strings.TrimSpace(b.String())
		confidence = 0.5
	}
	card := &model.AnswerCard{
		Conclusion: conclusion,
		TraceID:    traceID,
		Confidence: confidence,
		Fallback:   true,
		FollowUps: []string{
			"联系辅导员的方式是什么？",
			"学工办公室在哪里？",
			"怎么提交问题反馈？",
		},
	}

	// 附加搜索到的来源
	for _, r := range results {
		card.Sources = append(card.Sources, model.Source{
			ResourceID:     r.Resource.ResourceID,
			Title:          r.Resource.Title,
			ResourceType:   r.Resource.ResourceType,
			Version:        r.Resource.Version,
			SourceLink:     r.Resource.SourceLink,
			RelevanceScore: normalizeRelevanceScore(-r.Score),
			EffectiveAt:    r.Resource.EffectiveAt,
			Snippet:        r.Resource.Summary,
		})
	}

	// 按相关度降序排序
	sort.Slice(card.Sources, func(i, j int) bool {
		return card.Sources[i].RelevanceScore > card.Sources[j].RelevanceScore
	})

	return card
}
