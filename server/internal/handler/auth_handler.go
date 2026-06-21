package handler

import (
	"errors"
	"net/http"

	"github.com/dll/wxx/server/internal/auth"
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
	Role     string `json:"role"`     // 可选，新用户创建时的角色，默认 "student"
	Password string `json:"password"` // 可选，密码（未设置密码的用户可留空）
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

	result, err := h.authSvc.LoginByUsername(req.Username, req.Role, req.Password)
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
			"consented":    userCtx.Consented,
		},
	})
}

// Consent 记录用户同意隐私政策与用户协议
// POST /api/v1/user/consent
func (h *AuthHandler) Consent(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	// 更新用户同意状态
	if err := h.authSvc.RecordConsent(userCtx.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "记录同意状态失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "授权已记录",
	})
}

// changePasswordRequest 修改密码请求
type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 用户自助修改密码
// PUT /api/v1/user/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.authSvc.ChangePassword(userCtx.UserID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码修改成功",
	})
}

// GetVoiceConfig 获取用户语音开关配置
// GET /api/v1/user/voice-config
func (h *AuthHandler) GetVoiceConfig(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	enabled, err := h.authSvc.GetVoiceConfig(userCtx.UserID)
	if err != nil {
		// 用户不存在（Vercel 冷启动后 DB 重建场景）→ 返回默认 0，不当成错误
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data":    model.VoiceConfigResponse{VoiceEnabled: 0},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    model.VoiceConfigResponse{VoiceEnabled: enabled},
	})
}

// UpdateVoiceConfig 更新用户语音开关
// PUT /api/v1/user/voice-config
func (h *AuthHandler) UpdateVoiceConfig(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	var req model.VoiceConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.authSvc.UpdateVoiceConfig(userCtx.UserID, req.VoiceEnabled); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "语音配置已更新",
	})
}

// sendCodeRequest 发送验证码请求
type sendCodeRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// guestRegisterRequest 游客注册请求
type guestRegisterRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

// SendCode 发送短信验证码
// POST /api/v1/auth/send-code
func (h *AuthHandler) SendCode(c *gin.Context) {
	var req sendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	code, err := h.authSvc.SendCode(req.Phone)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "验证码已发送",
		"data":    gin.H{"code": code},
	})
}

// GuestRegister 游客注册
// POST /api/v1/auth/guest-register
func (h *AuthHandler) GuestRegister(c *gin.Context) {
	var req guestRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	result, err := h.authSvc.GuestRegister(req.DisplayName, req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "游客注册失败：" + err.Error(),
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "欢迎来到滁州学院！注册成功，请等待管理员审核",
		"data":    result,
	})
}

// GetCapabilities 获取当前用户拥有的能力列表（含继承）
// GET /api/v1/user/capabilities
// 前端登录后拉取一次，缓存用于菜单/按钮可见性判断
func (h *AuthHandler) GetCapabilities(c *gin.Context) {
	user := middleware.GetUserContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	caps := auth.CapabilitiesOf(user.Role)
	// 转 string 切片便于 JSON 序列化
	strCaps := make([]string, len(caps))
	for i, c := range caps {
		strCaps[i] = string(c)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"role":         user.Role,
			"capabilities": strCaps,
		},
	})
}
