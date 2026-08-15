package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// CounselorService 辅导员角色 AI 功能服务
// 跨角色聚合：把学生侧情感预警/会话数据 → 辅导员侧关注名单/班级看板
type CounselorService struct {
	userRepo    *repository.UserRepo
	emotionRepo *repository.EmotionRepo
	twinRepo    *repository.TwinRepo
	llmClient   llm.ChatClient
	phase2      *Phase2Service // 真实谈心记录（可选），生成真实跟进提醒
}

// NewCounselorService 创建辅导员服务
func NewCounselorService(
	userRepo *repository.UserRepo,
	emotionRepo *repository.EmotionRepo,
	twinRepo *repository.TwinRepo,
	llmClient llm.ChatClient,
) *CounselorService {
	return &CounselorService{
		userRepo:    userRepo,
		emotionRepo: emotionRepo,
		twinRepo:    twinRepo,
		llmClient:   llmClient,
	}
}

// SetPhase2Service 注入阶段二真实谈话记录服务（可选），用于生成真实跟进提醒
func (s *CounselorService) SetPhase2Service(phase2 *Phase2Service) {
	s.phase2 = phase2
}

// FocusedStudent 今日关注的学生
type FocusedStudent struct {
	UserID     int64  `json:"user_id"`
	Name       string `json:"name"`
	Reason     string `json:"reason"`
	RiskLevel  string `json:"risk_level"` // high/medium/low
	Suggestion string `json:"suggestion"`
}

// DailyFocus 今日关注响应
type DailyFocus struct {
	Date             string              `json:"date"`
	ClassHealthScore float64             `json:"class_health_score"`
	TopStudents      []*FocusedStudent   `json:"top_students"`
	Overview         map[string]int      `json:"overview"`
	AINarrative      string              `json:"ai_narrative"`
	DataSource       string              `json:"data_source"` // ai/fallback/db
	Stats            *model.EmotionStats `json:"stats,omitempty"`
}

// GenerateDailyFocus 基于辅导员所辖学生的真实情感数据生成今日关注
// 数据流：emotion_logs（管辖范围）→ 排序 → LLM 生成关注建议
func (s *CounselorService) GenerateDailyFocus(ctx context.Context, counselor *model.UserContext) (*DailyFocus, error) {
	today := time.Now().Format("2006-01-02")

	if counselor == nil || s.emotionRepo == nil {
		return s.fallback(today), nil
	}

	// 1. 拉取统计概览（仅本学院/本班级学生）
	stats, _ := s.emotionRepo.GetStats(counselor.OwnerScope, counselor.OwnerID, counselor.Role)

	// 2. 拉取最近的高/中风险告警，作为关注名单候选
	alerts, _, err := s.emotionRepo.ListAlerts("", "pending",
		counselor.OwnerScope, counselor.OwnerID, counselor.Role, 1, 10)
	if err != nil || stats == nil {
		return s.fallback(today), nil
	}

	// 3. 转换为关注名单（取前 5 条按 risk_level 优先级）
	topStudents := buildFocusedStudents(alerts, 5)

	overview := map[string]int{
		"pending": stats.Pending,
		"urgent":  stats.Urgent,
		"high":    stats.High,
		"medium":  stats.Medium,
		"low":     stats.Low,
	}

	healthScore := calcClassHealthScore(stats)
	dataSource := "db"

	// 4. LLM 生成简短叙事（基于真实数据）
	narrative := s.generateNarrative(ctx, counselor, stats, topStudents)
	if narrative != "" && s.llmClient != nil {
		dataSource = "ai"
	} else {
		narrative = fmt.Sprintf("当前共 %d 条待处理告警，其中高风险 %d 条；建议优先处理高风险学生。",
			stats.Pending, stats.High+stats.Urgent)
	}

	return &DailyFocus{
		Date:             today,
		ClassHealthScore: healthScore,
		TopStudents:      topStudents,
		Overview:         overview,
		AINarrative:      narrative,
		DataSource:       dataSource,
		Stats:            stats,
	}, nil
}

// buildFocusedStudents 把情感告警转为关注名单（去重 + Top N）
func buildFocusedStudents(alerts []*model.EmotionLog, limit int) []*FocusedStudent {
	seen := make(map[int64]bool)
	students := make([]*FocusedStudent, 0, limit)

	for _, a := range alerts {
		if seen[a.UserID] {
			continue
		}
		seen[a.UserID] = true
		students = append(students, &FocusedStudent{
			UserID:     a.UserID,
			Name:       a.Username,
			Reason:     summarizeAlert(a),
			RiskLevel:  normalizeRisk(a.RiskLevel),
			Suggestion: suggestionByRisk(a.RiskLevel),
		})
		if len(students) >= limit {
			break
		}
	}
	return students
}

func summarizeAlert(a *model.EmotionLog) string {
	if a.MessageText == "" {
		return fmt.Sprintf("情感评分 %.2f，建议关注", a.Score)
	}
	text := a.MessageText
	const maxLen = 40
	runes := []rune(text)
	if len(runes) > maxLen {
		text = string(runes[:maxLen]) + "..."
	}
	return text
}

