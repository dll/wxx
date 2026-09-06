package service

import (
	"context"
	"strings"
	"time"
)

// HotTopicsData 热点关注。
type HotTopicsData struct {
	Topics     []map[string]interface{} `json:"topics"`
	UpdatedAt  string                   `json:"updated_at"`
	DataSource string                   `json:"data_source"`
}

func (s *StudentService) GenerateHotTopics(ctx context.Context) *HotTopicsData {
	_ = ctx
	if s.kbRepo != nil {
		acts, err := s.kbRepo.List("", "", "published", "Activity", 0, 6)
		if err == nil && len(acts) > 0 {
			topics := make([]map[string]interface{}, 0, len(acts))
			heat := 95
			for i, a := range acts {
				summary := a.Summary
				if strings.TrimSpace(summary) == "" {
					summary = a.Title
				}
				if len([]rune(summary)) > 60 {
					summary = string([]rune(summary)[:60]) + "…"
				}
				trend := "stable"
				if i < 2 {
					trend = "rising"
				}
				topics = append(topics, map[string]interface{}{"id": a.ResourceID, "title": a.Title, "heat": heat, "trend": trend, "summary": summary, "source_link": a.SourceLink})
				heat -= 10
				if heat < 40 {
					heat = 40
				}
			}
			return &HotTopicsData{Topics: topics, UpdatedAt: time.Now().Format("2006-01-02 15:04"), DataSource: "real"}
		}
	}
	return fallbackHotTopics()
}
