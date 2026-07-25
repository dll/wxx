package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// CheckinRepo 学生打卡数据访问层
type CheckinRepo struct {
	db *sql.DB
}

// NewCheckinRepo 创建打卡仓库
func NewCheckinRepo(db *sql.DB) *CheckinRepo {
	return &CheckinRepo{db: db}
}

// CheckinRecord 单次打卡记录
type CheckinRecord struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	CheckDate string `json:"check_date"`
	Mood      string `json:"mood"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

// DoCheckin 执行打卡（幂等，当日重复打卡返回已有记录）
func (r *CheckinRepo) DoCheckin(userID int64, date, mood, note string) (*CheckinRecord, error) {
	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO student_checkins (user_id, check_date, mood, note)
		VALUES (?, ?, ?, ?)`, userID, date, mood, note)
	if err != nil {
		return nil, fmt.Errorf("打卡写入失败: %w", err)
	}

	// 读回当日记录
	rec := &CheckinRecord{}
	err = r.db.QueryRow(`
		SELECT id, user_id, check_date, mood, note, created_at
		FROM student_checkins WHERE user_id = ? AND check_date = ?`,
		userID, date).Scan(&rec.ID, &rec.UserID, &rec.CheckDate, &rec.Mood, &rec.Note, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("读取打卡记录失败: %w", err)
	}
	return rec, nil
}

// IsCheckedToday 查询今天是否已打卡
func (r *CheckinRepo) IsCheckedToday(userID int64) bool {
	today := time.Now().Format("2006-01-02")
	var count int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM student_checkins WHERE user_id = ? AND check_date = ?`,
		userID, today).Scan(&count)
	return count > 0
}

// GetRecentDates 获取最近 N 天的打卡日期列表（倒序）
func (r *CheckinRepo) GetRecentDates(userID int64, limit int) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT check_date FROM student_checkins
		WHERE user_id = ? ORDER BY check_date DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// CountTotal 打卡总天数
func (r *CheckinRepo) CountTotal(userID int64) int {
	var count int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM student_checkins WHERE user_id = ?`, userID).Scan(&count)
	return count
}

// CalcStreak 计算当前连续打卡天数（从今天/昨天往前推）
func (r *CheckinRepo) CalcStreak(userID int64) int {
	dates, err := r.GetRecentDates(userID, 365)
	if err != nil || len(dates) == 0 {
		return 0
	}

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 连续打卡必须从今天或昨天开始
	if dates[0] != today && dates[0] != yesterday {
		return 0
	}

	streak := 1
	for i := 1; i < len(dates); i++ {
		prev, _ := time.Parse("2006-01-02", dates[i-1])
		curr, _ := time.Parse("2006-01-02", dates[i])
		diff := prev.Sub(curr).Hours() / 24
		if diff == 1 {
			streak++
		} else {
			break
		}
	}
	return streak
}

// CalcLongestStreak 计算历史最长连续天数
func (r *CheckinRepo) CalcLongestStreak(userID int64) int {
	dates, err := r.GetRecentDates(userID, 9999)
	if err != nil || len(dates) == 0 {
		return 0
	}

	longest := 1
	current := 1
	for i := 1; i < len(dates); i++ {
		prev, _ := time.Parse("2006-01-02", dates[i-1])
		curr, _ := time.Parse("2006-01-02", dates[i])
		diff := prev.Sub(curr).Hours() / 24
		if diff == 1 {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 1
		}
	}
	return longest
}
