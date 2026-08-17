package handler

import (
	"database/sql"
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

// getUserInfo 获取用户信息
func (h *StudentHandler) getUserInfo(userID int64) HomeStudentUserInfo {
	info := HomeStudentUserInfo{
		Name:      "同学",
		StudentID: "",
		College:   "",
		Major:     "",
		Grade:     "",
	}
	if h.db == nil {
		return info
	}

	var displayName, username, college, major, enrollmentYear sql.NullString
	err := h.db.QueryRow(
		"SELECT display_name, username, college, major, enrollment_year FROM users WHERE id = ?",
		userID,
	).Scan(&displayName, &username, &college, &major, &enrollmentYear)
	if err != nil {
		log.Printf("获取用户信息失败 user_id=%d: %v", userID, err)
		return info
	}
	if displayName.Valid {
		info.Name = displayName.String
	}
	if username.Valid {
		info.StudentID = username.String
	}
	if college.Valid {
		info.College = college.String
	}
	if major.Valid {
		info.Major = major.String
	}
	if enrollmentYear.Valid && enrollmentYear.String != "" {
		info.Grade = enrollmentYear.String + "级"
	}
	return info
}

// resolveCurrentCalendar 获取当前学期校历与教学周
func (h *StudentHandler) resolveCurrentCalendar() (*AcademicCalendar, int) {
	if h.db == nil {
		return nil, 0
	}

	today := time.Now().Format("2006-01-02")

	calendar := &AcademicCalendar{}
	err := h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE start_date <= ? AND end_date >= ? ORDER BY id DESC LIMIT 1",
		today, today,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == nil {
		return calendar, calcHomeCurrentWeek(calendar.StartDate, today)
	}
	if err != sql.ErrNoRows {
		log.Printf("查询当前学期校历失败: %v", err)
		return nil, 0
	}

	calendar = &AcademicCalendar{}
	err = h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE start_date > ? ORDER BY start_date ASC LIMIT 1",
		today,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == nil {
		return calendar, 0
	}
	if err != sql.ErrNoRows {
		log.Printf("查询未来学期校历失败: %v", err)
		return nil, 0
	}

	calendar = &AcademicCalendar{}
	err = h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE end_date < ? ORDER BY end_date DESC LIMIT 1",
		today,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == sql.ErrNoRows {
		log.Printf("无任何校历记录")
		return nil, 0
	}
	if err != nil {
		log.Printf("查询过往学期校历失败: %v", err)
		return nil, 0
	}
	return calendar, 0
}

// calcHomeCurrentWeek 计算当前教学周
func calcHomeCurrentWeek(startDate, today string) int {
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

// getTodayCourses 获取今日课程
func (h *StudentHandler) getTodayCourses(userID int64, weekday int, calendar *AcademicCalendar) []HomeStudentCourse {
	courses := make([]HomeStudentCourse, 0)
	if h.db == nil || calendar == nil {
		return courses
	}

	rows, err := h.db.Query(
		"SELECT course_name, start_period, end_period, location, teacher, color "+
			"FROM course_schedules WHERE user_id = ? AND semester_code = ? AND weekday = ? "+
			"ORDER BY start_period ASC, id ASC",
		userID, calendar.SemesterCode, weekday,
	)
	if err != nil {
		return courses
	}
	defer rows.Close()

	periodTimes := []string{
		"", "08:00-08:45", "08:55-09:40",
		"10:00-10:45", "10:55-11:40",
		"14:00-14:45", "14:55-15:40",
		"16:00-16:45", "16:55-17:40",
		"19:00-19:45", "19:55-20:40",
	}

	for rows.Next() {
		var courseName, location, teacher, color sql.NullString
		var startPeriod, endPeriod int
		if err := rows.Scan(&courseName, &startPeriod, &endPeriod, &location, &teacher, &color); err != nil {
			continue
		}
		timeStr := ""
		if startPeriod >= 1 && startPeriod <= 10 && endPeriod >= startPeriod && endPeriod <= 10 {
			startTime := periodTimes[startPeriod][:5]
			endTime := periodTimes[endPeriod][6:]
			timeStr = startTime + "-" + endTime
		}
		courses = append(courses, HomeStudentCourse{
			CourseName: courseName.String,
			Time:       timeStr,
			Location:   location.String,
			Teacher:    teacher.String,
			Color:      color.String,
		})
	}
	return courses
}

// getTodayTasks 获取今日任务
func (h *StudentHandler) getTodayTasks(userID int64, todayStr string) []HomeStudentTask {
	tasks := make([]HomeStudentTask, 0)
	if h.db == nil {
		return tasks
	}

	rows, err := h.db.Query(
		"SELECT t.id, t.title, t.plan_id, t.status, t.scheduled_duration "+
			"FROM study_plan_tasks t JOIN study_plans p ON t.plan_id = p.id "+
			"WHERE p.user_id = ? AND t.scheduled_date = ? "+
			"ORDER BY t.sort_order ASC, t.id ASC",
		userID, todayStr,
	)
	if err != nil {
		log.Printf("查询今日任务失败 user_id=%d: %v", userID, err)
		return tasks
	}
	defer rows.Close()

	for rows.Next() {
		var title sql.NullString
		var duration sql.NullInt64
		task := HomeStudentTask{}
		if err := rows.Scan(&task.ID, &title, &task.PlanID, &task.Status, &duration); err != nil {
			continue
		}
		task.Title = title.String
		task.Duration = int(duration.Int64)
		tasks = append(tasks, task)
	}
	return tasks
}

// getUpcomingEvents 获取近期事件（未来7天 + 正在进行中的）
func (h *StudentHandler) getUpcomingEvents(todayStr string, calendar *AcademicCalendar) []HomeStudentEvent {
	events := make([]HomeStudentEvent, 0)
	if h.db == nil || calendar == nil {
		return events
	}

	fromDate := todayStr
	toDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	rows, err := h.db.Query(
		"SELECT id, event_name, event_type, start_date, end_date "+
			"FROM academic_calendar_events WHERE semester_code = ? "+
			"AND (start_date <= ? AND (end_date >= ? OR end_date IS NULL)) "+
			"ORDER BY start_date ASC, id ASC LIMIT 10",
		calendar.SemesterCode, toDate, fromDate,
	)
	if err != nil {
		return events
	}
	defer rows.Close()

	today, _ := time.Parse("2006-01-02", todayStr)

	for rows.Next() {
		var eventName, eventType, startDate sql.NullString
		var endDate sql.NullString
		var id int64
		if err := rows.Scan(&id, &eventName, &eventType, &startDate, &endDate); err != nil {
			continue
		}
		start, err := time.Parse("2006-01-02", startDate.String)
		if err != nil {
			continue
		}
		daysLeft := int(start.Sub(today).Hours() / 24)
		events = append(events, HomeStudentEvent{
			ID:        id,
			EventName: eventName.String,
			EventType: eventType.String,
			StartDate: startDate.String,
			DaysLeft:  daysLeft,
		})
	}
	return events
}

// getHomeStats 获取首页统计数据
func (h *StudentHandler) getHomeStats(userID int64) HomeStudentStats {
	stats := HomeStudentStats{}
	if h.db == nil {
		return stats
	}

	// 进行中的学习计划数
	countErr := h.db.QueryRow(
		"SELECT COUNT(*) FROM study_plans WHERE user_id = ? AND status = 'active'",
		userID,
	).Scan(&stats.PlansInProgress)
	if countErr != nil {
		log.Printf("获取首页统计失败 user_id=%d: %v", userID, countErr)
	}

	return stats
}
