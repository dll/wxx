package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// ── FAQ 持久化缓存 ──
//
// 设计要点：
// 1. 命中：在 resource_type='FAQ' 中按 BM25 搜索 question；得分阈值更严（≤ -8.0）
//    且字符级 Jaccard 相似度 > 0.6，避免假阳性。
// 2. 写入：仅在「单 LLM 调用 + 有引用 sources」时入库；纯生成无来源不写。
// 3. 失效：用户提交 category=answer_error 反馈时，由 FeedbackService 调
//    s.kbRepo.SetStatus(faqResourceIDFor(question), 'retired') 立刻失效。

const (
	faqScoreThreshold       = -8.0 // BM25 分数（越小越相关）
	faqMinJaccardSimilarity = 0.6  // 中文逐字 Jaccard 相似度
)

// faqResourceIDFor 由问题原文生成确定性的 FAQ 资源 ID（与内存哈希同源，便于失效）
func faqResourceIDFor(question string) string {
	return "faq-cached-" + cacheKeyForQuestion(question)
}

// faqLookup 在 FAQ 资源中检索，命中阈值与相似度后还原 AnswerCard
func (s *ChatService) faqLookup(question string, userCtx *model.UserContext) *model.AnswerCard {
	if s.kbRepo == nil {
		return nil
	}
	results, err := s.kbRepo.SearchFAQ(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, 1)
	if err != nil || len(results) == 0 {
		return nil
	}
	hit := results[0]
	if hit.Score > faqScoreThreshold {
		return nil
	}
	if jaccardSimilarity(question, hit.Resource.Summary) < faqMinJaccardSimilarity {
		return nil
	}

	// content 字段存的是 AnswerCard JSON
	var card model.AnswerCard
	if err := json.Unmarshal([]byte(hit.Resource.Content), &card); err != nil {
		log.Printf("FAQ 反序列化失败 resource_id=%s err=%v", hit.Resource.ResourceID, err)
		return nil
	}
	// 标记为来自历史问答缓存，便于前端展示
	card.Sources = append([]model.Source{{
		ResourceID:   hit.Resource.ResourceID,
		Title:        "历史问答缓存",
		ResourceType: "FAQ",
		Version:      hit.Resource.Version,
		SourceLink:   "",
		Snippet:      hit.Resource.Summary,
	}}, card.Sources...)
	return &card
}

// faqStore 把生成成功且有引用的回答持久化到知识库（待人工审核）
// 注意：自动入库的 FAQ 状态为 pending，需人工审核通过后才会被检索到
// 避免 LLM 生成错误 FAQ 导致"错误自我强化"
func (s *ChatService) faqStore(question string, card *model.AnswerCard, role string) {
	if s.kbRepo == nil || card == nil {
		return
	}
	body, err := json.Marshal(card)
	if err != nil {
		return
	}
	q := strings.TrimSpace(question)
	titleRunes := []rune(q)
	if len(titleRunes) > 60 {
		titleRunes = titleRunes[:60]
	}
	res := &model.KBResource{
		ResourceID:   faqResourceIDFor(q),
		ResourceType: "FAQ",
		OwnerScope:   "school",
		OwnerID:      "all",
		RoleScope:    "[\"" + role + "\"]",
		Version:      time.Now().Format("20060102.150405"),
		Status:       "pending",
		Title:        string(titleRunes),
		Summary:      q + "（AI自动生成，待人工审核）",
		Content:      string(body),
		SourceLink:   "",
		Tags:         "[\"faq-cached\",\"ai-generated\"]",
		UpdatedBy:    "auto",
	}
	if _, action, err := s.kbRepo.Upsert(res); err != nil {
		log.Printf("FAQ 入库失败 resource_id=%s err=%v", res.ResourceID, err)
	} else {
		log.Printf("FAQ 入库成功（待审核） resource_id=%s action=%s", res.ResourceID, action)
	}
}

// RetireFAQ 把指定问题对应的 FAQ 资源标为 retired（用户反馈"回答有误"时调用）
func (s *ChatService) RetireFAQ(question string) error {
	if s.kbRepo == nil {
		return nil
	}
	resourceID := faqResourceIDFor(question)
	if err := s.kbRepo.SetStatus(resourceID, "retired"); err != nil {
		return fmt.Errorf("撤回FAQ状态失败: %w", err)
	}
	log.Printf("FAQ 已失效 resource_id=%s", resourceID)
	return nil
}

// jaccardSimilarity 中文友好的字符级 Jaccard 相似度（去空白后按 rune 集合）
func jaccardSimilarity(a, b string) float64 {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return 0
	}
	setA := runeSet(a)
	setB := runeSet(b)
	inter := 0
	for r := range setA {
		if _, ok := setB[r]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func runeSet(s string) map[rune]struct{} {
	m := make(map[rune]struct{})
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		m[r] = struct{}{}
	}
	return m
}
