package util

import (
	"strings"
	"sync"
	"unicode/utf8"
)

// FilterAction 过滤动作
type FilterAction int

const (
	FilterPass  FilterAction = iota // 通过
	FilterBlock                     // 拦截（返回兜底回复）
	FilterFlag                      // 标记（放行但记录审计）
)

// FilterResult 过滤结果
type FilterResult struct {
	Action     FilterAction
	Reason     string  // 触发原因（仅用于审计，不返回给用户）
	Category   string  // 触发类别
	Confidence float64 // 置信度
	HitWord    string  // 命中的敏感词（用于审计）
}

// ContentFilter 内容安全过滤器
type ContentFilter struct {
	mu sync.RWMutex
	// 拦截词（命中即 Block）—— 按类别组织
	blockWords map[string][]string
	// 标记词（命中即 Flag，不拦截但记录）
	flagWords map[string][]string
}

// 全局默认过滤器
var defaultFilter = newDefaultFilter()

func newDefaultFilter() *ContentFilter {
	return &ContentFilter{
		blockWords: map[string][]string{
			// 政治敏感类
			"political": {
				"颠覆国家政权", "分裂国家", "恐怖主义", "极端主义",
				"煽动民族仇恨", "破坏民族团结", "邪教", "反动",
				"法轮功", "藏独", "疆独", "台独", "港独",
				"攻击党的领导", "攻击社会主义制度", "推翻政府",
			},
			// 色情低俗类
			"porn": {
				"色情", "淫秽", "裸体", "性交", "卖淫", "嫖娼",
				"黄色网站", "成人视频", "色情直播", "约炮",
				"一夜情", "援交", "情色", "艳照", "偷拍裙底",
			},
			// 暴力恐怖类
			"violence": {
				"制造炸弹", "杀人方法", "如何杀人", "枪支制造",
				"爆炸物制作", "投毒方法", "雇凶杀人", "买凶",
				"人体炸弹", "校园枪击", "恐怖袭击", "斩首",
			},
			// 违法类
			"illegal": {
				"代考", "替考", "作弊器材", "贩卖答案",
				"买卖论文", "代写论文", "办假证", "伪造学历",
				"贩卖毒品", "吸毒", "贩毒", "制毒",
				"网络诈骗", "电信诈骗", "刷单诈骗", "传销",
				"黑客", "盗号", "钓鱼网站", "木马病毒",
				"赌博", "赌球", "六合彩", "时时彩",
				"校园贷", "裸贷", "高利贷", "套路贷",
				"人肉搜索", "网络暴力", "造谣", "诽谤",
			},
			// 校园安全类
			"campus": {
				"强奸", "猥亵", "性骚扰", "性侵",
				"拐卖", "绑架", "非法拘禁",
				"投毒", "纵火", "故意伤害",
				"校园欺凌致死", "殴打致死",
			},
		},
		flagWords: map[string][]string{
			// 心理健康类（不拦截但标记关注）
			"psychological": {
				"自杀", "自残", "想不开", "不想活了", "活着没意思",
				"生无可恋", "轻生", "寻死", "自伤", "割腕",
				"校园暴力", "霸凌", "被欺负", "被孤立", "被排挤",
				"被威胁", "被恐吓", "被跟踪", "被骚扰",
				"抑郁", "焦虑", "失眠", "厌食", "暴食",
				"酗酒", "药瘾", "创伤", "PTSD",
				"没有人关心我", "没人理解我", "我很孤独",
				"我想离开", "撑不下去了", "太痛苦了",
			},
			// 情感风险类（标记关注）
			"emotional": {
				"失恋", "分手", "被甩", "劈腿", "出轨",
				"挂科太多", "要被退学", "毕不了业", "延毕",
				"家庭变故", "父母离异", "亲人去世",
				"经济困难", "交不起学费", "吃不起饭",
				"被处分", "被开除", "被通报批评",
			},
			// 学业风险类
			"academic": {
				"想退学", "不想上了", "读不下去了",
				"逃课太多", "旷考", "作弊", "抄袭",
				"实验做不出", "论文写不完", "答辩过不了",
			},
		},
	}
}

