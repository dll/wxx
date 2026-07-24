package service

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
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
	Reason       string  `json:"reason"` // 推荐理由
	Score        float64 `json:"score"`  // 推荐分数
}

// roleTypeWeights 角色对各资源类型的偏好权重
var roleTypeWeights = map[string]map[string]float64{
	"student":       {"Policy": 1.0, "Process": 1.2, "FAQ": 1.3, "Activity": 1.4},
	"student_union": {"Policy": 1.1, "Process": 1.0, "FAQ": 1.0, "Activity": 1.5},
	"counselor":     {"Policy": 1.5, "Process": 1.2, "FAQ": 0.8, "Activity": 0.8},
	"college_admin": {"Policy": 1.5, "Process": 1.3, "FAQ": 0.7, "Activity": 0.8},
	"school_admin":  {"Policy": 1.5, "Process": 1.3, "FAQ": 0.7, "Activity": 0.8},
	"sys_admin":     {"Policy": 1.3, "Process": 1.2, "FAQ": 1.0, "Activity": 1.0},
	"teacher":       {"Policy": 1.3, "Process": 1.2, "FAQ": 1.0, "Activity": 0.9},
	"assistant":     {"Policy": 1.3, "Process": 1.4, "FAQ": 1.1, "Activity": 0.9},
}

// seasonalTopics 季节性主题推荐（按月份映射到关键词和类别）
var seasonalTopics = map[time.Month]struct {
	Keywords []string
	Types    []string
}{
	time.January:   {Keywords: []string{"寒假", "复习", "补考"}, Types: []string{"Process", "FAQ"}},
	time.February:  {Keywords: []string{"开学", "选课", "补考"}, Types: []string{"Process", "Activity"}},
	time.March:     {Keywords: []string{"奖学金", "评优", "竞赛"}, Types: []string{"Policy", "Activity"}},
	time.April:     {Keywords: []string{"运动会", "期中", "体测"}, Types: []string{"Activity", "FAQ"}},
	time.May:       {Keywords: []string{"五四", "社团", "实习"}, Types: []string{"Activity", "Process"}},
	time.June:      {Keywords: []string{"毕业", "离校", "期末", "复习"}, Types: []string{"Process", "Policy"}},
	time.July:      {Keywords: []string{"暑假", "社会实践", "留校"}, Types: []string{"Activity", "Process"}},
	time.August:    {Keywords: []string{"暑假", "补考", "实习"}, Types: []string{"FAQ", "Process"}},
	time.September: {Keywords: []string{"入学", "军训", "选课", "社团"}, Types: []string{"Process", "Activity"}},
	time.October:   {Keywords: []string{"奖学金", "评优", "体测"}, Types: []string{"Policy", "Activity"}},
	time.November:  {Keywords: []string{"期中", "竞赛", "实习"}, Types: []string{"FAQ", "Activity"}},
	time.December:  {Keywords: []string{"期末", "四六级", "考研", "寒假"}, Types: []string{"FAQ", "Activity"}},
}

// RecommendResult 推荐结果
type RecommendResult struct {
	Items []RecommendItem `json:"items"`
	Total int             `json:"total"`
}

