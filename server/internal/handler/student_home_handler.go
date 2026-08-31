package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 学生首页数据接口
// ═══════════════════════════════════════════════

// HomeStudentCourse 今日课程
type HomeStudentCourse struct {
	CourseName string `json:"course_name"`
	Time       string `json:"time"`
	Location   string `json:"location"`
	Teacher    string `json:"teacher"`
	Color      string `json:"color"`
}

// HomeStudentTask 今日任务
type HomeStudentTask struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	PlanID   int64  `json:"plan_id"`
	Status   string `json:"status"`
	Duration int    `json:"duration"`
}

// HomeStudentEvent 近期事件
type HomeStudentEvent struct {
	ID        int64  `json:"id"`
	EventName string `json:"event_name"`
	EventType string `json:"event_type"`
	StartDate string `json:"start_date"`
	DaysLeft  int    `json:"days_left"`
}

// HomeStudentStats 统计数据
type HomeStudentStats struct {
	UnreadNotifications int `json:"unread_notifications"`
	PendingFeedback     int `json:"pending_feedback"`
	PlansInProgress     int `json:"plans_in_progress"`
}

// HomeStudentQuickEntry 功能入口
type HomeStudentQuickEntry struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Route string `json:"route"`
}

// HomeStudentUserInfo 用户信息
type HomeStudentUserInfo struct {
	Name      string `json:"name"`
	StudentID string `json:"student_id"`
	College   string `json:"college"`
	Major     string `json:"major"`
	Grade     string `json:"grade"`
}

// HomeStudentToday 今日信息
type HomeStudentToday struct {
	Date         string `json:"date"`
	Weekday      string `json:"weekday"`
	WeekNo       int    `json:"week_no"`
	SemesterName string `json:"semester_name"`
}

// Home 学生首页数据
// GET /api/v1/student/home
func (h *StudentHandler) Home(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    503,
			Message: "数据库未初始化",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	today := time.Now()
	todayStr := today.Format("2006-01-02")
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekdayNames := []string{"", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}

	// 1. 获取用户信息
	userInfo := h.getUserInfo(userCtx.UserID)

	// 2. 获取当前学期和教学周
	calendar, weekNo := h.resolveCurrentCalendar()
	semesterName := ""
	if calendar != nil {
		semesterName = calendar.SemesterName
		if calendar.Status == "completed" {
			semesterName += "（已结束）"
		} else if calendar.Status == "upcoming" {
			semesterName += "（未开始）"
		}
	}

	// 3. 获取今日课程
	todayCourses := h.getTodayCourses(userCtx.UserID, weekday, calendar)

	// 4. 获取今日任务
	todayTasks := h.getTodayTasks(userCtx.UserID, todayStr)

	// 5. 获取近期事件（未来7天 + 正在进行中的）
	upcomingEvents := h.getUpcomingEvents(todayStr, calendar)

	// 6. 获取统计数据
	stats := h.getHomeStats(userCtx.UserID)

	// 7. 功能入口（固定配置）
	quickEntries := []HomeStudentQuickEntry{
		{Icon: "chat", Title: "AI问答", Route: "/chat"},
		{Icon: "study_plan", Title: "学习计划", Route: "/student/study-plan"},
		{Icon: "timetable", Title: "我的课表", Route: "/student/timetable"},
		{Icon: "career", Title: "就业服务", Route: "/student/career"},
		{Icon: "study", Title: "学业服务", Route: "/student/study"},
		{Icon: "mental", Title: "心理健康", Route: "/student/mental"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"user_info": userInfo,
			"today": HomeStudentToday{
				Date:         todayStr,
				Weekday:      weekdayNames[weekday],
				WeekNo:       weekNo,
				SemesterName: semesterName,
			},
			"today_courses":   todayCourses,
			"today_tasks":     todayTasks,
			"upcoming_events": upcomingEvents,
			"stats":           stats,
			"quick_entries":   quickEntries,
		},
	})
}

// getUserInfo 获取用户信息（SQL 已下沉 StudentProfileRepo，P4-d）
func (h *StudentHandler) getUserInfo(userID int64) HomeStudentUserInfo {
	info := HomeStudentUserInfo{
		Name:      "同学",
		StudentID: "",
		College:   "",
		Major:     "",
		Grade:     "",
	}
	if h.profileRepo == nil {
		return info
	}

	u, err := h.profileRepo.GetHomeUserInfo(userID)
	if err != nil {
		log.Printf("获取用户信息失败 user_id=%d: %v", userID, err)
		return info
	}
	if u.DisplayName != "" {
		info.Name = u.DisplayName
	}
	info.StudentID = u.Username
	info.College = u.College
	info.Major = u.Major
	if u.EnrollmentYear != "" {
		info.Grade = u.EnrollmentYear + "级"
	}
	return info
}