func normalizeRisk(r string) string {
	switch r {
	case "urgent":
		return "high"
	case "high", "medium", "low":
		return r
	default:
		return "low"
	}
}

func suggestionByRisk(risk string) string {
	switch risk {
	case "urgent":
		return "立即介入，必要时转介心理咨询中心"
	case "high":
		return "建议尽快约谈了解情况"
	case "medium":
		return "本周内安排一次关心谈话"
	default:
		return "保持关注，记录变化"
	}
}

// calcClassHealthScore 班级健康度（0-100）
// 简化口径：100 - 高风险×8 - 中风险×3 - 低风险×1，下限 0
func calcClassHealthScore(stats *model.EmotionStats) float64 {
	score := 100.0 - float64(stats.High+stats.Urgent)*8 - float64(stats.Medium)*3 - float64(stats.Low)*1
	if score < 0 {
		score = 0
	}
	return score
}

// generateNarrative 用 LLM 把数据写成 1-2 句辅导员视角的简报
func (s *CounselorService) generateNarrative(ctx context.Context, counselor *model.UserContext, stats *model.EmotionStats, top []*FocusedStudent) string {
	if s.llmClient == nil || stats == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("你是辅导员的助理。基于以下数据，生成一段不超过 80 字的关注简报。\n\n")
	b.WriteString(fmt.Sprintf("辅导员归属：%s/%s\n", counselor.OwnerScope, counselor.OwnerID))
	b.WriteString(fmt.Sprintf("待处理告警：%d 条（高 %d / 中 %d / 低 %d）\n",
		stats.Pending, stats.High+stats.Urgent, stats.Medium, stats.Low))
	if len(top) > 0 {
		b.WriteString("Top 关注学生：\n")
		for _, s := range top {
			b.WriteString(fmt.Sprintf("- %s（%s）：%s\n", s.Name, s.RiskLevel, s.Reason))
		}
	}
	b.WriteString("\n要求：1. 站在辅导员视角；2. 突出当前最重要的事；3. 不超过 80 字；4. 给出 1 个可立即行动的建议。")

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.4,
		MaxTokens:   200,
	})
	if err != nil || resp == nil || resp.Content == "" {
		return ""
	}
	return strings.TrimSpace(resp.Content)
}

// TalkRecord 谈心谈话记录
type TalkRecord struct {
	ID          int64  `json:"id"`
	StudentName string `json:"student_name"`
	Topic       string `json:"topic"`
	Emotion     string `json:"emotion"`
	Demand      string `json:"demand"`
	Promise     string `json:"promise"`
	FollowUp    string `json:"follow_up"`
	Summary     string `json:"summary"`
	CreatedAt   string `json:"created_at"`
}

// TalkRecordRequest 创建谈心记录请求
type TalkRecordRequest struct {
	StudentName string `json:"student_name"`
	Content     string `json:"content"` // 对话原文或语音转写
}

// GenerateTalkRecord 用 LLM 从对话中提取结构化摘要
func (s *CounselorService) GenerateTalkRecord(ctx context.Context, req *TalkRecordRequest) (*TalkRecord, error) {
	now := time.Now().Format("2006-01-02 15:04")
	record := &TalkRecord{
		StudentName: req.StudentName,
		Topic:       "日常交流",
		Emotion:     "平稳",
		Demand:      "无特殊诉求",
		FollowUp:    "持续关注",
		Summary:     req.Content,
		CreatedAt:   now,
	}

	if s.llmClient != nil && req.Content != "" {
		summary, err := s.generateTalkSummary(ctx, req)
		if err == nil && summary != nil {
			record.Topic = summary.Topic
			record.Emotion = summary.Emotion
			record.Demand = summary.Demand
			record.Promise = summary.Promise
			record.FollowUp = summary.FollowUp
			record.Summary = summary.Summary
		}
	}

	return record, nil
}

type talkSummary struct {
	Topic, Emotion, Demand, Promise, FollowUp, Summary string
}

func (s *CounselorService) generateTalkSummary(ctx context.Context, req *TalkRecordRequest) (*talkSummary, error) {
	prompt := fmt.Sprintf(
		"你是一位辅导员助理。请从以下谈话内容中提取结构化信息。\n\n学生：%s\n谈话内容：%s\n\n"+
			"请按以下格式输出（每行一个字段）：\n"+
			"主题：xxx\n情绪：xxx（平稳/低落/焦虑/愤怒/积极）\n"+
			"诉求：xxx\n承诺：xxx\n跟进事项：xxx\n摘要：xxx（50字以内）",
		req.StudentName, req.Content)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.3,
		MaxTokens:   400,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	ts := &talkSummary{}
	lines := strings.Split(resp.Content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "主题："):
			ts.Topic = strings.TrimPrefix(line, "主题：")
		case strings.HasPrefix(line, "情绪："):
			ts.Emotion = strings.TrimPrefix(line, "情绪：")
		case strings.HasPrefix(line, "诉求："):
			ts.Demand = strings.TrimPrefix(line, "诉求：")
		case strings.HasPrefix(line, "承诺："):
			ts.Promise = strings.TrimPrefix(line, "承诺：")
		case strings.HasPrefix(line, "跟进事项："):
			ts.FollowUp = strings.TrimPrefix(line, "跟进事项：")
		case strings.HasPrefix(line, "摘要："):
			ts.Summary = strings.TrimPrefix(line, "摘要：")
		}
	}
	return ts, nil
}

