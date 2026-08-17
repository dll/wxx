package handler

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 入党教育
// ══════════════════════════════════════════════════════════════

// ListPartyStages 获取入党阶段列表
// GET /api/v1/party/stages
func (h *StudentFeaturesHandler) ListPartyStages(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListPartyStages()
	if err != nil {
		log.Printf("查询入党阶段失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// GetMyPartyProgress 获取我的入党进度
// GET /api/v1/party/my-progress
func (h *StudentFeaturesHandler) GetMyPartyProgress(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	item, err := h.svc.GetMyPartyProgress(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "暂无入党进度记录", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": item})
}

// UpdatePartyProgress 更新入党进度
// PUT /api/v1/party/my-progress
func (h *StudentFeaturesHandler) UpdatePartyProgress(c *gin.Context) {
	var req struct {
		Stage string `json:"stage" binding:"required"`
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	if err := h.svc.UpdatePartyProgress(userCtx.UserID, req.Stage, req.Notes); err != nil {
		log.Printf("更新入党进度失败: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "进度已更新"})
}

// ListMyStudyRecords 获取我的学习记录
// GET /api/v1/party/my-study-records
func (h *StudentFeaturesHandler) ListMyStudyRecords(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	items, err := h.svc.ListMyStudyRecords(userCtx.UserID)
	if err != nil {
		log.Printf("查询学习记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

// AddStudyRecord 添加学习记录
// POST /api/v1/party/study-record
func (h *StudentFeaturesHandler) AddStudyRecord(c *gin.Context) {
	var req struct {
		StudyType   string `json:"study_type"`
		Title       string `json:"title" binding:"required"`
		Content     string `json:"content"`
		Duration    int    `json:"duration"`
		StudyDate   string `json:"study_date"`
		Certificate string `json:"certificate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("study AddStudyRecord bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	id, err := h.svc.AddStudyRecord(userCtx.UserID, req.StudyType, req.Title, req.Content, req.Duration, req.StudyDate, req.Certificate)
	if err != nil {
		log.Printf("添加学习记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "添加失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录添加成功", "data": gin.H{"id": id}})
}

// GetPartyStats 入党统计
// GET /api/v1/party/stats
func (h *StudentFeaturesHandler) GetPartyStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	stats, err := h.svc.GetPartyStats()
	if err != nil {
		log.Printf("查询入党统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}
