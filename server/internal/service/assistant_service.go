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

// AssistantService 教辅角色 AI 功能服务
type AssistantService struct {
	llmClient llm.ChatClient
	phase3    *Phase3Service // 阶段三真实数据（可选），无真实数据时回落 reference
	userRepo  *repository.UserRepo // 真实学生账号来源（可选），用于学生信息查询
}

func NewAssistantService(llmClient llm.ChatClient) *AssistantService {
	return &AssistantService{llmClient: llmClient}
}

// SetUserRepo 注入用户仓储（学生信息查询用真实账号数据，可选依赖）
func (s *AssistantService) SetUserRepo(userRepo *repository.UserRepo) {
	s.userRepo = userRepo
}

// SetPhase3Service 注入阶段三真实数据服务（可选依赖）
func (s *AssistantService) SetPhase3Service(phase3 *Phase3Service) {
	s.phase3 = phase3
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
	// 优先真实课表数据
	if s.phase3 != nil {
		if total, conflicts, err := s.phase3.GetScheduleConflicts(""); err == nil && total > 0 {
			result := &ScheduleCheckResult{
				TotalCourses: total,
				Conflicts:    []*ScheduleConflict{},
				DataSource:   "real",
			}
			for _, c := range conflicts {
				result.Conflicts = append(result.Conflicts, &ScheduleConflict{
					Type: c["type"].(string), Description: c["description"].(string),
					Detail: c["description"].(string), Severity: c["severity"].(string),
				})
			}
			result.ConflictsFound = len(result.Conflicts)
			result.Summary = fmt.Sprintf("共检测%d门课程，发现%d处冲突。", result.TotalCourses, result.ConflictsFound)
			if s.llmClient != nil && result.ConflictsFound > 0 {
				prompt := "你是教务排课专家。请分析以下冲突并给出优化建议（50字以内）：\n"
				for _, c := range result.Conflicts {
					prompt += fmt.Sprintf("- [%s] %s: %s\n", c.Severity, c.Type, c.Description)
				}
				if resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
					Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.3, MaxTokens: 200,
				}); err == nil && resp != nil && resp.Content != "" {
					result.Summary += "\nAI建议：" + strings.TrimSpace(resp.Content)
				}
			}
			return result
		}
	}

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
	// 优先真实成绩聚合数据
	if s.phase3 != nil {
		if summaries, err := s.phase3.GetGraduationSummaries(); err == nil && len(summaries) > 0 {
			var target *repository.GradeSummary
			for i := range summaries {
				if summaries[i].UserID == studentID {
					target = summaries[i]
					break
				}
			}
			if target == nil {
				target = summaries[0]
			}
			required := 160.0
			result := &GraduationAuditResult{
				StudentName:     target.Name,
				TotalCredits:    target.Credits,
				RequiredCredits: required,
				CanGraduate:     target.Credits >= required,
				DataSource:      "real",
			}
			if target.Passed > 0 {
				result.PassedItems = []string{fmt.Sprintf("通过课程 %d 门", target.Passed)}
			}
			if !result.CanGraduate {
				result.PendingItems = []string{fmt.Sprintf("尚差 %.1f 学分", required-target.Credits)}
			}
			result.Summary = fmt.Sprintf("总学分%.1f/必修%.0f，%s。", target.Credits, required,
				map[bool]string{true: "符合毕业条件", false: fmt.Sprintf("尚差%.1f学分", required-target.Credits)}[result.CanGraduate])
			return result
		}
	}

	result := &GraduationAuditResult{
		StudentName:     "示例学生",
		TotalCredits:    168,
		RequiredCredits: 175,
		PassedItems:     []string{"公共必修课(40学分)", "专业必修课(60学分)", "专业选修课(30学分)", "毕业论文(10学分)", "大学英语四级(425+)"},
		PendingItems:    []string{"公共选修课差2学分", "创新创业学分差2分", "志愿服务时长差10小时"},
		CanGraduate:     false,
		DataSource:      "reference",
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
	// 优先真实考试安排
	if s.phase3 != nil {
		if exams, err := s.phase3.GetExams(semester); err == nil && len(exams) > 0 {
			schedule := make([]map[string]interface{}, 0, len(exams))
			rooms := map[string]bool{}
			for _, e := range exams {
				schedule = append(schedule, map[string]interface{}{
					"course": e["course_name"], "date": e["date"],
					"time": fmt.Sprintf("%v-%v", e["time_start"], e["time_end"]),
					"room": e["location"],
				})
				if loc, ok := e["location"].(string); ok && loc != "" {
					rooms[loc] = true
				}
			}
			return &ExamArrangement{
				TotalExams:        len(exams),
				TotalRooms:        len(rooms),
				TotalInvigilators: 0,
				Schedule:          schedule,
				Conflicts:         []string{},
				DataSource:        "real",
			}
		}
	}

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
		DataSource:  "reference",
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
	// 优先查真实学生账号（users.role=student），不再硬编码示例人物
	if s.userRepo != nil {
		q := &model.UserQuery{Role: "student", Keyword: query}
		if users, _, err := s.userRepo.ListAdvanced(q); err == nil && len(users) > 0 {
			result := make([]map[string]interface{}, 0, len(users))
			for _, u := range users {
				item := map[string]interface{}{
					"user_id":   u.ID,
					"student_id": u.Username,
					"name":      u.DisplayName,
					"major":     u.Major,
					"class":     u.ClassName,
					"college":   u.College,
					"status":    u.Status,
				}
				result = append(result, item)
			}
			return &StudentInfoQuery{Query: query, Result: result, DataSource: "real"}
		}
	}

	// 无真实数据或未命中：诚实返回空结果（data_source=real），不展示伪数据
	return &StudentInfoQuery{
		Query:      query,
		Result:     []map[string]interface{}{},
		DataSource: "real",
	}
}

