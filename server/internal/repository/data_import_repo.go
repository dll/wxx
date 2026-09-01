package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	dbutil "github.com/dll/wxx/server/internal/db"
)

// DataImportRepo 阶段三数据导入访问层（成绩 / 课表 / 考试）
type DataImportRepo struct {
	db *sql.DB

	// ownerUsernameOnce/hasOwnerUsername：course_schedules 是否含 owner_username 列的缓存探测结果
	// （2026-09-01：迁移 116 在生产会加此列；测试库可能不含，需运行时按列存在与否优雅降级）
	ownerOnce     sync.Once
	hasOwnerCol   bool
	ownerCheckErr error
}

// NewDataImportRepo 创建数据导入仓库
func NewDataImportRepo(db *sql.DB) *DataImportRepo {
	return &DataImportRepo{db: db}
}

// GradeRow 成绩导入行
// CreatedBy 记录成绩首声明人（教师录入时=教师 user_id；管理端导入默认 0）。
// UpdatedBy 记录最后写入/修改人（新增于 R1，审计可追溯）。
// 幂等键仍为 user_id+course_id+semester+grade_type='final'，CreatedBy/UpdatedBy 仅作审计追溯。
type GradeRow struct {
	UserID     string  `json:"user_id"`
	CourseID   string  `json:"course_id"`
	CourseName string  `json:"course_name"`
	Semester   string  `json:"semester"`
	Score      float64 `json:"score"`
	GPA        float64 `json:"gpa"`
	Passed     bool    `json:"passed"`
	Credits    float64 `json:"credits"`
	CreatedBy  int64   `json:"created_by"`
	UpdatedBy  int64   `json:"updated_by"`
}

