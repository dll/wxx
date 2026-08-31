package context_engine

import (
	"log"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// debugScores 评测调参开关（WXX_DEBUG_SCORES=1 时打印全部候选得分）
var debugScores = os.Getenv("WXX_DEBUG_SCORES") == "1"

// ── CE-09: 来源可信度分层 ──
// 按 resource_type 分层加权：Policy > Process > FAQ > Activity
var typeWeight = map[string]float64{
	"Policy":   1.0,
	"Process":  0.85,
	"FAQ":      0.7,
	"Activity": 0.5,
}

// computeTrustScore 计算加权置信度（BM25 得分 × 类型权重）
// BM25 分数在 SQLite FTS5 中为负数（越小越相关），归一化到 0~1
func computeTrustScore(r *SearchResult) float64 {
	// BM25 归一化：原始分范围约 [-20, 0]，映射到 [0, 1]
	normalized := 0.0
	if r.Score < 0 {
		normalized = -r.Score / 20.0
		if normalized > 1.0 {
			normalized = 1.0
		}
	}

	weight := typeWeight[r.ResourceType]
	if weight == 0 {
		weight = 0.5 // 未知类型默认权重
	}

	return normalized * weight
}

// sortByTrust 按 TrustScore 降序排列
func sortByTrust(results []*SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].TrustScore > results[j].TrustScore
	})
}

// ── CE-A2: 意图加权 + 可插拔重排 ──

// applyIntentBoost 按意图偏好资源类型对 TrustScore 加权（偏好 ×1.15，其余不变）。
// 意图分类已由 ClassifyIntent 完成；此处把分类结果反馈进排序，提升命中率。
func applyIntentBoost(results []*SearchResult, intent Intent) {
	preferred := IntentToResourceTypes(intent)
	if len(preferred) == 0 {
		return
	}
	match := make(map[string]bool, len(preferred))
	for _, t := range preferred {
		match[t] = true
	}
	for _, r := range results {
		if match[r.ResourceType] {
			r.TrustScore *= 1.15
		}
	}
}

// Reranker 可插拔重排器（CE-A2）：对初排（信任分 + 意图加权）后的结果重排。
// 默认无 rerank；可接入 LLM listwise / 交叉编码器实现（成本可控时启用）。
type Reranker interface {
	Rerank(question string, results []*SearchResult) []*SearchResult
}

// ── CE-02: 低相关性守卫（与 service 层 filterLowRelevanceResults 阈值语义一致） ──

// relevanceThreshold 相关性阈值：标题/摘要/全文对问题 bigram 的加权覆盖率低于此值视为弱相关
// （0.09：兼顾口语词面差异的容忍与弱相关拦截，评测集回归校准；泛词碰瓷由减半规则拦截）
const relevanceThreshold = 0.09

// minContentRatio 内容字段计入相关性所需的最小 bigram 覆盖率（防长文本单点误命中）
const minContentRatio = 0.20

// strongContentRatio 强正文证据阈值：内容覆盖率达标即视为相关（答案常在正文细节，
// 标题/摘要与口语问法天然词面不同，加权公式会结构性低估，评测集回归校准）
const strongContentRatio = 0.40

// filterByRelevance 过滤低相关结果；全部低于阈值时保留最佳 1 条并标记 LowConfidence，
// 让下游走兜底而非基于弱相关资料生成「确定」回答（CE-02）。
// 结构化命中（标题/标签精确匹配）天然豁免。
func filterByRelevance(results []*SearchResult, question string) []*SearchResult {
	if len(results) == 0 || strings.TrimSpace(question) == "" {
		return results
	}

	bigrams := extractChineseBigramsFromQuestion(question)
	if len(bigrams) == 0 {
		return results // 问题过短/无有效词组，不过滤
	}

	var filtered []*SearchResult
	for _, r := range results {
		// 不豁免结构化命中：结构化 LIKE 对泛化词召回偏宽，相关性公式自身
		// 足以保护真实命中（标题精确匹配 → bigram 覆盖率高）。
		score := calcDocRelevance(r, question, bigrams)
		// 强正文证据：问题 bigram 大半命中正文 → 直接视为相关（CE-A2 评测校准）
		if bigramMatchRatio(r.Content, bigrams) >= strongContentRatio {
			score = relevanceThreshold
		}
		if score >= relevanceThreshold {
			filtered = append(filtered, r)
			continue
		}
		log.Printf("检索守卫过滤低相关: title=%q score=%.3f question=%q",
			truncate(r.Title, 30), score, truncate(question, 30))
	}
	if debugScores {
		for _, r := range results {
			log.Printf("[评测调参] title=%q score=%.3f trust=%.3f question=%q",
				truncate(r.Title, 30), calcDocRelevance(r, question, bigrams), r.TrustScore, truncate(question, 30))
		}
	}
	if len(filtered) == 0 {
		// 全部弱相关：保留信任分最高的一条，标记低置信
		sortByTrust(results)
		results[0].LowConfidence = true
		filtered = results[:1]
	}
	return filtered
}

