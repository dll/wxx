package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// EmotionHandler 情感预警 HTTP handler
type EmotionHandler struct {
	emotionSvc *service.EmotionService
}

// NewEmotionHandler 创建情感预警 handler
func NewEmotionHandler(emotionSvc *service.EmotionService) *EmotionHandler {
	return &EmotionHandler{emotionSvc: emotionSvc}
}

// Analyze 分析消息情感
// POST /api/v1/emotion/analyze
func (h *EmotionHandler) Analyze(c *gin.Context) {
	var req model.EmotionAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("emotion Analyze bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
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

	logEntry, err := h.emotionSvc.AnalyzeAndLog(
		c.Request.Context(),
		userCtx.UserID,
		userCtx.Username,
		req.SessionID,
		req.MessageText,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "情感分析服务暂不可用，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.EmotionAnalyzeResponse{
		Code:    0,
		Message: "分析完成",
		Data:    logEntry,
	})
}

// ListAlerts 告警列表（辅导员及以上角色）
// GET /api/v1/emotion/alerts?risk_level=high&status=pending&page=1&page_size=20
func (h *EmotionHandler) ListAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	riskLevel := c.Query("risk_level")
	status := c.Query("status")

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	alerts, total, err := h.emotionSvc.ListAlerts(
		riskLevel, status,
		userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role,
		page, pageSize,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询告警列表失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.EmotionListResponse{
		Code:    0,
		Message: "success",
		Data:    alerts,
		Total:   total,
	})
}

// GetStats 获取告警统计
// GET /api/v1/emotion/stats
func (h *EmotionHandler) GetStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	stats, err := h.emotionSvc.GetStats(userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询统计失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.EmotionStatsResponse{
		Code:    0,
		Message: "success",
		Data:    stats,
	})
}

// Trends 获取情感趋势数据
// GET /api/v1/emotion/trends?days=30
func (h *EmotionHandler) Trends(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	report, err := h.emotionSvc.GetTrendReport(days, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取趋势数据失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.EmotionTrendResponse{
		Code:    0,
		Message: "success",
		Data:    report,
	})
}

// UpdateAlert 更新告警状态（确认/已处理）
// PUT /api/v1/emotion/alerts/:id
func (h *EmotionHandler) UpdateAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "告警 ID 不能为空",
		})
		return
	}

	var req model.EmotionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("emotion Update bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败",
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

	// P0-05：更新带用户上下文范围复核，防跨学院越权写/越权读原始文本
	logEntry, err := h.emotionSvc.UpdateAlertStatus(userCtx, alertID, req.Status)
	if err != nil {
		if errors.Is(err, model.ErrAlertNotFound) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Code:    404,
				Message: "告警不存在或无权访问",
				TraceID: middleware.GetTraceID(c),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "更新告警状态失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, model.EmotionAnalyzeResponse{
		Code:    0,
		Message: "更新成功",
		Data:    logEntry,
	})
}
