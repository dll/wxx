package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/util"
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

// loginRequest 账号密码登录请求。
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type consentRequest struct {
	PolicyVersion string `json:"policy_version"`
	Purpose       string `json:"purpose"`
	Vendor        string `json:"vendor"`
	Source        string `json:"source"`
}

// Login 账号密码登录。
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("auth Login bind err: %v", err)
		util.FailBadRequest(c, "请求参数错误")
		return
	}

	result, err := h.authSvc.LoginByUsername(req.Username, "", req.Password)
	if err != nil {
		message := "登录失败，请稍后重试"
		if errors.Is(err, service.ErrInvalidCredentials) {
			util.FailUnauthorized(c, "账号或密码错误")
			return
		} else if errors.Is(err, service.ErrAccountUnavailable) {
			log.Printf("账号不可用: %v", err)
			util.FailForbidden(c, "该账号已被禁用，请联系管理员")
			return
		}
		log.Printf("登录失败: %v", err)
		util.FailInternalError(c, message)
		return
	}

	util.SuccessWithMessage(c, result, "登录成功")
}

type ssoCallbackRequest struct {
	Ticket string `json:"ticket" binding:"required"`
}

// SSOCallback 统一身份认证票据换取 JWT。
// POST /api/v1/auth/sso/callback
func (h *AuthHandler) SSOCallback(c *gin.Context) {
	var req ssoCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.FailBadRequest(c, "请求参数错误")
		return
	}
	result, err := h.authSvc.LoginBySSOTicket(c.Request.Context(), req.Ticket)
	if err != nil {
		if errors.Is(err, service.ErrSSONotConfigured) {
			c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
				Code:    503,
				Message: "统一身份认证未配置，请联系管理员",
				TraceID: middleware.GetTraceID(c),
			})
			return
		}
		log.Printf("SSO 登录失败: %v", err)
		util.FailUnauthorized(c, "SSO 登录失败，请重试")
		return
	}
	util.SuccessWithMessage(c, result, "SSO 登录成功")
}

// Profile 获取当前用户信息
// GET /api/v1/user/profile
func (h *AuthHandler) Profile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}

	data := gin.H{
		"user_id":      userCtx.UserID,
		"username":     userCtx.Username,
		"display_name": userCtx.DisplayName,
		"role":         userCtx.Role,
		"owner_scope":  userCtx.OwnerScope,
		"owner_id":     userCtx.OwnerID,
		"consented":    userCtx.Consented,
	}

	// 补充学院/专业/入学年份等学业字段（用于前端年级主题自动切换等）
	if user, err := h.authSvc.GetProfile(userCtx.UserID); err == nil && user != nil {
		data["college"] = user.College
		data["major"] = user.Major
		data["class_name"] = user.ClassName
		data["enrollment_date"] = user.EnrollmentDate
		data["enrollment_year"] = user.EnrollmentYear
		data["status"] = user.Status
	}

	util.Success(c, data)
}

