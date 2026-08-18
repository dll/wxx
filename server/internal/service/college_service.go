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
//
// 【健康度口径锁定窗口 - 有意修复】
// 健康度口径由「最近 500 条快照求均值」改为「全部有快照学生聚合（按 user 去重取最新快照）」。
// 旧的 ListSnapshotsByScope(..., 500) 500 上限在院学生 >500 时会静默漏样本，导致全院健康度失真。
// 此处已改用 AggregateSnapshotsByScope（SQL AVG 聚合，无 limit 上限），口径窗口已锁定为全量有快照学生。
//
// 健康度口径（与既有行为一致）：先每人算五维均分再平均。因每人快照五维齐全，
// 该值 = (学业均值+能力均值+思想均值+情感均值+社交均值)/5，可由 SQL 聚合 AVG 直线得到。
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
		if agg, err := s.twinRepo.AggregateSnapshotsByScope("college", ownerID, "", "", ""); err == nil && agg != nil && agg.Overall.Count > 0 {
			o := agg.Overall
			m.HealthScore = (o.Academic + o.Ability + o.Ideological + o.Emotional + o.Social) / 5.0
			m.HasData = true
		}
	}
	return m
}

// FiveDimEntry 单个维度在学院层面的聚合结果。
// Score:=nil 表示该维 0 样本（data_source=not_available），绝不硬编码均值。
type FiveDimEntry struct {
	Key         string   `json:"key"`          // academic|ability|ideological|emotional|social
	Name        string   `json:"name"`         // 学业|能力|思想|情感|社交
	Score       *float64 `json:"score"`        // 院级均值；0 样本 → null
	Level       string   `json:"level"`        // 优秀/良好/待提升/数据积累中（沿用 scoreLevel 口径）
	SampleCount int      `json:"sample_count"` // 该维参与聚合的快照数
	DataSource  string   `json:"data_source"`  // real | not_available
}

// CollegeFiveDim 学院五维全院聚合结果。
type CollegeFiveDim struct {
	SampleCount int            `json:"sample_count"`        // 全院参与聚合的快照数
	Dimensions  []FiveDimEntry `json:"dimensions"`         // 5 维
	TrendNote   string         `json:"trend_note,omitempty"` // 趋势说明（无历史快照→数据积累中）
}

// fiveDimDefs 五维元数据（顺序固定，供前端雷达主轴排序）
var fiveDimDefs = []struct {
	Key  string
	Name string
}{
	{"academic", "学业"},
	{"ability", "能力"},
	{"ideological", "思想"},
	{"emotional", "情感"},
	{"social", "社交"},
}

// aggregateCollegeFiveDim 按学院归属聚合五维均值 + 各维样本数。
// SQL AVG 聚合天然无 limit 上限；返回各维 score 与 sample_count。
// 整体无快照时返回 (nil, nil)（调用方按 0 样本渲染「数据积累中」）。
func (s *CollegeService) aggregateCollegeFiveDim(ownerID, major, className string) *CollegeFiveDim {
	if s.twinRepo == nil {
		return nil
	}
	agg, err := s.twinRepo.AggregateSnapshotsByScope("college", ownerID, major, className, "")
	if err != nil || agg == nil || agg.Overall.Count == 0 {
		return nil
	}
	o := agg.Overall
	dims := make([]FiveDimEntry, 0, len(fiveDimDefs))
	// 各维均值按真实快照数平均；整体 sample_count 为该分组快照数。
	dimVals := map[string]float64{
		"academic": o.Academic, "ability": o.Ability, "ideological": o.Ideological,
		"emotional": o.Emotional, "social": o.Social,
	}
	for _, d := range fiveDimDefs {
		v := dimVals[d.Key]
		score := roundTo1(v)
		entry := FiveDimEntry{
			Key: d.Key, Name: d.Name, SampleCount: o.Count, DataSource: "real",
			Score: &score, Level: scoreLevel(score),
		}
		dims = append(dims, entry)
	}
	return &CollegeFiveDim{
		SampleCount: o.Count,
		Dimensions:  dims,
		TrendNote:   "趋势数据积累中（暂无可对比的历史快照）",
	}
}

// buildDepartments 按专业（major）填充学院大屏 departments 下钻条目。
// 每专业含样本数与五维均值；仅列有快照的 major，不编造不存在的专业。
func (s *CollegeService) buildDepartments(ownerID, className string) []map[string]interface{} {
	if s.twinRepo == nil {
		return []map[string]interface{}{}
	}
	agg, err := s.twinRepo.AggregateSnapshotsByScope("college", ownerID, "", className, "major")
	if err != nil || agg == nil || len(agg.ByGroup) == 0 {
		return []map[string]interface{}{}
	}
	out := make([]map[string]interface{}, 0, len(agg.ByGroup))
	for major, g := range agg.ByGroup {
		entry := map[string]interface{}{
			"name":         major,
			"sample_count": g.Count,
			"academic":     roundTo1(g.Academic),
			"ability":      roundTo1(g.Ability),
			"ideological":  roundTo1(g.Ideological),
			"emotional":    roundTo1(g.Emotional),
			"social":       roundTo1(g.Social),
			"data_source":  "real",
		}
		out = append(out, entry)
	}
	return out
}