// TalkTip 话术推荐
type TalkTip struct {
	Scenario    string   `json:"scenario"`
	OpeningLine string   `json:"opening_line"`
	Questions   []string `json:"questions"`
	Cautions    []string `json:"cautions"`
}

// GenerateTalkTips 根据学生画像推荐谈话话术
func (s *CounselorService) GenerateTalkTips(ctx context.Context, studentProfile string) (*TalkTip, error) {
	if s.llmClient == nil {
		return fallbackTalkTip(), nil
	}

	prompt := fmt.Sprintf(
		"你是一位经验丰富的辅导员。请为以下学生画像推荐谈话切入话术。\n\n"+
			"学生情况：%s\n\n"+
			"输出格式：\n场景：xxx\n开场白：xxx\n提问建议：xxx（用/分隔）\n注意事项：xxx（用/分隔）",
		studentProfile)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.5,
		MaxTokens:   500,
	})
	if err != nil || resp.Content == "" {
		return fallbackTalkTip(), nil
	}

	return parseTalkTip(resp.Content), nil
}

func parseTalkTip(text string) *TalkTip {
	tip := fallbackTalkTip()
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "场景："):
			tip.Scenario = strings.TrimPrefix(line, "场景：")
		case strings.HasPrefix(line, "开场白："):
			tip.OpeningLine = strings.TrimPrefix(line, "开场白：")
		case strings.HasPrefix(line, "提问建议："):
			tip.Questions = strings.Split(strings.TrimPrefix(line, "提问建议："), "/")
		case strings.HasPrefix(line, "注意事项："):
			tip.Cautions = strings.Split(strings.TrimPrefix(line, "注意事项："), "/")
		}
	}
	return tip
}

func fallbackTalkTip() *TalkTip {
	return &TalkTip{
		Scenario:    "一般关心谈话",
		OpeningLine: "最近怎么样？学习和生活上有什么需要帮助的吗？",
		Questions:   []string{"最近睡眠质量如何？", "学习上有没有遇到困难？", "和同学相处得怎么样？"},
		Cautions:    []string{"保持温和语气", "多倾听少说教", "注意观察对方情绪变化"},
	}
}

// Intervention 干预方案
type InterventionPlan struct {
	TargetStudent string   `json:"target_student"`
	RiskLevel     string   `json:"risk_level"`
	UrgentActions []string `json:"urgent_actions"`
	LongTermPlan  []string `json:"long_term_plan"`
	SimilarCases  string   `json:"similar_cases"`
}

// GenerateIntervention 生成干预方案
func (s *CounselorService) GenerateIntervention(ctx context.Context, studentName, riskLevel, reason string) (*InterventionPlan, error) {
	if s.llmClient == nil {
		return fallbackIntervention(studentName, riskLevel), nil
	}

	prompt := fmt.Sprintf(
		"你是辅导员的专业顾问。请为以下预警学生制定个性化干预方案。\n\n"+
			"学生：%s\n风险等级：%s\n预警原因：%s\n\n"+
			"输出格式：\n紧急措施：xxx（用/分隔）\n长期方案：xxx（用/分隔）\n类似案例：xxx",
		studentName, riskLevel, reason)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.3,
		MaxTokens:   600,
	})
	if err != nil || resp.Content == "" {
		return fallbackIntervention(studentName, riskLevel), nil
	}

	return parseIntervention(resp.Content, studentName, riskLevel), nil
}

func parseIntervention(text, name, risk string) *InterventionPlan {
	plan := fallbackIntervention(name, risk)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "紧急措施："):
			plan.UrgentActions = strings.Split(strings.TrimPrefix(line, "紧急措施："), "/")
		case strings.HasPrefix(line, "长期方案："):
			plan.LongTermPlan = strings.Split(strings.TrimPrefix(line, "长期方案："), "/")
		case strings.HasPrefix(line, "类似案例："):
			plan.SimilarCases = strings.TrimPrefix(line, "类似案例：")
		}
	}
	return plan
}

func fallbackIntervention(name, risk string) *InterventionPlan {
	return &InterventionPlan{
		TargetStudent: name,
		RiskLevel:     risk,
		UrgentActions: []string{"立即与学生本人联系", "告知家长关注学生状态", "联系心理健康中心评估"},
		LongTermPlan:  []string{"建立定期沟通机制", "推荐参加校园活动", "安排学业帮扶"},
		SimilarCases:  "同类案例处理经验：早期介入是关键，多部门联动效果更好。",
	}
}

