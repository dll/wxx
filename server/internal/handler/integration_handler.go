package handler

import (
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// IntegrationHandler 校外系统对接 HTTP handler（只读代理）
type IntegrationHandler struct {
	integrationSvc *service.IntegrationService
}

// NewIntegrationHandler 创建对接 handler
func NewIntegrationHandler(integrationSvc *service.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{integrationSvc: integrationSvc}
}

// ProxyXuegong 代理学工系统查询
// GET /api/v1/integration/xuegong/*path
func (h *IntegrationHandler) ProxyXuegong(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	// 获取路径（不含前缀）
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/v1/integration/xuegong")
	if path == "" {
		path = "/"
	}

	// 收集查询参数
	query := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	data, err := h.integrationSvc.ProxyXuegong(path, query)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.ErrorResponse{
			Code:    502,
			Message: "外部系统暂不可用，请稍后重试",
		})
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

// ProxyYBT 代理一表通查询
// GET /api/v1/integration/ybt/*path
func (h *IntegrationHandler) ProxyYBT(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	// 获取路径（不含前缀）
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/v1/integration/ybt")
	if path == "" {
		path = "/"
	}

	// 收集查询参数
	query := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	data, err := h.integrationSvc.ProxyYBT(path, query)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.ErrorResponse{
			Code:    502,
			Message: "外部系统暂不可用，请稍后重试",
		})
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

// Status 返回对接系统状态
// GET /api/v1/integration/status
func (h *IntegrationHandler) Status(c *gin.Context) {
	xgAvailable := h.integrationSvc.IsXuegongAvailable()
	ybtAvailable := h.integrationSvc.IsYBTAvailable()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"xuegong": gin.H{
				"available": xgAvailable,
				"label":     "学工系统",
			},
			"ybt": gin.H{
				"available": ybtAvailable,
				"label":     "一表通",
			},
		},
	})
}
