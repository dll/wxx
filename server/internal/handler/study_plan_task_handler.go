package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// 学习任务 + AI 生成计划 handler（从 study_plan_handler.go 按业务域拆分；SQL 已下沉 StudyPlanRepo）
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

	if _, err := h.studyPlanRepo.GetPlanByID(planID, userCtx.UserID); err == repository.ErrPlanNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

	var req CreatePlanTaskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("study_plan AddTask bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "title 不能为空"})
		return
	}

	taskID, err := h.studyPlanRepo.CreateTask(planID, req.CourseID, req.CourseName, req.Title, req.Description, req.ScheduledDate, req.ScheduledDuration, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "添加任务失败"})
		return
	}

	// 添加任务后，刷新计划进度
	h.studyPlanRepo.RecalcPlanProgress(planID)

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
	if _, err := h.studyPlanRepo.GetPlanByID(planID, userCtx.UserID); err == repository.ErrPlanNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "学习计划不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询学习计划失败"})
		return
	}

	var req struct {
		Title             string `json:"title"`
		Description       string `json:"description"`
		ScheduledDate     string `json:"scheduled_date"`
		ScheduledDuration *int   `json:"scheduled_duration"`
		ActualDuration    *int   `json:"actual_duration"`
		Status            string `json:"status"`
		Evidence          string `json:"evidence"`
		Reflection        string `json:"reflection"`
		SortOrder         *int   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("study_plan UpdateTask bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}
	if req.Status != "" && !isValidTaskStatus(req.Status) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "status 取值无效"})
		return
	}

	fields := repository.TaskUpdateFields{
		Title:             req.Title,
		Description:       req.Description,
		ScheduledDate:     req.ScheduledDate,
		ScheduledDuration: req.ScheduledDuration,
		ActualDuration:    req.ActualDuration,
		Status:            req.Status,
		Evidence:          req.Evidence,
		Reflection:        req.Reflection,
		SortOrder:         req.SortOrder,
	}
	affected, err := h.studyPlanRepo.UpdateTask(taskID, planID, fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新任务失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "任务不存在"})
		return
	}

	// 任务状态变更后，重新计算计划进度
	h.studyPlanRepo.RecalcPlanProgress(planID)

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
	overview, err := h.studyPlanRepo.PlansOverview(userCtx.UserID, planTypes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询计划概览失败"})
		return
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
	PlanType     string   `json:"plan_type"`     // 默认 weekly
	SemesterCode string   `json:"semester_code"` // 默认当前学期
	StartDate    string   `json:"start_date"`    // 可选，默认今天
	EndDate      string   `json:"end_date"`      // 可选，默认按 plan_type 推算
	Goals        []string `json:"goals"`         // 用户指定目标
	FocusCourses []string `json:"focus_courses"` // 关注的课程（可选）
}

// AIGeneratePlan AI生成学习计划
// POST /api/v1/study/plans/ai-generate
func (h *StudyPlanHandler) AIGeneratePlan(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	if !h.studyPlanSvc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{Code: 503, Message: "LLM 客户端未配置，AI 生成计划不可用"})
		return
	}

	var req AIGeneratePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		log.Printf("study_plan GeneratePlan bind err: %v", err)
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

	// 日期范围：未传则按 plan_type 推算
	today := time.Now()
	if req.StartDate == "" {
		req.StartDate = today.Format("2006-01-02")
	}
	if req.EndDate == "" {
		req.EndDate = calcDefaultEndDate(req.PlanType, today)
	}

	result, err := h.studyPlanSvc.AIGeneratePlan(
		c.Request.Context(),
		userCtx.UserID,
		req.PlanType,
		req.SemesterCode,
		req.StartDate,
		req.EndDate,
		req.Goals,
		req.FocusCourses,
	)
	if err != nil {
		log.Printf("study_plan GeneratePlan err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "生成学习计划失败"})
		return
	}

	plan, _ := h.studyPlanRepo.GetPlanByID(result.PlanID, userCtx.UserID)
	if plan != nil {
		tasks, _ := h.studyPlanRepo.ListTasksByPlan(result.PlanID)
		plan.Tasks = tasks
		h.studyPlanRepo.FillPlanTaskStats(plan)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"message":       "AI 学习计划生成成功",
		"data":          plan,
		"llm_provider":  result.LLMProvider,
		"prompt_tokens": result.PromptTokens,
		"output_tokens": result.OutputTokens,
	})
}