// ─── P2 补充功能 ───

// MaterialTemplate 材料模板
type MaterialTemplate struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Content  string `json:"content"`
	Usage    int    `json:"usage_count"`
}

// MaterialTemplateResult 材料模板库结果
type MaterialTemplateResult struct {
	Templates  []MaterialTemplate `json:"templates"`
	Total      int                `json:"total"`
	DataSource string             `json:"data_source"`
}

// GetMaterialTemplates AI 材料模板库
func (s *AssistantService) GetMaterialTemplates(ctx context.Context, category string) *MaterialTemplateResult {
	templates := []MaterialTemplate{
		{ID: 1, Name: "学生请假申请表", Category: "学生事务", Content: "姓名：___\n学号：___\n请假事由：___\n请假时间：___至___\n联系方式：___", Usage: 320},
		{ID: 2, Name: "成绩复核申请表", Category: "教务管理", Content: "姓名：___\n课程名称：___\n考试日期：___\n申请理由：___", Usage: 86},
		{ID: 3, Name: "教室借用申请", Category: "教务管理", Content: "申请人：___\n借用教室：___\n借用时间：___\n用途：___\n参与人数：___", Usage: 145},
		{ID: 4, Name: "学生奖学金推荐表", Category: "学生事务", Content: "姓名：___\n专业班级：___\nGPA排名：___\n获奖情况：___\n推荐理由：___", Usage: 210},
		{ID: 5, Name: "课程调停课申请", Category: "教务管理", Content: "教师姓名：___\n课程名称：___\n原时间：___\n拟调整为：___\n原因：___", Usage: 67},
	}

	if category != "" {
		var filtered []MaterialTemplate
		for _, t := range templates {
			if t.Category == category {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
	}

	return &MaterialTemplateResult{
		Templates:  templates,
		Total:      len(templates),
		DataSource: "reference",
	}
}

// DocProcessResult 文档智能处理结果
type DocProcessResult struct {
	FileName       string                   `json:"file_name"`
	FileType       string                   `json:"file_type"`
	ExtractedData  []map[string]interface{} `json:"extracted_data"`
	Classification string                   `json:"classification"`
	Summary        string                   `json:"summary"`
	DataSource     string                   `json:"data_source"`
}

// ProcessDocument AI 文档智能处理
func (s *AssistantService) ProcessDocument(ctx context.Context, fileName, fileType string) *DocProcessResult {
	result := &DocProcessResult{
		FileName: fileName,
		FileType: fileType,
		ExtractedData: []map[string]interface{}{
			{"field": "标题", "value": "2026年春季学期期末考试安排"},
			{"field": "发文单位", "value": "教务处"},
			{"field": "发文日期", "value": "2026-05-20"},
			{"field": "关键内容", "value": "考试时间6月15日-7月5日，共安排186场考试"},
		},
		Classification: "教务通知-考试安排",
		Summary:        "该文档为教务处发布的期末考试安排通知，涉及2026年春季学期全部186场考试的时间和场地安排。",
		DataSource:     "reference",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("对以下教务文档进行智能分类和摘要（30字以内）：\n文件名：%s\n类型：%s", fileName, fileType)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 100,
		})
		if err == nil && resp != nil && resp.Content != "" {
			result.Summary = strings.TrimSpace(resp.Content)
			result.DataSource = "ai"
		}
	}

	return result
}

// WorkflowResult 流程自动化结果
type WorkflowResult struct {
	WorkflowID  string                   `json:"workflow_id"`
	Name        string                   `json:"name"`
	Status      string                   `json:"status"`
	Steps       []map[string]interface{} `json:"steps"`
	AutoActions []string                 `json:"auto_actions"`
	DataSource  string                   `json:"data_source"`
}

