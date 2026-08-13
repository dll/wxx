package handler

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// statsLogScanError 记录统计查询行扫描失败（避免静默缺行）
func statsLogScanError(ctx string, err error) {
	log.Printf("[stats] 扫描 %s 行失败: %v", ctx, err)
}

// StatsHandler 统计数据 HTTP handler
type StatsHandler struct {
	db *sql.DB
}

// NewStatsHandler 创建统计 handler
func NewStatsHandler(db *sql.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	Users     UserStats      `json:"users"`
	Knowledge KnowledgeStats `json:"knowledge"`
	Chat      ChatStats      `json:"chat"`
	Feedback  FeedbackStats  `json:"feedback"`
}

// UserStats 用户统计
type UserStats struct {
	Total    int            `json:"total"`
	TodayNew int            `json:"today_new"`
	MonthNew int            `json:"month_new"`
	ByRole   map[string]int `json:"by_role"`
}

// KnowledgeStats 知识库统计
type KnowledgeStats struct {
	Total     int            `json:"total"`
	Draft     int            `json:"draft"`
	Pending   int            `json:"pending"`
	Published int            `json:"published"`
	Retired   int            `json:"retired"`
	ByType    map[string]int `json:"by_type"`
	WeekNew   int            `json:"week_new"`
}

// ChatStats 对话统计
type ChatStats struct {
	TotalSessions int            `json:"total_sessions"`
	TotalMessages int            `json:"total_messages"`
	TodaySessions int            `json:"today_sessions"`
	TodayMessages int            `json:"today_messages"`
	WeekTrend     []DayTrendItem `json:"week_trend"`
}

// DayTrendItem 每日趋势项
type DayTrendItem struct {
	Date     string `json:"date"`
	Sessions int    `json:"sessions"`
	Messages int    `json:"messages"`
}

// FeedbackStats 反馈统计
type FeedbackStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Resolved   int `json:"resolved"`
	Dismissed  int `json:"dismissed"`
	WeekNew    int `json:"week_new"`
}

// GetDashboardStats 获取仪表盘统计数据
// GET /api/v1/admin/stats/dashboard
func (h *StatsHandler) GetDashboardStats(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
		})
		return
	}

	stats := &DashboardStats{}

	// 1. 用户统计
	if err := h.getUserStats(stats); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取用户统计失败",
		})
		return
	}

	// 2. 知识库统计
	if err := h.getKnowledgeStats(stats); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取知识库统计失败",
		})
		return
	}

	// 3. 对话统计
	if err := h.getChatStats(stats); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取对话统计失败",
		})
		return
	}

	// 4. 反馈统计
	if err := h.getFeedbackStats(stats); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "获取反馈统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// getUserStats 获取用户统计
func (h *StatsHandler) getUserStats(stats *DashboardStats) error {
	now := time.Now()
	todayStart := now.Format("2006-01-02") + " 00:00:00"
	monthStart := now.Format("2006-01") + "-01 00:00:00"

	// 总用户数
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return err
	}
	stats.Users.Total = total

	// 今日新增
	var todayNew int
	err = h.db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= ?", todayStart).Scan(&todayNew)
	if err != nil {
		return err
	}
	stats.Users.TodayNew = todayNew

	// 本月新增
	var monthNew int
	err = h.db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= ?", monthStart).Scan(&monthNew)
	if err != nil {
		return err
	}
	stats.Users.MonthNew = monthNew

	// 按角色分布
	stats.Users.ByRole = make(map[string]int)
	rows, err := h.db.Query("SELECT role, COUNT(*) FROM users GROUP BY role")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			statsLogScanError("user-by-role", err)
			continue
		}
		stats.Users.ByRole[role] = count
	}

	return nil
}

// getKnowledgeStats 获取知识库统计
func (h *StatsHandler) getKnowledgeStats(stats *DashboardStats) error {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")

	// 总数
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM kb_resources").Scan(&total)
	if err != nil {
		return err
	}
	stats.Knowledge.Total = total

	// 按状态统计
	statusRows, err := h.db.Query("SELECT status, COUNT(*) FROM kb_resources GROUP BY status")
	if err != nil {
		return err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			statsLogScanError("statusRows", err)

		}
		switch status {
		case "draft":
			stats.Knowledge.Draft = count
		case "pending":
			stats.Knowledge.Pending = count
		case "published":
			stats.Knowledge.Published = count
		case "retired":
			stats.Knowledge.Retired = count
		}
	}

	// 按类型分布
	stats.Knowledge.ByType = make(map[string]int)
	typeRows, err := h.db.Query("SELECT resource_type, COUNT(*) FROM kb_resources GROUP BY resource_type")
	if err != nil {
		return err
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var rtype string
		var count int
		if err := typeRows.Scan(&rtype, &count); err != nil {
			statsLogScanError("typeRows", err)

		}
		// 转换为小写以匹配前端期望
		switch rtype {
		case "Policy":
			stats.Knowledge.ByType["policy"] = count
		case "Process":
			stats.Knowledge.ByType["process"] = count
		case "FAQ":
			stats.Knowledge.ByType["faq"] = count
		case "Activity":
			stats.Knowledge.ByType["activity"] = count
		default:
			stats.Knowledge.ByType[rtype] = count
		}
	}

	// 本周新增
	var weekNew int
	err = h.db.QueryRow("SELECT COUNT(*) FROM kb_resources WHERE created_at >= ?", weekAgo).Scan(&weekNew)
	if err != nil {
		return err
	}
	stats.Knowledge.WeekNew = weekNew

	return nil
}

