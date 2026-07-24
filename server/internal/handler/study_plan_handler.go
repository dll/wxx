package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// StudyPlanHandler 学习计划与校历 HTTP handler
// 直接依赖 *sql.DB（沿用 education_handler.go 的风格），同时注入可选的 LLM 客户端用于 AI 生成计划
type StudyPlanHandler struct {
	db        *sql.DB
	llmClient llm.ChatClient
}

// NewStudyPlanHandler 创建学习计划 handler
func NewStudyPlanHandler(db *sql.DB, llmClient llm.ChatClient) *StudyPlanHandler {
	return &StudyPlanHandler{db: db, llmClient: llmClient}
}

// ═══════════════════════════════════════════════
// 数据结构定义
// ═══════════════════════════════════════════════

// AcademicCalendar 学期校历
type AcademicCalendar struct {
	ID            int64  `json:"id"`
	AcademicYear  int    `json:"academic_year"`
	Semester      int    `json:"semester"`
	SemesterCode  string `json:"semester_code"`
	SemesterName  string `json:"semester_name"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	RegisterDate  string `json:"register_date,omitempty"`
	TotalWeeks    int    `json:"total_weeks"`
	WeekStartDay  string `json:"week_start_day"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
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
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	CourseID      string `json:"course_id"`
	CourseName    string `json:"course_name"`
	SemesterCode  string `json:"semester_code"`
	Weekday       int    `json:"weekday"`
	StartPeriod   int    `json:"start_period"`
	EndPeriod     int    `json:"end_period"`
	WeeksPattern  string `json:"weeks_pattern"`
	Location      string `json:"location,omitempty"`
	Teacher       string `json:"teacher,omitempty"`
	Color         string `json:"color"`
	CreatedAt     string `json:"created_at"`
}

// StudyPlan 学习计划
type StudyPlan struct {
	ID            int64           `json:"id"`
	UserID        int64           `json:"user_id"`
	Title         string          `json:"title"`
	PlanType      string          `json:"plan_type"`
	SemesterCode  string          `json:"semester_code,omitempty"`
	StartDate     string          `json:"start_date"`
	EndDate       string          `json:"end_date"`
	Goals         []string        `json:"goals"`
	Progress      float64         `json:"progress"`
	AIGenerated   bool            `json:"ai_generated"`
	Status        string          `json:"status"`
	LinkedPlanID  *int64          `json:"linked_plan_id,omitempty"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
	// 任务统计（列表场景）
	TaskTotal     int             `json:"task_total,omitempty"`
	TaskDone      int             `json:"task_done,omitempty"`
	TaskPending   int             `json:"task_pending,omitempty"`
	Tasks         []*StudyPlanTask `json:"tasks,omitempty"`
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
	Progress  float64 `json:"progress"`  // 平均进度
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
func (h *StudyPlanHandler) ListMyPlans(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	planType := c.Query("plan_type")

	var where []string
	var args []interface{}
	where = append(where, "user_id = ?")
	args = append(args, userCtx.UserID)
	if planType != "" {
		where = append(where, "plan_type = ?")
		args = append(args, planType)
	}
	whereSQL := strings.Join(where, " AND ")

	rows, err := h.db.Query(
		"SELECT id, user_id, title, plan_type, semester_code, start_date, end_date, goals_json, "+
			"progress, ai_generated, status, linked_plan_id, created_at, updated_at "+
			"FROM study_plans WHERE "+whereSQL+" ORDER BY created_at DESC, id DESC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}
	defer rows.Close()

	var list []*StudyPlan
	for rows.Next() {
		plan, err := scanStudyPlan(rows)
		if err != nil {
			continue
		}
		list = append(list, plan)
	}

	// 批量获取任务统计
	for _, plan := range list {
		h.fillPlanTaskStats(plan)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// GetPlan 获取计划详情（含所有任务）
// GET /api/v1/study/plans/:id
func (h *StudyPlanHandler) GetPlan(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "计划ID无效"})
		return
	}

	plan, err := h.getPlanByID(id, userCtx.UserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

	// 查询计划任务
	tasks, err := h.listTasksByPlan(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询计划任务失败"})
		return
	}
	plan.Tasks = tasks
	h.fillPlanTaskStats(plan)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    plan,
	})
}

// CreatePlanRequest 创建学习计划请求体
type CreatePlanRequest struct {
	Title        string                 `json:"title" binding:"required"`
	PlanType     string                 `json:"plan_type"`
	SemesterCode string                 `json:"semester_code"`
	StartDate    string                 `json:"start_date" binding:"required"`
	EndDate      string                 `json:"end_date" binding:"required"`
	Goals        []string               `json:"goals"`
	Tasks        []*CreatePlanTaskInput `json:"tasks"`
}

// CreatePlanTaskInput 创建计划任务输入
type CreatePlanTaskInput struct {
	CourseID          string `json:"course_id"`
	CourseName        string `json:"course_name"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	ScheduledDate     string `json:"scheduled_date"`
	ScheduledDuration int    `json:"scheduled_duration"`
	SortOrder         int    `json:"sort_order"`
}

