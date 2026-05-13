package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// AdminHandler 管理端 HTTP handler
type AdminHandler struct {
	adminSvc *service.AdminService
}

// NewAdminHandler 创建管理端 handler
func NewAdminHandler(adminSvc *service.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc}
}

// GetMetrics 质量看板 GET /api/v1/admin/metrics
func (h *AdminHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.adminSvc.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取质量看板数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.AdminMetricsResponse{
		Code:    0,
		Message: "success",
		Data:    metrics,
	})
}

// ListUsers 用户列表 GET /api/v1/admin/users?role=&owner_scope=&owner_id=&page=&page_size=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	ownerScope := c.Query("owner_scope")
	ownerID := c.Query("owner_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	// 范围过滤：college_admin 只能看本院用户
	queryScope := ownerScope
	queryOwnerID := ownerID
	if userCtx.Role == "college_admin" {
		queryScope = userCtx.OwnerScope
		queryOwnerID = userCtx.OwnerID
	}

	users, total, err := h.adminSvc.ListUsers(role, queryScope, queryOwnerID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询用户列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.UserListResponse{
		Code:     0,
		Message:  "success",
		Data:     users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// UpdateUser 修改用户 PUT /api/v1/admin/users/:id
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
		})
		return
	}

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
		})
		return
	}

	if req.Role == nil && req.OwnerScope == nil && req.OwnerID == nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "至少需要修改一项 (role/owner_scope/owner_id)",
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

	user, err := h.adminSvc.UpdateUser(userID, &req, userCtx.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户信息已更新",
		"data":    user,
	})
}

// ListAudit 审计日志列表 GET /api/v1/admin/audit?username=&action=&resource=&start_date=&end_date=&page=&page_size=
func (h *AdminHandler) ListAudit(c *gin.Context) {
	username := c.Query("username")
	action := c.Query("action")
	resource := c.Query("resource")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.adminSvc.ListAudit(username, action, resource, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询审计日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.AuditListResponse{
		Code:     0,
		Message:  "success",
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetSettings 获取系统配置 GET /api/v1/admin/settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.adminSvc.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取系统配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.SettingsResponse{
		Code:    0,
		Message: "success",
		Data:    settings,
	})
}

// UpdateSettings 更新系统配置 PUT /api/v1/admin/settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req model.SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
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

	if err := h.adminSvc.UpdateSettings(req.Settings, userCtx.Username); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "更新系统配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "系统配置已更新",
	})
}
