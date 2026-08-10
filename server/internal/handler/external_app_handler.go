package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ExternalAppHandler 第三方应用中心 HTTP handler
type ExternalAppHandler struct {
	appSvc *service.ExternalAppService
}

// NewExternalAppHandler 创建外部应用 handler
func NewExternalAppHandler(appSvc *service.ExternalAppService) *ExternalAppHandler {
	return &ExternalAppHandler{appSvc: appSvc}
}

// ListVisible 应用中心列表 GET /api/v1/apps（按当前用户角色过滤）
func (h *ExternalAppHandler) ListVisible(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}
	views, err := h.appSvc.ListForUser(userCtx.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询应用中心失败",
		})
		return
	}
	if views == nil {
		views = []model.ExternalAppView{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": views})
}

// ListAdmin 管理端应用列表 GET /api/v1/admin/apps
func (h *ExternalAppHandler) ListAdmin(c *gin.Context) {
	views, err := h.appSvc.ListAdmin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询应用列表失败",
		})
		return
	}
	if views == nil {
		views = []model.ExternalAppAdminView{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": views})
}

// Create 注册应用 POST /api/v1/admin/apps
func (h *ExternalAppHandler) Create(c *gin.Context) {
	var req model.ExternalAppCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败：manifest 必填",
		})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}
	view, err := h.appSvc.Create(req.Manifest, req.Enabled, userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "注册成功", "data": view})
}

// Update 更新应用 PUT /api/v1/admin/apps/:id
func (h *ExternalAppHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.ExternalAppCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
		})
		return
	}
	view, err := h.appSvc.Update(id, req.Manifest, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": view})
}

// Delete 删除应用 DELETE /api/v1/admin/apps/:id
func (h *ExternalAppHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.appSvc.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}