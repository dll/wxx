package context_engine

import (
	"sort"
	"strings"
	"unicode/utf8"
)

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
