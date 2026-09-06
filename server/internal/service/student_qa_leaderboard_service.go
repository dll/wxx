package service

import (
	"context"
)

// QALeaderboardData 问答排行榜。
type QALeaderboardData struct {
	HotQuestions []map[string]interface{} `json:"hot_questions"`
	TopAnswerers []map[string]interface{} `json:"top_answerers"`
	Contributors []map[string]interface{} `json:"contributors"`
	Period       string                   `json:"period"`
	DataSource   string                   `json:"data_source"`
}

func (s *StudentService) GenerateQALeaderboard(ctx context.Context) *QALeaderboardData {
	_ = ctx
	data := referenceQALeaderboard()
	if s.messageRepo != nil {
		if hot, err := s.messageRepo.GetHotQuestions(10); err == nil && len(hot) > 0 {
			questions := make([]map[string]interface{}, 0, len(hot))
			for i, h := range hot {
				title := h.Title
				if len([]rune(title)) > 40 {
					title = string([]rune(title)[:40]) + "…"
				}
				questions = append(questions, map[string]interface{}{"rank": i + 1, "title": title, "count": h.Count})
			}
			data.HotQuestions, data.DataSource = questions, "real"
		}
	}
	return data
}
