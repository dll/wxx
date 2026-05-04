package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证 handler
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler 创建认证 handler
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// loginRequest 登录请求（开发环境简化版）
type loginRequest struct {
	Username string `json:"username" binding:"required"`
}

// Login 开发环境简化登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	result, err := h.authSvc.LoginByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "登录失败：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "登录成功",
		"data":    result,
	})
}

// Profile 获取当前用户信息
// GET /api/v1/user/profile
func (h *AuthHandler) Profile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"user_id":      userCtx.UserID,
			"username":     userCtx.Username,
			"display_name": userCtx.DisplayName,
			"role":         userCtx.Role,
			"owner_scope":  userCtx.OwnerScope,
			"owner_id":     userCtx.OwnerID,
		},
	})
}
