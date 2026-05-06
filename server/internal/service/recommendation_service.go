package service

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"unicode"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
)

// RecommendationService 个性化推荐引擎
// 基于用户历史提问和知识库内容，生成个性化推荐列表
type RecommendationService struct {
	kbRepo      *repository.KBRepo
	messageRepo *repository.MessageRepo
}

// NewRecommendationService 创建推荐服务
func NewRecommendationService(kbRepo *repository.KBRepo, messageRepo *repository.MessageRepo) *RecommendationService {
	return &RecommendationService{kbRepo: kbRepo, messageRepo: messageRepo}
}

// RecommendItem 推荐条目
type RecommendItem struct {
	ResourceID   string  `json:"resource_id"`
	ResourceType string  `json:"resource_type"`
	Title        string  `json:"title"`
	Summary      string  `json:"summary"`
	Tags         string  `json:"tags"`
	SourceLink   string  `json:"source_link"`
	Reason       string  `json:"reason"`       // 推荐理由
	Score        float64 `json:"score"`        // 推荐分数
}

// RecommendResult 推荐结果
type RecommendResult struct {
	Items []RecommendItem `json:"items"`
	Total int             `json:"total"`
}

// GetRecommendations 获取个性化推荐
// 基于用户最近提问提取关键词 → FTS5 搜索知识库 → 冷启动热门兜底
func (s *RecommendationService) GetRecommendations(userCtx *model.UserContext, limit int) (*RecommendResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// ── 阶段 1：获取用户最近提问，提取关键词 ──
	questions, err := s.messageRepo.GetRecentQuestionsByUserID(userCtx.UserID, 20)
	if err != nil {
		log.Printf("[推荐引擎] 获取用户 %d 历史提问失败，降级为冷启动: %v", userCtx.UserID, err)
	}
	keywords := extractKeywords(questions)

	// ── 阶段 2：基于关键词搜索知识库 ──
	seen := make(map[string]bool) // 去重
	var items []RecommendItem

	if len(keywords) > 0 {
		// 合并所有关键词为一个搜索查询
		query := strings.Join(keywords, " ")
		results, err := s.kbRepo.Search(query, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, limit*2)
		if err != nil {
			log.Printf("[推荐引擎] FTS5 搜索失败，降级为冷启动: %v", err)
		} else {
			for _, r := range results {
				if seen[r.Resource.ResourceID] {
					continue
				}
				seen[r.Resource.ResourceID] = true
				items = append(items, RecommendItem{
					ResourceID:   r.Resource.ResourceID,
					ResourceType: r.Resource.ResourceType,
					Title:        r.Resource.Title,
					Summary:      r.Resource.Summary,
					Tags:         r.Resource.Tags,
					SourceLink:   r.Resource.SourceLink,
					Reason:       fmt.Sprintf("与你最近咨询的「%s」相关", util.TruncateString(questions[0], 15)),
					Score:        -r.Score, // BM25 分数取反
				})
			}
		}
	}

	// ── 阶段 3：冷启动兜底 — 热门/最新内容 ──
	if len(items) < limit {
		coldItems, err := s.getPopularItems(userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, limit-len(items), seen)
		if err != nil {
			log.Printf("[推荐引擎] 获取热门内容失败: %v", err)
		} else {
			items = append(items, coldItems...)
		}
	}

	// 如果仍然不足，用最新发布补足（与热门内容相同的逻辑，但允许第二次填充）
	if len(items) < limit {
		moreItems, err := s.getPopularItems(userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, limit-len(items), seen)
		if err != nil {
			log.Printf("[推荐引擎] 获取最新内容失败: %v", err)
		} else {
			items = append(items, moreItems...)
		}
	}

	// 按分数降序排列
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	if len(items) > limit {
		items = items[:limit]
	}

	return &RecommendResult{
		Items: items,
		Total: len(items),
	}, nil
}

// getPopularItems 获取热门内容（最近更新的已发布资源）
func (s *RecommendationService) getPopularItems(ownerScope, ownerID, role string, limit int, exclude map[string]bool) ([]RecommendItem, error) {
	// 冷启动不按 ownerID 细分，同一 scope 下所有已发布资源均可推荐
	resources, err := s.kbRepo.List(ownerScope, "", "published", "", 0, limit*2)
	if err != nil {
		return nil, err
	}

	var items []RecommendItem
	for _, r := range resources {
		if exclude[r.ResourceID] {
			continue
		}
		exclude[r.ResourceID] = true
		items = append(items, RecommendItem{
			ResourceID:   r.ResourceID,
			ResourceType: r.ResourceType,
			Title:        r.Title,
			Summary:      r.Summary,
			Tags:         r.Tags,
			SourceLink:   r.SourceLink,
			Reason:       "热门内容",
			Score:        0.1,
		})
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}

// ── 关键词提取 ──

// extractKeywords 从用户提问列表中提取关键词
// 简单策略：分词 + 去停用词 + 高频词优先
func extractKeywords(questions []string) []string {
	if len(questions) == 0 {
		return nil
	}

	freq := make(map[string]int)
	totalWords := 0

	for _, q := range questions {
		words := segmentChinese(q)
		for _, w := range words {
			if isStopWord(w) {
				continue
			}
			freq[w]++
			totalWords++
		}
	}

	if len(freq) == 0 {
		return nil
	}

	// 按频率排序，取前 5 个
	type kv struct {
		word  string
		count int
	}
	var sorted []kv
	for w, c := range freq {
		sorted = append(sorted, kv{w, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// 返回前 5 个有区分度的关键词
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}

	var keywords []string
	for i := 0; i < limit; i++ {
		keywords = append(keywords, sorted[i].word)
	}

	return keywords
}

// segmentChinese 简单中文分词：按标点/空格分割，提取 2-4 字片段
func segmentChinese(text string) []string {
	// 移除标点和空格
	var cleaned []rune
	for _, r := range text {
		if unicode.IsPunct(r) || unicode.IsSpace(r) || unicode.IsSymbol(r) {
			cleaned = append(cleaned, ' ')
			continue
		}
		cleaned = append(cleaned, r)
	}
	cleanedStr := string(cleaned)

	// 按空格分割
	parts := strings.Fields(cleanedStr)

	var words []string
	for _, part := range parts {
		runes := []rune(part)
		// 提取 2-gram 和单字关键词
		for i := 0; i < len(runes); i++ {
			// 单字
			if isContentChar(runes[i]) {
				words = append(words, string(runes[i]))
			}
			// 2-gram
			if i+1 < len(runes) && isContentChar(runes[i]) && isContentChar(runes[i+1]) {
				words = append(words, string(runes[i:i+2]))
			}
		}
	}

	return words
}

func isContentChar(r rune) bool {
	return unicode.Is(unicode.Han, r) || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isStopWord 判断是否为停用词
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"请问": true, "你好": true, "谢谢": true, "我想": true, "知道": true,
		"什么": true, "怎么": true, "如何": true, "哪里": true, "为什么": true,
		"可以": true, "是否": true, "有没有": true, "能不能": true, "可不可以": true,
		"一下": true, "一个": true, "这个": true, "那个": true, "帮我": true,
		"现在": true, "最近": true, "已经": true, "还有": true, "需要": true,
		"还是": true, "或者": true, "但是": true, "不过": true,
		"你好请问": true, "我是": true,
	}
	return stopWords[word]
}
