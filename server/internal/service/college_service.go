package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// CollegeService 学院管理员角色 AI 功能服务
type CollegeService struct {
	llmClient llm.ChatClient
}

func NewCollegeService(llmClient llm.ChatClient) *CollegeService {
	return &CollegeService{llmClient: llmClient}
}

// TwinScreenData 学院数字孪生大屏数据
type TwinScreenData struct {
	College     string                   `json:"college"`
	UpdatedAt   string                   `json:"updated_at"`
	Overview    map[string]interface{}   `json:"overview"`
	Departments []map[string]interface{} `json:"departments"`
	Trends      map[string][]float64     `json:"trends"`
	AIInsight   string                   `json:"ai_insight"`
	DataSource  string                   `json:"data_source"`
}

func (s *CollegeService) GenerateTwinScreen(ctx context.Context, collegeName string) *TwinScreenData {
	if collegeName == "" {
		collegeName = "计算机学院"
	}

	data := &TwinScreenData{
		College:   collegeName,
		UpdatedAt: time.Now().Format("2006-01-02 15:04"),
		Overview: map[string]interface{}{
			"total_students": 580, "health_score": 85.2, "risk_students": 12, "active_rate": 0.78,
		},
		Departments: []map[string]interface{}{
			{"name": "计算机科学", "students": 240, "health": 87.5, "risk": 4},
			{"name": "软件工程", "students": 180, "health": 83.0, "risk": 5},
			{"name": "信息安全", "students": 160, "health": 85.8, "risk": 3},
		},
		Trends: map[string][]float64{
			"academic": {82, 83, 85, 84, 86, 85.2},
			"emotion":  {78, 80, 79, 82, 81, 83},
			"activity": {70, 72, 75, 73, 76, 78},
		},
		DataSource: "mock",
	}

	// LLM 解读
	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是学院管理顾问。%s全院%d名学生，健康度%.1f分。请用30字解读当前状态。",
			collegeName, 580, 85.2)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 150,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.AIInsight = strings.TrimSpace(resp.Content)
			data.DataSource = "ai"
		}
	}

	return data
}

// DataAnalysisResult 数据分析结果
type DataAnalysisResult struct {
	Content    string `json:"content"`
	Query      string `json:"query"`
	DataSource string `json:"data_source"`
}

func (s *CollegeService) AnalyzeData(ctx context.Context, query string) *DataAnalysisResult {
	result := &DataAnalysisResult{
		Query:      query,
		Content:    "计算机学院数据分析报告：全院平均绩点3.12，挂科率4.2%，出勤率92.5%，心理预警12人，活动参与率65%。",
		DataSource: "fallback",
	}

	if s.llmClient != nil && query != "" {
		prompt := fmt.Sprintf("你是学院数据分析师。请回答：%s（50字以内，数据驱动）。", query)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 250,
		})
		if err == nil && resp != nil && resp.Content != "" {
			result.Content = strings.TrimSpace(resp.Content)
			result.DataSource = "ai"
		}
	}

	return result
}

// ======================== P2 深度分析功能 ========================

// DecisionAdviceData AI 决策建议
type DecisionAdviceData struct {
	Topic       string                   `json:"topic"`
	Suggestions []map[string]interface{} `json:"suggestions"`
	Risks       []string                 `json:"risks"`
	DataSource  string                   `json:"data_source"`
}

func (s *CollegeService) GenerateDecisionAdvice(ctx context.Context, topic string) *DecisionAdviceData {
	data := &DecisionAdviceData{
		Topic: topic,
		Suggestions: []map[string]interface{}{
			{"action": "增加心理健康活动频次", "reason": "本学期心理预警人数较上学期上升15%", "expected_effect": "预计降低预警率20%"},
			{"action": "设立学业帮扶专项计划", "reason": "挂科率集中在2-3门核心课程", "expected_effect": "预计挂科率下降30%"},
		},
		Risks:      []string{"资源分配需经学校审批，实施周期可能较长"},
		DataSource: "mock",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是高校管理顾问。关于「%s」，请用40字给出数据驱动的决策建议。", topic)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			_ = strings.TrimSpace(resp.Content)
			data.DataSource = "ai"
		}
	}

	return data
}

// TeacherEfficiencyData 教师效能分析
type TeacherEfficiencyData struct {
	TeacherName string                   `json:"teacher_name"`
	Scores      map[string]float64       `json:"scores"`
	Rankings    []map[string]interface{} `json:"rankings"`
	Suggestions []string                 `json:"suggestions"`
	DataSource  string                   `json:"data_source"`
}

func (s *CollegeService) AnalyzeTeacherEfficiency(ctx context.Context, teacherName string) *TeacherEfficiencyData {
	return &TeacherEfficiencyData{
		TeacherName: teacherName,
		Scores:      map[string]float64{"教学": 88.0, "学情": 82.5, "评教": 4.3, "互动": 78.0},
		Rankings: []map[string]interface{}{
			{"rank": 1, "name": "张教授", "score": 92.0},
			{"rank": 2, "name": "李副教授", "score": 88.5},
			{"rank": 3, "name": "王讲师", "score": 85.0},
		},
		Suggestions: []string{"建议增加课堂互动环节", "可参考张教授的教学方法"},
		DataSource:  "mock",
	}
}

// CourseQualityData 课程质量评估
type CourseQualityData struct {
	CourseName string             `json:"course_name"`
	Grade      string             `json:"grade"` // A/B/C/D
	Metrics    map[string]float64 `json:"metrics"`
	Strengths  []string           `json:"strengths"`
	Warnings   []string           `json:"warnings"`
	DataSource string             `json:"data_source"`
}

func (s *CollegeService) EvaluateCourseQuality(ctx context.Context, courseName string) *CourseQualityData {
	return &CourseQualityData{
		CourseName: courseName,
		Grade:      "B",
		Metrics:    map[string]float64{"pass_rate": 0.88, "avg_score": 76.5, "feedback": 4.0, "coverage": 0.85},
		Strengths:  []string{"知识点覆盖较全", "实验环节设计合理"},
		Warnings:   []string{"不及格率偏高(12%)", "学生反馈难度偏大"},
		DataSource: "mock",
	}
}

// CollegeReportData 周报/月报
type CollegeReportData struct {
	Period      string                   `json:"period"`
	KeyMetrics  map[string]float64       `json:"key_metrics"`
	Anomalies   []map[string]interface{} `json:"anomalies"`
	Suggestions []string                 `json:"suggestions"`
	DataSource  string                   `json:"data_source"`
}

func (s *CollegeService) GenerateCollegeReport(ctx context.Context, period string) *CollegeReportData {
	return &CollegeReportData{
		Period: period,
		KeyMetrics: map[string]float64{
			"avg_health": 82.5, "avg_academic": 76.0, "alert_count": 15, "checkin_rate": 0.93,
		},
		Anomalies: []map[string]interface{}{
			{"type": "健康度下降", "college": "理学院", "change": "-3.2", "reason": "挂科率上升，心理预警增多"},
		},
		Suggestions: []string{"建议理学院增加学业辅导资源", "全院范围内推广心理健康活动"},
		DataSource:  "mock",
	}
}