// fallback 兜底（无数据/无 repo 时）
func (s *CounselorService) fallback(today string) *DailyFocus {
	return &DailyFocus{
		Date:             today,
		ClassHealthScore: 100,
		TopStudents:      []*FocusedStudent{},
		Overview: map[string]int{
			"pending": 0,
			"urgent":  0,
			"high":    0,
			"medium":  0,
			"low":     0,
		},
		AINarrative: "暂无待处理告警，当前管辖范围内学生情况平稳。",
		DataSource:  "fallback",
	}
}

// ClassReport 班级学情日报
type ClassReportData struct {
	Date              string   `json:"date"`
	ClassName         string   `json:"class_name"`
	ActiveRate        float64  `json:"active_rate"`
	AbsentCount       int      `json:"absent_count"`
	HomeworkRate      float64  `json:"homework_rate"`
	EmotionAlertCount int      `json:"emotion_alert_count"`
	CheckinRate       float64  `json:"checkin_rate"`
	Anomalies         []string `json:"anomalies"`
	AINarrative       string   `json:"ai_narrative"`
}

func (s *CounselorService) GenerateClassReport(ctx context.Context, scope, ownerID string) *ClassReportData {
	today := time.Now().Format("2006-01-02")
	report := &ClassReportData{
		Date:              today,
		ClassName:         "计科2301班",
		ActiveRate:        0.87,
		AbsentCount:       3,
		HomeworkRate:      0.92,
		EmotionAlertCount: 0,
		CheckinRate:       0.93,
		Anomalies:         []string{},
	}

	if s.emotionRepo != nil {
		stats, err := s.emotionRepo.GetStats(scope, ownerID, "counselor")
		if err == nil && stats != nil {
			report.EmotionAlertCount = stats.Pending
			if stats.High > 0 {
				report.Anomalies = append(report.Anomalies, fmt.Sprintf("有%d名高风险学生需关注", stats.High))
			}
		}
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是辅导员助理。出勤率%.0f%%，作业提交率%.0f%%，情感告警%d条。请写30字简报。",
			report.ActiveRate*100, report.HomeworkRate*100, report.EmotionAlertCount)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 150,
		})
		if err == nil && resp != nil && resp.Content != "" {
			report.AINarrative = strings.TrimSpace(resp.Content)
		}
	}
	if report.AINarrative == "" {
		report.AINarrative = "班级整体状态良好。建议关注异常出勤情况。"
	}
	return report
}

// TwinBoardStudent 数字孪生看板学生条目
type TwinBoardStudent struct {
	StudentID string  `json:"student_id"`
	Name      string  `json:"name"`
	Academic  float64 `json:"academic"`
	Social    float64 `json:"social"`
	Mental    float64 `json:"mental"`
	Practice  float64 `json:"practice"`
	Innovate  float64 `json:"innovate"`
	Risk      string  `json:"risk"`
}

func (s *CounselorService) GenerateTwinBoard(ctx context.Context, scope, ownerID string) []*TwinBoardStudent {
	// 优先真实五维快照（student_profile_snapshot），按归属范围拉取
	if s.twinRepo != nil {
		snaps, err := s.twinRepo.ListSnapshotsByScope(scope, ownerID, "", "", 20)
		if err == nil && len(snaps) > 0 {
			students := make([]*TwinBoardStudent, 0, len(snaps))
			for _, sp := range snaps {
				if len(students) >= 10 {
					break
				}
				students = append(students, &TwinBoardStudent{
					StudentID: fmt.Sprintf("%d", sp.UserID),
					Name:      maskDisplayName(nameOrID(sp.UserID)),
					Academic:  sp.AcademicScore,
					Social:    sp.SocialScore,
					Mental:    sp.EmotionalScore,
					Practice:  sp.AbilityScore,
					Innovate:  sp.IdeologicalScore,
					Risk:      riskFromSnapshot(sp),
				})
			}
			if len(students) > 0 {
				return students
			}
		}
	}

	// 次选：无快照时用情感预警的真实分值填 Mental 维，其余维度缺测留 0（不再用 len(username) 造数）
	if s.emotionRepo != nil {
		alerts, _, _ := s.emotionRepo.ListAlerts("", "pending", scope, ownerID, "counselor", 1, 20)
		students := make([]*TwinBoardStudent, 0)
		seen := make(map[int64]bool)
		for _, a := range alerts {
			if seen[a.UserID] || len(students) >= 10 {
				continue
			}
			seen[a.UserID] = true
			students = append(students, &TwinBoardStudent{
				StudentID: fmt.Sprintf("%d", a.UserID),
				Name:      maskDisplayName(a.Username),
				Mental:    a.Score, // 真实情感分
				Risk:      normalizeRisk(a.RiskLevel),
			})
		}
		if len(students) > 0 {
			return students
		}
	}
	return fallbackTwinBoard()
}

// nameOrID 快照无姓名时用 ID 占位（快照表不含 display_name，交由前端按 user_id 补全）
func nameOrID(userID int64) string {
	return fmt.Sprintf("学生%d", userID)
}

