package context_engine

import (
	"strings"
)

// ── CE-10: 智能历史选取 ──
// 不再固定取最近 6 条，而是按相关性选取最相关的历史片段

// selectRelevantHistory 从最近消息中选取与当前问题最相关的 maxN 条
// 保持时间顺序（先旧后新），确保对话逻辑连贯
func selectRelevantHistory(msgs []HistoryMessage, question string, maxN int) []HistoryMessage {
	if len(msgs) == 0 || maxN <= 0 {
		return nil
	}

	// 如果消息数不超过 maxN，全部返回
	if len(msgs) <= maxN {
		return msgs
	}

	keywords := extractKeywords(question)
	if len(keywords) == 0 {
		// 无法提取关键词，取最近 maxN 条
		return msgs[len(msgs)-maxN:]
	}

	// 计算每条消息的相关性得分
	type scored struct {
		idx   int
		score int
	}
	scores := make([]scored, len(msgs))
	for i, m := range msgs {
		s := 0
		lower := strings.ToLower(m.Content)
		for _, kw := range keywords {
			s += strings.Count(lower, strings.ToLower(kw))
		}
		// 最近的消息加分（时间衰减逆向：越新分越高）
		recencyBonus := i // 越靠后（越新）index 越大
		scores[i] = scored{idx: i, score: s*10 + recencyBonus}
	}

	// 选择得分最高的 maxN 条
	// 用简单选择排序（数据量小，最多 10 条）
	selected := make([]int, 0, maxN)
	used := make(map[int]bool)

	for len(selected) < maxN {
		bestIdx := -1
		bestScore := -1
		for _, sc := range scores {
			if used[sc.idx] {
				continue
			}
			if sc.score > bestScore {
				bestScore = sc.score
				bestIdx = sc.idx
			}
		}
		if bestIdx < 0 {
			break
		}
		selected = append(selected, bestIdx)
		used[bestIdx] = true
	}

	// 按原始顺序排列（保持对话连贯性）
	sortInts(selected)

	result := make([]HistoryMessage, 0, len(selected))
	for _, idx := range selected {
		result = append(result, msgs[idx])
	}
	return result
}

// sortInts 简单升序排序（数据量 ≤ 10，无需导入 sort 包）
func sortInts(arr []int) {
	for i := 0; i < len(arr)-1; i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j] < arr[i] {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}
