package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 大学规划
// ══════════════════════════════════════════════════════════════

// ListPlanTemplates 获取规划模板列表
// GET /api/v1/plan/templates?category=
func (h *StudentFeaturesHandler) ListPlanTemplates(c *gin.Context) {
	category := c.Query("category")
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListPlanTemplates(category)
	if err != nil {
		log.Printf("查询规划模板失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// ListMyPlans 获取我的规划列表
// GET /api/v1/plan/my-plans
func (h *StudentFeaturesHandler) ListMyPlans(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListMyPlans(userCtx.UserID)
	if err != nil {
		log.Printf("查询我的规划失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// CreatePlan 创建规划
// POST /api/v1/plan/create
func (h *StudentFeaturesHandler) CreatePlan(c *gin.Context) {
	var req struct {
		TemplateID   int    `json:"template_id"`
		Title        string `json:"title" binding:"required"`
		Category     string `json:"category" binding:"required"`
		AcademicYear int    `json:"academic_year"`
		Semester     int    `json:"semester"`
		Goals        string `json:"goals"`
		Content      string `json:"content"` // 前端兼容字段
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("plan CreatePlan bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	goals := req.Goals
	if goals == "" && req.Content != "" {
		goals = req.Content
	}
	id, err := h.svc.CreatePlan(userCtx.UserID, req.TemplateID, req.Title, req.Category, req.AcademicYear, req.Semester, goals)
	if err != nil {
		log.Printf("创建规划失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "规划创建成功", "data": gin.H{"id": id}})
}

// SubmitPlan 提交规划审核
// PUT /api/v1/plan/:id/submit
func (h *StudentFeaturesHandler) SubmitPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if err := h.svc.SubmitPlan(id); err != nil {
		log.Printf("提交规划失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "提交失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已提交审核"})
}

// ReviewPlan 审核规划（管理员）
// PUT /api/v1/plan/:id/review
func (h *StudentFeaturesHandler) ReviewPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status  string `json:"status" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "counselor" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Code: 403, Message: "无权操作"})
		return
	}
	if err := h.svc.ReviewPlan(id, req.Status, req.Comment); err != nil {
		log.Printf("审核规划失败: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "审核失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审核完成"})
}