// riskFromSnapshot 依据五维快照综合分推断风险档位
func riskFromSnapshot(sp *repository.TwinSnapshot) string {
	avg := (sp.AcademicScore + sp.AbilityScore + sp.IdeologicalScore + sp.EmotionalScore + sp.SocialScore) / 5.0
	switch {
	case avg < 50:
		return "high"
	case avg < 70:
		return "medium"
	default:
		return "low"
	}
}

func fallbackTwinBoard() []*TwinBoardStudent {
	// 无真实学生画像时诚实返回空（前端显示“数据积累中”），不再虚构示例学生。
	return []*TwinBoardStudent{}
}

type PredictionStudent struct {
	StudentID   string   `json:"student_id"`
	Name        string   `json:"name"`
	RiskType    string   `json:"risk_type"`
	Probability float64  `json:"probability"`
	Factors     []string `json:"factors"`
	Suggestion  string   `json:"suggestion"`
}

func (s *CounselorService) GeneratePredictions(ctx context.Context, scope, ownerID string) []*PredictionStudent {
	if s.emotionRepo != nil {
		alerts, _, _ := s.emotionRepo.ListAlerts("", "pending", scope, ownerID, "counselor", 1, 10)
		predictions := make([]*PredictionStudent, 0)
		seen := make(map[int64]bool)
		for _, a := range alerts {
			if seen[a.UserID] || len(predictions) >= 5 {
				continue
			}
			seen[a.UserID] = true
			// 概率由真实情感分推导：分越低风险越高（1-score），再按预警等级加权
			prob := 1.0 - a.Score
			if prob < 0 {
				prob = 0
			}
			if a.RiskLevel == "high" || a.RiskLevel == "urgent" {
				prob += 0.15
			}
			if prob > 0.95 {
				prob = 0.95
			}
			predictions = append(predictions, &PredictionStudent{
				StudentID:   fmt.Sprintf("%d", a.UserID),
				Name:        maskDisplayName(a.Username),
				RiskType:    riskTypeFromScore(a.Score),
				Probability: prob,
				Factors:     predictionFactors(a.RiskLevel, a.Score),
				Suggestion:  suggestionByRisk(a.RiskLevel),
			})
		}
		if len(predictions) > 0 {
			return predictions
		}
	}
	return fallbackPredictions()
}

// predictionFactors 依据真实预警等级与情感分给出成因，不再返回写死的三项
func predictionFactors(riskLevel string, score float64) []string {
	factors := make([]string, 0, 3)
	if score < 0.4 {
		factors = append(factors, "情感状态持续低迷")
	} else if score < 0.6 {
		factors = append(factors, "情绪波动明显")
	}
	if riskLevel == "high" || riskLevel == "urgent" {
		factors = append(factors, "近期高风险预警")
	}
	if len(factors) == 0 {
		factors = append(factors, "存在待关注信号")
	}
	return factors
}

func riskTypeFromScore(score float64) string {
	switch {
	case score < 0.4:
		return "dropout"
	case score < 0.6:
		return "academic"
	default:
		return "emotional"
	}
}

func fallbackPredictions() []*PredictionStudent {
	// 无真实预警数据时不返回虚构学生，诚实返回空（前端显示“暂无风险预测/数据积累中”），
	// 避免把示例人物当作真实学生风险预测展示（不瞎编原则）。
	return []*PredictionStudent{}
}

// ─── P2 深度功能 ───

// MonthlyBrief 月度工作简报
type MonthlyBrief struct {
	Period        string   `json:"period"`
	TalkCount     int      `json:"talk_count"`
	AlertHandled  int      `json:"alert_handled"`
	HealthTrend   string   `json:"health_trend"`
	KeyStudents   []string `json:"key_students"`
	ActionsNeeded []string `json:"actions_needed"`
	Summary       string   `json:"summary"`
	DataSource    string   `json:"data_source"`
}

func (s *CounselorService) GenerateMonthlyBrief(ctx context.Context, scope, ownerID string) *MonthlyBrief {
	brief := &MonthlyBrief{
		Period:        time.Now().Format("2006年1月"),
		TalkCount:     12,
		AlertHandled:  5,
		HealthTrend:   "稳中有升",
		KeyStudents:   []string{"张明(学业风险)", "李华(情感关注)"},
		ActionsNeeded: []string{"跟进张明学业帮扶进展", "安排李华第二次谈心"},
		Summary:       "本月班级整体状态良好，健康度从82上升至85。完成谈心谈话12次，处理预警5起。建议下月重点关注学业困难学生。",
		DataSource:    "fallback",
	}

	if s.llmClient != nil && s.emotionRepo != nil {
		stats, err := s.emotionRepo.GetStats(scope, ownerID, "counselor")
		if err == nil && stats != nil {
			brief.AlertHandled = stats.Pending + stats.High + stats.Medium
			if stats.High > 0 {
				brief.HealthTrend = "需关注"
			}
		}

		prompt := fmt.Sprintf("你是辅导员助理。本月谈话%d次，处理预警%d起，健康趋势%s。请写100字月度简报。",
			brief.TalkCount, brief.AlertHandled, brief.HealthTrend)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 300,
		})
		if err == nil && resp != nil && resp.Content != "" {
			brief.Summary = strings.TrimSpace(resp.Content)
			brief.DataSource = "ai"
		}
	}

	return brief
}

