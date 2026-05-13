package util

import (
	"strings"
	"sync"
)

// FilterAction 过滤动作
type FilterAction int

const (
	FilterPass   FilterAction = iota // 通过
	FilterBlock                      // 拦截（返回兜底回复）
	FilterFlag                       // 标记（放行但记录审计）
)

// FilterResult 过滤结果
type FilterResult struct {
	Action     FilterAction
	Reason     string // 触发原因（仅用于审计，不返回给用户）
	Category   string // 触发类别：political/porn/violence/illegal
	Confidence float64
}

// ContentFilter 内容安全过滤器
type ContentFilter struct {
	mu sync.RWMutex
	// 拦截词（命中即 Block）
	blockWords []string
	// 标记词（命中即 Flag，不放行但记录）
	flagWords []string
}

// 全局默认过滤器
var defaultFilter = newDefaultFilter()

func newDefaultFilter() *ContentFilter {
	return &ContentFilter{
		blockWords: []string{
			// 政治敏感类（示例，实际应根据法规完善）
			"颠覆国家政权", "分裂国家", "恐怖主义",
			// 色情类
			"色情", "淫秽",
			// 暴力类
			"制造炸弹", "杀人方法",
			// 违法类
			"代考", "替考", "作弊器材",
		},
		flagWords: []string{
			// 需关注但不直接拦截
			"自杀", "自残", "想不开", "不想活了",
			"校园暴力", "霸凌", "被欺负",
		},
	}
}

// DefaultFilter 返回全局默认过滤器
func DefaultFilter() *ContentFilter {
	return defaultFilter
}

// Check 检查文本是否包含敏感内容
func (f *ContentFilter) Check(text string) FilterResult {
	lower := strings.ToLower(text)

	// 先检查拦截词
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, word := range f.blockWords {
		if strings.Contains(lower, strings.ToLower(word)) {
			return FilterResult{
				Action:     FilterBlock,
				Reason:     "命中拦截词",
				Category:   categorize(word),
				Confidence: 0.95,
			}
		}
	}

	// 检查标记词
	for _, word := range f.flagWords {
		if strings.Contains(lower, strings.ToLower(word)) {
			return FilterResult{
				Action:     FilterFlag,
				Reason:     "命中标记词",
				Category:   "psychological",
				Confidence: 0.7,
			}
		}
	}

	return FilterResult{Action: FilterPass}
}

// CheckContent 便捷函数：使用默认过滤器检查文本
func CheckContent(text string) FilterResult {
	return defaultFilter.Check(text)
}

// AddBlockWord 动态添加拦截词
func (f *ContentFilter) AddBlockWord(word string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockWords = append(f.blockWords, word)
}

// AddFlagWord 动态添加标记词
func (f *ContentFilter) AddFlagWord(word string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flagWords = append(f.flagWords, word)
}

// categorize 根据关键词推断内容类别
func categorize(word string) string {
	switch strings.ToLower(word) {
	case "颠覆国家政权", "分裂国家", "恐怖主义":
		return "political"
	case "色情", "淫秽":
		return "porn"
	case "制造炸弹", "杀人方法":
		return "violence"
	case "代考", "替考", "作弊器材":
		return "illegal"
	default:
		return "unknown"
	}
}

// FilterBlockResponse 拦截时返回的兜底回复
const FilterBlockResponse = "抱歉，该问题涉及的内容暂不支持回答。如有需要，请联系辅导员或学工办公室。"