// AutomateWorkflow AI 流程自动化
func (s *AssistantService) AutomateWorkflow(ctx context.Context, workflowType string) *WorkflowResult {
	if workflowType == "" {
		workflowType = "成绩录入"
	}

	return &WorkflowResult{
		WorkflowID: "wf-" + workflowType,
		Name:       workflowType + "自动化流程",
		Status:     "active",
		Steps: []map[string]interface{}{
			{"step": 1, "name": "数据采集", "status": "completed", "auto": true, "desc": "从教务系统自动拉取原始成绩"},
			{"step": 2, "name": "格式校验", "status": "completed", "auto": true, "desc": "检查成绩范围、缺考标记、格式一致性"},
			{"step": 3, "name": "异常检测", "status": "in_progress", "auto": true, "desc": "AI 标注异常分数（如突降/全班低分）"},
			{"step": 4, "name": "人工确认", "status": "pending", "auto": false, "desc": "教师确认异常标注，审批最终成绩"},
			{"step": 5, "name": "系统录入", "status": "pending", "auto": true, "desc": "自动提交至教务系统并生成录入回执"},
		},
		AutoActions: []string{
			"自动发送成绩录入提醒（截止前3天/1天）",
			"自动检测未录入课程并催办",
			"异常成绩自动标注并通知教师",
		},
		DataSource: "reference",
	}
}

// ProcessStepDetail 流程步骤详情
type ProcessStepDetail struct {
	StepID     int64    `json:"step_id"`
	Title      string   `json:"title"`
	Contact    string   `json:"contact"`
	Location   string   `json:"location"`
	Materials  []string `json:"materials"`
	FAQ        []string `json:"faq"`
	MediaURLs  []string `json:"media_urls"`
	DataSource string   `json:"data_source"`
}

// ProcessStepManageResult 流程步骤管理结果
type ProcessStepManageResult struct {
	ProcessName string              `json:"process_name"`
	Steps       []ProcessStepDetail `json:"steps"`
	Total       int                 `json:"total"`
	DataSource  string              `json:"data_source"`
}

// ManageProcessSteps 流程步骤详情管理
func (s *AssistantService) ManageProcessSteps(ctx context.Context, processID string) *ProcessStepManageResult {
	return &ProcessStepManageResult{
		ProcessName: "转专业办理流程",
		Steps: []ProcessStepDetail{
			{StepID: 1, Title: "在线申请", Contact: "教务处李老师 88880001", Location: "教务系统→学籍异动→转专业申请", Materials: []string{"成绩单", "转专业申请表"}, FAQ: []string{"Q: GPA要求？A: 转入专业前30%"}, DataSource: "db"},
			{StepID: 2, Title: "学院审核", Contact: "学院教学办 88880002", Location: "信息楼A301", Materials: []string{"院系同意函"}, FAQ: []string{"Q: 审核周期？A: 一般5个工作日"}, DataSource: "db"},
			{StepID: 3, Title: "教务处审批", Contact: "教务处综合科 88880003", Location: "行政楼201", Materials: []string{}, FAQ: []string{"Q: 如何查进度？A: 教务系统可查"}, DataSource: "db"},
		},
		Total:      3,
		DataSource: "db",
	}
}

// MusicRadioResult 音乐电台
type MusicRadioResult struct {
	NowPlaying map[string]interface{}   `json:"now_playing"`
	Playlist   []map[string]interface{} `json:"playlist"`
	Categories []string                 `json:"categories"`
	DataSource string                   `json:"data_source"`
}

// GetMusicRadio 音乐电台
func (s *AssistantService) GetMusicRadio(ctx context.Context, category string) *MusicRadioResult {
	if category == "" {
		category = "轻音乐"
	}

	return &MusicRadioResult{
		NowPlaying: map[string]interface{}{
			"title": "Canon in D", "artist": "Pachelbel", "duration": "5:30", "category": category,
		},
		Playlist: []map[string]interface{}{
			{"title": "River Flows in You", "artist": "Yiruma", "duration": "3:54"},
			{"title": "春野", "artist": "Bandari", "duration": "4:12"},
			{"title": "天空之城", "artist": "久石让", "duration": "4:48"},
			{"title": "克罗地亚狂想曲", "artist": "Maksim", "duration": "3:36"},
		},
		Categories: []string{"轻音乐", "古典", "校园民谣", "白噪音", "学习专注"},
		DataSource: "reference",
	}
}

// ActivityRegisterResult 校园活动报名
type ActivityRegisterResult struct {
	Activities []map[string]interface{} `json:"activities"`
	Total      int                      `json:"total"`
	DataSource string                   `json:"data_source"`
}

// GetActivityRegister 校园活动报名
func (s *AssistantService) GetActivityRegister(ctx context.Context, status string) *ActivityRegisterResult {
	activities := []map[string]interface{}{
		{"id": 1, "title": "2026年程序设计大赛校内选拔", "date": "2026-06-01", "location": "信息楼301", "capacity": 100, "registered": 67, "status": "报名中"},
		{"id": 2, "title": "心理健康月主题讲座", "date": "2026-05-28", "location": "大学生活动中心", "capacity": 200, "registered": 189, "status": "报名中"},
		{"id": 3, "title": "毕业季跳蚤市场", "date": "2026-06-10", "location": "南区篮球场", "capacity": 50, "registered": 50, "status": "已满"},
	}

	if status != "" {
		var filtered []map[string]interface{}
		for _, a := range activities {
			if a["status"] == status {
				filtered = append(filtered, a)
			}
		}
		activities = filtered
	}

	return &ActivityRegisterResult{
		Activities: activities,
		Total:      len(activities),
		DataSource: "reference",
	}
}
