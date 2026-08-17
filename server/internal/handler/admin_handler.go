package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
	"github.com/gin-gonic/gin"
)

// AdminHandler 管理端 HTTP handler
type AdminHandler struct {
	adminSvc *service.AdminService
	authSvc  *service.AuthService
}

// NewAdminHandler 创建管理端 handler
func NewAdminHandler(adminSvc *service.AdminService, authSvc *service.AuthService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, authSvc: authSvc}
}

// GetPublicFeatureSwitches 公开功能开关（登录用户可读）
// GET /api/v1/public/feature-switches
// 返回管理员配置的全局功能开关（feature.* 前缀），用于前端控制模块显示。
func (h *AdminHandler) GetPublicFeatureSwitches(c *gin.Context) {
	switches, err := h.adminSvc.GetFeatureSwitches()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取功能开关失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": switches})
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

// TopFallbackQuestions 高频兜底问题清单（知识治理：命中失败高的问题应补录知识库）
// GET /api/v1/admin/metrics/fallback-questions?days=7&top=20
func (h *AdminHandler) TopFallbackQuestions(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	top, _ := strconv.Atoi(c.DefaultQuery("top", "20"))
	list, err := h.adminSvc.TopFallbackQuestions(days, top)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取高频兜底问题失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// ListUsers 用户列表 GET /api/v1/admin/users?role=&owner_scope=&owner_id=&page=&page_size=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	ownerScope := c.Query("owner_scope")
	ownerID := c.Query("owner_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

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

	// 隐私脱敏：非辅导员/管理员角色不返回联系方式与出生年月
	for _, u := range users {
		u.SanitizePrivate(userCtx.Role)
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
		log.Printf("admin UpdateUser bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
		})
		return
	}

	if req.DisplayName == nil && req.Role == nil && req.OwnerScope == nil && req.OwnerID == nil && req.Status == nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "至少需要修改一项用户信息",
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

	user, err := h.adminSvc.UpdateUser(userID, &req, userCtx)
	if err != nil {
		log.Printf("更新用户失败 user_id=%d: %v", userID, err)
		util.FailBadRequest(c, "更新用户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户信息已更新",
		"data":    user,
	})
}

// DeleteUser 删除用户 DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "用户 ID 格式错误",
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

	// 不允许删除自己
	if userCtx.UserID == userID {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "不允许删除当前登录账户",
		})
		return
	}

	if err := h.adminSvc.DeleteUser(userID, userCtx.Username); err != nil {
		log.Printf("删除用户失败 user_id=%d: %v", userID, err)
		util.FailBadRequest(c, "删除用户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户已删除",
	})
}

// ListUsersAdvanced 高级用户查询 GET /api/v1/admin/users/advanced
// ?keyword=&role=&owner_scope=&owner_id=&college=&major=&class_name=&enrollment_year=&status=&sort_by=&sort_order=&page=&page_size=
func (h *AdminHandler) ListUsersAdvanced(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	ownerScope := c.Query("owner_scope")
	ownerID := c.Query("owner_id")
	if userCtx.Role == "college_admin" {
		ownerScope = userCtx.OwnerScope
		ownerID = userCtx.OwnerID
	}

	q := &model.UserQuery{
		Keyword:        c.Query("keyword"),
		Role:           c.Query("role"),
		OwnerScope:     ownerScope,
		OwnerID:        ownerID,
		College:        c.Query("college"),
		Major:          c.Query("major"),
		ClassName:      c.Query("class_name"),
		EnrollmentYear: c.Query("enrollment_year"),
		Status:         c.Query("status"),
		SortBy:         c.DefaultQuery("sort_by", "id"),
		SortOrder:      c.DefaultQuery("sort_order", "asc"),
		Offset:         offset,
		Limit:          pageSize,
	}

	users, total, err := h.adminSvc.ListUsersAdvanced(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询用户列表失败",
		})
		return
	}

	// 隐私脱敏：非辅导员/管理员角色不返回联系方式与出生年月
	for _, u := range users {
		u.SanitizePrivate(userCtx.Role)
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

// GetUserDict 获取用户字典值 GET /api/v1/admin/users/dict?column=college
func (h *AdminHandler) GetUserDict(c *gin.Context) {
	column := c.Query("column")
	role := c.Query("role")

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	ownerScope := ""
	ownerID := ""
	if userCtx.Role == "college_admin" {
		ownerScope = userCtx.OwnerScope
		ownerID = userCtx.OwnerID
	}

	values, err := h.adminSvc.GetUserDictValues(column, role, ownerScope, ownerID)
	if err != nil {
		log.Printf("获取字典值失败 column=%s: %v", column, err)
		util.FailBadRequest(c, "获取字典值失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    values,
	})
}

// BatchUpdateStatus 批量更新用户状态 POST /api/v1/admin/users/batch/status
func (h *AdminHandler) BatchUpdateStatus(c *gin.Context) {
	var req struct {
		Ids    []int64 `json:"ids" binding:"required,min=1"`
		Status string  `json:"status" binding:"required,oneof=active disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin BatchUpdateStatus bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
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

	count, err := h.adminSvc.BatchUpdateStatus(req.Ids, req.Status, userCtx.Username)
	if err != nil {
		log.Printf("批量更新用户状态失败: %v", err)
		util.FailBadRequest(c, "批量更新状态失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data":    gin.H{"affected": count},
	})
}

// BatchResetPassword 批量重置密码 POST /api/v1/admin/users/batch/password
func (h *AdminHandler) BatchResetPassword(c *gin.Context) {
	var req struct {
		Ids         []int64 `json:"ids" binding:"required,min=1"`
		NewPassword string  `json:"new_password" binding:"required,min=6,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin BatchResetPassword bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
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

	count, err := h.adminSvc.BatchResetPassword(req.Ids, req.NewPassword, userCtx.Username)
	if err != nil {
		log.Printf("批量重置密码失败: %v", err)
		util.FailBadRequest(c, "批量重置密码失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码重置成功",
		"data":    gin.H{"affected": count},
	})
}

// BatchDelete 批量删除用户 POST /api/v1/admin/users/batch/delete
func (h *AdminHandler) BatchDelete(c *gin.Context) {
	var req struct {
		Ids []int64 `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("admin BatchDelete bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数错误",
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

	// 检查是否包含自己
	for _, id := range req.Ids {
		if id == userCtx.UserID {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Code:    400,
				Message: "不允许删除当前登录账户",
			})
			return
		}
	}

	count, err := h.adminSvc.BatchDelete(req.Ids, userCtx.Username)
	if err != nil {
		log.Printf("批量删除用户失败: %v", err)
		util.FailBadRequest(c, "批量删除用户失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data":    gin.H{"affected": count},
	})
}