// calcDocRelevance 文档相关性（0-1）：标题 60% + 摘要 25% + 全文 15%。
// 规则：① 内容字段覆盖率 < minContentRatio 时不计分（防长文本单点误命中）；
// ② 仅标题命中且标题覆盖率不足（<0.4）时减半：如问"转专业管理办法"时所有含
// "管理办法"的标题都会弱命中，但没有实质内容支撑，属泛词碰瓷。
func calcDocRelevance(r *SearchResult, question string, bigrams []string) float64 {
	titleScore := bigramMatchRatio(r.Title, bigrams)
	summaryScore := bigramMatchRatio(r.Summary, bigrams)
	contentScore := bigramMatchRatio(r.Content, bigrams)
	if question != "" && strings.Contains(r.Title, question) {
		titleScore = 1.0
	}
	if contentScore < minContentRatio {
		contentScore = 0
	}
	score := titleScore*0.6 + summaryScore*0.25 + contentScore*0.15
	if titleScore > 0 && titleScore < 0.4 && summaryScore == 0 && contentScore == 0 {
		score *= 0.5
	}
	return score
}

// bigramMatchRatio 计算文本中命中的 bigram 比例
func bigramMatchRatio(text string, bigrams []string) float64 {
	if len(bigrams) == 0 {
		return 0
	}
	matched := 0
	for _, bg := range bigrams {
		if strings.Contains(text, bg) {
			matched++
		}
	}
	return float64(matched) / float64(len(bigrams))
}

// extractChineseBigramsFromQuestion 从问题提取中文二元词组（去停用词后）
func extractChineseBigramsFromQuestion(q string) []string {
	cleaned := q
	stopwords := []string{"什么", "怎么", "如何", "为什么", "哪", "哪里", "哪个",
		"吗", "呢", "啊", "吧", "了", "的", "是", "有", "在", "我", "你", "他",
		"要", "需要", "可以", "能", "能够", "请问", "麻烦", "一下", "它", "这个", "那个"}
	for _, sw := range stopwords {
		cleaned = strings.ReplaceAll(cleaned, sw, "")
	}
	runes := []rune(strings.TrimSpace(cleaned))
	seen := make(map[string]bool)
	var bigrams []string
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] >= 0x4E00 && runes[i] <= 0x9FFF && runes[i+1] >= 0x4E00 && runes[i+1] <= 0x9FFF {
			bg := string(runes[i : i+2])
			if !seen[bg] {
				seen[bg] = true
				bigrams = append(bigrams, bg)
			}
		}
	}
	return bigrams
}

// ── CE-07: 命中片段提取 ──

// extractSnippet 从内容中提取与查询最相关的片段
func extractSnippet(content, question string) string {
	if content == "" || question == "" {
		return ""
	}

	// 提取查询关键词（去停用词后的字符）
	keywords := extractKeywords(question)
	if len(keywords) == 0 {
		// 无有效关键词，取前 200 字
		return truncate(content, 200)
	}

	// 滑动窗口找最佳片段（窗口 200 字符）
	runes := []rune(content)
	windowSize := 200
	if len(runes) <= windowSize {
		return content
	}

	bestStart := 0
	bestScore := 0

	for start := 0; start <= len(runes)-windowSize; start += 20 {
		window := string(runes[start : start+windowSize])
		score := 0
		for _, kw := range keywords {
			score += strings.Count(strings.ToLower(window), strings.ToLower(kw))
		}
		if score > bestScore {
			bestScore = score
			bestStart = start
		}
	}

	end := bestStart + windowSize
	if end > len(runes) {
		end = len(runes)
	}

	snippet := string(runes[bestStart:end])
	if bestStart > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
		snippet = snippet + "…"
	}
	return snippet
}

// extractKeywords 提取查询中的关键词（简易中文分词：连续非标点字符段）
func extractKeywords(question string) []string {
	// 去除常见中文停用词和标点
	stopwords := map[string]bool{
		"的": true, "了": true, "吗": true, "呢": true, "是": true,
		"在": true, "有": true, "和": true, "与": true, "或": true,
		"我": true, "你": true, "他": true, "她": true, "它": true,
		"这": true, "那": true, "什么": true, "怎么": true, "如何": true,
		"请问": true, "能": true, "可以": true, "想": true, "要": true,
	}

	// 按标点和空格分割
	segments := strings.FieldsFunc(question, func(r rune) bool {
		return r == '，' || r == '。' || r == '？' || r == '！' ||
			r == ',' || r == '.' || r == '?' || r == '!' ||
			r == ' ' || r == '\n' || r == '\t' ||
			r == '、' || r == '：' || r == '；'
	})

	var keywords []string
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" || stopwords[seg] {
			continue
		}
		// 如果是长段（>4字符），按 2-gram 切分
		runesSeg := []rune(seg)
		if len(runesSeg) > 4 {
			for i := 0; i < len(runesSeg)-1; i += 2 {
				end := i + 2
				if end > len(runesSeg) {
					end = len(runesSeg)
				}
				keywords = append(keywords, string(runesSeg[i:end]))
			}
		} else if len(runesSeg) >= 2 {
			keywords = append(keywords, seg)
		}
	}
	return keywords
}

// truncate 截取前 n 个字符
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}
