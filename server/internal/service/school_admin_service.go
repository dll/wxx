package service

import (
	"context"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// SchoolAdminService 学校管理员角色 AI 功能服务
type SchoolAdminService struct {
	llmClient llm.ChatClient
}

func NewSchoolAdminService(llmClient llm.ChatClient) *SchoolAdminService {
	return &SchoolAdminService{llmClient: llmClient}
}

// SchoolPanorama 全校数字孪生全景
type SchoolPanorama struct {
	TotalStudents int                      `json:"total_students"`
	TotalColleges int                      `json:"total_colleges"`
	HealthScore   float64                  `json:"health_score"`
	RiskStudents  int                      `json:"risk_students"`
	Colleges      []map[string]interface{} `json:"colleges"`
	Trends        map[string][]float64     `json:"trends"`
	AIInsight     string                   `json:"ai_insight"`
	DataSource    string                   `json:"data_source"`
}

func (s *SchoolAdminService) GenerateSchoolPanorama(ctx context.Context) *SchoolPanorama {
	data := &SchoolPanorama{
		TotalStudents: 8000,
		TotalColleges: 8,
		HealthScore:   82.5,
		RiskStudents:  85,
		Colleges: []map[string]interface{}{
			{"name": "计算机学院", "students": 1200, "health": 85.2, "risk": 12, "trend": "up"},
			{"name": "经管学院", "students": 1500, "health": 80.5, "risk": 18, "trend": "stable"},
			{"name": "文学院", "students": 900, "health": 88.0, "risk": 5, "trend": "up"},
			{"name": "理学院", "students": 800, "health": 79.0, "risk": 15, "trend": "down"},
		},
		Trends: map[string][]float64{
			"health":   {78, 79, 80, 81, 82, 82.5},
			"academic": {72, 73, 74, 75, 76, 77},
			"emotion":  {80, 79, 81, 80, 82, 81},
		},
		DataSource: "mock",
	}

	if s.llmClient != nil {
		prompt := "你是高校管理顾问。全校8000学生，8个学院，健康度82.5分。请用40字给出校级宏观感知。"
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.AIInsight = strings.TrimSpace(resp.Content)
			data.DataSource = "ai"
		}
	}

	return data
}

// PolicySimulation 政策影响模拟
type PolicySimulation struct {
	Policy            string   `json:"policy"`
	Adjustment        string   `json:"adjustment"`
	BeneficiaryChange string   `json:"beneficiary_change"`
	RiskPrediction    string   `json:"risk_prediction"`
	ResourceNeeds     []string `json:"resource_needs"`
	DataSource        string   `json:"data_source"`
}

func (s *SchoolAdminService) SimulatePolicy(ctx context.Context, policy, adjustment string) *PolicySimulation {
	return &PolicySimulation{
		Policy:            policy,
		Adjustment:        adjustment,
		BeneficiaryChange: "预计受益学生从1200人增加至1500人(+25%)",
		RiskPrediction:    "可能存在经费缺口约5万元，建议分两期实施",
		ResourceNeeds:     []string{"新增2名辅导教师", "扩充心理咨询室1间", "开发线上申请系统模块"},
		DataSource:        "mock",
	}
}

// CrossCollegeComparison 跨学院对比分析
type CrossCollegeComparison struct {
	Metric      string                   `json:"metric"`
	Rankings    []map[string]interface{} `json:"rankings"`
	Anomalies   []map[string]interface{} `json:"anomalies"`
	Suggestions []string                 `json:"suggestions"`
	DataSource  string                   `json:"data_source"`
}

func (s *SchoolAdminService) CompareColleges(ctx context.Context, metric string) *CrossCollegeComparison {
	if metric == "" {
		metric = "学业健康度"
	}

	return &CrossCollegeComparison{
		Metric: metric,
		Rankings: []map[string]interface{}{
			{"rank": 1, "college": "文学院", "score": 88.0, "change": "+2.5"},
			{"rank": 2, "college": "计算机学院", "score": 85.2, "change": "+1.8"},
			{"rank": 3, "college": "经管学院", "score": 80.5, "change": "-0.5"},
			{"rank": 4, "college": "理学院", "score": 79.0, "change": "-3.2"},
		},
		Anomalies: []map[string]interface{}{
			{"college": "理学院", "metric": metric, "value": 79.0, "deviation": "较均值低3.8分", "reason": "挂科率上升，心理预警增多"},
		},
		Suggestions: []string{
			"建议理学院增加学业辅导资源",
			"推广计算机学院的导师制经验到其他学院",
			"建立学院间帮扶结对机制",
		},
		DataSource: "mock",
	}
}

// SchoolAcademicOverview 校级学情总览
type SchoolAcademicOverview struct {
	Date                string                   `json:"date"`
	CollegeRankings     []map[string]interface{} `json:"college_rankings"`
	CounselorEfficiency []map[string]interface{} `json:"counselor_efficiency"`
	KeyStudentTypes     map[string]int           `json:"key_student_types"`
	IdeologicalCoverage float64                  `json:"ideological_coverage"`
	DataSource          string                   `json:"data_source"`
}

func (s *SchoolAdminService) GenerateAcademicOverview(ctx context.Context) *SchoolAcademicOverview {
	return &SchoolAcademicOverview{
		Date: time.Now().Format("2006-01-02"),
		CollegeRankings: []map[string]interface{}{
			{"college": "计算机学院", "health": 85.2, "academic": 77, "activity": 78, "rank": 2},
			{"college": "经管学院", "health": 80.5, "academic": 75, "activity": 72, "rank": 3},
			{"college": "文学院", "health": 88.0, "academic": 82, "activity": 85, "rank": 1},
		},
		CounselorEfficiency: []map[string]interface{}{
			{"name": "李辅导员", "college": "计算机学院", "talks_monthly": 15, "alerts_handled": 8, "score": 92},
			{"name": "王辅导员", "college": "经管学院", "talks_monthly": 10, "alerts_handled": 5, "score": 78},
		},
		KeyStudentTypes: map[string]int{
			"学业困难": 120, "心理关注": 85, "经济困难": 200, "优秀学生": 350,
		},
		IdeologicalCoverage: 0.95,
		DataSource:          "mock",
	}
}
