package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// CampusHandler 校园报到步骤 HTTP handler
type CampusHandler struct {
	repo *repository.CampusRepository
}

// NewCampusHandler 构造函数
func NewCampusHandler(repo *repository.CampusRepository) *CampusHandler {
	return &CampusHandler{repo: repo}
}

// ListPublicSteps GET /api/v1/campus/steps?campus=huifeng
// 公开接口，返回已发布步骤
func (h *CampusHandler) ListPublicSteps(c *gin.Context) {
	campus := c.DefaultQuery("campus", "huifeng")
	steps, err := h.repo.ListPublished(campus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	if steps == nil {
		steps = []model.CampusStep{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": steps})
}

// ListAdminSteps GET /api/v1/admin/campus/steps?campus=huifeng
// 管理端，返回全部步骤（含草稿）
func (h *CampusHandler) ListAdminSteps(c *gin.Context) {
	campus := c.DefaultQuery("campus", "huifeng")
	steps, err := h.repo.ListAll(campus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	if steps == nil {
		steps = []model.CampusStep{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": steps})
}

// CreateStep POST /api/v1/admin/campus/steps
func (h *CampusHandler) CreateStep(c *gin.Context) {
	var req model.CampusStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数校验失败: " + err.Error()})
		return
	}
	user := middleware.GetUserContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	id, err := h.repo.Create(&req, user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": id}})
}

// UpdateStep PUT /api/v1/admin/campus/steps/:id
func (h *CampusHandler) UpdateStep(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的步骤 ID"})
		return
	}
	var req model.CampusStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数校验失败: " + err.Error()})
		return
	}
	if err := h.repo.Update(id, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// SubmitStep POST /api/v1/admin/campus/steps/:id/submit（draft → pending_review）
func (h *CampusHandler) SubmitStep(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的步骤 ID"})
		return
	}
	if err := h.repo.Submit(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已提交审核"})
}

// PublishStep POST /api/v1/admin/campus/steps/:id/publish（pending_review → published）
func (h *CampusHandler) PublishStep(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的步骤 ID"})
		return
	}
	user := middleware.GetUserContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	if err := h.repo.Publish(id, user.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发布成功"})
}

// DeleteStep DELETE /api/v1/admin/campus/steps/:id（仅 draft）
func (h *CampusHandler) DeleteStep(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的步骤 ID"})
		return
	}
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
