package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
)

// CollegeService 学院管理员角色 AI 功能服务
type CollegeService struct {
	userRepo    *repository.UserRepo
	emotionRepo *repository.EmotionRepo
	twinRepo    *repository.TwinRepo
	llmClient   llm.ChatClient
}

func NewCollegeService(
	userRepo *repository.UserRepo,
	emotionRepo *repository.EmotionRepo,
	twinRepo *repository.TwinRepo,
	llmClient llm.ChatClient,
) *CollegeService {
	return &CollegeService{
		userRepo:    userRepo,
		emotionRepo: emotionRepo,
		twinRepo:    twinRepo,
		llmClient:   llmClient,
	}
}

// collegeMetrics 从真实数据聚合学院概览指标
type collegeMetrics struct {
	TotalStudents int
	RiskStudents  int
	HealthScore   float64
	HasData       bool
}

// aggregateCollegeMetrics 按学院归属聚合真实指标：学生数、风险数、健康度
func (s *CollegeService) aggregateCollegeMetrics(ownerID string) collegeMetrics {
	m := collegeMetrics{}
	if s.userRepo != nil {
		if total, err := s.userRepo.Count("student", "college", ownerID); err == nil && total > 0 {
			m.TotalStudents = total
			m.HasData = true
		}
	}
	if s.emotionRepo != nil {
		if stats, err := s.emotionRepo.GetStats("college", ownerID, "college_admin"); err == nil && stats != nil {
			m.RiskStudents = stats.Urgent + stats.High
			m.HasData = true
		}
	}
	if s.twinRepo != nil {
		if snaps, err := s.twinRepo.ListSnapshotsByScope("college", ownerID, "", "", 500); err == nil && len(snaps) > 0 {
			var sum float64
			for _, sp := range snaps {
				sum += (sp.AcademicScore + sp.AbilityScore + sp.IdeologicalScore + sp.EmotionalScore + sp.SocialScore) / 5.0
			}
			m.HealthScore = sum / float64(len(snaps))
			m.HasData = true
		}
	}
	return m
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

func (s *CollegeService) GenerateTwinScreen(ctx context.Context, collegeName, ownerID string) *TwinScreenData {
	if collegeName == "" {
		collegeName = "计算机学院"
	}
	if ownerID == "" {
		ownerID = collegeName
	}

	m := s.aggregateCollegeMetrics(ownerID)

	data := &TwinScreenData{
		College:   collegeName,
		UpdatedAt: time.Now().Format("2006-01-02 15:04"),
	}

	if m.HasData {
		// 真实聚合数据
		activeRate := 0.0
		if m.TotalStudents > 0 {
			activeRate = float64(m.TotalStudents-m.RiskStudents) / float64(m.TotalStudents)
		}
		data.Overview = map[string]interface{}{
			"total_students": m.TotalStudents,
			"health_score":   roundTo1(m.HealthScore),
			"risk_students":  m.RiskStudents,
			"active_rate":    roundTo2(activeRate),
		}
		data.Departments = []map[string]interface{}{}
		data.Trends = map[string][]float64{}
		data.DataSource = "real"
	} else {
		// 兜底：无任何真实数据时给占位并明确标注
		data.Overview = map[string]interface{}{
			"total_students": 0, "health_score": 0.0, "risk_students": 0, "active_rate": 0.0,
		}
		data.Departments = []map[string]interface{}{}
		data.Trends = map[string][]float64{}
		data.DataSource = "fallback"
	}

	// LLM 解读（基于真实指标）
	if s.llmClient != nil && m.HasData {
		prompt := fmt.Sprintf("你是学院管理顾问。%s全院%d名学生，风险关注%d人，健康度%.1f分。请用30字解读当前状态。",
			collegeName, m.TotalStudents, m.RiskStudents, m.HealthScore)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 150,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.AIInsight = strings.TrimSpace(resp.Content)
		}
	}

	return data
}

// roundTo1 保留一位小数
func roundTo1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}

// roundTo2 保留两位小数
func roundTo2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// DataAnalysisResult 数据分析结果
type DataAnalysisResult struct {
	Content    string `json:"content"`
	Query      string `json:"query"`
	DataSource string `json:"data_source"`
}

func (s *CollegeService) AnalyzeData(ctx context.Context, query, ownerID string) *DataAnalysisResult {
	result := &DataAnalysisResult{
		Query:      query,
		Content:    "暂无足够数据生成分析，请先完成学生数据与画像同步。",
		DataSource: "fallback",
	}

	m := s.aggregateCollegeMetrics(ownerID)

	if s.llmClient != nil && query != "" {
		// 将真实聚合指标作为事实注入，避免模型编造数字
		facts := "（暂无结构化统计数据）"
		if m.HasData {
			facts = fmt.Sprintf("在校学生%d人，风险关注%d人，综合健康度%.1f分",
				m.TotalStudents, m.RiskStudents, m.HealthScore)
		}
		prompt := fmt.Sprintf(
			"你是学院数据分析师。已知本学院真实数据：%s。请仅基于这些数据回答：%s（50字以内，不得编造未提供的具体数字）。",
			facts, query)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 250,
		})
		if err == nil && resp != nil && resp.Content != "" {
			result.Content = strings.TrimSpace(resp.Content)
			if m.HasData {
				result.DataSource = "real+ai"
			} else {
				result.DataSource = "ai"
			}
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
	AIAdvice    string                   `json:"ai_advice"`
	DataSource  string                   `json:"data_source"`
}

func (s *CollegeService) GenerateDecisionAdvice(ctx context.Context, topic string) *DecisionAdviceData {
	if topic == "" {
		topic = "学院管理决策"
	}
	data := &DecisionAdviceData{
		Topic:       topic,
		Suggestions: []map[string]interface{}{},
		Risks:       []string{},
		DataSource:  "fallback",
	}

	// 决策建议不臆造具体百分比，交由 LLM 基于问题给定性建议；无 LLM 则返回空建议
	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是高校管理顾问。关于「%s」，请给出2条数据驱动的定性决策建议和主要风险，每条不超过30字，不得编造具体数字。", topic)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 300,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.AIAdvice = strings.TrimSpace(resp.Content)
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
		DataSource:  "reference",
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
		DataSource: "reference",
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
		DataSource:  "reference",
	}
}

// ProcessStepData 流程步骤管理数据
type ProcessStepData struct {
	ProcessID   string                   `json:"process_id"`
	ProcessName string                   `json:"process_name"`
	Steps       []map[string]interface{} `json:"steps"`
	Total       int                      `json:"total"`
	DataSource  string                   `json:"data_source"`
}

// ManageProcessSteps 学院流程步骤编辑（学院管辖范围）
func (s *CollegeService) ManageProcessSteps(ctx context.Context, processID, ownerID string) *ProcessStepData {
	if processID == "" {
		processID = "transfer"
	}

	steps := []map[string]interface{}{
		{"step": 1, "title": "学生在线申请", "handler": "学生本人", "editable": false, "status": "系统自动"},
		{"step": 2, "title": "学院审核", "handler": "学院教学办", "editable": true, "status": "待配置审核人"},
		{"step": 3, "title": "教务处审批", "handler": "教务处", "editable": false, "status": "上级流程"},
	}

	return &ProcessStepData{
		ProcessID:   processID,
		ProcessName: "转专业办理流程",
		Steps:       steps,
		Total:       len(steps),
		DataSource:  "reference",
	}
}
