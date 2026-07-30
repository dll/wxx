package context_engine

import (
	"testing"
)

func makeMsg(role, content string) HistoryMessage {
	return HistoryMessage{Role: role, Content: content}
}

// TestSelectRelevantHistory_不超限不截断
func TestSelectRelevantHistory_不超限不截断(t *testing.T) {
	msgs := []HistoryMessage{makeMsg("user", "奖学金"), makeMsg("assistant", "申请流程")}
	got := selectRelevantHistory(msgs, "奖学金条件", 10)
	if len(got) != 2 {
		t.Errorf("消息数(%d) <= maxN(10) 时应全部返回，得到 %d 条", 2, len(got))
	}
}

// TestSelectRelevantHistory_超限按相关性截断
func TestSelectRelevantHistory_超限按相关性截断(t *testing.T) {
	msgs := make([]HistoryMessage, 8)
	for i := range msgs {
		msgs[i] = makeMsg("user", "无关话题")
	}
	msgs[6] = makeMsg("user", "奖学金申请")
	msgs[7] = makeMsg("assistant", "奖学金条件如下")
	got := selectRelevantHistory(msgs, "奖学金", 3)
	if len(got) != 3 {
		t.Errorf("期望返回 3 条，得到 %d 条", len(got))
	}
}

// TestSelectRelevantHistory_空消息列表
func TestSelectRelevantHistory_空消息列表(t *testing.T) {
	got := selectRelevantHistory(nil, "奖学金", 5)
	if got != nil && len(got) != 0 {
		t.Errorf("空输入应返回 nil 或空 slice，得到 %v", got)
	}
}

// TestSelectRelevantHistory_maxN零
func TestSelectRelevantHistory_maxN为零(t *testing.T) {
	msgs := []HistoryMessage{makeMsg("user", "测试")}
	got := selectRelevantHistory(msgs, "测试", 0)
	if len(got) != 0 {
		t.Errorf("maxN=0 应返回空 slice，得到 %d 条", len(got))
	}
}

// TestSelectRelevantHistory_无法提取关键词时取最近
func TestSelectRelevantHistory_无关键词取最近(t *testing.T) {
	msgs := []HistoryMessage{
		makeMsg("user", "abc"),
		makeMsg("user", "def"),
		makeMsg("user", "ghi"),
		makeMsg("user", "jkl"),
		makeMsg("user", "mno"),
	}
	// 纯字母可能无法分词，应取最近 maxN 条而不崩溃
	got := selectRelevantHistory(msgs, "xyz", 2)
	if len(got) != 2 {
		t.Errorf("无关键词时应取最近 2 条，得到 %d 条", len(got))
	}
}

// TestSortInts_正确排序
func TestSortInts_正确排序(t *testing.T) {
	arr := []int{5, 3, 8, 1, 4}
	sortInts(arr)
	for i := 1; i < len(arr); i++ {
		if arr[i] < arr[i-1] {
			t.Errorf("sortInts 结果未升序：arr[%d]=%d < arr[%d]=%d", i, arr[i], i-1, arr[i-1])
		}
	}
}

// TestSortInts_空和单元素不崩溃
func TestSortInts_边界不崩溃(t *testing.T) {
	sortInts([]int{})
	sortInts([]int{42})
}