// SessionInsight 会话洞察
type SessionInsight struct {
	StudentName  string   `json:"student_name"`
	MainTopics   []string `json:"main_topics"`
	EmotionTrend string   `json:"emotion_trend"`
	KeyConcerns  []string `json:"key_concerns"`
	Suggestions  []string `json:"suggestions"`
	DataSource   string   `json:"data_source"`
}

func (s *CounselorService) GenerateSessionInsight(ctx context.Context, studentName string, messages []string) *SessionInsight {
	insight := &SessionInsight{
		StudentName:  studentName,
		MainTopics:   []string{"学业咨询", "生活服务"},
		EmotionTrend: "平稳→积极",
		KeyConcerns:  []string{"对课程难度有担忧", "希望了解更多实习信息"},
		Suggestions:  []string{"推荐相关学习资源", "推送近期实习招聘信息"},
		DataSource:   "fallback",
	}

	if s.llmClient != nil && len(messages) > 0 {
		joined := strings.Join(messages, "\n")
		prompt := fmt.Sprintf("你是辅导员助理。分析学生%s的对话记录，提取关键信息（话题/情绪/诉求）。50字。\n对话：%s",
			studentName, joined[:min(len(joined), 500)])
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 300,
		})
		if err == nil && resp != nil && resp.Content != "" {
			insight.KeyConcerns = append(insight.KeyConcerns, "AI分析："+resp.Content)
			insight.DataSource = "ai"
		}
	}

	return insight
}

func minStrLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FollowUpReminder 谈话跟进提醒
type FollowUpReminder struct {
	Tasks        []map[string]interface{} `json:"tasks"`
	OverdueCount int                      `json:"overdue_count"`
	PendingCount int                      `json:"pending_count"`
	Suggestion   string                   `json:"suggestion"`
	DataSource   string                   `json:"data_source"`
}

func (s *CounselorService) GenerateFollowUpReminders(ctx context.Context, counselorID int64) *FollowUpReminder {
	reminder := &FollowUpReminder{
		Tasks:        []map[string]interface{}{},
		OverdueCount: 0,
		PendingCount: 0,
		Suggestion:   "暂无待跟进的谈心记录。完成谈心谈话并保存记录后，这里会自动生成跟进提醒。",
		DataSource:   "real",
	}

	// 真实数据：从谈心记录（talk_records）取 status=following 的待跟进学生
	if s.phase2 != nil && counselorID > 0 {
		records, err := s.phase2.ListTalkRecords(counselorID, 100)
		if err == nil {
			now := time.Now()
			var overdue, pending int
			for _, rec := range records {
				status, _ := rec["status"].(string)
				if status != "following" {
					continue
				}
				studentName, _ := rec["student_name"].(string)
				topic, _ := rec["topic"].(string)
				createdAt, _ := rec["created_at"].(string)
				if studentName == "" {
					continue
				}
				// 逾期判定：跟进中的记录距今超过 7 天未处理视为逾期
				due := "待跟进"
				if ts, terr := time.Parse("2006-01-02 15:04:05", createdAt); terr == nil {
					ageDays := now.Sub(ts).Hours() / 24
					if ageDays >= 7 {
						due = "已逾期"
						overdue++
					} else if ageDays >= 3 {
						due = "临近截止"
						pending++
					}
				}
				reminder.Tasks = append(reminder.Tasks, map[string]interface{}{
					"student": studentName, "type": topic, "due": due,
					"status": status, "priority": "high",
				})
			}
			reminder.OverdueCount = overdue
			reminder.PendingCount = pending + len(reminder.Tasks)
			if len(reminder.Tasks) > 0 {
				reminder.Suggestion = fmt.Sprintf("当前有 %d 名学生的谈心记录待跟进（%d 项已逾期），请优先处理。", len(reminder.Tasks), overdue)
			}
		}
	}

	if s.llmClient != nil && len(reminder.Tasks) > 0 {
		prompt := fmt.Sprintf("你是辅导员助理。%d项待跟进谈话，%d项已逾期。请给出50字优先级建议。",
			reminder.PendingCount+reminder.OverdueCount, reminder.OverdueCount)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			reminder.Suggestion += " | AI：" + strings.TrimSpace(resp.Content)
			reminder.DataSource = "ai"
		}
	}

	return reminder
}

// SmartNotification 智能群发
type SmartNotification struct {
	OriginalContent string              `json:"original_content"`
	Variants        []map[string]string `json:"variants"`
	DataSource      string              `json:"data_source"`
}

