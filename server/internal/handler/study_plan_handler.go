package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// StudyPlanHandler 学习计划与校历 HTTP handler
type StudyPlanHandler struct {
	db           *sql.DB
	studyPlanSvc *service.StudyPlanService
}

// NewStudyPlanHandler 创建学习计划 handler
func NewStudyPlanHandler(db *sql.DB, studyPlanSvc *service.StudyPlanService) *StudyPlanHandler {
	return &StudyPlanHandler{db: db, studyPlanSvc: studyPlanSvc}
}

// ═══════════════════════════════════════════════
// 数据结构定义
// ═══════════════════════════════════════════════

// AcademicCalendar 学期校历
type AcademicCalendar struct {
	ID           int64  `json:"id"`
	AcademicYear int    `json:"academic_year"`
	Semester     int    `json:"semester"`
	SemesterCode string `json:"semester_code"`
	SemesterName string `json:"semester_name"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	RegisterDate string `json:"register_date,omitempty"`
	TotalWeeks   int    `json:"total_weeks"`
	WeekStartDay string `json:"week_start_day"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// CalendarEvent 校历事件
type CalendarEvent struct {
	ID             int64  `json:"id"`
	SemesterCode   string `json:"semester_code"`
	EventName      string `json:"event_name"`
	EventType      string `json:"event_type"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date,omitempty"`
	WeekNo         int    `json:"week_no,omitempty"`
	AffectsClasses int    `json:"affects_classes"`
	Description    string `json:"description,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// CourseScheduleItem 课表项
type CourseScheduleItem struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	CourseID     string `json:"course_id"`
	CourseName   string `json:"course_name"`
	SemesterCode string `json:"semester_code"`
	Weekday      int    `json:"weekday"`
	StartPeriod  int    `json:"start_period"`
	EndPeriod    int    `json:"end_period"`
	WeeksPattern string `json:"weeks_pattern"`
	Location     string `json:"location,omitempty"`
	Teacher      string `json:"teacher,omitempty"`
	Color        string `json:"color"`
	CreatedAt    string `json:"created_at"`
}

// StudyPlan 学习计划
type StudyPlan struct {
	ID           int64    `json:"id"`
	UserID       int64    `json:"user_id"`
	Title        string   `json:"title"`
	PlanType     string   `json:"plan_type"`
	SemesterCode string   `json:"semester_code,omitempty"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	Goals        []string `json:"goals"`
	Progress     float64  `json:"progress"`
	AIGenerated  bool     `json:"ai_generated"`
	Status       string   `json:"status"`
	LinkedPlanID *int64   `json:"linked_plan_id,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	// 任务统计（列表场景）
	TaskTotal   int              `json:"task_total,omitempty"`
	TaskDone    int              `json:"task_done,omitempty"`
	TaskPending int              `json:"task_pending,omitempty"`
	Tasks       []*StudyPlanTask `json:"tasks,omitempty"`
}

// StudyPlanTask 学习计划任务
type StudyPlanTask struct {
	ID                int64  `json:"id"`
	PlanID            int64  `json:"plan_id"`
	CourseID          string `json:"course_id,omitempty"`
	CourseName        string `json:"course_name,omitempty"`
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	ScheduledDate     string `json:"scheduled_date,omitempty"`
	ScheduledDuration int    `json:"scheduled_duration"`
	ActualDuration    int    `json:"actual_duration"`
	Status            string `json:"status"`
	Evidence          string `json:"evidence,omitempty"`
	Reflection        string `json:"reflection,omitempty"`
	SortOrder         int    `json:"sort_order"`
	CreatedAt         string `json:"created_at"`
}

// PlanOverviewItem 计划概览项（用于多Tab首页）
type PlanOverviewItem struct {
	PlanType  string  `json:"plan_type"`
	PlanCount int     `json:"plan_count"`
	Progress  float64 `json:"progress"` // 平均进度
	TaskTotal int     `json:"task_total"`
	TaskDone  int     `json:"task_done"`
}

// ═══════════════════════════════════════════════
// 一、校历接口 /api/v1/study/calendar
// ═══════════════════════════════════════════════

// GetCurrentCalendar 获取当前学期校历（含近期事件）
// GET /api/v1/study/calendar/current
func (h *StudyPlanHandler) GetCurrentCalendar(c *gin.Context) {
	calendar, currentWeek, err := h.resolveCurrentCalendar()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询当前校历失败: " + err.Error()})
		return
	}

	// 近期事件：当前日期前后 30 天内
	today := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	to := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	events := make([]*CalendarEvent, 0)
	if calendar != nil {
		rows, err := h.db.Query(
			"SELECT id, semester_code, event_name, event_type, start_date, end_date, week_no, affects_classes, description, created_at "+
				"FROM academic_calendar_events WHERE semester_code = ? AND start_date <= ? AND end_date >= ? "+
				"ORDER BY start_date ASC, id ASC",
			calendar.SemesterCode, to, from,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询校历事件失败"})
			return
		}
		defer rows.Close()
		events = scanCalendarEvents(rows)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"message":      "success",
		"data":         calendar,
		"current_week": currentWeek,
		"today":        today,
		"events":       events,
	})
}

// GetCalendarBySemester 获取指定学期校历（含全部事件）
// GET /api/v1/study/calendar/:semester_code
func (h *StudyPlanHandler) GetCalendarBySemester(c *gin.Context) {
	semesterCode := c.Param("semester_code")
	if semesterCode == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "semester_code 不能为空"})
		return
	}

	calendar := &AcademicCalendar{}
	err := h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE semester_code = ?",
		semesterCode,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学期校历不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询校历失败"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, semester_code, event_name, event_type, start_date, end_date, week_no, affects_classes, description, created_at "+
			"FROM academic_calendar_events WHERE semester_code = ? ORDER BY start_date ASC, id ASC",
		semesterCode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询校历事件失败"})
		return
	}
	defer rows.Close()
	events := scanCalendarEvents(rows)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    calendar,
		"events":  events,
	})
}

// ═══════════════════════════════════════════════
// 二、课表接口 /api/v1/study/timetable
// ═══════════════════════════════════════════════

// GetMyTimetable 获取我的课表（按 weekday 分组）
// GET /api/v1/study/timetable?semester_code=
func (h *StudyPlanHandler) GetMyTimetable(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	semesterCode := c.Query("semester_code")
	// 默认使用当前学期
	if semesterCode == "" {
		calendar, _, err := h.resolveCurrentCalendar()
		if err != nil || calendar == nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "无法确定当前学期"})
			return
		}
		semesterCode = calendar.SemesterCode
	}

	rows, err := h.db.Query(
		"SELECT id, user_id, course_id, course_name, semester_code, weekday, start_period, end_period, "+
			"weeks_pattern, location, teacher, color, created_at "+
			"FROM course_schedules WHERE user_id = ? AND semester_code = ? ORDER BY weekday ASC, start_period ASC, id ASC",
		userCtx.UserID, semesterCode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询课表失败"})
		return
	}
	defer rows.Close()

	// 按 weekday 分组：1-7 → []
	grouped := make(map[int][]*CourseScheduleItem)
	all := make([]*CourseScheduleItem, 0)
	for rows.Next() {
		item := &CourseScheduleItem{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.CourseID, &item.CourseName,
			&item.SemesterCode, &item.Weekday, &item.StartPeriod, &item.EndPeriod,
			&item.WeeksPattern, &item.Location, &item.Teacher, &item.Color, &item.CreatedAt); err != nil {
			continue
		}
		grouped[item.Weekday] = append(grouped[item.Weekday], item)
		all = append(all, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"message":       "success",
		"semester_code": semesterCode,
		"data":          all,
		"grouped":       grouped,
		"total":         len(all),
	})
}

// ═══════════════════════════════════════════════
// 三、学习计划接口 /api/v1/study/plans
// ═══════════════════════════════════════════════

// ListMyPlans 获取我的学习计划列表
// GET /api/v1/study/plans?plan_type=

func (h *StudyPlanHandler) resolveCurrentCalendar() (*AcademicCalendar, int, error) {
	today := time.Now().Format("2006-01-02")

	// 1. 当前在某个学期内
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
		return calendar, calcCurrentWeek(calendar.StartDate, today), nil
	}
	if err != sql.ErrNoRows {
		return nil, 0, err
	}

	// 2. 不在任何学期：返回最近的即将开始学期
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
		return calendar, 0, nil
	}
	if err != sql.ErrNoRows {
		return nil, 0, err
	}

	// 3. 都没有：返回最近的已完成学期
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
		return nil, 0, nil // 数据库无任何校历数据
	}
	if err != nil {
		return nil, 0, err
	}
	return calendar, 0, nil
}

// calcCurrentWeek 计算当前教学周：(today - start_date) / 7 + 1
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

// calcDefaultEndDate 根据 plan_type 推算默认结束日期
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

// scanCalendarEvents 扫描行集为事件列表
func scanCalendarEvents(rows *sql.Rows) []*CalendarEvent {
	var list []*CalendarEvent
	for rows.Next() {
		e := &CalendarEvent{}
		var endDate, description sql.NullString
		var weekNo sql.NullInt64
		if err := rows.Scan(&e.ID, &e.SemesterCode, &e.EventName, &e.EventType,
			&e.StartDate, &endDate, &weekNo, &e.AffectsClasses, &description, &e.CreatedAt); err != nil {
			continue
		}
		if endDate.Valid {
			e.EndDate = endDate.String
		}
		if weekNo.Valid {
			e.WeekNo = int(weekNo.Int64)
		}
		if description.Valid {
			e.Description = description.String
		}
		list = append(list, e)
	}
	return list
}

// scanStudyPlan 扫描一行学习计划
func scanStudyPlan(rows *sql.Rows) (*StudyPlan, error) {
	plan := &StudyPlan{}
	var semesterCode sql.NullString
	var linkedID sql.NullInt64
	var goalsJSON string
	var aiGen int
	if err := rows.Scan(&plan.ID, &plan.UserID, &plan.Title, &plan.PlanType, &semesterCode,
		&plan.StartDate, &plan.EndDate, &goalsJSON, &plan.Progress, &aiGen, &plan.Status,
		&linkedID, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return nil, err
	}
	if semesterCode.Valid {
		plan.SemesterCode = semesterCode.String
	}
	if linkedID.Valid {
		v := linkedID.Int64
		plan.LinkedPlanID = &v
	}
	plan.AIGenerated = aiGen == 1
	_ = json.Unmarshal([]byte(goalsJSON), &plan.Goals)
	if plan.Goals == nil {
		plan.Goals = []string{}
	}
	return plan, nil
}

// getPlanByID 按 ID+user_id 查询计划（校验归属）
func (h *StudyPlanHandler) getPlanByID(id, userID int64) (*StudyPlan, error) {
	plan := &StudyPlan{}
	var semesterCode sql.NullString
	var linkedID sql.NullInt64
	var goalsJSON string
	var aiGen int
	err := h.db.QueryRow(
		"SELECT id, user_id, title, plan_type, semester_code, start_date, end_date, goals_json, "+
			"progress, ai_generated, status, linked_plan_id, created_at, updated_at "+
			"FROM study_plans WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&plan.ID, &plan.UserID, &plan.Title, &plan.PlanType, &semesterCode,
		&plan.StartDate, &plan.EndDate, &goalsJSON, &plan.Progress, &aiGen, &plan.Status,
		&linkedID, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if semesterCode.Valid {
		plan.SemesterCode = semesterCode.String
	}
	if linkedID.Valid {
		v := linkedID.Int64
		plan.LinkedPlanID = &v
	}
	plan.AIGenerated = aiGen == 1
	_ = json.Unmarshal([]byte(goalsJSON), &plan.Goals)
	if plan.Goals == nil {
		plan.Goals = []string{}
	}
	return plan, nil
}

// listTasksByPlan 查询指定计划下的所有任务（按 sort_order、id 排序）
func (h *StudyPlanHandler) listTasksByPlan(planID int64) ([]*StudyPlanTask, error) {
	rows, err := h.db.Query(
		"SELECT id, plan_id, course_id, course_name, title, description, scheduled_date, "+
			"scheduled_duration, actual_duration, status, evidence, reflection, sort_order, created_at "+
			"FROM study_plan_tasks WHERE plan_id = ? ORDER BY sort_order ASC, id ASC",
		planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*StudyPlanTask
	for rows.Next() {
		t := &StudyPlanTask{}
		var courseID, courseName, description, scheduledDate, evidence, reflection sql.NullString
		if err := rows.Scan(&t.ID, &t.PlanID, &courseID, &courseName, &t.Title, &description,
			&scheduledDate, &t.ScheduledDuration, &t.ActualDuration, &t.Status,
			&evidence, &reflection, &t.SortOrder, &t.CreatedAt); err != nil {
			continue
		}
		if courseID.Valid {
			t.CourseID = courseID.String
		}
		if courseName.Valid {
			t.CourseName = courseName.String
		}
		if description.Valid {
			t.Description = description.String
		}
		if scheduledDate.Valid {
			t.ScheduledDate = scheduledDate.String
		}
		if evidence.Valid {
			t.Evidence = evidence.String
		}
		if reflection.Valid {
			t.Reflection = reflection.String
		}
		list = append(list, t)
	}
	return list, nil
}

// fillPlanTaskStats 填充计划的任务完成统计
func (h *StudyPlanHandler) fillPlanTaskStats(plan *StudyPlan) {
	if plan == nil {
		return
	}
	_ = h.db.QueryRow(
		"SELECT COUNT(*), SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), "+
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) "+
			"FROM study_plan_tasks WHERE plan_id = ?",
		plan.ID,
	).Scan(&plan.TaskTotal, &plan.TaskDone, &plan.TaskPending)
}

// recalcPlanProgress 根据任务完成率重新计算计划进度
func (h *StudyPlanHandler) recalcPlanProgress(planID int64) {
	var total, done int
	_ = h.db.QueryRow(
		"SELECT COUNT(*), SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) FROM study_plan_tasks WHERE plan_id = ?",
		planID,
	).Scan(&total, &done)
	var progress float64
	if total > 0 {
		progress = float64(done) / float64(total) * 100
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, _ = h.db.Exec(
		"UPDATE study_plans SET progress = ?, updated_at = ? WHERE id = ?",
		progress, now, planID,
	)
}

// isValidPlanType 校验计划类型
func isValidPlanType(t string) bool {
	switch t {
	case "weekly", "monthly", "quarterly", "semester", "yearly", "four_year":
		return true
	}
	return false
}

// isValidPlanStatus 校验计划状态
func isValidPlanStatus(s string) bool {
	switch s {
	case "active", "completed", "archived":
		return true
	}
	return false
}

// isValidTaskStatus 校验任务状态
func isValidTaskStatus(s string) bool {
	switch s {
	case "pending", "in_progress", "done", "skipped":
		return true
	}
	return false
}