// getChatStats 获取对话统计
func (h *StatsHandler) getChatStats(stats *DashboardStats) error {
	todayStart := time.Now().Format("2006-01-02") + " 00:00:00"

	// 总会话数
	var totalSessions int
	err := h.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&totalSessions)
	if err != nil {
		return err
	}
	stats.Chat.TotalSessions = totalSessions

	// 总消息数
	var totalMessages int
	err = h.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&totalMessages)
	if err != nil {
		return err
	}
	stats.Chat.TotalMessages = totalMessages

	// 今日会话数
	var todaySessions int
	err = h.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE created_at >= ?", todayStart).Scan(&todaySessions)
	if err != nil {
		return err
	}
	stats.Chat.TodaySessions = todaySessions

	// 今日消息数
	var todayMessages int
	err = h.db.QueryRow("SELECT COUNT(*) FROM messages WHERE created_at >= ?", todayStart).Scan(&todayMessages)
	if err != nil {
		return err
	}
	stats.Chat.TodayMessages = todayMessages

	// 近 7 天趋势
	stats.Chat.WeekTrend = h.getWeekTrend()

	return nil
}

// getWeekTrend 获取近 7 天趋势数据
func (h *StatsHandler) getWeekTrend() []DayTrendItem {
	now := time.Now()
	result := make([]DayTrendItem, 7)

	// 生成近 7 天的日期序列
	for i := 0; i < 7; i++ {
		day := now.AddDate(0, 0, -6+i)
		dateStr := day.Format("2006-01-02")
		result[i] = DayTrendItem{
			Date:     dateStr,
			Sessions: 0,
			Messages: 0,
		}
	}

	// 统计每天的会话数
	// 用 Go 侧计算起始日期，避免 SQLite 专有 date('now',...) 在 MySQL 下失效
	since := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	sessionRows, err := h.db.Query(`
		SELECT date(created_at) as d, COUNT(*) 
		FROM sessions 
		WHERE created_at >= ?
		GROUP BY date(created_at)
	`, since)
	if err == nil {
		defer sessionRows.Close()
		sessionMap := make(map[string]int)
		for sessionRows.Next() {
			var date string
			var count int
			if err := sessionRows.Scan(&date, &count); err == nil {
				sessionMap[date] = count
			} else {
				statsLogScanError("session", err)
			}
		}
		for i := range result {
			if cnt, ok := sessionMap[result[i].Date]; ok {
				result[i].Sessions = cnt
			}
		}
	}

	// 统计每天的消息数
	messageRows, err := h.db.Query(`
		SELECT date(created_at) as d, COUNT(*) 
		FROM messages 
		WHERE created_at >= ?
		GROUP BY date(created_at)
	`, since)
	if err == nil {
		defer messageRows.Close()
		messageMap := make(map[string]int)
		for messageRows.Next() {
			var date string
			var count int
			if err := messageRows.Scan(&date, &count); err == nil {
				messageMap[date] = count
			} else {
				statsLogScanError("message", err)
			}
		}
		for i := range result {
			if cnt, ok := messageMap[result[i].Date]; ok {
				result[i].Messages = cnt
			}
		}
	}

	return result
}

// getFeedbackStats 获取反馈统计
func (h *StatsHandler) getFeedbackStats(stats *DashboardStats) error {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")

	// 总反馈数
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM feedback").Scan(&total)
	if err != nil {
		return err
	}
	stats.Feedback.Total = total

	// 按状态统计
	statusRows, err := h.db.Query("SELECT status, COUNT(*) FROM feedback GROUP BY status")
	if err != nil {
		return err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			statsLogScanError("statusRows", err)

		}
		switch status {
		case "pending":
			stats.Feedback.Pending = count
		case "processing":
			stats.Feedback.Processing = count
		case "resolved":
			stats.Feedback.Resolved = count
		case "dismissed":
			stats.Feedback.Dismissed = count
		}
	}

	// 本周新增
	var weekNew int
	err = h.db.QueryRow("SELECT COUNT(*) FROM feedback WHERE created_at >= ?", weekAgo).Scan(&weekNew)
	if err != nil {
		return err
	}
	stats.Feedback.WeekNew = weekNew

	return nil
}

// RequireAdminStatsRead 检查是否有仪表盘读取权限（college_admin 及以上）
func RequireAdminStatsRead() gin.HandlerFunc {
	return auth.RequireCapability(auth.CollegeMetricsRead)
}
