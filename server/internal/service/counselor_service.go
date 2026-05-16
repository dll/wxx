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
	llmClient   llm.ChatClient
}

// NewCounselorService 创建辅导员服务
func NewCounselorService(
	userRepo *repository.UserRepo,
	emotionRepo *repository.EmotionRepo,
	llmClient llm.ChatClient,
) *CounselorService {
	return &CounselorService{
		userRepo:    userRepo,
		emotionRepo: emotionRepo,
		llmClient:   llmClient,
	}
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
	Date             string                 `json:"date"`
	ClassHealthScore float64                `json:"class_health_score"`
	TopStudents      []*FocusedStudent      `json:"top_students"`
	Overview         map[string]int         `json:"overview"`
	AINarrative      string                 `json:"ai_narrative"`
	DataSource       string                 `json:"data_source"` // ai/fallback/db
	Stats            *model.EmotionStats    `json:"stats,omitempty"`
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
		"pending":   stats.Pending,
		"urgent":    stats.Urgent,
		"high":      stats.High,
		"medium":    stats.Medium,
		"low":       stats.Low,
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
