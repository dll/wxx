package service

import (
	"encoding/json"
)

// parseExamPaper parses an LLM exam response and applies safe defaults.
func parseExamPaper(text, courseName string) *ExamPaper {
	jsonStr := extractJSON(text)
	var parsed struct {
		Title           string                   `json:"title"`
		TotalScore      int                      `json:"total_score"`
		Duration        int                      `json:"duration"`
		Sections        []map[string]interface{} `json:"sections"`
		SampleQuestions []map[string]interface{} `json:"sample_questions"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil
	}
	paper := &ExamPaper{Title: parsed.Title, TotalScore: parsed.TotalScore, Duration: parsed.Duration, Sections: parsed.Sections, SampleQuestions: parsed.SampleQuestions}
	if paper.Title == "" {
		paper.Title = courseName + "期中考试"
	}
	if paper.TotalScore == 0 {
		paper.TotalScore = 100
	}
	if paper.Duration == 0 {
		paper.Duration = 120
	}
	if len(paper.SampleQuestions) == 0 {
		return nil
	}
	return paper
}

func (s *TeacherService) fallbackExam(courseName string) *ExamPaper {
	return &ExamPaper{
		Title: courseName + "期中考试", TotalScore: 100, Duration: 120,
		Sections: []map[string]interface{}{
			{"type": "选择题", "count": 10, "score_each": 3, "subtotal": 30},
			{"type": "填空题", "count": 5, "score_each": 4, "subtotal": 20},
			{"type": "简答题", "count": 3, "score_each": 10, "subtotal": 30},
			{"type": "编程题", "count": 2, "score_each": 10, "subtotal": 20},
		},
		SampleQuestions: []map[string]interface{}{
			{"type": "选择题", "question": "在一棵完全二叉树中，若第i层有k个节点，则第i+1层最多有几个节点？", "options": []string{"k", "2k", "k+1", "2k+1"}, "answer": "B"},
			{"type": "编程题", "question": "实现二叉树的层次遍历，返回每层节点值的列表", "answer": "使用队列BFS实现"},
		},
		DataSource: "fallback",
	}
}
