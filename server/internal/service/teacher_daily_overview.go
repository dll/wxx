package service

import (
	"context"
	"time"
)

// DailyOverview 今日授课概览
type DailyOverview struct {
	Date           string   `json:"date"`
	Greeting       string   `json:"greeting"`
	CourseName     string   `json:"course_name"`
	ClassName      string   `json:"class_name"`
	StudentCount   int      `json:"student_count"`
	LastReflection string   `json:"last_reflection"`
	KeyKnowledge   []string `json:"key_knowledge"`
	DataSource     string   `json:"data_source"`
}

// GenerateDailyOverview keeps the honest empty-data behavior until teaching assignments are connected.
func (s *TeacherService) GenerateDailyOverview(ctx context.Context) *DailyOverview {
	_ = ctx
	hour := time.Now().Hour()
	greeting := "下午好！今天的教学工作辛苦了。"
	if hour < 11 {
		greeting = "早上好！今天的课堂准备好了吗？"
	} else if hour < 14 {
		greeting = "中午好！下午的课要加油哦。"
	}
	return &DailyOverview{Date: time.Now().Format("2006-01-02"), Greeting: greeting, DataSource: "real", KeyKnowledge: []string{}}
}
