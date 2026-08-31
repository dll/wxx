// Package repository 学生画像与头像仓库（P4-d：从 student_profile_handler 下沉的 9 处裸 SQL）。
package repository

import (
	"database/sql"
	"errors"
)

// ErrProfileNotFound 画像资源不存在。
var ErrProfileNotFound = errors.New("画像资源不存在")

// StudentProfileRepo 学生画像/头像数据访问层。
type StudentProfileRepo struct {
	db *sql.DB
}

// NewStudentProfileRepo 创建学生画像仓库。
func NewStudentProfileRepo(db *sql.DB) *StudentProfileRepo {
	return &StudentProfileRepo{db: db}
}

// BasicInfo 学生基础信息（users 表）。
type BasicInfo struct {
	College        string
	Major          string
	ClassName      string
	EnrollmentDate string
	EnrollmentYear string
	Status         string
}

// GetBasicInfo 基础信息（无记录返回 ErrProfileNotFound）。
func (r *StudentProfileRepo) GetBasicInfo(userID int64) (*BasicInfo, error) {
	info := &BasicInfo{}
	err := r.db.QueryRow(
		"SELECT college, major, class_name, enrollment_date, enrollment_year, status FROM users WHERE id = ?",
		userID,
	).Scan(&info.College, &info.Major, &info.ClassName, &info.EnrollmentDate, &info.EnrollmentYear, &info.Status)
	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

// AcademicSummary 学业成绩汇总。
type AcademicSummary struct {
	CourseCount int
	TotalCredit float64
	AvgScore    float64
	AvgGPA      float64
	PassCount   int
	TotalCount  int
}

// PassRate 及格率（%）。
func (a *AcademicSummary) PassRate() float64 {
	if a.TotalCount == 0 {
		return 0
	}
	return float64(a.PassCount) / float64(a.TotalCount) * 100
}

// GetAcademicSummary 学业成绩汇总（student_grades）。
func (r *StudentProfileRepo) GetAcademicSummary(userID int64) (*AcademicSummary, error) {
	s := &AcademicSummary{}
	err := r.db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(credits_earned),0), COALESCE(AVG(score),0), "+
			"COALESCE(AVG(gpa),0), COALESCE(SUM(CASE WHEN passed=1 THEN 1 ELSE 0 END),0), COUNT(*) "+
			"FROM student_grades WHERE user_id = ?",
		userID,
	).Scan(&s.CourseCount, &s.TotalCredit, &s.AvgScore, &s.AvgGPA, &s.PassCount, &s.TotalCount)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// CompetitionRow 竞赛报名行。
type CompetitionRow struct {
	CompetitionID int64
	StudentName   string
	TeamName      string
	AdvisorName   string
	Status        string
	AwardLevel    string
}

// ListCompetitions 最近竞赛报名（最多 limit 条）。
func (r *StudentProfileRepo) ListCompetitions(userID int64, limit int) ([]*CompetitionRow, error) {
	rows, err := r.db.Query(
		"SELECT competition_id, student_name, team_name, advisor_name, status, award_level FROM competition_registrations WHERE user_id = ? ORDER BY id DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*CompetitionRow, 0)
	for rows.Next() {
		row := &CompetitionRow{}
		if err := rows.Scan(&row.CompetitionID, &row.StudentName, &row.TeamName, &row.AdvisorName, &row.Status, &row.AwardLevel); err != nil {
			continue
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// PartyProgress 入党进度（无记录返回零值而非错误）。
type PartyProgress struct {
	CurrentStage string
	Status       string
	ApplyDate    string
}

// GetPartyProgress 入党进度。
func (r *StudentProfileRepo) GetPartyProgress(userID int64) (*PartyProgress, error) {
	p := &PartyProgress{}
	err := r.db.QueryRow(
		"SELECT current_stage, status, COALESCE(apply_date,'') FROM party_progress WHERE user_id = ? ORDER BY id DESC LIMIT 1",
		userID,
	).Scan(&p.CurrentStage, &p.Status, &p.ApplyDate)
	if err == sql.ErrNoRows {
		return &PartyProgress{}, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ClubRow 社团参与行。
type ClubRow struct {
	ClubID   int64
	Role     string
	JoinDate string
}

// ListClubs 在读社团（未退团，最多 limit 条）。
func (r *StudentProfileRepo) ListClubs(userID int64, limit int) ([]*ClubRow, error) {
	rows, err := r.db.Query(
		"SELECT club_id, role, join_date FROM club_members WHERE user_id = ? AND (leave_date IS NULL OR leave_date = '') ORDER BY join_date DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*ClubRow, 0)
	for rows.Next() {
		row := &ClubRow{}
		if err := rows.Scan(&row.ClubID, &row.Role, &row.JoinDate); err != nil {
			continue
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// CheckinSummary 打卡汇总。
type CheckinSummary struct {
	TotalDays int
	LastDate  string
}

// GetCheckinSummary 打卡汇总。
func (r *StudentProfileRepo) GetCheckinSummary(userID int64) (*CheckinSummary, error) {
	s := &CheckinSummary{}
	err := r.db.QueryRow(
		"SELECT COUNT(*), COALESCE(MAX(check_date),'') FROM student_checkins WHERE user_id = ?",
		userID,
	).Scan(&s.TotalDays, &s.LastDate)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetPointsTotal 积分合计。
func (r *StudentProfileRepo) GetPointsTotal(userID int64) (int, error) {
	var total int
	err := r.db.QueryRow("SELECT COALESCE(SUM(points),0) FROM student_points WHERE user_id = ?", userID).Scan(&total)
	return total, err
}

// UpdateAvatar 更新用户头像（base64 + mime）。
func (r *StudentProfileRepo) UpdateAvatar(userID int64, encoded, mime string) error {
	_, err := r.db.Exec(
		"UPDATE users SET avatar_base64 = ?, avatar_mime = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		encoded, mime, userID,
	)
	return err
}

// GetAvatar 读取用户头像（base64 + mime；无头像返回 ErrProfileNotFound）。
func (r *StudentProfileRepo) GetAvatar(userID string) (string, string, error) {
	var b64, mime string
	err := r.db.QueryRow(
		"SELECT COALESCE(avatar_base64,''), COALESCE(avatar_mime,'image/png') FROM users WHERE id = ?",
		userID,
	).Scan(&b64, &mime)
	if err == sql.ErrNoRows {
		return "", "", ErrProfileNotFound
	}
	if err != nil {
		return "", "", err
	}
	if b64 == "" {
		return "", "", ErrProfileNotFound
	}
	return b64, mime, nil
}

// HomeUserInfo 首页用户信息。
type HomeUserInfo struct {
	DisplayName    string
	Username       string
	College        string
	Major          string
	EnrollmentYear string
}

// GetHomeUserInfo 首页用户信息（无记录返回 ErrProfileNotFound）。
func (r *StudentProfileRepo) GetHomeUserInfo(userID int64) (*HomeUserInfo, error) {
	info := &HomeUserInfo{}
	var displayName, username, college, major, enrollmentYear sql.NullString
	err := r.db.QueryRow(
		"SELECT display_name, username, college, major, enrollment_year FROM users WHERE id = ?",
		userID,
	).Scan(&displayName, &username, &college, &major, &enrollmentYear)
	if err == sql.ErrNoRows {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	if displayName.Valid {
		info.DisplayName = displayName.String
	}
	if username.Valid {
		info.Username = username.String
	}
	if college.Valid {
		info.College = college.String
	}
	if major.Valid {
		info.Major = major.String
	}
	if enrollmentYear.Valid {
		info.EnrollmentYear = enrollmentYear.String
	}
	return info, nil
}