// DefaultFilter 返回全局默认过滤器
func DefaultFilter() *ContentFilter {
	return defaultFilter
}

// 安全语境豁免词：这些词出现在拦截词附近时，判定为“防范/提醒”正常语境而非违规。
// 例如"谨防电信诈骗""远离校园贷""防范网络诈骗"等安全教育内容不应被拦截。
var safetyContextWords = []string{
	"防", "谨防", "防范", "警惕", "预防", "抵制", "远离", "拒绝",
	"举报", "提醒", "安全教育", "识破", "不上当", "避免被骗", "反诈", "防骗",
	"增强", "提高警惕", "注意安全", "不要相信", "谨记",
}

// Check 检查文本是否包含敏感内容
// [allowSafetyContext] 为 true 时（LLM 输出检查），命中拦截词但相邻为安全语境（如"谨防诈骗"）则放行，
// 避免正常的安全教育内容被误判为违规。
func (f *ContentFilter) Check(text string, allowSafetyContext bool) FilterResult {
	lower := strings.ToLower(text)

	f.mu.RLock()
	defer f.mu.RUnlock()

	// 先检查拦截词（按类别检查）
	for category, words := range f.blockWords {
		for _, word := range words {
			if strings.Contains(lower, strings.ToLower(word)) {
				// LLM 输出检查：若该词处于安全语境（前缀含"防/警惕"等）则放行
				if allowSafetyContext && inSafetyContext(lower, word) {
					continue
				}
				return FilterResult{
					Action:     FilterBlock,
					Reason:     "命中拦截词: " + word,
					Category:   category,
					Confidence: 0.95,
					HitWord:    word,
				}
			}
		}
	}

	// 检查标记词（按类别检查）
	for category, words := range f.flagWords {
		for _, word := range words {
			if strings.Contains(lower, strings.ToLower(word)) {
				return FilterResult{
					Action:     FilterFlag,
					Reason:     "命中标记词: " + word,
					Category:   category,
					Confidence: 0.7,
					HitWord:    word,
				}
			}
		}
	}

	return FilterResult{Action: FilterPass}
}

// inSafetyContext 判断拦截词是否处于安全提醒语境（词前方存在安全前缀词）
func inSafetyContext(text, word string) bool {
	idx := strings.Index(text, strings.ToLower(word))
	if idx < 0 {
		return false
	}
	// 取拦截词前的一段文本（不含该词本身）作为安全语境。
	// 用整段而非固定窗口，兼容"谨防电信诈骗和网络诈骗"等连词连接的多个拦截词。
	prefix := text[:idx]
	for _, sw := range safetyContextWords {
		if strings.Contains(prefix, strings.ToLower(sw)) {
			return true
		}
	}
	return false
}

// CheckInput 检查用户输入（与 Check 相同，但可扩展特殊逻辑）
func (f *ContentFilter) CheckInput(text string) FilterResult {
	return f.Check(text, false)
}

// CheckOutput 检查 LLM 输出（更严格的检查）
func (f *ContentFilter) CheckOutput(text string) FilterResult {
	// 对输出用相同的检查逻辑，但放行安全语境（如"谨防电信诈骗"）
	return f.Check(text, true)
}

// CheckCombined 同时检查输入和输出，返回更严格的判定
func (f *ContentFilter) CheckCombined(input, output string) FilterResult {
	// 先检查输入
	if result := f.CheckInput(input); result.Action == FilterBlock {
		return result
	}

	// 再检查输出
	if result := f.CheckOutput(output); result.Action == FilterBlock {
		return result
	}

	// Flag 情况取输入输出的组合
	inputResult := f.CheckInput(input)
	outputResult := f.CheckOutput(output)

	if inputResult.Action == FilterFlag || outputResult.Action == FilterFlag {
		action := FilterFlag
		reason := ""
		category := ""
		hitWord := ""

		if inputResult.Action == FilterFlag {
			reason = "用户输入命中标记词: " + inputResult.HitWord
			category = inputResult.Category
			hitWord = inputResult.HitWord
		}

		if outputResult.Action == FilterFlag {
			if reason != "" {
				reason += "; 模型输出命中标记词: " + outputResult.HitWord
			} else {
				reason = "模型输出命中标记词: " + outputResult.HitWord
			}
			if category == "" {
				category = outputResult.Category
			}
			if hitWord == "" {
				hitWord = outputResult.HitWord
			}
		}

		return FilterResult{
			Action:     action,
			Reason:     reason,
			Category:   category,
			Confidence: 0.7,
			HitWord:    hitWord,
		}
	}

	return FilterResult{Action: FilterPass}
}

