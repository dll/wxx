package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
)

// TestStudentService_GreetingByHour 各时段问候语
func TestStudentService_GreetingByHour(t *testing.T) {
	cases := []struct {
		hour int
		want string
	}{
		{3, "夜深"},
		{9, "早上好"},
		{12, "中午好"},
		{15, "下午好"},
		{20, "晚上好"},
	}
	for _, c := range cases {
		got := greetingByHour(c.hour, "张三")
		if !contains(got, c.want) {
			t.Errorf("hour=%d 期望含 %q，实际 %q", c.hour, c.want, got)
		}
	}
}

// TestStudentService_ParseGreetingAndMotto 解析 LLM 输出
func TestStudentService_ParseGreetingAndMotto(t *testing.T) {
	greeting, motto := parseGreetingAndMotto("问候语：早上好啊，张三\n激励语：今天也是充满希望的一天")
	if greeting != "早上好啊，张三" {
		t.Errorf("问候语解析错误：%s", greeting)
	}
	if motto != "今天也是充满希望的一天" {
		t.Errorf("激励语解析错误：%s", motto)
	}
}

// TestStudentService_FallbackBriefing 用户不存在时兜底
func TestStudentService_FallbackBriefing(t *testing.T) {
	s := &StudentService{} // nil repos
	briefing, err := s.GenerateDailyBriefing(context.Background(), 999)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if briefing.DataSource != "fallback" {
		t.Errorf("应返回 fallback 标记，实际：%s", briefing.DataSource)
	}
	if briefing.Motto == "" {
		t.Error("应返回兜底激励语")
	}
}

// TestCounselorService_BuildFocusedStudents 关注名单去重 + Top N
func TestCounselorService_BuildFocusedStudents(t *testing.T) {
	alerts := []*model.EmotionLog{
		{UserID: 1, Username: "张三", RiskLevel: "high", MessageText: "焦虑严重"},
		{UserID: 1, Username: "张三", RiskLevel: "medium", MessageText: "重复"}, // 应被去重
		{UserID: 2, Username: "李四", RiskLevel: "medium", MessageText: "压力大"},
		{UserID: 3, Username: "王五", RiskLevel: "low", MessageText: "ok"},
		{UserID: 4, Username: "赵六", RiskLevel: "urgent", MessageText: "紧急"},
	}
	students := buildFocusedStudents(alerts, 3)
	if len(students) != 3 {
		t.Errorf("期望返回 3 人，实际 %d", len(students))
	}
	// 第 1 条应是 UserID=1（去重后）
	if students[0].UserID != 1 {
		t.Errorf("首条应为 UserID=1，实际 %d", students[0].UserID)
	}
}

// TestCounselorService_CalcClassHealthScore 班级健康度计算
func TestCounselorService_CalcClassHealthScore(t *testing.T) {
	cases := []struct {
		stats *model.EmotionStats
		want  float64
	}{
		{&model.EmotionStats{}, 100},
		{&model.EmotionStats{High: 1}, 92},                    // 100-8
		{&model.EmotionStats{High: 1, Medium: 2, Low: 3}, 83}, // 100-8-6-3
		{&model.EmotionStats{High: 100}, 0},                   // 下限 0
	}
	for _, c := range cases {
		got := calcClassHealthScore(c.stats)
		if got != c.want {
			t.Errorf("stats=%+v 期望 %.1f，实际 %.1f", c.stats, c.want, got)
		}
	}
}

// TestCounselorService_SuggestionByRisk 不同风险级别的建议差异化
func TestCounselorService_SuggestionByRisk(t *testing.T) {
	if suggestionByRisk("urgent") == suggestionByRisk("low") {
		t.Error("不同风险级别建议不应相同")
	}
	if suggestionByRisk("high") == "" {
		t.Error("high 建议不应为空")
	}
}

// contains 已在 emotion_service_test.go 中定义，此处复用
