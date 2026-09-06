package service

import (
	"context"
	"strings"
)

// QAPlazaData 问答广场。
type QAPlazaData struct {
	HotQuestions []map[string]interface{} `json:"hot_questions"`
	Categories   []string                 `json:"categories"`
	MyPosts      int                      `json:"my_posts"`
	MyAnswers    int                      `json:"my_answers"`
	DataSource   string                   `json:"data_source"`
}

func (s *StudentService) GenerateQAPlaza(ctx context.Context) *QAPlazaData {
	_ = ctx
	if s.kbRepo != nil {
		faqs, err := s.kbRepo.List("", "", "published", "FAQ", 0, 8)
		if err == nil && len(faqs) > 0 {
			hot := make([]map[string]interface{}, 0, len(faqs))
			for _, f := range faqs {
				ans := f.Summary
				if strings.TrimSpace(ans) == "" {
					ans = f.Content
				}
				if len([]rune(ans)) > 120 {
					ans = string([]rune(ans)[:120]) + "…"
				}
				hot = append(hot, map[string]interface{}{"id": f.ResourceID, "title": f.Title, "author": "知识库", "answers": 1, "views": 0, "ai_answer": ans, "tags": parseKnowledgeTags(f.Tags), "source_link": f.SourceLink})
			}
			return &QAPlazaData{HotQuestions: hot, Categories: []string{"学业", "生活", "政策", "心理", "就业", "竞赛"}, DataSource: "real"}
		}
	}
	return fallbackQAPlaza()
}
