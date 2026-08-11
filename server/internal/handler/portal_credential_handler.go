package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// PortalCredentialHandler 学校门户凭证 HTTP handler
type PortalCredentialHandler struct {
	svc *service.PortalCredentialService
}

// NewPortalCredentialHandler 创建门户凭证 handler
func NewPortalCredentialHandler(svc *service.PortalCredentialService) *PortalCredentialHandler {
	return &PortalCredentialHandler{svc: svc}
}

// Get 查询绑定状态 GET /api/v1/user/portal-credential
func (h *PortalCredentialHandler) Get(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	view, err := h.svc.Get(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询门户绑定状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": view})
}

// Save 保存凭证 PUT /api/v1/user/portal-credential
// body: {"portal_url":"https://my0.chzu.edu.cn/","portal_account":"...","portal_password":"..."}
func (h *PortalCredentialHandler) Save(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	var req struct {
		PortalURL     string `json:"portal_url"`
		PortalAccount string `json:"portal_account"`
		PortalPassword string `json:"portal_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	if err := h.svc.Save(userCtx.UserID, req.PortalURL, req.PortalAccount, req.PortalPassword); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	// 审计：记录保存/更新门户绑定（不含密码）
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已保存，门户凭证加密存储，仅你本人可见"})
}

// Delete 清除凭证 DELETE /api/v1/user/portal-credential
func (h *PortalCredentialHandler) Delete(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if err := h.svc.Delete(userCtx.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "清除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已清除"})
}
