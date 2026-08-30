// Package repository 管理端统计仓库（P4-d：从 stats_handler 下沉的 17 处裸 SQL）。
package repository

import (
	"database/sql"
	"log"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// StatsRepo 统计数据仓库。
type StatsRepo struct {
	db *sql.DB
}

// NewStatsRepo 创建统计仓库。
func NewStatsRepo(db *sql.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

// statsLogScanError 记录统计查询行扫描失败（避免静默缺行）。
func statsLogScanError(ctx string, err error) {
	log.Printf("[stats] 扫描 %s 行失败: %v", ctx, err)
}

// GetUserStats 用户统计。
func (r *StatsRepo) GetUserStats() (*model.UserStats, error) {
	now := time.Now()
	todayStart := now.Format("2006-01-02") + " 00:00:00"
	monthStart := now.Format("2006-01") + "-01 00:00:00"

	stats := &model.UserStats{ByRole: make(map[string]int)}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.Total); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= ?", todayStart).Scan(&stats.TodayNew); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= ?", monthStart).Scan(&stats.MonthNew); err != nil {
		return nil, err
	}

	rows, err := r.db.Query("SELECT role, COUNT(*) FROM users GROUP BY role")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			statsLogScanError("user-by-role", err)
			continue
		}
		stats.ByRole[role] = count
	}
	return stats, rows.Err()
}

// GetKnowledgeStats 知识库统计。
func (r *StatsRepo) GetKnowledgeStats() (*model.KnowledgeStats, error) {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")

	stats := &model.KnowledgeStats{ByType: make(map[string]int)}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM kb_resources").Scan(&stats.Total); err != nil {
		return nil, err
	}

	statusRows, err := r.db.Query("SELECT status, COUNT(*) FROM kb_resources GROUP BY status")
	if err != nil {
		return nil, err
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
			stats.Draft = count
		case "pending":
			stats.Pending = count
		case "published":
			stats.Published = count
		case "retired":
			stats.Retired = count
		}
	}

	typeRows, err := r.db.Query("SELECT resource_type, COUNT(*) FROM kb_resources GROUP BY resource_type")
	if err != nil {
		return nil, err
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
			stats.ByType["policy"] = count
		case "Process":
			stats.ByType["process"] = count
		case "FAQ":
			stats.ByType["faq"] = count
		case "Activity":
			stats.ByType["activity"] = count
		default:
			stats.ByType[rtype] = count
		}
	}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM kb_resources WHERE created_at >= ?", weekAgo).Scan(&stats.WeekNew); err != nil {
		return nil, err
	}
	return stats, nil
}

// GetChatStats 对话统计。
func (r *StatsRepo) GetChatStats() (*model.ChatStats, error) {
	todayStart := time.Now().Format("2006-01-02") + " 00:00:00"

	stats := &model.ChatStats{}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&stats.TotalSessions); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&stats.TotalMessages); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE created_at >= ?", todayStart).Scan(&stats.TodaySessions); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow("SELECT COUNT(*) FROM messages WHERE created_at >= ?", todayStart).Scan(&stats.TodayMessages); err != nil {
		return nil, err
	}

	trend, err := r.weekTrend()
	if err != nil {
		return nil, err
	}
	stats.WeekTrend = trend
	return stats, nil
}

// weekTrend 近 7 天趋势。
func (r *StatsRepo) weekTrend() ([]model.DayTrendItem, error) {
	now := time.Now()
	result := make([]model.DayTrendItem, 7)
	for i := 0; i < 7; i++ {
		result[i] = model.DayTrendItem{Date: now.AddDate(0, 0, -6+i).Format("2006-01-02")}
	}

	// 用 Go 侧计算起始日期，避免 SQLite 专有 date('now',...) 在 MySQL 下失效
	since := now.AddDate(0, 0, -6).Format("2006-01-02")

	fill := func(query string, apply func(date string, count int)) error {
		rows, err := r.db.Query(query, since)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var date string
			var count int
			if err := rows.Scan(&date, &count); err != nil {
				statsLogScanError("week-trend", err)
				continue
			}
			apply(date, count)
		}
		return rows.Err()
	}

	if err := fill(`SELECT date(created_at), COUNT(*) FROM sessions WHERE created_at >= ? GROUP BY date(created_at)`,
		func(date string, count int) {
			for i := range result {
				if result[i].Date == date {
					result[i].Sessions = count
				}
			}
		}); err != nil {
		return nil, err
	}

	if err := fill(`SELECT date(created_at), COUNT(*) FROM messages WHERE created_at >= ? GROUP BY date(created_at)`,
		func(date string, count int) {
			for i := range result {
				if result[i].Date == date {
					result[i].Messages = count
				}
			}
		}); err != nil {
		return nil, err
	}

	return result, nil
}

// GetFeedbackStats 反馈统计。
func (r *StatsRepo) GetFeedbackStats() (*model.DashboardFeedbackStats, error) {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")

	stats := &model.DashboardFeedbackStats{}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM feedback").Scan(&stats.Total); err != nil {
		return nil, err
	}

	statusRows, err := r.db.Query("SELECT status, COUNT(*) FROM feedback GROUP BY status")
	if err != nil {
		return nil, err
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
			stats.Pending = count
		case "processing":
			stats.Processing = count
		case "resolved":
			stats.Resolved = count
		case "dismissed":
			stats.Dismissed = count
		}
	}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM feedback WHERE created_at >= ?", weekAgo).Scan(&stats.WeekNew); err != nil {
		return nil, err
	}
	return stats, nil
}