// GetRecommendations 获取个性化推荐
// 推荐策略：用户历史关键词 → FTS5 搜索 → 角色加权 → 季节感知 → 类别多样性 → 冷启动兜底
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

	// 注入季节性关键词
	now := time.Now()
	if seasonal, ok := seasonalTopics[now.Month()]; ok {
		keywords = append(keywords, seasonal.Keywords...)
	}

	seen := make(map[string]bool)
	var items []RecommendItem

	// ── 阶段 2：基于关键词搜索知识库 ──
	if len(keywords) > 0 {
		query := strings.Join(dedupeKeywords(keywords), " ")
		results, err := s.kbRepo.Search(query, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, limit*2)
		if err != nil {
			log.Printf("[推荐引擎] FTS5 搜索失败，降级为冷启动: %v", err)
		} else {
			weights := roleTypeWeights[userCtx.Role]
			for _, r := range results {
				if seen[r.Resource.ResourceID] {
					continue
				}
				seen[r.Resource.ResourceID] = true
				score := -r.Score // BM25 分数取反后越高越相关
				// 应用角色权重
				if w, ok := weights[r.Resource.ResourceType]; ok {
					score *= w
				}
				var reason string
				if len(questions) > 0 {
					reason = fmt.Sprintf("与你最近咨询的「%s」相关", util.TruncateString(questions[0], 15))
				} else {
					reason = getSeasonalReason(now)
				}
				items = append(items, RecommendItem{
					ResourceID:   r.Resource.ResourceID,
					ResourceType: r.Resource.ResourceType,
					Title:        r.Resource.Title,
					Summary:      r.Resource.Summary,
					Tags:         r.Resource.Tags,
					SourceLink:   r.Resource.SourceLink,
					Reason:       reason,
					Score:        score,
				})
			}
		}
	}

	// ── 阶段 3：类别多样性补全 ──
	items = s.ensureDiversity(items, userCtx, limit, seen)

	// ── 阶段 4：冷启动兜底 ──
	if len(items) < limit {
		coldItems, _ := s.getPopularItems(userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, limit-len(items), seen)
		items = append(items, coldItems...)
	}
	if len(items) < limit {
		moreItems, _ := s.getPopularItems(userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, limit-len(items), seen)
		items = append(items, moreItems...)
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

// ensureDiversity 确保推荐列表涵盖至少 3 种资源类型
func (s *RecommendationService) ensureDiversity(items []RecommendItem, userCtx *model.UserContext, targetLimit int, seen map[string]bool) []RecommendItem {
	typeCounts := make(map[string]int)
	for _, it := range items {
		typeCounts[it.ResourceType]++
	}

	// 需要补全的类型
	allTypes := []string{"Policy", "Process", "FAQ", "Activity"}
	diverseCount := 0
	for _, t := range allTypes {
		if typeCounts[t] > 0 {
			diverseCount++
		}
	}

	if diverseCount >= 3 || len(items) >= targetLimit {
		return items
	}

	// 为缺少的类型各取一条
	for _, t := range allTypes {
		if typeCounts[t] > 0 {
			continue
		}
		resources, err := s.kbRepo.List(userCtx.OwnerScope, userCtx.OwnerID, "published", t, 0, 3)
		if err != nil || len(resources) == 0 {
			continue
		}
		for _, r := range resources {
			if seen[r.ResourceID] {
				continue
			}
			seen[r.ResourceID] = true
			items = append(items, RecommendItem{
				ResourceID:   r.ResourceID,
				ResourceType: r.ResourceType,
				Title:        r.Title,
				Summary:      r.Summary,
				Tags:         r.Tags,
				SourceLink:   r.SourceLink,
				Reason:       "你可能感兴趣",
				Score:        0.3,
			})
			break
		}
	}

	return items
}

// getSeasonalReason 返回季节性推荐理由
func getSeasonalReason(now time.Time) string {
	switch now.Month() {
	case time.January, time.February:
		return "开学季推荐"
	case time.June:
		return "毕业季必读"
	case time.July, time.August:
		return "暑期相关"
	case time.September:
		return "新生入学必读"
	case time.December:
		return "期末季备考"
	default:
		return "近期热门"
	}
}

// dedupeKeywords 关键词去重
func dedupeKeywords(keywords []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, k := range keywords {
		if !seen[k] {
			seen[k] = true
			result = append(result, k)
		}
	}
	return result
}

// getPopularItems 获取推荐内容（角色偏好加权的最新已发布资源）
func (s *RecommendationService) getPopularItems(ownerScope, ownerID, role string, limit int, exclude map[string]bool) ([]RecommendItem, error) {
	resources, err := s.kbRepo.List(ownerScope, "", "published", "", 0, limit*2)
	if err != nil {
		return nil, err
	}

	weights := roleTypeWeights[role]
	var items []RecommendItem
	for _, r := range resources {
		if exclude[r.ResourceID] {
			continue
		}
		exclude[r.ResourceID] = true
		score := 0.1
		if w, ok := weights[r.ResourceType]; ok {
			score *= w
		}
		items = append(items, RecommendItem{
			ResourceID:   r.ResourceID,
			ResourceType: r.ResourceType,
			Title:        r.Title,
			Summary:      r.Summary,
			Tags:         r.Tags,
			SourceLink:   r.SourceLink,
			Reason:       getSeasonalReason(time.Now()),
			Score:        score,
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
