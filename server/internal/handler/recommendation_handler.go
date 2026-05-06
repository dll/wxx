package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	recSvc *service.RecommendationService
}

func NewRecommendationHandler(recSvc *service.RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{recSvc: recSvc}
}

// GetRecommendations 获取个性化推荐
// GET /api/v1/recommendations?limit=10
func (h *RecommendationHandler) GetRecommendations(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	result, err := h.recSvc.GetRecommendations(userCtx, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取推荐内容失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result.Items,
		"total":   result.Total,
	})
}
