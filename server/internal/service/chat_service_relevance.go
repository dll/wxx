package service

import (
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/repository"
)

// mergeStructuredAndFTS 合并结构化结果与 FTS 结果，结构化在前、去重（MED-KB1）
func mergeStructuredAndFTS(structured, fts []*repository.SearchResult) []*repository.SearchResult {
	seen := make(map[string]bool)
	var merged []*repository.SearchResult
	for _, r := range structured {
		key := r.Resource.ResourceID + r.Resource.Version
		if !seen[key] {
			seen[key] = true
			merged = append(merged, r)
		}
	}
	for _, r := range fts {
		key := r.Resource.ResourceID + r.Resource.Version
		if !seen[key] {
			seen[key] = true
			merged = append(merged, r)
		}
	}
	return merged
}

// filterLowRelevanceResults 过滤低相关性检索结果
// 综合使用：标题二元词组匹配、Jaccard 相似度、关键词覆盖率
// 目的：在送入 LLM 前就把明显不相关的内容过滤掉，避免误导
func filterLowRelevanceResults(results []*repository.SearchResult, question string) []*repository.SearchResult {
	if len(results) == 0 {
		return results
	}

	q := strings.TrimSpace(question)
	if q == "" {
		return results
	}

	// 提取问题中的中文二元词组（核心语义单元）
	qBigrams := extractChineseBigramsFromQuestion(q)
	if len(qBigrams) == 0 {
		// 问题太短，不做过滤
		return results
	}

	var filtered []*repository.SearchResult
	for _, r := range results {
		// 计算相关性得分
		score := calcRelevanceScore(r.Resource.Title, r.Resource.Summary, r.Resource.Content, q, qBigrams)

		// 阈值：至少 0.15 分才认为相关
		if score >= 0.15 {
			filtered = append(filtered, r)
		} else {
			log.Printf("过滤低相关性结果: title=%q score=%.3f question=%q",
				truncateContent(r.Resource.Title, 30), score, truncateContent(q, 30))
		}
	}

	// 如果全部被过滤了，保留分数最高的1条（避免完全无结果），但标记为低置信，
	// 让下游走兜底而非基于弱相关资料生成「确定」回答（CE-02）。
	if len(filtered) == 0 && len(results) > 0 {
		bestIdx := 0
		bestScore := -1.0
		for i, r := range results {
			s := calcRelevanceScore(r.Resource.Title, r.Resource.Summary, r.Resource.Content, q, qBigrams)
			if s > bestScore {
				bestScore = s
				bestIdx = i
			}
		}
		results[bestIdx].LowConfidence = true
		filtered = append(filtered, results[bestIdx])
		log.Printf("所有结果相关性均较低，保留最佳(标记低置信): title=%q score=%.3f",
			truncateContent(results[bestIdx].Resource.Title, 30), bestScore)
	}

	return filtered
}

// calcRelevanceScore 计算文档与问题的相关性得分（0-1）
// 权重：标题60% + 摘要25% + 全文15%
func calcRelevanceScore(title, summary, content, question string, qBigrams []string) float64 {
	titleScore := bigramMatchRatio(title, qBigrams)
	summaryScore := bigramMatchRatio(summary, qBigrams)
	contentScore := bigramMatchRatio(content, qBigrams)

	// 标题中精确匹配整个问题，额外加分
	if strings.Contains(title, question) {
		titleScore = 1.0
	}

	return titleScore*0.6 + summaryScore*0.25 + contentScore*0.15
}

// bigramMatchRatio 计算文本中匹配的二元词组比例
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

// extractChineseBigramsFromQuestion 从问题中提取中文二元词组（去停用词后）
func extractChineseBigramsFromQuestion(q string) []string {
	// 先去除常见停用词和疑问词
	stopWords := []string{"什么", "怎么", "如何", "为什么", "哪", "哪里", "哪个",

		"吗", "呢", "啊", "吧", "了", "的", "是", "有", "在", "我", "你", "他",
		"要", "需要", "可以", "能", "能够", "请问", "麻烦", "一下"}
	cleaned := q
	for _, sw := range stopWords {
		cleaned = strings.ReplaceAll(cleaned, sw, "")
	}
	cleaned = strings.TrimSpace(cleaned)

	runes := []rune(cleaned)
	var bigrams []string
	seen := make(map[string]bool)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] >= 0x4E00 && runes[i] <= 0x9FFF &&
			runes[i+1] >= 0x4E00 && runes[i+1] <= 0x9FFF {
			bg := string(runes[i : i+2])
			if !seen[bg] {
				seen[bg] = true
				bigrams = append(bigrams, bg)
			}
		}
	}
	return bigrams
}

// normalizeRelevanceScore 将 BM25 原始分数归一化到 0~1 范围
// SQLite FTS5 的 BM25 分数为负数（越小越相关），转换后范围约为 0~20
// 将其映射到 0~1，使得前端显示百分比时不会超过 100%
func normalizeRelevanceScore(rawScore float64) float64 {
	if rawScore <= 0 {
		return 0
	}
	// BM25 原始分数范围约为 0~20，映射到 0~1
	normalized := rawScore / 20.0
	if normalized > 1.0 {
		normalized = 1.0
	}
	return normalized
}
