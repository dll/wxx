// Package model 管理端统计 DTO（P4-d：从 handler 下沉，供 repository 填充与 handler 序列化）。
package model

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	Users     UserStats              `json:"users"`
	Knowledge KnowledgeStats         `json:"knowledge"`
	Chat      ChatStats              `json:"chat"`
	Feedback  DashboardFeedbackStats `json:"feedback"`
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
type DashboardFeedbackStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Resolved   int `json:"resolved"`
	Dismissed  int `json:"dismissed"`
	WeekNew    int `json:"week_new"`
}
