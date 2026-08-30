package context_engine

import (
	"strings"
)

// ── CE-A2: 查询改写（规则式，零 LLM 成本） ──
//
// 目标：提升 FTS/BM25 召回。三类处理：
//  1. 剥离口语装饰词（"请问""帮我查一下"等）——降低噪声词对 BM25 的稀释；
//  2. 指代消解——"它/这个/上面说的"结合最近历史话题补全检索词；
//  3. 空白归一。
//
// 原始问题仍用于意图分类与结构化检索；改写结果仅用于 FTS 召回。

// decoratePrefixes 口语装饰词（出现在开头时剥离）
var decoratePrefixes = []string{
	"请问", "帮忙", "麻烦", "我想了解", "我想知道", "我想问", "想了解", "想知道",
	"告诉我", "你好", "您好", "hi", "hello", "哈喽", "在吗",
}

// coreferenceWords 指代词：出现时需要结合历史话题补全
var coreferenceWords = []string{"它", "他们", "这个", "那个", "上面说的", "刚才说的", "这个流程", "该流程"}

// synonyms 领域同义扩展：命中左侧词时，把右侧词追加进检索查询（不改动原问题）。
// 解决词面不匹配（口语 vs 制度用语），如 借书/借阅、挂科/不及格。
var synonyms = map[string][]string{
	"借书":   {"借阅"},
	"还书":   {"归还"},
	"开门":   {"开放"},
	"开馆":   {"开放"},
	"关门":   {"闭馆"},
	"挂科":   {"不及格", "补考", "重修"},
	"补考":   {"重修"},
	"评奖":   {"评定"},
	"找谁":   {"咨询", "联系"},
	"宿舍":   {"寝室", "住宿"},
	"退学":   {"休学", "保留学籍"},
	"转专业":  {"专业调整"},
	"奖助学金": {"奖学金", "助学金"},
	"辅导员":  {"学工办"},
	"怎么办":  {"流程", "手续"},
	"在哪办":  {"办理地点"},
	"多少钱":  {"收费标准"},
	"几点":   {"开放时间", "工作时间"},
}

// RewriteQuery 生成用于 FTS 召回的改写查询。
// lastUserMsg 为最近一条用户消息（可空；用于指代消解）。
func RewriteQuery(question, lastUserMsg string) string {
	q := strings.TrimSpace(question)
	if q == "" {
		return q
	}

	// 1. 剥离开头的口语装饰词（可能连续出现，如"你好，请问"）
	for changed := true; changed; {
		changed = false
		lower := strings.ToLower(q)
		for _, p := range decoratePrefixes {
			if strings.HasPrefix(lower, p) {
				q = strings.TrimSpace(q[len(p):])
				q = strings.TrimLeft(q, "，,。.？?！!、：: ")
				changed = true
				break
			}
		}
	}

	// 2. 指代消解：当前问题过短或含指代词时，拼接历史话题
	if lastUserMsg != "" && needsCoreference(q) {
		topic := extractTopic(lastUserMsg)
		if topic != "" && !strings.Contains(q, topic) {
			q = topic + " " + q
		}
	}

	// 3. 领域同义扩展（追加不改写，保留原词面）
	expanded := q
	for term, extra := range synonyms {
		if strings.Contains(q, term) {
			for _, e := range extra {
				if !strings.Contains(q, e) && !strings.Contains(expanded, " "+e+" ") {
					expanded += " " + e
				}
			}
		}
	}
	q = expanded

	// 3.5 中文数字归一：三天→3天（制度文本用阿拉伯数字，如"请假 3 天"）
	q = normalizeChineseNumerals(q)

	// 4. 空白归一
	q = strings.Join(strings.Fields(q), " ")
	return q
}

// normalizeChineseNumerals 将"天/周/个月/年"前的中文数字转阿拉伯数字
func normalizeChineseNumerals(q string) string {
	numMap := map[rune]rune{'零': '0', '一': '1', '两': '2', '二': '2', '三': '3', '四': '4', '五': '5', '六': '6', '七': '7', '八': '8', '九': '9'}
	runes := []rune(q)
	units := map[rune]bool{'天': true, '日': true, '周': true, '月': true, '年': true, '个': true}
	for i := 0; i < len(runes)-1; i++ {
		if repl, ok := numMap[runes[i]]; ok && units[runes[i+1]] {
			runes[i] = repl
		}
	}
	return string(runes)
}

// needsCoreference 判断问题是否依赖上下文（含指代词或过短）
func needsCoreference(q string) bool {
	for _, w := range coreferenceWords {
		if strings.Contains(q, w) {
			return true
		}
	}
	// 过短问题（≤6 字符）通常省略了主语，如"在哪办""需要什么"
	runes := []rune(q)
	return len(runes) <= 6
}

// extractTopic 从历史用户消息提取检索话题：取最长连续中文语段（截断至 8 字，
// 并裁掉结尾虚词），作为问题主语补全。
func extractTopic(msg string) string {
	runes := []rune(msg)
	best := []rune{}
	current := []rune{}
	for _, r := range runes {
		if (r >= 0x4E00 && r <= 0x9FFF) {
			current = append(current, r)
		} else {
			if len(current) > len(best) {
				best = current
			}
			current = nil
		}
	}
	if len(current) > len(best) {
		best = current
	}
	if len(best) > 8 {
		best = best[:8]
	}
	topic := strings.TrimRight(string(best), "的吗呢吧啊是")
	return strings.TrimSpace(topic)
}