// resolveCurrentCalendar 获取当前学期校历与教学周（复用 StudyPlanRepo，P4-d 消重）
func (h *StudentHandler) resolveCurrentCalendar() (*model.AcademicCalendar, int) {
	if h.studyPlanRepo == nil {
		return nil, 0
	}
	calendar, week, err := h.studyPlanRepo.ResolveCurrentCalendar()
	if err != nil {
		log.Printf("解析当前校历失败: %v", err)
		return nil, 0
	}
	return calendar, week
}

// getTodayCourses 获取今日课程（节次→时间段映射留在展示层）
func (h *StudentHandler) getTodayCourses(userID int64, weekday int, calendar *model.AcademicCalendar) []HomeStudentCourse {
	courses := make([]HomeStudentCourse, 0)
	if h.studyPlanRepo == nil || calendar == nil {
		return courses
	}

	rows, err := h.studyPlanRepo.ListTodayCourses(userID, calendar.SemesterCode, weekday)
	if err != nil {
		return courses
	}

	periodTimes := []string{
		"", "08:00-08:45", "08:55-09:40",
		"10:00-10:45", "10:55-11:40",
		"14:00-14:45", "14:55-15:40",
		"16:00-16:45", "16:55-17:40",
		"19:00-19:45", "19:55-20:40",
	}

	for _, row := range rows {
		timeStr := ""
		if row.StartPeriod >= 1 && row.StartPeriod <= 10 && row.EndPeriod >= row.StartPeriod && row.EndPeriod <= 10 {
			startTime := periodTimes[row.StartPeriod][:5]
			endTime := periodTimes[row.EndPeriod][6:]
			timeStr = startTime + "-" + endTime
		}
		courses = append(courses, HomeStudentCourse{
			CourseName: row.CourseName,
			Time:       timeStr,
			Location:   row.Location,
			Teacher:    row.Teacher,
			Color:      row.Color,
		})
	}
	return courses
}

// getTodayTasks 获取今日任务（SQL 已下沉 StudyPlanRepo，P4-d）
func (h *StudentHandler) getTodayTasks(userID int64, todayStr string) []HomeStudentTask {
	tasks := make([]HomeStudentTask, 0)
	if h.studyPlanRepo == nil {
		return tasks
	}

	rows, err := h.studyPlanRepo.ListTodayTasks(userID, todayStr)
	if err != nil {
		log.Printf("查询今日任务失败 user_id=%d: %v", userID, err)
		return tasks
	}

	for _, row := range rows {
		task := HomeStudentTask{}
		task.ID = row.ID
		task.Title = row.Title
		task.PlanID = row.PlanID
		task.Status = row.Status
		task.Duration = row.Duration
		tasks = append(tasks, task)
	}
	return tasks
}

// getUpcomingEvents 获取近期事件（未来7天 + 正在进行中的）
func (h *StudentHandler) getUpcomingEvents(todayStr string, calendar *model.AcademicCalendar) []HomeStudentEvent {
	events := make([]HomeStudentEvent, 0)
	if h.studyPlanRepo == nil || calendar == nil {
		return events
	}

	toDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	rows, err := h.studyPlanRepo.ListUpcomingEvents(calendar.SemesterCode, todayStr, toDate)
	if err != nil {
		return events
	}

	today, _ := time.Parse("2006-01-02", todayStr)

	for _, row := range rows {
		start, err := time.Parse("2006-01-02", row.StartDate)
		if err != nil {
			continue
		}
		daysLeft := int(start.Sub(today).Hours() / 24)
		events = append(events, HomeStudentEvent{
			ID:        row.ID,
			EventName: row.EventName,
			EventType: row.EventType,
			StartDate: row.StartDate,
			DaysLeft:  daysLeft,
		})
	}
	return events
}

// getHomeStats 获取首页统计数据（SQL 已下沉 StudyPlanRepo，P4-d）
func (h *StudentHandler) getHomeStats(userID int64) HomeStudentStats {
	stats := HomeStudentStats{}
	if h.studyPlanRepo == nil {
		return stats
	}

	count, err := h.studyPlanRepo.CountActivePlans(userID)
	if err != nil {
		log.Printf("获取首页统计失败 user_id=%d: %v", userID, err)
		return stats
	}
	stats.PlansInProgress = count

	return stats
}