// ProfileDetail 个人详细信息（基本信息 + 联系方式 + 组织关系）
// GET /api/v1/user/profile/detail
func (h *AuthHandler) ProfileDetail(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}
	detail, err := h.authSvc.GetProfileDetail(userCtx.UserID)
	if err != nil {
		if err == service.ErrUserNotFound {
			util.FailUnauthorized(c, "用户不存在")
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询个人信息失败"})
		return
	}
	util.Success(c, detail)
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

	var req consentRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			util.FailBadRequest(c, "授权参数格式错误")
			return
		}
	}
	if err := h.authSvc.RecordConsent(userCtx.UserID, req.PolicyVersion, req.Purpose, req.Vendor, req.Source, middleware.GetTraceID(c)); err != nil {
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
		log.Printf("auth ChangePassword bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.authSvc.ChangePassword(userCtx.UserID, req.OldPassword, req.NewPassword); err != nil {
		log.Printf("修改密码失败 user_id=%d: %v", userCtx.UserID, err)
		util.FailBadRequest(c, "密码修改失败，请检查原密码是否正确")
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
		log.Printf("获取语音配置失败 user_id=%d: %v", userCtx.UserID, err)
		util.FailInternalError(c, "获取语音配置失败")
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
		log.Printf("auth UpdateVoiceConfig bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	if err := h.authSvc.UpdateVoiceConfig(userCtx.UserID, req.VoiceEnabled); err != nil {
		log.Printf("更新语音配置失败 user_id=%d: %v", userCtx.UserID, err)
		util.FailBadRequest(c, "更新语音配置失败")
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
	// 预研期游客注册默认关闭（无短信通道）；需开放时设 ENABLE_GUEST_REGISTER=true
	if !h.authSvc.GuestRegisterEnabled() {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "暂未开放注册，请联系管理员获取账号",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	var req sendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("auth SendCode bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	code, err := h.authSvc.SendCode(req.Phone)
	if err != nil {
		log.Printf("发送验证码失败 phone=%s: %v", req.Phone, err)
		util.FailBadRequest(c, "发送验证码失败")
		return
	}

	// 安全约束：生产环境绝不在响应中回显验证码，仅通过真实短信通道下发。
	// 仅在 debug 模式下回显，便于本地联调。
	data := gin.H{}
	if h.authSvc.DebugCodeEcho() {
		data["code"] = code
		data["debug_note"] = "验证码仅在调试模式回显，生产环境不返回"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "验证码已发送",
		"data":    data,
	})
}

// GuestRegister 游客注册
// POST /api/v1/auth/guest-register
func (h *AuthHandler) GuestRegister(c *gin.Context) {
	// 预研期游客注册默认关闭（无短信通道）；需开放时设 ENABLE_GUEST_REGISTER=true
	if !h.authSvc.GuestRegisterEnabled() {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "暂未开放注册，请联系管理员获取账号",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	var req guestRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("auth GuestRegister bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "请求参数错误",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	result, err := h.authSvc.GuestRegister(req.DisplayName, req.Phone, req.Code)
	if err != nil {
		log.Printf("游客注册失败 phone=%s: %v", req.Phone, err)
		util.FailInternalError(c, "注册失败，请稍后重试")
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

	// 多角色（2026-09-01）：能力取全部角色的并集；单角色用户 Roles 为空时回退主角色
	caps := auth.CapabilitiesOfAny(user.Roles)
	if len(user.Roles) == 0 {
		caps = auth.CapabilitiesOf(user.Role)
	}
	// 转 string 切片便于 JSON 序列化
	strCaps := make([]string, len(caps))
	for i, c := range caps {
		strCaps[i] = string(c)
	}

	roles := user.Roles
	if roles == nil {
		roles = []string{user.Role}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"role":         user.Role,
			"roles":        roles,
			"capabilities": strCaps,
		},
	})
}

// GetAIKey 获取当前用户 AI Key 绑定状态 GET /api/v1/user/ai-key
func (h *AuthHandler) GetAIKey(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}
	info, err := h.authSvc.GetAIKeyInfo(userCtx.UserID)
	if err != nil {
		util.FailInternalError(c, "查询 AI Key 失败")
		return
	}
	util.Success(c, info)
}

// SaveAIKey 保存用户自备 AI Key PUT /api/v1/user/ai-key
func (h *AuthHandler) SaveAIKey(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}
	var req struct {
		Provider string `json:"provider"`
		ApiKey   string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.FailBadRequest(c, "参数错误")
		return
	}
	if err := h.authSvc.SaveAIKey(userCtx.UserID, req.Provider, req.ApiKey); err != nil {
		util.FailBadRequest(c, err.Error())
		return
	}
	util.Success(c, gin.H{"bound": true})
}

// ClearAIKey 清除用户自备 AI Key DELETE /api/v1/user/ai-key
func (h *AuthHandler) ClearAIKey(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		util.FailUnauthorized(c, "未认证")
		return
	}
	if err := h.authSvc.ClearAIKey(userCtx.UserID); err != nil {
		util.FailInternalError(c, "清除 AI Key 失败")
		return
	}
	util.Success(c, gin.H{"bound": false})
}
