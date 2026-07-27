package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// StudyPlanService 学习计划业务服务（AI 生成相关）
type StudyPlanService struct {
	db        *sql.DB
	llmClient llm.ChatClient
}

// NewStudyPlanService 创建学习计划服务
func NewStudyPlanService(db *sql.DB, llmClient llm.ChatClient) *StudyPlanService {
	return &StudyPlanService{db: db, llmClient: llmClient}
}

// IsAvailable 检查 LLM 客户端是否可用
func (s *StudyPlanService) IsAvailable() bool {
	return s.llmClient != nil
}

// AIGeneratePlanResult AI 生成结果
type AIGeneratePlanResult struct {
	PlanID       int64
	LLMProvider  string
	PromptTokens int
	OutputTokens int
}

// AIGeneratePlan AI 生成学习计划完整流程
func (s *StudyPlanService) AIGeneratePlan(
	ctx context.Context,
	userID int64,
	planType string,
	semesterCode string,
	startDate string,
	endDate string,
	goals []string,
	focusCourses []string,
) (*AIGeneratePlanResult, error) {
	calendar, currentWeek, err := s.resolveCurrentCalendar()
	if err != nil {
		return nil, fmt.Errorf("查询当前校历失败: %w", err)
	}
	if semesterCode == "" && calendar != nil {
		semesterCode = calendar.SemesterCode
	}

	today := time.Now()
	if startDate == "" {
		startDate = today.Format("2006-01-02")
	}
	if endDate == "" {
		endDate = calcDefaultEndDate(planType, today)
	}

	timetable := s.listUserTimetable(userID, semesterCode)
	upcomingExams := s.listUpcomingExams(semesterCode)

	prompt := buildAIGeneratePrompt(planType, semesterCode, startDate, endDate, goals, focusCourses, calendar, currentWeek, timetable, upcomingExams)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "你是一名高校学业规划助手，根据学生的课表、教学周和考试安排生成可执行的学习计划。请严格按 JSON 格式返回，不要包含额外说明或代码块标记。",
			},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   2500,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	planSchema, err := parseAIGeneratedPlan(resp.Content)
	if err != nil {
		log.Printf("解析 AI 学习计划失败: %v, 原始内容: %s", err, resp.Content)
		return nil, fmt.Errorf("解析 AI 生成的计划失败: %w", err)
	}

	planGoals := planSchema.Goals
	if len(planGoals) == 0 {
		planGoals = goals
	}
	if planGoals == nil {
		planGoals = []string{}
	}
	goalsJSON, _ := json.Marshal(planGoals)

	title := planSchema.Title
	if title == "" {
		title = "AI 生成 " + planType + " 计划"
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	var semCode sql.NullString
	if semesterCode != "" {
		semCode = sql.NullString{String: semesterCode, Valid: true}
	}

	res, err := s.db.Exec(
		"INSERT INTO study_plans (user_id, title, plan_type, semester_code, start_date, end_date, goals_json, progress, ai_generated, status, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 1, 'active', ?, ?)",
		userID, title, planType, semCode, startDate, endDate,
		string(goalsJSON), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("保存 AI 计划失败: %w", err)
	}
	planID, _ := res.LastInsertId()

	for _, t := range planSchema.Tasks {
		_, _ = s.db.Exec(
			"INSERT INTO study_plan_tasks (plan_id, course_id, course_name, title, description, scheduled_date, scheduled_duration, actual_duration, status, sort_order, created_at) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 'pending', ?, ?)",
			planID, t.CourseID, t.CourseName, t.Title, t.Description, t.ScheduledDate,
			t.ScheduledDuration, t.SortOrder, now,
		)
	}

	return &AIGeneratePlanResult{
		PlanID:       planID,
		LLMProvider:  s.llmClient.Name(),
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
	}, nil
}

// ─── 内部类型（AI 生成专用，不跨包导出） ───

type aiCalendar struct {
	SemesterCode string
	SemesterName string
	StartDate    string
	EndDate      string
}

type aiCourseItem struct {
	Weekday      int
	StartPeriod  int
	EndPeriod    int
	CourseName   string
	Teacher      string
	Location     string
	WeeksPattern string
}

