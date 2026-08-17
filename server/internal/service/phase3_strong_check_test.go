package service

import (
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// setupStrongCheckTestDB 强校验测试库：临时 DB，不发 seeded approved，供四分支测试自建状态。
func setupStrongCheckTestDB(t *testing.T) (*Phase3Service, *sql.DB) {
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
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (3,'tea_a','teacher','王老师')`)

	repo := repository.NewDataImportRepo(db)
	tcRepo := repository.NewTeacherCourseRepo(db)
	svc := NewPhase3Service(repo)
	svc.SetTeacherCourseRepo(tcRepo)
	return svc, db
}

// 一条合法成绩（学生 1 + 指定课程/学期）
func strongGrade(courseID, semester string) *repository.GradeRow {
	return &repository.GradeRow{UserID: "1", CourseID: courseID, CourseName: "课程" + courseID, Semester: semester, Score: 88, GPA: 3.5, Passed: true, Credits: 4}
}

// TestPhase3_ImportTeacherGrades_StrongCheck 强校验四分支：无申报/pending/rejected/approved
func TestPhase3_ImportTeacherGrades_StrongCheck(t *testing.T) {
	svc, db := setupStrongCheckTestDB(t)
	tcRepo := repository.NewTeacherCourseRepo(db)

	// ① 无申报 → 拒绝
	r := svc.ImportTeacherGrades([]*repository.GradeRow{strongGrade("C1", "S1")}, 3)
	if r.Created != 0 || len(r.Errors) != 1 {
		t.Fatalf("无申报应拒绝: %+v", r)
	}

	// 建一条 pending
	id, _, _ := tcRepo.SubmitTeacherCourse(&repository.TeacherCourse{TeacherID: 3, CourseID: "C2", Semester: "S1", CreatedBy: 3})
	// ② pending → 拒绝
	r = svc.ImportTeacherGrades([]*repository.GradeRow{strongGrade("C2", "S1")}, 3)
	if r.Created != 0 || len(r.Errors) != 1 {
		t.Fatalf("pending 应拒绝: %+v", r)
	}

	// ③ rejected → 拒绝
	_, _ = db.Exec(`DELETE FROM teacher_courses WHERE id=?`, id)
	_, _, _ = tcRepo.SubmitTeacherCourse(&repository.TeacherCourse{TeacherID: 3, CourseID: "C3", Semester: "S1", CreatedBy: 3})
	var id3 int64
	_ = db.QueryRow(`SELECT id FROM teacher_courses WHERE course_id='C3'`).Scan(&id3)
	_ = tcRepo.ReviewTeacherCourse(id3, 9, "教务", repository.CourseStatusRejected, "不符")
	r = svc.ImportTeacherGrades([]*repository.GradeRow{strongGrade("C3", "S1")}, 3)
	if r.Created != 0 || len(r.Errors) != 1 {
		t.Fatalf("rejected 应拒绝: %+v", r)
	}

	// ④ approved → 放行
	_, _, _ = tcRepo.SubmitTeacherCourse(&repository.TeacherCourse{TeacherID: 3, CourseID: "C4", Semester: "S1", CreatedBy: 3})
	var id4 int64
	_ = db.QueryRow(`SELECT id FROM teacher_courses WHERE course_id='C4'`).Scan(&id4)
	_ = tcRepo.ReviewTeacherCourse(id4, 9, "教务", repository.CourseStatusApproved, "确认")
	r = svc.ImportTeacherGrades([]*repository.GradeRow{strongGrade("C4", "S1")}, 3)
	if r.Created != 1 || len(r.Errors) != 0 {
		t.Fatalf("approved 应放行写入: %+v", r)
	}
}

// TestPhase3_ImportTeacherGrades_StrongCheck_SingleErrorNotBatchRollback
// 单条校验失败仅记入 Errors，不影响其它已通过行写入。
func TestPhase3_ImportTeacherGrades_StrongCheck_SingleErrorNotBatchRollback(t *testing.T) {
	svc, db := setupStrongCheckTestDB(t)
	tcRepo := repository.NewTeacherCourseRepo(db)

	// C5 approved、C6 无申报
	_, _, _ = tcRepo.SubmitTeacherCourse(&repository.TeacherCourse{TeacherID: 3, CourseID: "C5", Semester: "S1", CreatedBy: 3})
	var id5 int64
	_ = db.QueryRow(`SELECT id FROM teacher_courses WHERE course_id='C5'`).Scan(&id5)
	_ = tcRepo.ReviewTeacherCourse(id5, 9, "教务", repository.CourseStatusApproved, "")

	r := svc.ImportTeacherGrades([]*repository.GradeRow{
		strongGrade("C5", "S1"),
		strongGrade("C6", "S1"),
	}, 3)
	// C5 写入 +1，C6 记 1 错
	if r.Created != 1 || len(r.Errors) != 1 {
		t.Fatalf("单条错误不应整批回滚，approved(C5) 应写入、无申报(C6) 应记错: %+v", r)
	}

	// C6 补成 approved 后再导同一条学生成绩 → 更新，非新增
	_, _, _ = tcRepo.SubmitTeacherCourse(&repository.TeacherCourse{TeacherID: 3, CourseID: "C6", Semester: "S1", CreatedBy: 3})
	var id6 int64
	_ = db.QueryRow(`SELECT id FROM teacher_courses WHERE course_id='C6'`).Scan(&id6)
	_ = tcRepo.ReviewTeacherCourse(id6, 9, "教务", repository.CourseStatusApproved, "")
	// C6 补 approved 后导同一条学生成绩 → 首次成功写入（新增）。
	// （此前 C6 因无 approved 被拒绝，未落任何行，故为 Created 而非 Updated）
	r = svc.ImportTeacherGrades([]*repository.GradeRow{strongGrade("C6", "S1")}, 3)
	if r.Created != 1 || len(r.Errors) != 0 {
		t.Fatalf("C6 获 approved 后应首次写入学生成绩: %+v", r)
	}
}

// TestPhase3_ImportTeacherGrades_NoTcRepo_NoCrash 防御：tcRepo 未注入时（旧路径）不 panic，仍可写
func TestPhase3_ImportTeacherGrades_NoTcRepo_NoCrash(t *testing.T) {
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS student_grades (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL, course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '', semester TEXT NOT NULL, grade_type TEXT NOT NULL DEFAULT 'final',
			score REAL DEFAULT 0, gpa REAL DEFAULT 0, rank INTEGER DEFAULT 0, grade_level TEXT DEFAULT '',
			passed INTEGER NOT NULL DEFAULT 0, credits_earned REAL NOT NULL DEFAULT 0,
			created_by INTEGER NOT NULL DEFAULT 0, updated_by INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP), updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
			UNIQUE(user_id, course_id, semester, grade_type)
		);
		CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL, role TEXT NOT NULL, display_name TEXT NOT NULL DEFAULT '');
	`)
	db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (1,'st','student','生')`)
	svc := NewPhase3Service(repository.NewDataImportRepo(db)) // 未注入 tcRepo
	r := svc.ImportTeacherGrades([]*repository.GradeRow{strongGrade("C9", "S1")}, 3)
	if r.Created != 1 || len(r.Errors) != 0 {
		t.Fatalf("未注入 tcRepo 时应可写入（防御路径）: %+v", r)
	}
}