// UpsertGrade 按 UNIQUE(user_id, course_id, semester, grade_type='final') 幂等写入
func (r *DataImportRepo) UpsertGrade(g *GradeRow) (bool, error) {
	passed := 0
	if g.Passed {
		passed = 1
	}
	gradeLevel := gradeLevelOf(g.Score)

	// SQLite upsert 无法用 RowsAffected 区分插入/更新，先做存在性预检
	var exists int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM student_grades WHERE user_id=? AND course_id=? AND semester=? AND grade_type='final'`,
		g.UserID, g.CourseID, g.Semester).Scan(&exists); err != nil {
		return false, fmt.Errorf("成绩存在性检查失败: %w", err)
	}

	stmt := `
		INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, gpa, grade_level, passed, credits_earned, created_by, updated_by)
		VALUES (?, ?, ?, ?, 'final', ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, course_id, semester, grade_type) DO UPDATE SET
			course_name=excluded.course_name, score=excluded.score, gpa=excluded.gpa,
			grade_level=excluded.grade_level, passed=excluded.passed, credits_earned=excluded.credits_earned,
			created_by=excluded.created_by, updated_by=excluded.updated_by,
			updated_at=CURRENT_TIMESTAMP`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		g.UserID, g.CourseID, g.CourseName, g.Semester, g.Score, g.GPA, gradeLevel, passed, g.Credits, g.CreatedBy, g.UpdatedBy)
	if err != nil {
		return false, fmt.Errorf("成绩写入失败: %w", err)
	}
	return exists == 0, nil
}

// GetUserRoleByUserID 查询用户角色（成绩写入时校验 target 为学生，防捞非学生）
func (r *DataImportRepo) GetUserRoleByUserID(userID string) (string, error) {
	var role string
	err := r.db.QueryRow(`SELECT role FROM users WHERE CAST(id AS TEXT) = ?`, userID).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

// ListedGrade 已录入成绩记录（教师端查看本人声明）
type ListedGrade struct {
	UserID     string  `json:"user_id"`
	Username   string  `json:"username"`
	Name       string  `json:"name"`
	CourseID   string  `json:"course_id"`
	CourseName string  `json:"course_name"`
	Semester   string  `json:"semester"`
	Score      float64 `json:"score"`
	GPA        float64 `json:"gpa"`
	Passed     bool    `json:"passed"`
	Credits    float64 `json:"credits"`
}

// ListGradesByCreator 查询指定声明人（教师）录入的成绩记录，读取边界=created_by
func (r *DataImportRepo) ListGradesByCreator(creatorID int64) ([]*ListedGrade, error) {
	rows, err := r.db.Query(`
		SELECT g.user_id, COALESCE(u.username,''), COALESCE(u.display_name,''),
		       g.course_id, g.course_name, g.semester, g.score, g.gpa,
		       g.passed, g.credits_earned
		FROM student_grades g
		LEFT JOIN users u ON CAST(g.user_id AS INTEGER) = u.id
		WHERE g.created_by = ?
		ORDER BY g.semester DESC, g.course_id`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ListedGrade
	for rows.Next() {
		g := &ListedGrade{}
		var passed int
		if err := rows.Scan(&g.UserID, &g.Username, &g.Name, &g.CourseID, &g.CourseName,
			&g.Semester, &g.Score, &g.GPA, &passed, &g.Credits); err != nil {
			return nil, err
		}
		g.Passed = passed == 1
		list = append(list, g)
	}
	return list, rows.Err()
}

// gradeLevelOf 分数分档
func gradeLevelOf(score float64) string {
	switch {
	case score >= 90:
		return "优秀"
	case score >= 80:
		return "良好"
	case score >= 60:
		return "及格"
	default:
		return "不及格"
	}
}

// ScheduleRow 课表导入行
// 归属解析（2026-09-01 修复）：优先按 Username（学号/工号，稳定键）定位账号，
// 避免运营按内部自增 user_id 导入教师/学生课表时填错导致课程挂到错误账号；
// 二者都提供时以 Username 解析出的 user_id 为权威（忽略传入 user_id，防止不一致）。
type ScheduleRow struct {
	Username      string `json:"username"` // 学号/工号（稳定键，推荐）；与 user_id 二选一
	UserID        int64  `json:"user_id"`  // 内部自增 ID（兼容旧接口，username 缺省/为空时使用）
	CourseID      string `json:"course_id"`
	CourseName    string `json:"course_name"`
	SemesterCode  string `json:"semester_code"`
	Weekday       int    `json:"weekday"`
	StartPeriod   int    `json:"start_period"`
	EndPeriod     int    `json:"end_period"`
	WeeksPattern  string `json:"weeks_pattern"`
	Location      string `json:"location"`
	Teacher       string `json:"teacher"`
}

// UpsertSchedule 幂等写入课表（按 user+course+weekday+period 去重）。
// userID 由调用方（service 层）解析传入，确保挂到正确账号（2026-09-01）。
// owner_username：记录课表本应归属的学号/工号，供按工号归位接口追溯修正历史错挂数据。
// 若 course_schedules 尚无 owner_username 列（迁移 116 未生效的库），优雅降级仍可导入。
func (r *DataImportRepo) UpsertSchedule(s *ScheduleRow, userID int64) error {
	owner := strings.TrimSpace(s.Username)
	tableHasOwner := r.ownerUsernameAvailable()
	var stmt string
	args := []interface{}{userID, s.CourseID, s.CourseName, s.SemesterCode, s.Weekday, s.StartPeriod, s.EndPeriod, s.WeeksPattern, s.Location, s.Teacher}
	if tableHasOwner {
		stmt = `
		INSERT INTO course_schedules (user_id, course_id, course_name, semester_code, weekday, start_period, end_period, weeks_pattern, location, teacher, owner_username)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, course_id, weekday, start_period, semester_code) DO UPDATE SET
			course_name=excluded.course_name, semester_code=excluded.semester_code,
			end_period=excluded.end_period, weeks_pattern=excluded.weeks_pattern,
			location=excluded.location, teacher=excluded.teacher, owner_username=excluded.owner_username`
		args = append(args, owner)
	} else {
		stmt = `
		INSERT INTO course_schedules (user_id, course_id, course_name, semester_code, weekday, start_period, end_period, weeks_pattern, location, teacher)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, course_id, weekday, start_period, semester_code) DO UPDATE SET
			course_name=excluded.course_name, semester_code=excluded.semester_code,
			end_period=excluded.end_period, weeks_pattern=excluded.weeks_pattern,
			location=excluded.location, teacher=excluded.teacher`
	}
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)), args...)
	if err != nil {
		return fmt.Errorf("user_id=%d 课表写入失败: %w", userID, err)
	}
	return nil
}

// ownerUsernameAvailable 探测 course_schedules 是否含 owner_username 列（结果缓存）。
// 迁移 116（生产）会加该列；未应用迁移的库返回 false，相关操作优雅降级。
func (r *DataImportRepo) ownerUsernameAvailable() bool {
	r.ownerOnce.Do(func() {
		// 先试 information_schema（MySQL）；SQLite 驱动通常返回空行或报错，均回退 PRAGMA
		var n int
		if err := r.db.QueryRow(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_name='course_schedules' AND column_name='owner_username'`,
		).Scan(&n); err == nil && n > 0 {
			r.hasOwnerCol = true
			r.ownerCheckErr = nil
			return
		}
		// SQLite：PRAGMA table_info 为准（information_schema 可能静默返回空）
		rows, err := r.db.Query(`PRAGMA table_info(course_schedules)`)
		if err != nil {
			r.ownerCheckErr = err
			return
		}
		defer rows.Close()
		for rows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				continue
			}
			if name == "owner_username" {
				r.hasOwnerCol = true
				break
			}
		}
		r.ownerCheckErr = nil
	})
	return r.hasOwnerCol
}

