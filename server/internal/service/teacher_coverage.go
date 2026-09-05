package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

type KnowledgeCoverage struct {
	CourseName     string   `json:"course_name"`
	SyllabusPoints int      `json:"syllabus_points"`
	ExamPoints     int      `json:"exam_points"`
	TaughtPoints   int      `json:"taught_points"`
	CoverageRate   float64  `json:"coverage_rate"`
	Gaps           []string `json:"gaps"`
	Suggestion     string   `json:"suggestion"`
	DataSource     string   `json:"data_source"`
}

func (s *TeacherService) CheckKnowledgeCoverage(ctx context.Context, courseName string) *KnowledgeCoverage {
	if courseName == "" {
		courseName = "数据结构"
	}
	coverage := &KnowledgeCoverage{CourseName: courseName, SyllabusPoints: 24, ExamPoints: 20, TaughtPoints: 18, CoverageRate: 0.90, Gaps: []string{"红黑树(大纲有/考试有/未讲)", "B+树索引(大纲有/考试有/简略)", "外部排序(大纲有/考试有/未讲)"}, Suggestion: "建议在第12-13周补充红黑树和外部排序内容，B+树部分可结合数据库索引案例加深理解。", DataSource: "fallback"}
	if s.llmClient != nil {
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: fmt.Sprintf("你是课程教学专家。%s课程教学大纲24点，考试覆盖20点，已讲授18点，覆盖率90%%。存在3个知识缺口。请给出30字补课建议。", courseName)}}, Temperature: 0.3, MaxTokens: 200})
		if err == nil && resp != nil && resp.Content != "" {
			coverage.Suggestion += " | AI建议：" + strings.TrimSpace(resp.Content)
			coverage.DataSource = "ai"
		}
	}
	return coverage
}

type IdeologicalSuggestion struct {
	CourseName string              `json:"course_name"`
	Topics     []map[string]string `json:"topics"`
	Materials  []string            `json:"materials"`
	DataSource string              `json:"data_source"`
}

func (s *TeacherService) GenerateIdeologicalSuggestions(ctx context.Context, courseName string) *IdeologicalSuggestion {
	_ = ctx
	if courseName == "" {
		courseName = "数据结构"
	}
	return &IdeologicalSuggestion{CourseName: courseName, Topics: []map[string]string{{"element": "家国情怀", "point": "介绍国产数据库系统发展成就", "method": "案例教学"}, {"element": "科学精神", "point": "算法效率分析中的求真态度", "method": "课堂讨论"}, {"element": "职业道德", "point": "代码规范与软件工程师职业素养", "method": "实践环节"}}, Materials: []string{"《中国计算机学科发展史》选段", "《软件工程师职业道德规范》", "华为高斯数据库技术白皮书"}, DataSource: "reference"}
}
