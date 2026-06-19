package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// GraduationHandler 毕设选题 HTTP handler
type GraduationHandler struct {
	graduationRepo *repository.GraduationRepo
}

// NewGraduationHandler 创建毕设选题 handler
func NewGraduationHandler(graduationRepo *repository.GraduationRepo) *GraduationHandler {
	return &GraduationHandler{graduationRepo: graduationRepo}
}

// ListAdvisors 获取导师列表
// GET /api/v1/graduation/advisors?college=&page=&page_size=
func (h *GraduationHandler) ListAdvisors(c *gin.Context) {
	college := c.Query("college")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	// college_admin 只能查看本院导师
	if userCtx.Role == "college_admin" && college == "" {
		college = userCtx.OwnerScope
	}

	advisors, total, err := h.graduationRepo.ListAdvisors(college, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "success",
		"data": advisors, "total": total, "page": page, "page_size": pageSize,
	})
}

// ListTopics 获取选题列表
// GET /api/v1/graduation/topics?college=&major=&difficulty=&status=&batch=&page=&page_size=
func (h *GraduationHandler) ListTopics(c *gin.Context) {
	college := c.Query("college")
	major := c.Query("major")
	difficulty := c.Query("difficulty")
	status := c.Query("status")
	batch, _ := strconv.Atoi(c.DefaultQuery("batch", "2026"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	// college_admin 只能查看本院选题
	if userCtx.Role == "college_admin" && college == "" {
		college = userCtx.OwnerScope
	}

	topics, total, err := h.graduationRepo.ListTopics(college, major, difficulty, status, batch, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "success",
		"data": topics, "total": total, "page": page, "page_size": pageSize,
	})
}

// GetTopic 获取选题详情
// GET /api/v1/graduation/topics/:id
func (h *GraduationHandler) GetTopic(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	topic, err := h.graduationRepo.GetTopic(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "选题不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": topic})
}

// GetMySelection 获取我的选题
// GET /api/v1/graduation/my-selection
func (h *GraduationHandler) GetMySelection(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	selection, err := h.graduationRepo.GetUserSelection(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "暂无选题记录", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": selection})
}

// SelectTopic 学生选题
// POST /api/v1/graduation/select
func (h *GraduationHandler) SelectTopic(c *gin.Context) {
	var req struct {
		TopicID int64  `json:"topic_id" binding:"required"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	// 检查是否已有选题记录
	existing, _ := h.graduationRepo.GetUserSelection(userCtx.UserID)
	if existing != nil && existing.Status != "changed" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "已有选题记录，如需改题请联系导师"})
		return
	}

	// 获取选题信息
	topic, err := h.graduationRepo.GetTopic(req.TopicID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "选题不存在"})
		return
	}

	// 检查选题是否已满
	if topic.SelectedCount >= topic.MaxStudents {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "该选题已满，请选择其他题目"})
		return
	}

	selection := &model.StudentTopicSelection{
		UserID:          userCtx.UserID,
		StudentID:       userCtx.Username,
		StudentName:     userCtx.DisplayName,
		College:         userCtx.OwnerScope,
		Batch:           2026,
		TopicID:         req.TopicID,
		AdvisorID:       topic.AdvisorID,
		Status:          "pending",
		PreferenceOrder: 1,
		Reason:          req.Reason,
	}

	id, err := h.graduationRepo.CreateSelection(selection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "选题失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "选题成功", "data": gin.H{"id": id}})
}

// ListMilestones 获取里程碑列表
// GET /api/v1/graduation/milestones?batch=2026
func (h *GraduationHandler) ListMilestones(c *gin.Context) {
	batch, _ := strconv.Atoi(c.DefaultQuery("batch", "2026"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	milestones, err := h.graduationRepo.ListMilestones(batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": milestones})
}

// GetStats 获取选题统计
// GET /api/v1/graduation/stats?batch=2026
func (h *GraduationHandler) GetStats(c *gin.Context) {
	batch, _ := strconv.Atoi(c.DefaultQuery("batch", "2026"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	stats, err := h.graduationRepo.GetTopicStats(batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询统计失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}

// ListSelections 获取选题记录列表（管理员）
// GET /api/v1/graduation/selections?topic_id=&batch=&page=&page_size=
func (h *GraduationHandler) ListSelections(c *gin.Context) {
	topicID, _ := strconv.ParseInt(c.Query("topic_id"), 10, 64)
	batch, _ := strconv.Atoi(c.DefaultQuery("batch", "2026"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	// 权限检查
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "counselor" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: 403, Message: "无权访问"})
		return
	}

	selections, total, err := h.graduationRepo.ListSelections(topicID, batch, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "success",
		"data": selections, "total": total, "page": page, "page_size": pageSize,
	})
}

// ConfirmSelection 确认选题（管理员）
// PUT /api/v1/graduation/selections/:id/confirm
func (h *GraduationHandler) ConfirmSelection(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	// 权限检查
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "counselor" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: 403, Message: "无权操作"})
		return
	}

	err := h.graduationRepo.UpdateSelectionStatus(id, "confirmed")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "确认失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "选题已确认"})
}