// ReassignSchedulesByOwnerUsername 按课表 owner_username 批量归位 user_id。
// 用于修正历史课表填错内部 user_id 的问题：运营提供 工号，此处把 course_schedules.owner_username
// 匹配该工号的所有行更新为正确 user_id。返回受影响行数。
// 无 owner_username 列（迁移 116 未生效）时返回错误。
func (r *DataImportRepo) ReassignSchedulesByOwnerUsername(username string, correctUserID int64) (int64, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, fmt.Errorf("工号/学号不能为空")
	}
	if !r.ownerUsernameAvailable() {
		return 0, fmt.Errorf("课表未启用按工号归位（缺少 owner_username 列，需执行迁移 116）")
	}
	res, err := r.db.Exec(
		`UPDATE course_schedules SET user_id = ? WHERE owner_username = ?`,
		correctUserID, username,
	)
	if err != nil {
		return 0, fmt.Errorf("归位课表失败: %w", err)
	}
	return res.RowsAffected()
}

// CountSchedulesByOwnerUsername 统计某工号名下课表条数（归位前核对用）
func (r *DataImportRepo) CountSchedulesByOwnerUsername(username string) (int, error) {
	if !r.ownerUsernameAvailable() {
		return 0, fmt.Errorf("缺少 owner_username 列")
	}
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM course_schedules WHERE owner_username = ?`, strings.TrimSpace(username)).Scan(&n)
	return n, err
}

// ListSchedules 读取课表（用于排课冲突检测）
func (r *DataImportRepo) ListSchedules(semester string) ([]map[string]interface{}, error) {
	query := `SELECT user_id, course_id, course_name, semester_code, weekday, start_period, end_period, weeks_pattern, location, teacher
		FROM course_schedules WHERE 1=1`
	var args []interface{}
	if semester != "" {
		query += " AND semester_code = ?"
		args = append(args, semester)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var userID int64
		var courseID, courseName, semCode, weeks, loc, teacher string
		var weekday, startP, endP int
		if err := rows.Scan(&userID, &courseID, &courseName, &semCode, &weekday, &startP, &endP, &weeks, &loc, &teacher); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"user_id": userID, "course_id": courseID, "course_name": courseName,
			"semester_code": semCode, "weekday": weekday, "start_period": startP, "end_period": endP,
			"weeks_pattern": weeks, "location": loc, "teacher": teacher,
		})
	}
	return list, rows.Err()
}

// ListExams 读取考试安排
func (r *DataImportRepo) ListExams(semester string) ([]map[string]interface{}, error) {
	query := `SELECT course_id, course_name, semester, date, time_start, time_end, location, seat
		FROM exam_schedules WHERE 1=1`
	var args []interface{}
	if semester != "" {
		query += " AND semester = ?"
		args = append(args, semester)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var courseID, courseName, sem, date, start, end, location, seat string
		if err := rows.Scan(&courseID, &courseName, &sem, &date, &start, &end, &location, &seat); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"course_id": courseID, "course_name": courseName, "semester": sem, "date": date,
			"time_start": start, "time_end": end, "location": location, "seat": seat,
		})
	}
	return list, rows.Err()
}

// GradeSummary 学生成绩汇总（毕业审核用）
type GradeSummary struct {
	UserID   string  `json:"user_id"`
	Name     string  `json:"name"`
	Credits  float64 `json:"credits"`
	AvgScore float64 `json:"avg_score"`
	Passed   int     `json:"passed"`
	Total    int     `json:"total"`
}

// ListGradeSummaries 按学生聚合成绩（毕业资格判断数据源）
func (r *DataImportRepo) ListGradeSummaries() ([]*GradeSummary, error) {
	rows, err := r.db.Query(`
		SELECT g.user_id, COALESCE(u.display_name,''), SUM(g.credits_earned), AVG(g.score), SUM(CASE WHEN g.passed=1 THEN 1 ELSE 0 END), COUNT(*)
		FROM student_grades g LEFT JOIN users u ON CAST(g.user_id AS INTEGER) = u.id
		GROUP BY g.user_id ORDER BY g.user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*GradeSummary
	for rows.Next() {
		var s GradeSummary
		var credits float64
		var avg float64
		if err := rows.Scan(&s.UserID, &s.Name, &credits, &avg, &s.Passed, &s.Total); err != nil {
			return nil, err
		}
		s.Credits = credits
		s.AvgScore = avg
		list = append(list, &s)
	}
	return list, rows.Err()
}
