package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
)

// SchoolAdminService 学校管理员角色 AI 功能服务
type SchoolAdminService struct {
	userRepo    *repository.UserRepo
	emotionRepo *repository.EmotionRepo
	twinRepo    *repository.TwinRepo
	llmClient   llm.ChatClient
}

func NewSchoolAdminService(
	userRepo *repository.UserRepo,
	emotionRepo *repository.EmotionRepo,
	twinRepo *repository.TwinRepo,
	llmClient llm.ChatClient,
) *SchoolAdminService {
	return &SchoolAdminService{
		userRepo:    userRepo,
		emotionRepo: emotionRepo,
		twinRepo:    twinRepo,
		llmClient:   llmClient,
	}
}

// aggregateSchoolReal 聚合全校真实概览：学生总数、风险数、学院数、按学院明细
func (s *SchoolAdminService) aggregateSchoolReal() (total, risk, colleges int, perCollege []map[string]interface{}, hasData bool) {
	perCollege = []map[string]interface{}{}
	if s.userRepo != nil {
		if t, err := s.userRepo.Count("student", "", ""); err == nil && t > 0 {
			total = t
			hasData = true
		}
		// 按学院明细
		if names, err := s.userRepo.GetDistinctValues("college", "student", "", ""); err == nil {
			colleges = len(names)
			for _, name := range names {
				if name == "" {
					continue
				}
				cnt, _ := s.userRepo.Count("student", "college", name)
				entry := map[string]interface{}{"name": name, "students": cnt}
				if s.emotionRepo != nil {
					if st, e := s.emotionRepo.GetStats("college", name, "school_admin"); e == nil && st != nil {
						entry["risk"] = st.Urgent + st.High
					}
				}
				perCollege = append(perCollege, entry)
			}
		}
	}
	if s.emotionRepo != nil {
		if st, err := s.emotionRepo.GetStats("school", "", "school_admin"); err == nil && st != nil {
			risk = st.Urgent + st.High
			hasData = true
		}
	}
	return
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
	total, risk, colleges, perCollege, hasData := s.aggregateSchoolReal()

	data := &SchoolPanorama{
		TotalStudents: total,
		TotalColleges: colleges,
		RiskStudents:  risk,
		Colleges:      perCollege,
		Trends:        map[string][]float64{},
		DataSource:    "fallback",
	}
	if hasData {
		data.DataSource = "real"
	}

	if s.llmClient != nil && hasData {
		prompt := fmt.Sprintf("你是高校管理顾问。全校%d名学生，%d个学院，风险关注%d人。请用40字给出校级宏观感知（不得编造未提供的数字）。",
			total, colleges, risk)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.AIInsight = strings.TrimSpace(resp.Content)
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
	result := &PolicySimulation{
		Policy:            policy,
		Adjustment:        adjustment,
		BeneficiaryChange: "需结合具体政策口径与学生数据进一步测算",
		RiskPrediction:    "暂无足够数据进行量化风险预测",
		ResourceNeeds:     []string{},
		DataSource:        "fallback",
	}

	total, _, colleges, _, hasData := s.aggregateSchoolReal()
	if s.llmClient != nil {
		facts := "（暂无结构化统计数据）"
		if hasData {
			facts = fmt.Sprintf("全校在校学生%d人，%d个学院", total, colleges)
		}
		prompt := fmt.Sprintf(
			"你是高校政策分析师。已知真实数据：%s。拟对政策「%s」做调整：%s。请基于数据给出：1)受益范围变化定性判断 2)主要风险 3)所需资源清单。用简洁中文，不得编造未提供的具体数字。",
			facts, policy, adjustment)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.4, MaxTokens: 400,
		})
		if err == nil && resp != nil && resp.Content != "" {
			result.RiskPrediction = strings.TrimSpace(resp.Content)
			result.BeneficiaryChange = ""
			result.ResourceNeeds = nil
			if hasData {
				result.DataSource = "real+ai"
			} else {
				result.DataSource = "ai"
			}
		}
	}

	return result
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
		metric = "学生规模与风险"
	}

	result := &CrossCollegeComparison{
		Metric:      metric,
		Rankings:    []map[string]interface{}{},
		Anomalies:   []map[string]interface{}{},
		Suggestions: []string{},
		DataSource:  "fallback",
	}

	_, _, _, perCollege, hasData := s.aggregateSchoolReal()
	if hasData && len(perCollege) > 0 {
		// 按在校学生数降序排名（真实数据）
		sortByStudentsDesc(perCollege)
		for i, c := range perCollege {
			entry := map[string]interface{}{
				"rank":     i + 1,
				"college":  c["name"],
				"students": c["students"],
			}
			if r, ok := c["risk"]; ok {
				entry["risk"] = r
			}
			result.Rankings = append(result.Rankings, entry)
		}
		result.DataSource = "real"

		// LLM 基于真实排名给建议
		if s.llmClient != nil {
			prompt := fmt.Sprintf("你是高校管理顾问。以下是各学院真实学生规模与风险数据（JSON）：%s。请就「%s」给出3条精炼建议，每条不超过25字，只依据数据、不得编造。",
				jsonCompact(result.Rankings), metric)
			resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
				Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
				Temperature: 0.4, MaxTokens: 220,
			})
			if err == nil && resp != nil && resp.Content != "" {
				result.Suggestions = splitSuggestions(resp.Content)
				result.DataSource = "real+ai"
			}
		}
	}

	return result
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
	overview := &SchoolAcademicOverview{
		Date:                time.Now().Format("2006-01-02"),
		CollegeRankings:     []map[string]interface{}{},
		CounselorEfficiency: []map[string]interface{}{},
		KeyStudentTypes:     map[string]int{},
		DataSource:          "fallback",
	}

	_, risk, _, perCollege, hasData := s.aggregateSchoolReal()
	if hasData {
		sortByStudentsDesc(perCollege)
		for i, c := range perCollege {
			entry := map[string]interface{}{
				"college":  c["name"],
				"students": c["students"],
				"rank":     i + 1,
			}
			if r, ok := c["risk"]; ok {
				entry["risk"] = r
			}
			overview.CollegeRankings = append(overview.CollegeRankings, entry)
		}
		// 关键学生类型：以真实风险预警人数作为「心理关注」项，其余待接入相应数据源
		overview.KeyStudentTypes["心理关注"] = risk
		overview.DataSource = "real"
	}

	return overview
}

// sortByStudentsDesc 按 students 字段降序排序（就地）
func sortByStudentsDesc(items []map[string]interface{}) {
	sort.SliceStable(items, func(i, j int) bool {
		return toInt(items[i]["students"]) > toInt(items[j]["students"])
	})
}

// toInt 宽松地把 interface{} 转成 int
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// jsonCompact 把任意值序列化为紧凑 JSON 字符串（失败返回空串）
func jsonCompact(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// splitSuggestions 把模型返回的多行/编号文本拆成建议条目
func splitSuggestions(text string) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		// 去掉常见前缀：1. 2、 - •
		ln = strings.TrimLeft(ln, "0123456789.、)-•　 \t")
		if ln != "" {
			out = append(out, ln)
		}
	}
	if len(out) == 0 && strings.TrimSpace(text) != "" {
		return []string{strings.TrimSpace(text)}
	}
	return out
}
