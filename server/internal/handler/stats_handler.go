package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// StatsHandler 统计数据 HTTP handler（P4-d：查询逻辑已下沉 repository.StatsRepo）
type StatsHandler struct {
	statsRepo *repository.StatsRepo
}

// NewStatsHandler 创建统计 handler
func NewStatsHandler(statsRepo *repository.StatsRepo) *StatsHandler {
	return &StatsHandler{statsRepo: statsRepo}
}

// GetDashboardStats 获取仪表盘统计数据
// GET /api/v1/admin/stats/dashboard
func (h *StatsHandler) GetDashboardStats(c *gin.Context) {
	if middleware.GetUserContext(c) == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	stats := &model.DashboardStats{}

	if users, err := h.statsRepo.GetUserStats(); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取用户统计失败"})
		return
	} else {
		stats.Users = *users
	}

	if knowledge, err := h.statsRepo.GetKnowledgeStats(); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取知识库统计失败"})
		return
	} else {
		stats.Knowledge = *knowledge
	}

	if chat, err := h.statsRepo.GetChatStats(); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取对话统计失败"})
		return
	} else {
		stats.Chat = *chat
	}

	if feedback, err := h.statsRepo.GetFeedbackStats(); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取反馈统计失败"})
		return
	} else {
		stats.Feedback = *feedback
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// RequireAdminStatsRead 检查是否有仪表盘读取权限（college_admin 及以上）
func RequireAdminStatsRead() gin.HandlerFunc {
	return auth.RequireCapability(auth.CollegeMetricsRead)
}
