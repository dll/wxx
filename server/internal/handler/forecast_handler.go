package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// ForecastHandler 问题预案 HTTP handler
type ForecastHandler struct {
	forecastSvc *service.ForecastService
}

// NewForecastHandler 创建问题预案 handler
func NewForecastHandler(forecastSvc *service.ForecastService) *ForecastHandler {
	return &ForecastHandler{forecastSvc: forecastSvc}
}

// AnalysisRequest 分析请求
type AnalysisRequest struct {
	CollegeID    string   `json:"college_id"`
	TimeRange    string   `json:"time_range"`
	AnalysisType string   `json:"analysis_type"`
	DataSources  []string `json:"data_sources"`
}

// AnalysisResponse 分析响应
type AnalysisResponse struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    *service.AnalysisResponse `json:"data,omitempty"`
}

// Analyze 执行问题分析
// POST /api/v1/forecast/analysis
func (h *ForecastHandler) Analyze(c *gin.Context) {
	var req AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
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

	// 权限检查：仅 sys_admin、college_admin 可访问
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "school_admin" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "无权访问问题预案功能",
		})
		return
	}

	// 如果是 college_admin，只能查看本学院数据
	if userCtx.Role == "college_admin" && req.CollegeID == "" {
		req.CollegeID = userCtx.OwnerScope
	}

	// 设置默认时间范围
	if req.TimeRange == "" {
		req.TimeRange = "last_30_days"
	}

	// 设置默认分析类型
	if req.AnalysisType == "" {
		req.AnalysisType = "comprehensive"
	}

	// 转换为 service 请求
	serviceReq := &service.AnalysisRequest{
		CollegeID:    req.CollegeID,
		TimeRange:    req.TimeRange,
		AnalysisType: req.AnalysisType,
		DataSources:  req.DataSources,
	}

	// 执行分析
	result, err := h.forecastSvc.Analyze(serviceReq, userCtx.UserID, userCtx.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "分析失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AnalysisResponse{
		Code:    0,
		Message: "分析完成",
		Data:    result,
	})
}

// ListForecasts 问题预案列表
// GET /api/v1/forecast/issues?category=emotion&risk_level=high&status=pending&page=1&page_size=20
func (h *ForecastHandler) ListForecasts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")
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

	// 权限检查
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "school_admin" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "无权访问问题预案功能",
		})
		return
	}

	// college_admin 只能查看本学院数据
	collegeID := ""
	if userCtx.Role == "college_admin" {
		collegeID = userCtx.OwnerScope
	}

	forecasts, total, err := h.forecastSvc.ListForecasts(collegeID, category, riskLevel, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"data":      forecasts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetForecast 获取问题预案详情
// GET /api/v1/forecast/issues/:id
func (h *ForecastHandler) GetForecast(c *gin.Context) {
	forecastID := c.Param("id")

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	// 权限检查
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "school_admin" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "无权访问问题预案功能",
		})
		return
	}

	forecast, err := h.forecastSvc.GetForecast(forecastID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    404,
			Message: "问题预案不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    forecast,
	})
}

// UpdateStatus 更新问题预案状态
// PUT /api/v1/forecast/issues/:id/status
func (h *ForecastHandler) UpdateStatus(c *gin.Context) {
	forecastID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "参数校验失败: " + err.Error(),
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

	// 权限检查
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "school_admin" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "无权更新问题预案状态",
		})
		return
	}

	// 验证状态值
	validStatuses := map[string]bool{
		"pending":    true,
		"processing": true,
		"resolved":   true,
		"archived":   true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "无效的状态值",
		})
		return
	}

	err := h.forecastSvc.UpdateStatus(forecastID, req.Status, userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "更新状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "状态更新成功",
	})
}

// GetStatistics 获取问题预案统计
// GET /api/v1/forecast/statistics?days=30
func (h *ForecastHandler) GetStatistics(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	// 权限检查
	if userCtx.Role != "sys_admin" && userCtx.Role != "college_admin" && userCtx.Role != "school_admin" {
		c.JSON(http.StatusForbidden, model.ErrorResponse{
			Code:    403,
			Message: "无权访问问题预案统计",
		})
		return
	}

	// college_admin 只能查看本学院数据
	collegeID := ""
	if userCtx.Role == "college_admin" {
		collegeID = userCtx.OwnerScope
	}

	// 获取风险分布
	riskDistribution, err := h.forecastSvc.GetRiskDistribution(collegeID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取风险分布失败: " + err.Error(),
		})
		return
	}

	// 获取分类分布
	categoryDistribution, err := h.forecastSvc.GetCategoryDistribution(collegeID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取分类分布失败: " + err.Error(),
		})
		return
	}

	// 获取每日趋势
	dailyTrend, err := h.forecastSvc.GetDailyTrend(collegeID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取每日趋势失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"risk_distribution":     riskDistribution,
			"category_distribution": categoryDistribution,
			"daily_trend":           dailyTrend,
		},
	})
}