type aiCalendarEvent struct {
	StartDate string
	EndDate   string
	EventName string
	EventType string
}

type aiGeneratedPlanSchema struct {
	Title string   `json:"title"`
	Goals []string `json:"goals"`
	Tasks []struct {
		CourseID          string `json:"course_id"`
		CourseName        string `json:"course_name"`
		Title             string `json:"title"`
		Description       string `json:"description"`
		ScheduledDate     string `json:"scheduled_date"`
		ScheduledDuration int    `json:"scheduled_duration"`
		SortOrder         int    `json:"sort_order"`
	} `json:"tasks"`
}

// ─── DB 查询 ───

func (s *StudyPlanService) resolveCurrentCalendar() (*aiCalendar, int, error) {
	today := time.Now().Format("2006-01-02")
	cal := &aiCalendar{}
	var semesterCode, semesterName, startDate, endDate string
	var id int64
	err := s.db.QueryRow(
		"SELECT id, semester_code, semester_name, start_date, end_date "+
			"FROM academic_calendars WHERE start_date <= ? AND end_date >= ? ORDER BY id DESC LIMIT 1",
		today, today,
	).Scan(&id, &semesterCode, &semesterName, &startDate, &endDate)
	if err == nil {
		cal.SemesterCode = semesterCode
		cal.SemesterName = semesterName
		cal.StartDate = startDate
		cal.EndDate = endDate
		return cal, calcCurrentWeek(startDate, today), nil
	}
	if err != sql.ErrNoRows {
		return nil, 0, err
	}

	err = s.db.QueryRow(
		"SELECT id, semester_code, semester_name, start_date, end_date "+
			"FROM academic_calendars WHERE start_date > ? ORDER BY start_date ASC LIMIT 1",
		today,
	).Scan(&id, &semesterCode, &semesterName, &startDate, &endDate)
	if err == nil {
		cal.SemesterCode = semesterCode
		cal.SemesterName = semesterName
		cal.StartDate = startDate
		cal.EndDate = endDate
		return cal, 0, nil
	}
	if err != sql.ErrNoRows {
		return nil, 0, err
	}

	err = s.db.QueryRow(
		"SELECT id, semester_code, semester_name, start_date, end_date "+
			"FROM academic_calendars WHERE end_date < ? ORDER BY end_date DESC LIMIT 1",
		today,
	).Scan(&id, &semesterCode, &semesterName, &startDate, &endDate)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	cal.SemesterCode = semesterCode
	cal.SemesterName = semesterName
	cal.StartDate = startDate
	cal.EndDate = endDate
	return cal, 0, nil
}

