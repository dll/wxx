package handler

import (
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

func (h *StudentHandler) Checkin(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	var req struct {
		Mood string `json:"mood"` // happy/normal/tired/sad
		Note string `json:"note"` // 一句话感想
	}
	_ = c.ShouldBindJSON(&req)

	if h.checkinSvc != nil {
		alreadyChecked := h.checkinSvc.GetHistory(userCtx.UserID).TodayChecked
		result := h.checkinSvc.DoCheckin(userCtx.UserID, req.Mood, req.Note)
		// 首次打卡加分（当日幂等，避免重复刷积分）
		if !alreadyChecked && h.phase2Svc != nil && result.Success {
			_ = h.phase2Svc.AddPoints(userCtx.UserID, 5, "今日学习打卡", "checkin")
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": result.Message, "data": result})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "打卡成功"})
}

// CheckinHistory 打卡历史
func (h *StudentHandler) CheckinHistory(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if h.checkinSvc != nil {
		result := h.checkinSvc.GetHistory(userCtx.UserID)
		// 附加最近 30 天打卡日期（供日历渲染）
		dates, _ := h.checkinSvc.GetRecentDates(userCtx.UserID, 30)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "recent_dates": dates})
		return
	}

	// 兜底 mock
	today := time.Now().Format("2006-01-02")
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"date":           today,
			"streak":         0,
			"total_days":     0,
			"longest_streak": 0,
			"today_checked":  false,
		},
		"recent_dates": []string{},
	})
}

// CheckinMakeup 补签（断签保护，每月 2 次）
// POST /student/checkin/makeup  body: {date: "YYYY-MM-DD", mood, note}
func (h *StudentHandler) CheckinMakeup(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	var req struct {
		Date string `json:"date"`
		Mood string `json:"mood"`
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择要补签的日期"})
		return
	}

	if h.checkinSvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "补签成功"})
		return
	}

	result := h.checkinSvc.MakeupCheckin(userCtx.UserID, req.Date, req.Mood, req.Note)
	if !result.Success {
		c.JSON(http.StatusOK, gin.H{"code": 1, "message": result.Message, "data": result})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": result.Message, "data": result})
}