// TwinScreenData 学院数字孪生大屏数据
type TwinScreenData struct {
	College     string                   `json:"college"`
	UpdatedAt   string                   `json:"updated_at"`
	Overview    map[string]interface{}   `json:"overview"`
	Departments []map[string]interface{} `json:"departments"`
	Trends      map[string][]float64     `json:"trends"`
	FiveDim     *CollegeFiveDim          `json:"five_dim"` // 五维全院聚合；无快照→nil
	AIInsight   string                   `json:"ai_insight"`
	DataSource  string                   `json:"data_source"`
}

// GenerateTwinScreen 生成学院数字孪生大屏数据。
// major/className 为可选下钻过滤：传空则统计全院（行为与旧版一致）。
func (s *CollegeService) GenerateTwinScreen(ctx context.Context, collegeName, ownerID, major, className string) *TwinScreenData {
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
		Trends:    map[string][]float64{},
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
		data.DataSource = "real"
	} else {
		// 兜底：无任何真实数据时给占位并明确标注
		data.Overview = map[string]interface{}{
			"total_students": 0, "health_score": 0.0, "risk_students": 0, "active_rate": 0.0,
		}
		data.DataSource = "fallback"
	}

	// 五维全院聚合（仅基于归属本院快照，绝不跨院读取）
	data.FiveDim = s.aggregateCollegeFiveDim(ownerID, major, className)
	// 按 major 下钻填充 departments（P1 直接按 major；无快照→如实空）
	data.Departments = s.buildDepartments(ownerID, className)
	// 趋势：当前快照表按 user_id 唯一、无历史版本 → 如实空 map（trend_note 已诚实标注）

	// LLM 解读（基于真实指标；无快照/无 LLM 时走现有规则降级）
	if s.llmClient != nil && m.HasData {
		prompt := fmt.Sprintf("你是学院管理顾问。%s全院%d名学生，风险关注%d人，健康度%.1f分。",
			collegeName, m.TotalStudents, m.RiskStudents, m.HealthScore)
		// 注入五维均值（仅真实数字，绝不编造）：增强解读依据，又不改动既有数字口径
		if data.FiveDim != nil && len(data.FiveDim.Dimensions) > 0 {
			var dimParts []string
			for _, d := range data.FiveDim.Dimensions {
				if d.Score != nil {
					dimParts = append(dimParts, fmt.Sprintf("%s%.1f分", d.Name, *d.Score))
				}
			}
			if len(dimParts) > 0 {
				prompt += "五维均值：" + strings.Join(dimParts, ",") + "。"
			}
		}
		prompt += "请用30字解读当前状态（仅依据上述真实数字，不编造）。"
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

func (s *CollegeService) GenerateDecisionAdvice(ctx context.Context, topic, ownerID string) *DecisionAdviceData {
	if topic == "" {
		topic = "学院管理决策"
	}
	data := &DecisionAdviceData{
		Topic:       topic,
		Suggestions: []map[string]interface{}{},
		Risks:       []string{},
		DataSource:  "fallback",
	}

	// 决策建议不臆造百分比：优先注入真实聚合指标，由 LLM 据实定性地给建议
	if s.llmClient != nil {
		m := s.aggregateCollegeMetrics(ownerID)
		var prompt string
		if m.HasData {
			prompt = fmt.Sprintf(
				"你是高校管理顾问。关于「%s」，已掌握以下真实数据：学生总数=%d，心理风险学生数=%d，五维健康均分=%.1f。请给出2条数据驱动的定性决策建议和主要风险，每条不超过30字；只基于已给数据，不得再编造其他数字。",
				topic, m.TotalStudents, m.RiskStudents, m.HealthScore)
		} else {
			prompt = fmt.Sprintf(
				"你是高校管理顾问。关于「%s」，暂无真实聚合数据，请给出2条通用的定性管理决策建议和主要风险，每条不超过30字，并说明缺少数据支撑。",
				topic)
		}
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 300,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.AIAdvice = strings.TrimSpace(resp.Content)
			if m.HasData {
				data.DataSource = "ai+real"
			} else {
				data.DataSource = "ai"
			}
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
		Scores:      map[string]float64{},
		Rankings:    []map[string]interface{}{},
		Suggestions: []string{"暂无真实学情/评教聚合数据，教师效能分析暂不可用。"},
		DataSource:  "real", // 诚实空：未接真实评教与学情聚合时不再编造教师排名
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
		Grade:      "",
		Metrics:    map[string]float64{},
		Strengths:  []string{},
		Warnings:   []string{"暂无真实成绩与评教聚合数据，课程质量评估暂不可用。"},
		DataSource: "real", // 诚实空：未接真实成绩/评教聚合时不再编造课程评级
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
