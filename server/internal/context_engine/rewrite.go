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

	// 3. 空白归一
	q = strings.Join(strings.Fields(q), " ")
	return q
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

// extractTopic 从历史用户消息提取核心话题（取最长语段，截断至 12 字）
func extractTopic(msg string) string {
	segments := strings.FieldsFunc(msg, func(r rune) bool {
		return r == '，' || r == '。' || r == '？' || r == '！' ||
			r == ',' || r == '.' || r == '?' || r == '!' ||
			r == ' ' || r == '\n' || r == '\t' || r == '、' || r == '：' || r == '；'
	})
	best := ""
	for _, seg := range segments {
		if len([]rune(seg)) > len([]rune(best)) {
			best = seg
		}
	}
	return truncate(best, 12)
}
