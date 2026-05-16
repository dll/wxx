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

// StudentService 学生角色 AI 功能服务
// 整合用户画像 + 历史会话 + 情感数据 + LLM，生成真实个性化的学生输出
// （区别于早期 handler 中的硬编码 mock）
type StudentService struct {
	userRepo    *repository.UserRepo
	sessionRepo *repository.SessionRepo
	messageRepo *repository.MessageRepo
	emotionRepo *repository.EmotionRepo
	llmClient   llm.ChatClient
}

// NewStudentService 创建学生服务（llmClient 可为 nil，对应 LLM 失败时走兜底）
func NewStudentService(
	userRepo *repository.UserRepo,
	sessionRepo *repository.SessionRepo,
	messageRepo *repository.MessageRepo,
	emotionRepo *repository.EmotionRepo,
	llmClient llm.ChatClient,
) *StudentService {
	return &StudentService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		emotionRepo: emotionRepo,
		llmClient:   llmClient,
	}
}

// DailyBriefing 今日速览结构
type DailyBriefing struct {
	Date            string                   `json:"date"`
	Greeting        string                   `json:"greeting"`
	UserDisplayName string                   `json:"user_display_name"`
	RecentQuestions []string                 `json:"recent_questions"` // 最近 5 个提问
	SessionCount    int                      `json:"session_count"`    // 历史会话数
	EmotionRisk     string                   `json:"emotion_risk"`     // none/low/medium/high
	Courses         []map[string]interface{} `json:"courses"`
	Deadlines       []map[string]interface{} `json:"deadlines"`
	Activities      []map[string]interface{} `json:"activities"`
	Weather         string                   `json:"weather"`
	Motto           string                   `json:"motto"` // LLM 生成的个性化激励语
	DataSource      string                   `json:"data_source"` // ai/fallback
}

// GenerateDailyBriefing 生成学生今日速览
// 真实数据流：用户信息 + 最近提问 + 情感风险 → LLM → 个性化激励语
func (s *StudentService) GenerateDailyBriefing(ctx context.Context, userID int64) (*DailyBriefing, error) {
	today := time.Now().Format("2006-01-02")

	// 防御性 nil 检查（测试或异常场景）
	if s.userRepo == nil {
		return s.fallbackBriefing(today, "同学"), nil
	}

	// 读取用户信息
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		// 用户不存在（Vercel 冷启动场景）→ 兜底
		return s.fallbackBriefing(today, "同学"), nil
	}

	// 读取最近提问（用于 LLM 个性化）
	var recentQs []string
	if s.messageRepo != nil {
		recentQs, _ = s.messageRepo.GetRecentQuestionsByUserID(userID, 5)
	}

	// 读取会话总数
	var sessionCount int
	if s.sessionRepo != nil {
		sessions, _ := s.sessionRepo.ListByUserID(userID, 100)
		sessionCount = len(sessions)
	}

	// 读取情感风险（最近 7 天高风险告警）
	emotionRisk := s.evaluateEmotionRisk(user.OwnerScope, user.OwnerID, user.Role)

	briefing := &DailyBriefing{
		Date:            today,
		UserDisplayName: user.DisplayName,
		RecentQuestions: recentQs,
		SessionCount:    sessionCount,
		EmotionRisk:     emotionRisk,
		Courses:         defaultCourses(),
		Deadlines:       defaultDeadlines(),
		Activities:      defaultActivities(),
		Weather:         "晴 26°C",
		DataSource:      "fallback",
	}

	// LLM 生成个性化问候语 + 激励语
	greeting, motto := s.generatePersonalizedGreeting(ctx, user, recentQs, emotionRisk)
	briefing.Greeting = greeting
	briefing.Motto = motto
	if s.llmClient != nil && motto != defaultMotto {
		briefing.DataSource = "ai"
	}

	return briefing, nil
}

// evaluateEmotionRisk 评估当前用户的情感风险水平
func (s *StudentService) evaluateEmotionRisk(ownerScope, ownerID, role string) string {
	if s.emotionRepo == nil {
		return "none"
	}
	stats, err := s.emotionRepo.GetStats(ownerScope, ownerID, role)
	if err != nil || stats == nil {
		return "none"
	}
	if stats.High > 0 {
		return "high"
	}
	if stats.Medium > 0 {
		return "medium"
	}
	if stats.Low > 0 {
		return "low"
	}
	return "none"
}