// CreatePlan 创建学习计划
// POST /api/v1/study/plans
func (h *StudyPlanHandler) CreatePlan(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}

	if req.PlanType == "" {
		req.PlanType = "weekly"
	}
	// 校验 plan_type 合法性
	if !isValidPlanType(req.PlanType) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "plan_type 取值无效"})
		return
	}

	goals := req.Goals
	if goals == nil {
		goals = []string{}
	}
	goalsJSON, _ := json.Marshal(goals)
	now := time.Now().Format("2006-01-02 15:04:05")
	var semesterCode sql.NullString
	if req.SemesterCode != "" {
		semesterCode = sql.NullString{String: req.SemesterCode, Valid: true}
	}

	res, err := h.db.Exec(
		"INSERT INTO study_plans (user_id, title, plan_type, semester_code, start_date, end_date, goals_json, progress, ai_generated, status, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, 'active', ?, ?)",
		userCtx.UserID, req.Title, req.PlanType, semesterCode, req.StartDate, req.EndDate,
		string(goalsJSON), now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "创建学习计划失败"})
		return
	}
	planID, _ := res.LastInsertId()

	// 批量插入任务
	for _, t := range req.Tasks {
		if t == nil {
			continue
		}
		_, _ = h.db.Exec(
			"INSERT INTO study_plan_tasks (plan_id, course_id, course_name, title, description, scheduled_date, scheduled_duration, actual_duration, status, sort_order, created_at) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 'pending', ?, ?)",
			planID, t.CourseID, t.CourseName, t.Title, t.Description, t.ScheduledDate,
			t.ScheduledDuration, t.SortOrder, now,
		)
	}

	plan, _ := h.getPlanByID(planID, userCtx.UserID)
	if plan != nil {
		tasks, _ := h.listTasksByPlan(planID)
		plan.Tasks = tasks
		h.fillPlanTaskStats(plan)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "学习计划创建成功",
		"data":    plan,
	})
}