func (s *StudyPlanService) listUserTimetable(userID int64, semesterCode string) []*aiCourseItem {
	if semesterCode == "" {
		return nil
	}
	rows, err := s.db.Query(
		"SELECT course_name, weekday, start_period, end_period, weeks_pattern, location, teacher "+
			"FROM course_schedules WHERE user_id = ? AND semester_code = ? ORDER BY weekday ASC, start_period ASC",
		userID, semesterCode,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var list []*aiCourseItem
	for rows.Next() {
		item := &aiCourseItem{}
		var location, teacher sql.NullString
		if err := rows.Scan(&item.CourseName, &item.Weekday, &item.StartPeriod, &item.EndPeriod,
			&item.WeeksPattern, &location, &teacher); err != nil {
			continue
		}
		if location.Valid {
			item.Location = location.String
		}
		if teacher.Valid {
			item.Teacher = teacher.String
		}
		list = append(list, item)
	}
	return list
}

func (s *StudyPlanService) listUpcomingExams(semesterCode string) []*aiCalendarEvent {
	if semesterCode == "" {
		return nil
	}
	rows, err := s.db.Query(
		"SELECT event_name, event_type, start_date, end_date "+
			"FROM academic_calendar_events WHERE semester_code = ? AND event_type IN ('exam','deadline') "+
			"ORDER BY start_date ASC, id ASC",
		semesterCode,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var list []*aiCalendarEvent
	for rows.Next() {
		e := &aiCalendarEvent{}
		if err := rows.Scan(&e.EventName, &e.EventType, &e.StartDate, &e.EndDate); err != nil {
			continue
		}
		list = append(list, e)
	}
	return list
}

// ─── Prompt 构建 ───

func buildAIGeneratePrompt(planType, semesterCode, startDate, endDate string, goals, focusCourses []string, cal *aiCalendar, currentWeek int, timetable []*aiCourseItem, exams []*aiCalendarEvent) string {
	var sb strings.Builder
	sb.WriteString("请根据以下信息生成一份" + planType + "学习计划：\n\n")

	sb.WriteString(fmt.Sprintf("计划类型：%s\n", planType))
	sb.WriteString(fmt.Sprintf("开始日期：%s\n", startDate))
	sb.WriteString(fmt.Sprintf("结束日期：%s\n", endDate))
	if cal != nil {
		sb.WriteString(fmt.Sprintf("当前学期：%s（%s ~ %s）\n", cal.SemesterName, cal.StartDate, cal.EndDate))
	}
	if currentWeek > 0 {
		sb.WriteString(fmt.Sprintf("当前教学周：第 %d 周\n", currentWeek))
	}

	if len(goals) > 0 {
		sb.WriteString("用户目标：\n")
		for i, g := range goals {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, g))
		}
	}
	if len(focusCourses) > 0 {
		sb.WriteString("重点关注课程：" + strings.Join(focusCourses, "、") + "\n")
	}

	if len(timetable) > 0 {
		sb.WriteString("\n学生本周课表：\n")
		for _, t := range timetable {
			sb.WriteString(fmt.Sprintf("  周%d 第%d-%d节 %s（%s, %s）周次:%s\n",
				t.Weekday, t.StartPeriod, t.EndPeriod, t.CourseName, t.Teacher, t.Location, t.WeeksPattern))
		}
	}

	if len(exams) > 0 {
		sb.WriteString("\n近期考试/事件安排：\n")
		for _, e := range exams {
			sb.WriteString(fmt.Sprintf("  %s ~ %s %s（%s）\n", e.StartDate, e.EndDate, e.EventName, e.EventType))
		}
	}

	sb.WriteString(`
请按以下 JSON 格式返回（只返回 JSON，不要 Markdown 代码块）：
{
  "title": "计划标题",
  "goals": ["目标1", "目标2"],
  "tasks": [
    {
      "course_id": "课程ID（如无则留空）",
      "course_name": "课程名称",
      "title": "任务标题",
      "description": "任务描述",
      "scheduled_date": "YYYY-MM-DD",
      "scheduled_duration": 60,
      "sort_order": 1
    }
  ]
}

要求：
1. 任务要具体可执行，结合课表与考试安排合理分配时间
2. 优先安排考试周前的复习任务
3. 每个 task 的 scheduled_date 必须在计划起止日期内
4. 任务总时长不要超过计划周期可用时间
`)
	return sb.String()
}

// ─── 解析 ───

func parseAIGeneratedPlan(content string) (*aiGeneratedPlanSchema, error) {
	s := strings.TrimSpace(content)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = s[:len(s)-3]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("AI 响应中未找到 JSON 对象")
	}
	jsonStr := s[start : end+1]

	plan := &aiGeneratedPlanSchema{}
	if err := json.Unmarshal([]byte(jsonStr), plan); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	return plan, nil
}

// ─── 工具函数 ───

func calcCurrentWeek(startDate, today string) int {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0
	}
	now, err := time.Parse("2006-01-02", today)
	if err != nil {
		return 0
	}
	if now.Before(start) {
		return 0
	}
	days := int(now.Sub(start).Hours() / 24)
	return days/7 + 1
}

func calcDefaultEndDate(planType string, today time.Time) string {
	switch planType {
	case "weekly":
		return today.AddDate(0, 0, 7).Format("2006-01-02")
	case "monthly":
		return today.AddDate(0, 1, 0).Format("2006-01-02")
	case "quarterly":
		return today.AddDate(0, 3, 0).Format("2006-01-02")
	case "semester":
		return today.AddDate(0, 5, 0).Format("2006-01-02")
	case "yearly":
		return today.AddDate(1, 0, 0).Format("2006-01-02")
	case "four_year":
		return today.AddDate(4, 0, 0).Format("2006-01-02")
	default:
		return today.AddDate(0, 0, 7).Format("2006-01-02")
	}
}
