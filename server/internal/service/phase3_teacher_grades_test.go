package service

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// 最小成绩导入测试库（对齐 data_import_repo_test 的表结构，含 created_by + R3 teacher_courses）
func setupTeacherGradesTestDB(t *testing.T) *Phase3Service {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS student_grades (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '',
			semester TEXT NOT NULL,
			grade_type TEXT NOT NULL DEFAULT 'final',
			score REAL DEFAULT 0,
			gpa REAL DEFAULT 0,
			rank INTEGER DEFAULT 0,
			grade_level TEXT DEFAULT '',
			passed INTEGER NOT NULL DEFAULT 0,
			credits_earned REAL NOT NULL DEFAULT 0,
			created_by INTEGER NOT NULL DEFAULT 0,
			updated_by INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
			updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
			UNIQUE(user_id, course_id, semester, grade_type)
		);
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			role TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS teacher_courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			teacher_id INTEGER NOT NULL,
			course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '',
			semester TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_by INTEGER NOT NULL,
			reviewed_by INTEGER NOT NULL DEFAULT 0,
			reviewed_name TEXT NOT NULL DEFAULT '',
			review_note TEXT NOT NULL DEFAULT '',
			reviewed_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			UNIQUE(teacher_id, course_id, semester)
		)
	`)
	_, _ = db.Exec(`DELETE FROM users; INSERT INTO users (id, username, role, display_name) VALUES (1,'stu_a','student','张明远')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (2,'stu_b','student','李华')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (3,'tea_a','teacher','王老师')`)

	repo := repository.NewDataImportRepo(db)
	svc := NewPhase3Service(repo)
	// R3 强校验接线：注入 teacher_courses 访问层，并 seed 教师 3 已确认(approved)的课程组合
	tcRepo := repository.NewTeacherCourseRepo(db)
	svc.SetTeacherCourseRepo(tcRepo)
	// 教师 3 已 approved 的授课组合（供既有成绩校验测试沿用，只测其它校验维度）
	seedApprovedTeacherCourse(t, db, 3, "CS101", "数据结构", "2025-2026-2")
	seedApprovedTeacherCourse(t, db, 3, "CS102", "操作系统", "2025-2026-2")
	seedApprovedTeacherCourse(t, db, 3, "CS103", "网络", "2025-2026-2")
	return svc
}

// seedApprovedTeacherCourse 直接落一条 approved 授课关系（测试专用）：
// 模拟教辅已真实审核确认，供成绩录入测试按 approved 放行。
func seedApprovedTeacherCourse(t *testing.T, db *sql.DB, teacherID int64, courseID, courseName, semester string) {
	t.Helper()
	// course_id 写入规范化（对齐 repo.normalizeCourseID：trim+大写）
	courseID = strings.TrimSpace(strings.ToUpper(courseID))
	_, err := db.Exec(
		`INSERT OR IGNORE INTO teacher_courses (teacher_id, course_id, course_name, semester, status, created_by, reviewed_by, reviewed_name, reviewed_at)
		  VALUES (?, ?, ?, ?, 'approved', ?, 999, '测试审核人', datetime('now','localtime'))`,
		teacherID, courseID, courseName, semester, teacherID,
	)
	if err != nil {
		t.Fatalf("seed 测试授课关系失败: %v", err)
	}
}

func TestPhase3Service_ImportTeacherGrades_StudentOnlyAndAudit(t *testing.T) {
	svc := setupTeacherGradesTestDB(t)

	// 教师 user_id=3 录入：1 个学生合法 + 1 个教师目标（应被拒绝）+ 1 个不存在学号（应报错）
	res := svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 88, GPA: 3.5, Passed: true, Credits: 4},
		{UserID: "3", CourseID: "CS102", CourseName: "操作系统", Semester: "2025-2026-2", Score: 99, GPA: 4.0, Passed: true, Credits: 4},
		{UserID: "999", CourseID: "CS103", CourseName: "网络", Semester: "2025-2026-2", Score: 60, GPA: 1.0, Passed: true, Credits: 3},
	}, 3)

	if res.Created != 1 {
		t.Fatalf("仅 1 条学生记录应新增，实际 created=%d errors=%v", res.Created, res.Errors)
	}
	if res.Updated != 0 {
		t.Fatalf("不应有更新，实际 updated=%d", res.Updated)
	}
	if len(res.Errors) != 2 {
		t.Fatalf("教师目标与非学生学号应各记 1 错，实际 errors=%v", res.Errors)
	}

	// 教师只能查到自己声明（读取边界 created_by=3）
	mine, err := svc.ListTeacherGrades(3)
	if err != nil || len(mine) != 1 || mine[0].Score != 88 || mine[0].Name != "张明远" {
		t.Fatalf("教师 3 应看到 1 条声明成绩: %+v err=%v", mine, err)
	}
	// 教师目标 3 虽被拒绝写入学生成绩，但教师 4 无任何声明
	other, err := svc.ListTeacherGrades(4)
	if err != nil || len(other) != 0 {
		t.Fatalf("教师 4 应无声明: %+v err=%v", other, err)
	}

	// 幂等：同一教师重录同一学生 → 更新，不新增
	res2 := svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 90, GPA: 3.8, Passed: true, Credits: 4},
	}, 3)
	if res2.Created != 0 || res2.Updated != 1 {
		t.Fatalf("幂等重录应更新而非新增: %+v", res2)
	}
}

// TestPhase3Service_ImportTeacherGrades_ScoreAndPassedValidation
// R2：成绩范围(0-100) + passed↔score 一致性校验（防误录/恶意污染学生端与毕业审核）
func TestPhase3Service_ImportTeacherGrades_ScoreAndPassedValidation(t *testing.T) {
	svc := setupTeacherGradesTestDB(t)

	// 成绩 > 100 → 拒绝
	res := svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 120, Passed: true, Credits: 4},
	}, 3)
	if len(res.Errors) != 1 || res.Created != 0 {
		t.Fatalf("成绩>100 应被拒绝: %+v", res)
	}
	// 成绩 < 0 → 拒绝
	res = svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: -5, Passed: false, Credits: 4},
	}, 3)
	if len(res.Errors) != 1 || res.Created != 0 {
		t.Fatalf("成绩<0 应被拒绝: %+v", res)
	}
	// passed 与成绩不一致（score>=60 但 passed=false）→ 拒绝
	res = svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 90, Passed: false, Credits: 4},
	}, 3)
	if len(res.Errors) != 1 || res.Created != 0 {
		t.Fatalf("score>=60 但 passed=false 应被拒绝: %+v", res)
	}
	// score<60 但 passed=true → 拒绝
	res = svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 50, Passed: true, Credits: 4},
	}, 3)
	if len(res.Errors) != 1 || res.Created != 0 {
		t.Fatalf("score<60 但 passed=true 应被拒绝: %+v", res)
	}
	// 合法数据应通过（score=90 passed=true）
	res = svc.ImportTeacherGrades([]*repository.GradeRow{
		{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 90, GPA: 3.8, Passed: true, Credits: 4},
	}, 3)
	if res.Created != 1 || len(res.Errors) != 0 {
		t.Fatalf("合法成绩应新增: %+v", res)
	}
}
