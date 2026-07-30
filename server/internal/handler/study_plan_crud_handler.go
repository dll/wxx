package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 学习计划 CRUD handler（从 study_plan_handler.go 按业务域拆分）
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