// CheckContent 便捷函数：使用默认过滤器检查文本
func CheckContent(text string) FilterResult {
	return defaultFilter.Check(text, false)
}

// CheckUserInput 便捷函数：检查用户输入
func CheckUserInput(text string) FilterResult {
	return defaultFilter.CheckInput(text)
}

// CheckLLMOutput 便捷函数：检查 LLM 输出
func CheckLLMOutput(text string) FilterResult {
	return defaultFilter.CheckOutput(text)
}

// AddBlockWord 动态添加拦截词。省略 word 时使用 custom 类别，兼容旧调用。
func (f *ContentFilter) AddBlockWord(category string, words ...string) {
	word := category
	if len(words) > 0 {
		word = words[0]
	} else {
		category = "custom"
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockWords[category] = append(f.blockWords[category], word)
}

// AddFlagWord 动态添加标记词。省略 word 时使用 custom 类别，兼容旧调用。
func (f *ContentFilter) AddFlagWord(category string, words ...string) {
	word := category
	if len(words) > 0 {
		word = words[0]
	} else {
		category = "custom"
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flagWords[category] = append(f.flagWords[category], word)
}

// RemoveWord 移除敏感词
func (f *ContentFilter) RemoveWord(category, word string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for cat, words := range f.blockWords {
		if cat == category {
			var newWords []string
			for _, w := range words {
				if w != word {
					newWords = append(newWords, w)
				}
			}
			f.blockWords[cat] = newWords
			return
		}
	}

	for cat, words := range f.flagWords {
		if cat == category {
			var newWords []string
			for _, w := range words {
				if w != word {
					newWords = append(newWords, w)
				}
			}
			f.flagWords[cat] = newWords
			return
		}
	}
}

// GetStats 获取过滤器统计信息
func (f *ContentFilter) GetStats() map[string]int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := map[string]int{}
	for cat, words := range f.blockWords {
		stats["block_"+cat] = len(words)
	}
	for cat, words := range f.flagWords {
		stats["flag_"+cat] = len(words)
	}

	totalBlock := 0
	for _, words := range f.blockWords {
		totalBlock += len(words)
	}
	totalFlag := 0
	for _, words := range f.flagWords {
		totalFlag += len(words)
	}
	stats["total_block"] = totalBlock
	stats["total_flag"] = totalFlag
	return stats
}

// FilterBlockResponse 拦截时返回的兜底回复
const FilterBlockResponse = "抱歉，该问题涉及的内容暂不支持回答。如有需要，请联系辅导员或学工办公室。"

// 各类型的兜底回复
var filterFallbackResponses = map[string]string{
	"political": "根据相关规定，该内容不予显示。如有疑问请联系辅导员。",
	"porn":      "该内容违反社区规范，不予显示。",
	"violence":  "该内容涉及不当信息，不予显示。如有困惑请寻求辅导员帮助。",
	"illegal":   "该内容涉及违法违规信息，不予显示。",
	"campus":    "该内容涉及校园安全问题，已转交相关部门关注。如遇紧急情况请立即拨打校园保卫处电话。",
}

// GetFallbackResponse 根据类别获取合适的兜底回复
func GetFallbackResponse(category string) string {
	if resp, ok := filterFallbackResponses[category]; ok {
		return resp
	}
	return FilterBlockResponse
}

// IsFlagCategory 判断是否为标记类（心理健康/情感/学业风险）
// 标记类需要通知辅导员关注，但不拦截
func IsFlagCategory(category string) bool {
	switch category {
	case "psychological", "emotional", "academic":
		return true
	default:
		return false
	}
}

// TruncateText 安全截断文本
func TruncateText(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen]) + "..."
}