const defaultMotto = "学如逆水行舟，不进则退。"

// generatePersonalizedGreeting 用 LLM 生成符合学生当下情况的问候和激励语
func (s *StudentService) generatePersonalizedGreeting(ctx context.Context, user *model.User, recentQs []string, emotionRisk string) (greeting, motto string) {
	// 默认问候（按时段）
	greeting = greetingByHour(time.Now().Hour(), user.DisplayName)
	motto = defaultMotto

	if s.llmClient == nil {
		return
	}

	// 拼装提示词
	var b strings.Builder
	b.WriteString("你是一个温和的校园 AI 学工助手。请为以下学生生成今天的简短个性化问候和一句激励语。\n\n")
	b.WriteString(fmt.Sprintf("学生姓名：%s\n", user.DisplayName))
	b.WriteString(fmt.Sprintf("当前时段：%s\n", timeOfDay(time.Now().Hour())))
	if emotionRisk != "none" {
		b.WriteString(fmt.Sprintf("学生最近情绪风险：%s（请用更温和、更关怀的语气）\n", emotionRisk))
	}
	if len(recentQs) > 0 {
		b.WriteString("最近提问：\n")
		for _, q := range recentQs {
			b.WriteString("- " + q + "\n")
		}
	}
	b.WriteString("\n输出格式（严格遵守，两行）：\n问候语：xxx\n激励语：xxx\n要求：每行不超过 30 字，平实、不说教。")

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.6,
		MaxTokens:   200,
	})
	if err != nil || resp.Content == "" {
		return
	}

	g, m := parseGreetingAndMotto(resp.Content)
	if g != "" {
		greeting = g
	}
	if m != "" {
		motto = m
	}
	return
}

// parseGreetingAndMotto 解析 LLM 输出的"问候语：xxx\n激励语：xxx"格式
func parseGreetingAndMotto(text string) (greeting, motto string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "问候语：") || strings.HasPrefix(line, "问候语:") {
			greeting = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "问候语："), "问候语:"))
		} else if strings.HasPrefix(line, "激励语：") || strings.HasPrefix(line, "激励语:") {
			motto = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "激励语："), "激励语:"))
		}
	}
	return
}

func greetingByHour(hour int, name string) string {
	switch {
	case hour < 6:
		return fmt.Sprintf("夜深了，%s 注意休息哦", name)
	case hour < 11:
		return fmt.Sprintf("早上好，%s！新的一天开始了", name)
	case hour < 14:
		return fmt.Sprintf("中午好，%s，吃饭了吗？", name)
	case hour < 18:
		return fmt.Sprintf("下午好，%s，继续加油", name)
	default:
		return fmt.Sprintf("晚上好，%s，今天辛苦了", name)
	}
}

func timeOfDay(hour int) string {
	switch {
	case hour < 6:
		return "凌晨"
	case hour < 11:
		return "上午"
	case hour < 14:
		return "中午"
	case hour < 18:
		return "下午"
	default:
		return "晚上"
	}
}

// fallbackBriefing 兜底数据（用户不存在或异常时）
func (s *StudentService) fallbackBriefing(today, name string) *DailyBriefing {
	return &DailyBriefing{
		Date:            today,
		Greeting:        greetingByHour(time.Now().Hour(), name),
		UserDisplayName: name,
		Courses:         defaultCourses(),
		Deadlines:       defaultDeadlines(),
		Activities:      defaultActivities(),
		Weather:         "晴 26°C",
		Motto:           defaultMotto,
		DataSource:      "fallback",
		EmotionRisk:     "none",
	}
}

func defaultCourses() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "数据结构", "subtitle": "第8周 · 二叉树遍历", "time": "08:00-09:40", "icon": "book"},
		{"title": "操作系统", "subtitle": "第8周 · 进程调度", "time": "10:00-11:40", "icon": "computer"},
	}
}

func defaultDeadlines() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "数据结构实验报告", "subtitle": "二叉树实现", "time": "今天 23:59", "icon": "assignment"},
	}
}

func defaultActivities() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "ACM 训练赛", "subtitle": "信息楼 301", "time": "19:00", "icon": "emoji_events"},
	}
}
