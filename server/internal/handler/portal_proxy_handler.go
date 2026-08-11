package handler

import (
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// PortalProxyHandler 学校门户页面代理 handler
type PortalProxyHandler struct {
	proxySvc *service.PortalProxyService
}

// NewPortalProxyHandler 创建门户代理 handler
func NewPortalProxyHandler(proxySvc *service.PortalProxyService) *PortalProxyHandler {
	return &PortalProxyHandler{proxySvc: proxySvc}
}

// Proxy 代理访问校内页面
// GET /api/v1/user/portal/*path   （path 为门户站点路径，如 /home /edu/course）
func (h *PortalProxyHandler) Proxy(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	path := strings.TrimPrefix(c.Request.URL.Path, "/api/v1/user/portal")
	if path == "" {
		path = "/"
	}

	status, respHeader, body, err := h.proxySvc.Proxy(userCtx.UserID, path, c.Request.URL.Query(), c.Request.Header)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "未绑定") {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: msg})
			return
		}
		if strings.Contains(msg, "登录") {
			c.JSON(http.StatusBadGateway, model.ErrorResponse{Code: 502, Message: msg})
			return
		}
		c.JSON(http.StatusBadGateway, model.ErrorResponse{Code: 502, Message: "门户暂不可用，请稍后重试"})
		return
	}

	// 透传 Content-Type 等
	ct := respHeader.Get("Content-Type")
	if ct == "" {
		ct = "text/html; charset=utf-8"
	}
	c.Data(status, ct, body)
}
