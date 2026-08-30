package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// 学习计划 CRUD handler（从 study_plan_handler.go 按业务域拆分；SQL 已下沉 StudyPlanRepo）
func (h *StudyPlanHandler) ListMyPlans(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	list, err := h.studyPlanRepo.ListPlansByUser(userCtx.UserID, c.Query("plan_type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

	// 批量获取任务统计
	for _, plan := range list {
		h.studyPlanRepo.FillPlanTaskStats(plan)
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

	plan, err := h.studyPlanRepo.GetPlanByID(id, userCtx.UserID)
	if err == repository.ErrPlanNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

	tasks, err := h.studyPlanRepo.ListTasksByPlan(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询计划任务失败"})
		return
	}
	plan.Tasks = tasks
	h.studyPlanRepo.FillPlanTaskStats(plan)

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
		log.Printf("study_plan CreatePlan bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}

	if req.PlanType == "" {
		req.PlanType = "weekly"
	}
	if !isValidPlanType(req.PlanType) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "plan_type 取值无效"})
		return
	}

	goals := req.Goals
	if goals == nil {
		goals = []string{}
	}
	goalsJSON, _ := json.Marshal(goals)

	planID, err := h.studyPlanRepo.CreatePlan(userCtx.UserID, req.Title, req.PlanType, req.SemesterCode, req.StartDate, req.EndDate, string(goalsJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "创建学习计划失败"})
		return
	}

	// 批量插入任务
	for _, t := range req.Tasks {
		if t == nil {
			continue
		}
		if _, err := h.studyPlanRepo.CreateTask(planID, t.CourseID, t.CourseName, t.Title, t.Description, t.ScheduledDate, t.ScheduledDuration, t.SortOrder); err != nil {
			log.Printf("[WARN] 计划任务创建失败 plan_id=%d: %v", planID, err)
		}
	}

	plan, _ := h.studyPlanRepo.GetPlanByID(planID, userCtx.UserID)
	if plan != nil {
		tasks, _ := h.studyPlanRepo.ListTasksByPlan(planID)
		plan.Tasks = tasks
		h.studyPlanRepo.FillPlanTaskStats(plan)
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
	if _, err := h.studyPlanRepo.GetPlanByID(id, userCtx.UserID); err == repository.ErrPlanNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

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
		log.Printf("study_plan UpdatePlan bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
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

	goalsJSON := ""
	if req.Goals != nil {
		b, _ := json.Marshal(req.Goals)
		goalsJSON = string(b)
	}

	fields := repository.PlanUpdateFields{
		Title:        req.Title,
		PlanType:     req.PlanType,
		SemesterCode: req.SemesterCode,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		GoalsJSON:    goalsJSON,
		Status:       req.Status,
		Progress:     req.Progress,
	}
	if err := h.studyPlanRepo.UpdatePlan(id, userCtx.UserID, fields); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新学习计划失败"})
		return
	}

	plan, _ := h.studyPlanRepo.GetPlanByID(id, userCtx.UserID)
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

	affected, err := h.studyPlanRepo.DeletePlan(id, userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除学习计划失败"})
		return
	}
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
