package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 学习任务 + AI 生成计划 handler（从 study_plan_handler.go 按业务域拆分）
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

	plan, _ := h.getPlanByID(result.PlanID, userCtx.UserID)
	if plan != nil {
		tasks, _ := h.listTasksByPlan(result.PlanID)
		plan.Tasks = tasks
		h.fillPlanTaskStats(plan)
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

// ═══════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════

// resolveCurrentCalendar 解析当前学期校历与教学周
// 优先返回 start_date <= today <= end_date 的学期；若不在任何学期内（如暑假），返回最近的已完成或即将开始的学期
