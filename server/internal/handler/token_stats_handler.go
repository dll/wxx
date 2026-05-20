package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// TokenStatsHandler 词元统计 handler
type TokenStatsHandler struct {
	tokenSvc *service.TokenStatsService
}

// NewTokenStatsHandler 创建词元统计 handler
func NewTokenStatsHandler(tokenSvc *service.TokenStatsService) *TokenStatsHandler {
	return &TokenStatsHandler{tokenSvc: tokenSvc}
}

// GetMyStats 获取当前用户词元统计 GET /api/v1/token-stats/my?days=30
func (h *TokenStatsHandler) GetMyStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	data, err := h.tokenSvc.GetMyStats(userCtx.UserID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取词元统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.TokenStatsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// GetSubordinateStats 获取下级用户词元统计 GET /api/v1/token-stats/subordinates?days=30
func (h *TokenStatsHandler) GetSubordinateStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	data, err := h.tokenSvc.GetSubordinateStats(userCtx, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取下级词元统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, model.SubordinateTokenStatsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}
