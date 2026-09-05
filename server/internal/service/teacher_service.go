// LessonPlan/ExamPaper/Grading 在真实数据未配置时返回明确标注的 fallback，
// 接入真实备课、题库和批改数据后可替换数据源，不影响现有调用契约。
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// TeacherService 教师角色 AI 功能服务
type TeacherService struct {
	llmClient llm.ChatClient
}

// NewTeacherService 创建教师服务
func NewTeacherService(llmClient llm.ChatClient) *TeacherService {
	return &TeacherService{llmClient: llmClient}
}

// LessonPlan 教案结构
type LessonPlan struct {
	Topic        string   `json:"topic"`
	Outline      string   `json:"outline"`
	KeyPoints    []string `json:"key_points"`
	Difficulties []string `json:"difficulties"`
	Strategies   []string `json:"strategies"`
	Interactions []string `json:"interactions"`
	Homework     []string `json:"homework"`
	DataSource   string   `json:"data_source"` // ai/fallback
}

// GenerateLessonPlan 用 LLM 生成教案
func (s *TeacherService) GenerateLessonPlan(ctx context.Context, topic, courseID string) (*LessonPlan, error) {
	if topic == "" {
		topic = "二叉树遍历"
	}

	if s.llmClient != nil {
		plan, err := s.generatePlanWithLLM(ctx, topic, courseID)
		if err == nil && plan != nil {
			return plan, nil
		}
	}

	return s.fallbackPlan(topic), nil
}

func (s *TeacherService) generatePlanWithLLM(ctx context.Context, topic, courseID string) (*LessonPlan, error) {
	var b strings.Builder
	b.WriteString("你是一位经验丰富的高校教师。请为以下课程主题生成一份教案。\n\n")
	b.WriteString(fmt.Sprintf("课程主题：%s\n", topic))
	if courseID != "" {
		b.WriteString(fmt.Sprintf("课程编号：%s\n", courseID))
	}
	b.WriteString("\n请按以下 JSON 格式输出（严格遵守）：\n")
	b.WriteString(`{
  "outline": "教案大纲（含教学目标、重难点、教学过程）",
  "key_points": ["重点1", "重点2", "重点3"],
  "difficulties": ["难点1", "难点2"],
  "strategies": ["教学策略1", "教学策略2"],
  "interactions": ["互动设计1", "互动设计2"],
  "homework": ["课后作业1", "课后作业2"]
}`)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.4,
		MaxTokens:   1200,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	plan := parseLessonPlanJSON(resp.Content, topic)
	plan.DataSource = "ai"
	return plan, nil
}

// ExamPaper AI 考试出题结果
type ExamPaper struct {
	Title           string                   `json:"title"`
	TotalScore      int                      `json:"total_score"`
	Duration        int                      `json:"duration"`
	Sections        []map[string]interface{} `json:"sections"`
	SampleQuestions []map[string]interface{} `json:"sample_questions"`
	DataSource      string                   `json:"data_source"`
}

// GenerateExam 用 LLM 生成考试试卷
func (s *TeacherService) GenerateExam(ctx context.Context, courseName string) (*ExamPaper, error) {
	if courseName == "" {
		courseName = "数据结构"
	}

	if s.llmClient != nil {
		paper, err := s.generateExamWithLLM(ctx, courseName)
		if err == nil && paper != nil {
			return paper, nil
		}
	}

	return s.fallbackExam(courseName), nil
}

func (s *TeacherService) generateExamWithLLM(ctx context.Context, courseName string) (*ExamPaper, error) {
	prompt := fmt.Sprintf(
		"你是一位高校教师。请为「%s」课程设计一份期中考试试卷。\n"+
			"要求：满分100分，时长120分钟，包含选择题(10题x3分)、填空题(5题x4分)、简答题(3题x10分)、编程题(2题x10分)。\n"+
			"请给出各题型的一个样题（含题干、选项、答案）。\n"+
			"严格按以下 JSON 结构输出：{\"title\":\"...\",\"total_score\":100,\"duration\":120,\"sections\":[{\"type\":\"选择题\",\"count\":10,\"score_each\":3,\"subtotal\":30}],\"sample_questions\":[{\"type\":\"选择题\",\"question\":\"...\",\"options\":[\"A\",\"B\",\"C\",\"D\"],\"answer\":\"B\"}]}",
		courseName)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.4,
		MaxTokens:   1000,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	paper := parseExamPaper(resp.Content, courseName)
	if paper == nil {
		return nil, fmt.Errorf("LLM 试卷解析失败")
	}
	paper.DataSource = "ai"
	return paper, nil
}

// parseExamPaper 解析 LLM 返回的试卷 JSON（兼容 markdown 代码块包裹），解析失败返回 nil
// ─── P2 深度功能 ───

// ======================== P1 剩余方法 ========================