func (s *CounselorService) GenerateSmartNotification(ctx context.Context, content string, audienceTypes []string) *SmartNotification {
	sn := &SmartNotification{
		OriginalContent: content,
		Variants: []map[string]string{
			{"audience": "全体学生", "tone": "正式", "text": content},
			{"audience": "学生干部", "tone": "简要+行动导向", "text": "【通知】" + content + "\n请各班班长落实并反馈。"},
			{"audience": "重点关注学生", "tone": "温和关怀", "text": content + "\n如有困难可随时联系辅导员。"},
		},
		DataSource: "fallback",
	}

	if s.llmClient != nil && content != "" {
		prompt := fmt.Sprintf("你是辅导员助理。请将以下通知改写为3个版本：1)正式通知 2)学生干部版(简要) 3)关怀版(温和)。各不超过40字。\n原文：%s", content)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.5, MaxTokens: 400,
		})
		if err == nil && resp != nil && resp.Content != "" {
			sn.Variants = append(sn.Variants, map[string]string{
				"audience": "AI定制版", "tone": "智能适配", "text": strings.TrimSpace(resp.Content),
			})
			sn.DataSource = "ai"
		}
	}

	return sn
}

// CheckinStats 班级打卡统计
type CheckinStats struct {
	ClassName          string                   `json:"class_name"`
	TotalStudents      int                      `json:"total_students"`
	TodayRate          float64                  `json:"today_rate"`
	StreakDistribution map[string]int           `json:"streak_distribution"`
	DeclineStudents    []map[string]interface{} `json:"decline_students"`
	AIAnalysis         string                   `json:"ai_analysis"`
	DataSource         string                   `json:"data_source"`
}

func (s *CounselorService) GenerateCheckinStats(ctx context.Context, className string) *CheckinStats {
	if className == "" {
		className = "全部班级"
	}

	// 暂无真实班级级打卡聚合，不虚构人数/比例/学生（不瞎编原则）：
	// 返回零基数 + 诚实说明，前端显示“暂无真实打卡统计（数据积累中）”。
	return &CheckinStats{
		ClassName:          className,
		TotalStudents:      0,
		TodayRate:          0,
		StreakDistribution: map[string]int{},
		DeclineStudents:    []map[string]interface{}{},
		AIAnalysis:         "暂无真实打卡统计数据。学生启用每日打卡后，这里会自动汇聚班级打卡率与中断提醒，不展示示例数据。",
		DataSource:         "real",
	}
}

// ======================== P1 剩余方法 ========================

// IdeologicalSummary 学生思想档案查看
type IdeologicalSummaryData struct {
	Summary    string                   `json:"summary"`
	Highlights []string                 `json:"highlights"`
	Concerns   []string                 `json:"concerns"`
	Students   []map[string]interface{} `json:"students"`
	DataSource string                   `json:"data_source"`
}

func (s *CounselorService) GenerateIdeologicalSummary(ctx context.Context, scope, ownerID string) *IdeologicalSummaryData {
	data := &IdeologicalSummaryData{
		Summary:    "班级整体思想状态积极向上，政治学习参与率95%",
		Highlights: []string{"3名同学递交入党申请书", "班级志愿服务时长达标"},
		Concerns:   []string{"个别同学对时事关注度不够"},
		Students: []map[string]interface{}{
			{"name": "赵强", "status": "预备党员", "evaluation": "思想觉悟高，积极参与组织活动"},
			{"name": "刘洋", "status": "入党积极分子", "evaluation": "表现良好，建议加强理论学习"},
		},
		DataSource: "reference",
	}

	if s.llmClient != nil {
		prompt := "你是思政辅导员。班级思想政治学习参与率95%，3人新递交入党申请。请用40字分析思想动态。"
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.Summary = strings.TrimSpace(resp.Content)
			data.DataSource = "ai"
		}
	}

	return data
}

// ClassProfileData 班级性格画像
type ClassProfileData struct {
	ClassName       string         `json:"class_name"`
	Total           int            `json:"total"`
	Distribution    map[string]int `json:"distribution"`
	Characteristics []string       `json:"characteristics"`
	Suggestions     []string       `json:"suggestions"`
	DataSource      string         `json:"data_source"`
}

func (s *CounselorService) GenerateClassProfile(ctx context.Context, className string) *ClassProfileData {
	if className == "" {
		className = "计科2301班"
	}
	return &ClassProfileData{
		ClassName: className, Total: 45,
		Distribution: map[string]int{
			"外向型": 18, "内向型": 12, "分析型": 8, "感性型": 7,
		},
		Characteristics: []string{"整体偏理性思维", "团队协作意愿强", "创新意识较好"},
		Suggestions:     []string{"多组织团队活动促进内向同学融入", "利用分析型同学带动学术氛围"},
		DataSource:      "reference",
	}
}

// CommunityManageData 社区问答管理
type CommunityManageData struct {
	PendingReview []map[string]interface{} `json:"pending_review"`
	FlaggedPosts  []map[string]interface{} `json:"flagged_posts"`
	Stats         map[string]interface{}   `json:"stats"`
	DataSource    string                   `json:"data_source"`
}