// UpdatePlan 更新学习计划
// PUT /api/v1/study/plans/:id
func (h *StudyPlanHandler) UpdatePlan(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "计划ID无效"})
		return
	}

	// 校验所属权
	existing, err := h.getPlanByID(id, userCtx.UserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}
	_ = existing

	var req struct {
		Title        string   `json:"title"`
		PlanType     string   `json:"plan_type"`
		SemesterCode string   `json:"semester_code"`
		StartDate    string   `json:"start_date"`
		EndDate      string   `json:"end_date"`
		Goals        []string `json:"goals"`
		Status       string   `json:"status"`
		Progress     *float64 `json:"progress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}

	if req.PlanType != "" && !isValidPlanType(req.PlanType) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "plan_type 取值无效"})
		return
	}
	if req.Status != "" && !isValidPlanStatus(req.Status) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "status 取值无效"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	goalsJSON := ""
	if req.Goals != nil {
		b, _ := json.Marshal(req.Goals)
		goalsJSON = string(b)
	}

	var progress sql.NullFloat64
	if req.Progress != nil {
		progress = sql.NullFloat64{Float64: *req.Progress, Valid: true}
	}

	_, err = h.db.Exec(
		"UPDATE study_plans SET title = COALESCE(NULLIF(?, ''), title), "+
			"plan_type = COALESCE(NULLIF(?, ''), plan_type), "+
			"semester_code = CASE WHEN ? <> '' THEN ? ELSE semester_code END, "+
			"start_date = COALESCE(NULLIF(?, ''), start_date), "+
			"end_date = COALESCE(NULLIF(?, ''), end_date), "+
			"goals_json = CASE WHEN ? <> '' THEN ? ELSE goals_json END, "+
			"status = COALESCE(NULLIF(?, ''), status), "+
			"progress = CASE WHEN ? IS NOT NULL THEN ? ELSE progress END, "+
			"updated_at = ? WHERE id = ? AND user_id = ?",
		req.Title, req.PlanType,
		req.SemesterCode, req.SemesterCode,
		req.StartDate, req.EndDate,
		goalsJSON, goalsJSON,
		req.Status,
		progress, progress,
		now, id, userCtx.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新学习计划失败"})
		return
	}

	plan, _ := h.getPlanByID(id, userCtx.UserID)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "学习计划更新成功",
		"data":    plan,
	})
}

// DeletePlan 删除学习计划（连带任务通过外键 ON DELETE CASCADE 删除）
// DELETE /api/v1/study/plans/:id
func (h *StudyPlanHandler) DeletePlan(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "计划ID无效"})
		return
	}

	res, err := h.db.Exec("DELETE FROM study_plans WHERE id = ? AND user_id = ?", id, userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除学习计划失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在或无权删除"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "学习计划已删除",
		"id":      id,
	})
}

// AddTask 添加任务到指定计划
// POST /api/v1/study/plans/:id/tasks
func (h *StudyPlanHandler) AddTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || planID <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "计划ID无效"})
		return
	}

	existing, err := h.getPlanByID(planID, userCtx.UserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}
	_ = existing

	var req CreatePlanTaskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "title 不能为空"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := h.db.Exec(
		"INSERT INTO study_plan_tasks (plan_id, course_id, course_name, title, description, scheduled_date, scheduled_duration, actual_duration, status, sort_order, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 'pending', ?, ?)",
		planID, req.CourseID, req.CourseName, req.Title, req.Description, req.ScheduledDate,
		req.ScheduledDuration, req.SortOrder, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "添加任务失败"})
		return
	}
	taskID, _ := res.LastInsertId()

	// 添加任务后，刷新计划进度
	h.recalcPlanProgress(planID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "任务添加成功",
		"data": gin.H{
			"id":             taskID,
			"plan_id":        planID,
			"title":          req.Title,
			"course_id":      req.CourseID,
			"course_name":    req.CourseName,
			"scheduled_date": req.ScheduledDate,
			"status":         "pending",
			"sort_order":     req.SortOrder,
			"created_at":     now,
		},
	})
}

// UpdateTask 更新任务（含状态变更打卡）
// PUT /api/v1/study/plans/:id/tasks/:task_id
func (h *StudyPlanHandler) UpdateTask(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || planID <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "计划ID无效"})
		return
	}
	taskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "任务ID无效"})
		return
	}

	// 校验计划归属
	if _, err := h.getPlanByID(planID, userCtx.UserID); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

	var req struct {
		Title             string  `json:"title"`
		Description       string  `json:"description"`
		ScheduledDate     string  `json:"scheduled_date"`
		ScheduledDuration *int    `json:"scheduled_duration"`
		ActualDuration    *int    `json:"actual_duration"`
		Status            string  `json:"status"`
		Evidence          string  `json:"evidence"`
		Reflection        string  `json:"reflection"`
		SortOrder         *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}
	if req.Status != "" && !isValidTaskStatus(req.Status) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "status 取值无效"})
		return
	}

	var scheduledDuration, actualDuration, sortOrder sql.NullInt64
	if req.ScheduledDuration != nil {
		scheduledDuration = sql.NullInt64{Int64: int64(*req.ScheduledDuration), Valid: true}
	}
	if req.ActualDuration != nil {
		actualDuration = sql.NullInt64{Int64: int64(*req.ActualDuration), Valid: true}
	}
	if req.SortOrder != nil {
		sortOrder = sql.NullInt64{Int64: int64(*req.SortOrder), Valid: true}
	}

	res, err := h.db.Exec(
		"UPDATE study_plan_tasks SET "+
			"title = COALESCE(NULLIF(?, ''), title), "+
			"description = COALESCE(NULLIF(?, ''), description), "+
			"scheduled_date = COALESCE(NULLIF(?, ''), scheduled_date), "+
			"scheduled_duration = CASE WHEN ? IS NOT NULL THEN ? ELSE scheduled_duration END, "+
			"actual_duration = CASE WHEN ? IS NOT NULL THEN ? ELSE actual_duration END, "+
			"status = COALESCE(NULLIF(?, ''), status), "+
			"evidence = COALESCE(NULLIF(?, ''), evidence), "+
			"reflection = COALESCE(NULLIF(?, ''), reflection), "+
			"sort_order = CASE WHEN ? IS NOT NULL THEN ? ELSE sort_order END "+
			"WHERE id = ? AND plan_id = ?",
		req.Title, req.Description, req.ScheduledDate,
		scheduledDuration, scheduledDuration,
		actualDuration, actualDuration,
		req.Status,
		req.Evidence, req.Reflection,
		sortOrder, sortOrder,
		taskID, planID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新任务失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "任务不存在"})
		return
	}

	// 任务状态变更后，重新计算计划进度
	h.recalcPlanProgress(planID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "任务更新成功",
		"plan_id": planID,
		"task_id": taskID,
	})
}

// ═══════════════════════════════════════════════
// 四、学习计划概览接口 /api/v1/study/plans/overview
// ═══════════════════════════════════════════════

// GetPlansOverview 获取各时间维度计划概览（用于多Tab首页）
// GET /api/v1/study/plans/overview
func (h *StudyPlanHandler) GetPlansOverview(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	planTypes := []string{"weekly", "monthly", "quarterly", "semester", "yearly", "four_year"}
	overview := make([]*PlanOverviewItem, 0, len(planTypes))

	for _, pt := range planTypes {
		item := &PlanOverviewItem{PlanType: pt}
		// 计划数与平均进度
		_ = h.db.QueryRow(
			"SELECT COUNT(*), COALESCE(AVG(progress), 0) FROM study_plans WHERE user_id = ? AND plan_type = ?",
			userCtx.UserID, pt,
		).Scan(&item.PlanCount, &item.Progress)

		// 任务统计：通过 JOIN 关联到该用户该类型的所有计划
		var done int
		_ = h.db.QueryRow(
			"SELECT COUNT(*), COALESCE(SUM(CASE WHEN t.status = 'done' THEN 1 ELSE 0 END), 0) "+
				"FROM study_plan_tasks t JOIN study_plans p ON t.plan_id = p.id "+
				"WHERE p.user_id = ? AND p.plan_type = ?",
			userCtx.UserID, pt,
		).Scan(&item.TaskTotal, &done)
		item.TaskDone = done

		if item.TaskDone > 0 && item.TaskTotal > 0 {
			// 任务完成进度优先于平均进度
			item.Progress = float64(item.TaskDone) / float64(item.TaskTotal) * 100
		}
		overview = append(overview, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    overview,
	})
}

// ═══════════════════════════════════════════════
// 五、AI 生成学习计划
// ═══════════════════════════════════════════════

// AIGeneratePlanRequest AI生成学习计划请求
type AIGeneratePlanRequest struct {
	PlanType     string `json:"plan_type"`      // 默认 weekly
	SemesterCode string `json:"semester_code"`  // 默认当前学期
	StartDate    string `json:"start_date"`     // 可选，默认今天
	EndDate      string `json:"end_date"`       // 可选，默认按 plan_type 推算
	Goals        []string `json:"goals"`        // 用户指定目标
	FocusCourses []string `json:"focus_courses"` // 关注的课程（可选）
}

// aiGeneratedPlanSchema LLM 返回的 JSON 计划结构
type aiGeneratedPlanSchema struct {
	Title  string `json:"title"`
	Goals  []string `json:"goals"`
	Tasks  []struct {
		CourseID          string `json:"course_id"`
		CourseName        string `json:"course_name"`
		Title             string `json:"title"`
		Description       string `json:"description"`
		ScheduledDate     string `json:"scheduled_date"`
		ScheduledDuration int    `json:"scheduled_duration"`
		SortOrder         int    `json:"sort_order"`
	} `json:"tasks"`
}

// AIGeneratePlan AI生成学习计划
// POST /api/v1/study/plans/ai-generate
func (h *StudyPlanHandler) AIGeneratePlan(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	if h.llmClient == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Code: 503, Message: "LLM 客户端未配置，AI 生成计划不可用"})
		return
	}

	var req AIGeneratePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}
	if req.PlanType == "" {
		req.PlanType = "weekly"
	}
	if !isValidPlanType(req.PlanType) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "plan_type 取值无效"})
		return
	}

	// 收集上下文：当前学期、教学周、课表、考试事件
	calendar, currentWeek, err := h.resolveCurrentCalendar()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询当前校历失败: " + err.Error()})
		return
	}
	if req.SemesterCode == "" && calendar != nil {
		req.SemesterCode = calendar.SemesterCode
	}

	// 日期范围：未传则按 plan_type 推算
	today := time.Now()
	if req.StartDate == "" {
		req.StartDate = today.Format("2006-01-02")
	}
	if req.EndDate == "" {
		req.EndDate = calcDefaultEndDate(req.PlanType, today)
	}

	timetable, _ := h.listUserTimetable(userCtx.UserID, req.SemesterCode)
	upcomingExams := h.listUpcomingExams(req.SemesterCode)

	// 构建 Prompt
	prompt := h.buildAIGeneratePrompt(&req, calendar, currentWeek, timetable, upcomingExams)

	// 调用 LLM
	resp, err := h.llmClient.Chat(context.Background(), &llm.ChatRequest{
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
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "LLM 调用失败: " + err.Error()})
		return
	}

	// 解析 JSON
	planSchema, err := parseAIGeneratedPlan(resp.Content)
	if err != nil {
		log.Printf("解析 AI 学习计划失败: %v, 原始内容: %s", err, resp.Content)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "解析 AI 生成的计划失败"})
		return
	}

	// 持久化到 study_plans + study_plan_tasks
	goals := planSchema.Goals
	if len(goals) == 0 {
		goals = req.Goals
	}
	if goals == nil {
		goals = []string{}
	}
	goalsJSON, _ := json.Marshal(goals)

	title := planSchema.Title
	if title == "" {
		title = "AI 生成 " + req.PlanType + " 计划"
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	var semesterCode sql.NullString
	if req.SemesterCode != "" {
		semesterCode = sql.NullString{String: req.SemesterCode, Valid: true}
	}

	res, err := h.db.Exec(
		"INSERT INTO study_plans (user_id, title, plan_type, semester_code, start_date, end_date, goals_json, progress, ai_generated, status, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 1, 'active', ?, ?)",
		userCtx.UserID, title, req.PlanType, semesterCode, req.StartDate, req.EndDate,
		string(goalsJSON), now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存 AI 计划失败"})
		return
	}
	planID, _ := res.LastInsertId()

	for _, t := range planSchema.Tasks {
		_, _ = h.db.Exec(
			"INSERT INTO study_plan_tasks (plan_id, course_id, course_name, title, description, scheduled_date, scheduled_duration, actual_duration, status, sort_order, created_at) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 'pending', ?, ?)",
			planID, t.CourseID, t.CourseName, t.Title, t.Description, t.ScheduledDate,
			t.ScheduledDuration, t.SortOrder, now,
		)
	}

	plan, _ := h.getPlanByID(planID, userCtx.UserID)
	if plan != nil {
		tasks, _ := h.listTasksByPlan(planID)
		plan.Tasks = tasks
		h.fillPlanTaskStats(plan)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"message":       "AI 学习计划生成成功",
		"data":          plan,
		"llm_provider":  h.llmClient.Name(),
		"prompt_tokens": resp.PromptTokens,
		"output_tokens": resp.OutputTokens,
	})
}

// buildAIGeneratePrompt 构建 AI 生成学习计划的 Prompt
func (h *StudyPlanHandler) buildAIGeneratePrompt(req *AIGeneratePlanRequest, cal *AcademicCalendar, currentWeek int, timetable []*CourseScheduleItem, exams []*CalendarEvent) string {
	var sb strings.Builder
	sb.WriteString("请根据以下信息生成一份" + req.PlanType + "学习计划：\n\n")

	sb.WriteString(fmt.Sprintf("计划类型：%s\n", req.PlanType))
	sb.WriteString(fmt.Sprintf("开始日期：%s\n", req.StartDate))
	sb.WriteString(fmt.Sprintf("结束日期：%s\n", req.EndDate))
	if cal != nil {
		sb.WriteString(fmt.Sprintf("当前学期：%s（%s ~ %s）\n", cal.SemesterName, cal.StartDate, cal.EndDate))
	}
	if currentWeek > 0 {
		sb.WriteString(fmt.Sprintf("当前教学周：第 %d 周\n", currentWeek))
	}

	if len(req.Goals) > 0 {
		sb.WriteString("用户目标：\n")
		for i, g := range req.Goals {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, g))
		}
	}
	if len(req.FocusCourses) > 0 {
		sb.WriteString("重点关注课程：" + strings.Join(req.FocusCourses, "、") + "\n")
	}

	// 课表概要
	if len(timetable) > 0 {
		sb.WriteString("\n学生本周课表：\n")
		for _, t := range timetable {
			sb.WriteString(fmt.Sprintf("  周%d 第%d-%d节 %s（%s, %s）周次:%s\n",
				t.Weekday, t.StartPeriod, t.EndPeriod, t.CourseName, t.Teacher, t.Location, t.WeeksPattern))
		}
	}

	// 考试事件
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

// ═══════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════

// resolveCurrentCalendar 解析当前学期校历与教学周
// 优先返回 start_date <= today <= end_date 的学期；若不在任何学期内（如暑假），返回最近的已完成或即将开始的学期
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

// listUserTimetable 查询学生课表
func (h *StudyPlanHandler) listUserTimetable(userID int64, semesterCode string) ([]*CourseScheduleItem, error) {
	if semesterCode == "" {
		return nil, nil
	}
	rows, err := h.db.Query(
		"SELECT id, user_id, course_id, course_name, semester_code, weekday, start_period, end_period, "+
			"weeks_pattern, location, teacher, color, created_at "+
			"FROM course_schedules WHERE user_id = ? AND semester_code = ? ORDER BY weekday ASC, start_period ASC, id ASC",
		userID, semesterCode,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CourseScheduleItem
	for rows.Next() {
		item := &CourseScheduleItem{}
		var location, teacher sql.NullString
		if err := rows.Scan(&item.ID, &item.UserID, &item.CourseID, &item.CourseName,
			&item.SemesterCode, &item.Weekday, &item.StartPeriod, &item.EndPeriod,
			&item.WeeksPattern, &location, &teacher, &item.Color, &item.CreatedAt); err != nil {
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
	return list, nil
}

// listUpcomingExams 查询学期内的考试事件
func (h *StudyPlanHandler) listUpcomingExams(semesterCode string) []*CalendarEvent {
	if semesterCode == "" {
		return nil
	}
	rows, err := h.db.Query(
		"SELECT id, semester_code, event_name, event_type, start_date, end_date, week_no, affects_classes, description, created_at "+
			"FROM academic_calendar_events WHERE semester_code = ? AND event_type IN ('exam','deadline') "+
			"ORDER BY start_date ASC, id ASC",
		semesterCode,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanCalendarEvents(rows)
}

// parseAIGeneratedPlan 解析 LLM 返回的 JSON 计划（容忍代码块标记）
func parseAIGeneratedPlan(content string) (*aiGeneratedPlanSchema, error) {
	s := strings.TrimSpace(content)
	// 去除可能的 markdown 代码块
	if strings.HasPrefix(s, "```") {
		// 移除首行 ``` 或 ```json
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = s[:len(s)-3]
		}
		s = strings.TrimSpace(s)
	}
	// 提取首个 { 到最后一个 }
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
