package service

import "testing"

func TestRankForecastTopics(t *testing.T) {
	got := rankForecastTopics(map[string]int{
		"课程":  2,
		"奖学金": 5,
		"成绩":  5,
		"毕业":  1,
	}, 3)
	want := []string{"奖学金", "成绩", "课程"}
	if len(got) != len(want) {
		t.Fatalf("热点数量=%d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("热点[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestRankForecastTopicsIgnoresZeroAndHonorsLimit(t *testing.T) {
	got := rankForecastTopics(map[string]int{"心理": 0, "就业": 1, "宿舍": 2}, 1)
	if len(got) != 1 || got[0] != "宿舍" {
		t.Fatalf("应过滤 0 次并限制数量: %#v", got)
	}
}