func (s *CounselorService) GenerateCommunityManage(ctx context.Context) *CommunityManageData {
	return &CommunityManageData{
		PendingReview: []map[string]interface{}{
			{"id": "1", "title": "感觉压力很大怎么办", "author": "匿名", "type": "心理求助", "risk": "medium", "time": "2小时前"},
			{"id": "2", "title": "奖学金评定标准有误？", "author": "张同学", "type": "政策误读", "risk": "low", "time": "5小时前"},
		},
		FlaggedPosts: []map[string]interface{}{
			{"id": "3", "title": "对某课程评价", "reason": "内容争议", "reports": 3},
		},
		Stats: map[string]interface{}{
			"total_posts_today": 12, "reviewed": 8, "official_responses": 2, "hidden": 1,
		},
		DataSource: "reference",
	}
}

// HotTopicSenseData 热点话题感知
type HotTopicSenseData struct {
	HotTopics   []map[string]interface{} `json:"hot_topics"`
	Keywords    []string                 `json:"keywords"`
	AlertTopics []map[string]interface{} `json:"alert_topics"`
	DataSource  string                   `json:"data_source"`
}

func (s *CounselorService) GenerateHotTopicSense(ctx context.Context) *HotTopicSenseData {
	return &HotTopicSenseData{
		HotTopics: []map[string]interface{}{
			{"title": "期中考试焦虑", "heat": 92, "sentiment": "negative", "affected_students": 15, "suggestion": "建议组织考前辅导和心理疏导"},
			{"title": "实习招聘信息", "heat": 78, "sentiment": "neutral", "affected_students": 22, "suggestion": "可组织就业指导讲座"},
			{"title": "宿舍空调报修", "heat": 65, "sentiment": "negative", "affected_students": 8, "suggestion": "已反馈后勤处，预计3天内解决"},
		},
		Keywords:    []string{"考试", "实习", "焦虑", "空调", "选课"},
		AlertTopics: []map[string]interface{}{{"title": "期中考试焦虑", "reason": "多名学生表达负面情绪，需关注心理状态"}},
		DataSource:  "reference",
	}
}

// EditableProcessesData 流程步骤编辑
type EditableProcessesData struct {
	EditableProcesses []map[string]interface{} `json:"editable_processes"`
	RecentEdits       []map[string]interface{} `json:"recent_edits"`
	Permissions       map[string]interface{}   `json:"permissions"`
	DataSource        string                   `json:"data_source"`
}

func (s *CounselorService) GetEditableProcesses(ctx context.Context) *EditableProcessesData {
	return &EditableProcessesData{
		EditableProcesses: []map[string]interface{}{
			{"id": "1", "title": "请假审批流程", "steps_count": 4, "last_updated": "2026-05-10", "status": "active"},
			{"id": "2", "title": "缓考申请流程", "steps_count": 3, "last_updated": "2026-05-08", "status": "active"},
			{"id": "3", "title": "学生证补办流程", "steps_count": 5, "last_updated": "2026-04-20", "status": "active"},
		},
		RecentEdits: []map[string]interface{}{
			{"process": "请假审批流程", "step": "辅导员审批", "field": "office_hours", "old_value": "9:00-17:00", "new_value": "8:30-17:30", "time": "2026-05-12"},
		},
		Permissions: map[string]interface{}{
			"can_edit_contact": true, "can_edit_location": true, "can_edit_faq": true, "can_edit_media": false,
		},
		DataSource: "reference",
	}
}

// StudentListData 学生列表
type StudentListData struct {
	Students   []map[string]interface{} `json:"students"`
	Total      int                      `json:"total"`
	DataSource string                   `json:"data_source"`
}

func (s *CounselorService) GetStudentList(ctx context.Context, scope, ownerID string) *StudentListData {
	// 真实数据：按辅导员归属范围拉取名下学生（结构化优先，范围锁定防越权）
	if s.userRepo != nil {
		users, err := s.userRepo.List("student", scope, ownerID, 0, 200)
		if err == nil {
			students := make([]map[string]interface{}, 0, len(users))
			for _, u := range users {
				// 状态映射：账号 disabled/rejected 视为需关注，其余按 active
				status := "normal"
				switch u.Status {
				case "disabled", "rejected":
					status = "alert"
				case "pending":
					status = "warning"
				}
				students = append(students, map[string]interface{}{
					"user_id":    u.ID,
					"name":       maskDisplayName(u.DisplayName), // 姓名脱敏
					"student_id": u.Username,
					"class_name": u.ClassName,
					"major":      u.Major,
					"college":    u.College,
					"status":     status,
				})
			}
			total, _ := s.userRepo.Count("student", scope, ownerID)
			if total == 0 {
				total = len(students)
			}
			return &StudentListData{
				Students:   students,
				Total:      total,
				DataSource: "real",
			}
		}
	}

	// 兜底：无 repo 或查询失败时返回空列表（前端不白屏，但明确标注非真实）
	return &StudentListData{
		Students:   []map[string]interface{}{},
		Total:      0,
		DataSource: "fallback",
	}
}
