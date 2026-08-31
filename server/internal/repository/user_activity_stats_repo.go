package repository

import (
	"database/sql"
	"time"
)

// UserActivityStatsRepo 学生注册/登录/打卡统计仓库（任务5，2026-09-01）。
//
// 数据源：
//   - 注册：users.created_at（含角色过滤 guest/student）
//   - 登录：audit_logs（action='POST' AND resource='/api/v1/auth/login'），
//     登录请求在 JWTAuth 之前无法带 user_id，按 username 聚合后再关联 users.id
//   - 打卡：student_checkins（每人每天至多一条，UNIQUE(user_id, check_date)）
type UserActivityStatsRepo struct {
	db *sql.DB
}

// NewUserActivityStatsRepo 创建学生活动统计仓库。
func NewUserActivityStatsRepo(db *sql.DB) *UserActivityStatsRepo {
	return &UserActivityStatsRepo{db: db}
}

// UserActivityStats 学生注册/登录/打卡聚合统计。
type UserActivityStats struct {
	// RegisteredTotal 学生累计注册数（guest+student，含待审核）
	RegisteredTotal int `json:"registered_total"`
	// PendingApproval 待审核数（status=pending 的 guest）
	PendingApproval int `json:"pending_approval"`
	// RegisteredToday 今日新增注册
	RegisteredToday int `json:"registered_today"`
	// RegisteredMonth 本月新增注册
	RegisteredMonth int `json:"registered_month"`
	// LoginToday 今日登录人数（去重）
	LoginTodayUsers int `json:"login_today_users"`
	// LoginTodayCount 今日登录次数
	LoginTodayCount int `json:"login_today_count"`
	// Login7dUsers 近7日活跃（登录去重人数）
	Login7dUsers int `json:"login_7d_users"`
	// CheckinToday 今日打卡人数
	CheckinToday int `json:"checkin_today"`
	// CheckinYesterday 昨日打卡人数
	CheckinYesterday int `json:"checkin_yesterday"`
	// Checkin7dAvg 近7日日均打卡人数
	Checkin7dAvg float64 `json:"checkin_7d_avg"`
	// Daily 近7日趋势（按日）
	Daily []UserActivityDayItem `json:"daily"`
}

// UserActivityDayItem 单日活动趋势项。
type UserActivityDayItem struct {
	Date       string `json:"date"`        // YYYY-MM-DD
	Registers  int    `json:"registers"`   // 当日新增注册
	Logins     int    `json:"logins"`      // 当日登录次数
	LoginUsers int    `json:"login_users"` // 当日登录人数（去重）
	Checkins   int    `json:"checkins"`    // 当日打卡人数
}

const loginResourceFilter = `action = 'POST' AND resource = '/api/v1/auth/login'`

// GetUserActivityStats 汇总学生注册/登录/打卡统计（近7日趋势 + 当日概览）。
func (r *UserActivityStatsRepo) GetUserActivityStats() (*UserActivityStats, error) {
	s := &UserActivityStats{}
	day := func(offset int) string { return nowDate(-offset) }

	// ── 注册统计 ──
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role IN ('guest','student')`,
	).Scan(&s.RegisteredTotal); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = 'guest' AND status = 'pending'`,
	).Scan(&s.PendingApproval); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role IN ('guest','student') AND date(created_at) >= ?`, day(0),
	).Scan(&s.RegisteredToday); err != nil {
		return nil, err
	}
	monthStart := day(0)[:7] + "-01"
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role IN ('guest','student') AND date(created_at) >= ?`, monthStart,
	).Scan(&s.RegisteredMonth); err != nil {
		return nil, err
	}

	// ── 登录统计（audit_logs 按 username 关联 users）──
	if err := r.db.QueryRow(
		`SELECT COUNT(*), COUNT(DISTINCT a.username) FROM audit_logs a
		 WHERE `+loginResourceFilter+` AND a.result_code = 200 AND date(a.created_at) >= ?`, day(0),
	).Scan(&s.LoginTodayCount, &s.LoginTodayUsers); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(
		`SELECT COUNT(DISTINCT a.username) FROM audit_logs a
		 WHERE `+loginResourceFilter+` AND a.result_code = 200 AND date(a.created_at) >= ?`, day(6),
	).Scan(&s.Login7dUsers); err != nil {
		return nil, err
	}

	// ── 打卡统计 ──
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM student_checkins WHERE check_date = ?`, day(0),
	).Scan(&s.CheckinToday); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM student_checkins WHERE check_date = ?`, day(1),
	).Scan(&s.CheckinYesterday); err != nil {
		return nil, err
	}
	var checkin7d int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM student_checkins WHERE check_date >= ?`, day(6),
	).Scan(&checkin7d); err != nil {
		return nil, err
	}
	s.Checkin7dAvg = float64(checkin7d) / 7.0

	// ── 近7日趋势 ──
	regs := map[string]int{}
	rows, err := r.db.Query(
		`SELECT date(created_at), COUNT(*) FROM users
		 WHERE role IN ('guest','student') AND date(created_at) >= ? GROUP BY date(created_at)`, day(6),
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			break
		}
		regs[d] = n
	}
	rows.Close()

	logins := map[string]int{}
	loginUsers := map[string]int{}
	rows, err = r.db.Query(
		`SELECT date(a.created_at), COUNT(*), COUNT(DISTINCT a.username) FROM audit_logs a
		 WHERE `+loginResourceFilter+` AND a.result_code = 200 AND date(a.created_at) >= ? GROUP BY date(a.created_at)`, day(6),
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var n, u int
		if err := rows.Scan(&d, &n, &u); err != nil {
			break
		}
		logins[d] = n
		loginUsers[d] = u
	}
	rows.Close()

	checkins := map[string]int{}
	rows, err = r.db.Query(
		`SELECT check_date, COUNT(*) FROM student_checkins WHERE check_date >= ? GROUP BY check_date`, day(6),
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			break
		}
		checkins[d] = n
	}
	rows.Close()

	for i := 6; i >= 0; i-- {
		d := day(i)
		s.Daily = append(s.Daily, UserActivityDayItem{
			Date:       d,
			Registers:  regs[d],
			Logins:     logins[d],
			LoginUsers: loginUsers[d],
			Checkins:   checkins[d],
		})
	}
	return s, nil
}

// nowDate 返回当天（含 offset 天偏移）的 YYYY-MM-DD。
func nowDate(offsetDays int) string {
	t := time.Now().AddDate(0, 0, -offsetDays)
	return t.Format("2006-01-02")
}
