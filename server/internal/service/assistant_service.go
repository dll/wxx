package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// AssistantService 教辅角色 AI 功能服务
type AssistantService struct {
	llmClient llm.ChatClient
}

func NewAssistantService(llmClient llm.ChatClient) *AssistantService {
	return &AssistantService{llmClient: llmClient}
}

// ScheduleConflict 排课冲突
type ScheduleConflict struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
	Severity    string `json:"severity"` // high/medium/low
}

// ScheduleCheckResult 排课冲突检测结果
type ScheduleCheckResult struct {
	TotalCourses   int                 `json:"total_courses"`
	ConflictsFound int                 `json:"conflicts_found"`
	Conflicts      []*ScheduleConflict `json:"conflicts"`
	Summary        string              `json:"summary"`
	DataSource     string              `json:"data_source"`
}

func (s *AssistantService) CheckSchedule(ctx context.Context) *ScheduleCheckResult {
	result := &ScheduleCheckResult{
		TotalCourses: 48,
		Conflicts: []*ScheduleConflict{
			{Type: "教师冲突", Description: "张教授周一上午同时排了计科2301和计科2302的课", Detail: "信息楼301 vs 信息楼201，同一时段", Severity: "high"},
			{Type: "教室冲突", Description: "信息楼301周三下午被两门课程同时预定", Detail: "数据结构 vs 操作系统，请协调调整", Severity: "high"},
			{Type: "逻辑冲突", Description: "高等数学(一)安排了高等数学(二)为前置课程", Detail: "应先修(一)再修(二)，但当前安排在同学期", Severity: "medium"},
		},
		DataSource: "reference",
	}
	result.ConflictsFound = len(result.Conflicts)
	result.Summary = fmt.Sprintf("共检测%d门课程，发现%d处冲突（高优先级%d处）。", result.TotalCourses, result.ConflictsFound, 2)

	if s.llmClient != nil {
		prompt := "你是教务排课专家。请分析以下冲突并给出优化建议（50字以内）：\n"
		for _, c := range result.Conflicts {
			prompt += fmt.Sprintf("- [%s] %s: %s\n", c.Severity, c.Type, c.Description)
		}
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			result.Summary += "\nAI建议：" + strings.TrimSpace(resp.Content)
			result.DataSource = "ai"
		}
	}

	return result
}

// GraduationAuditResult 毕业资格审核结果
type GraduationAuditResult struct {
	StudentName     string   `json:"student_name"`
	TotalCredits    float64  `json:"total_credits"`
	RequiredCredits float64  `json:"required_credits"`
	PassedItems     []string `json:"passed_items"`
	PendingItems    []string `json:"pending_items"`
	CanGraduate     bool     `json:"can_graduate"`
	Summary         string   `json:"summary"`
	DataSource      string   `json:"data_source"`
}

func (s *AssistantService) AuditGraduation(ctx context.Context, studentID string) *GraduationAuditResult {
	result := &GraduationAuditResult{
		StudentName:     "示例学生",
		TotalCredits:    168,
		RequiredCredits: 175,
		PassedItems:     []string{"公共必修课(40学分)", "专业必修课(60学分)", "专业选修课(30学分)", "毕业论文(10学分)", "大学英语四级(425+)"},
		PendingItems:    []string{"公共选修课差2学分", "创新创业学分差2分", "志愿服务时长差10小时"},
		CanGraduate:     false,
		DataSource: "reference",
	}

	remaining := result.RequiredCredits - result.TotalCredits
	result.Summary = fmt.Sprintf("总学分%d/必修%d，尚差%.0f学分。%d项未达标需修补。",
		int(result.TotalCredits), int(result.RequiredCredits), remaining, len(result.PendingItems))

	return result
}

// ExamArrangement 考试安排
type ExamArrangement struct {
	TotalExams        int                      `json:"total_exams"`
	TotalRooms        int                      `json:"total_rooms"`
	TotalInvigilators int                      `json:"total_invigilators"`
	Schedule          []map[string]interface{} `json:"schedule"`
	Conflicts         []string                 `json:"conflicts"`
	DataSource        string                   `json:"data_source"`
}

func (s *AssistantService) ArrangeExams(ctx context.Context, semester string) *ExamArrangement {
	return &ExamArrangement{
		TotalExams:        12,
		TotalRooms:        8,
		TotalInvigilators: 24,
		Schedule: []map[string]interface{}{
			{"course": "数据结构", "date": "2026-06-15", "time": "08:30-10:30", "room": "信息楼301", "invigilators": []string{"张老师", "李老师"}, "students": 45},
			{"course": "操作系统", "date": "2026-06-16", "time": "08:30-10:30", "room": "信息楼201", "invigilators": []string{"王老师", "赵老师"}, "students": 42},
		},
		Conflicts:  []string{},
		DataSource: "reference",
	}
}

// ─── P2 深度功能 ───

// NotificationTemplate 通知模板
type NotificationTemplate struct {
	Channel     string `json:"channel"`
	Content     string `json:"content"`
	SendTime    string `json:"send_time"`
	TargetCount int    `json:"target_count"`
	DataSource  string `json:"data_source"`
}

func (s *AssistantService) GenerateNotification(ctx context.Context, channel, topic string) *NotificationTemplate {
	if channel == "" {
		channel = "班级群"
	}

	return &NotificationTemplate{
		Channel:     channel,
		Content:     "【通知】" + topic + "：请大家注意相关安排，按时完成。详情请查看教务系统公告。",
		SendTime:    time.Now().Add(2 * time.Hour).Format("15:04"),
		TargetCount: 240,
		DataSource: "reference",
	}
}

// TeachingCalendar 教学日历
type TeachingCalendar struct {
	Semester    string                   `json:"semester"`
	KeyDates    []map[string]interface{} `json:"key_dates"`
	Suggestions []string                 `json:"suggestions"`
	DataSource  string                   `json:"data_source"`
}

func (s *AssistantService) GenerateTeachingCalendar(ctx context.Context, semester string) *TeachingCalendar {
	if semester == "" {
		semester = "2025-2026-2"
	}

	return &TeachingCalendar{
		Semester: semester,
		KeyDates: []map[string]interface{}{
			{"date": "2026-05-25", "event": "期中考试周开始", "type": "考试", "remind": true},
			{"date": "2026-06-15", "event": "课程设计提交截止", "type": "deadline", "remind": true},
			{"date": "2026-07-01", "event": "期末考试周开始", "type": "考试", "remind": true},
			{"date": "2026-07-15", "event": "成绩录入截止", "type": "admin", "remind": false},
		},
		Suggestions: []string{
			"5月下旬：开始准备期中考试出题",
			"6月上旬：提醒学生提交课程设计",
			"7月：做好期末监考和成绩录入安排",
		},
		DataSource: "reference",
	}
}

// StudentInfoQuery 学生信息查询结果
type StudentInfoQuery struct {
	Query      string                   `json:"query"`
	Result     []map[string]interface{} `json:"result"`
	DataSource string                   `json:"data_source"`
}

func (s *AssistantService) QueryStudentInfo(ctx context.Context, query string) *StudentInfoQuery {
	return &StudentInfoQuery{
		Query: query,
		Result: []map[string]interface{}{
			{"student_id": "202301001", "name": "张明", "major": "计算机科学与技术", "class": "计科2301", "gpa": 2.8, "status": "在读"},
			{"student_id": "202301002", "name": "李华", "major": "计算机科学与技术", "class": "计科2301", "gpa": 3.5, "status": "在读"},
		},
		DataSource: "reference",
	}
}
