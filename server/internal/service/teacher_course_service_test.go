package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// setupTeacherCourseServiceTestDB 授课申报服务测试库（含 courses + users + teacher_courses）。
// 补 M 校验：申报入参 course_id 必须存在于 courses 主数据表，拒绝虚构课程号。
func setupTeacherCourseServiceTestDB(t *testing.T) (*TeacherCourseService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			course_id TEXT NOT NULL UNIQUE,
			course_name TEXT NOT NULL,
			semester TEXT DEFAULT ''
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
		);
	`)
	_, _ = db.Exec(`INSERT INTO courses (course_id, course_name) VALUES ('CS101','数据结构'),('CS102','操作系统')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (3,'tea_a','teacher','王老师')`)
	svc := NewTeacherCourseService(repository.NewTeacherCourseRepo(db))
	return svc, db
}

// TestTeacherCourseService_Submit_CourseIDExists 申报校验（M）：course_id 存在于 courses → 放行
func TestTeacherCourseService_Submit_CourseIDExists(t *testing.T) {
	svc, _ := setupTeacherCourseServiceTestDB(t)
	id, status, err := svc.SubmitTeacherCourse(context.Background(), 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil || id <= 0 || status != repository.CourseStatusPending {
		t.Fatalf("存在的课程应放行申报: id=%d status=%s err=%v", id, status, err)
	}
	// 大小写不敏感：cs102 应命中 courses.CS102
	_, status, err = svc.SubmitTeacherCourse(context.Background(), 3, "cs102", "操作系统", "2025-2026-2")
	if err != nil || status != repository.CourseStatusPending {
		t.Fatalf("大小写归一后应命中存在的课程: status=%s err=%v", status, err)
	}
}

// TestTeacherCourseService_Submit_CourseIDNotExists 申报校验（M）：course_id 不存在于 courses → 拒绝
func TestTeacherCourseService_Submit_CourseIDNotExists(t *testing.T) {
	svc, _ := setupTeacherCourseServiceTestDB(t)
	_, _, err := svc.SubmitTeacherCourse(context.Background(), 3, "CS999", "不存在的课", "2025-2026-2")
	if err == nil || !strings.Contains(err.Error(), "课程不存在") {
		t.Fatalf("虚构课程号应被拒绝并提示核对课程ID，实际 err=%v", err)
	}
	// 空课程/空学期也被拒绝
	if _, _, err = svc.SubmitTeacherCourse(context.Background(), 3, "", "sem", "2025-2026-2"); err == nil {
		t.Fatalf("空课程ID应被拒绝")
	}
	// 教师为空拒绝
	if _, _, err = svc.SubmitTeacherCourse(context.Background(), 0, "CS101", "数据结构", "2025-2026-2"); err == nil {
		t.Fatalf("空教师应被拒绝")
	}
}

// TestTeacherCourseService_Review_StateMachine 审核流服务层回归：approved/rejected + 角标
func TestTeacherCourseService_Review_StateMachine(t *testing.T) {
	svc, _ := setupTeacherCourseServiceTestDB(t)
	ctx := context.Background()

	id, _, err := svc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil {
		t.Fatalf("申报失败: %v", err)
	}
	// 驳回
	if err := svc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusRejected, "信息不符"); err != nil {
		t.Fatalf("驳回失败: %v", err)
	}
	// 驳回后可重申报（置回 pending）
	_, status, err := svc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil || status != repository.CourseStatusPending {
		t.Fatalf("驳回后重申报应置回 pending: status=%s err=%v", status, err)
	}
	// 通过
	if err := svc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusApproved, "确认"); err != nil {
		t.Fatalf("通过失败: %v", err)
	}
	// approved 后再审核应拒绝（仅 pending 可审）
	if err := svc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusRejected, ""); err == nil {
		t.Fatalf("approved 后不可再审核")
	}
	// 待审角标：已有 1 条 approved，0 pending
	n, _ := svc.CountPending(ctx)
	if n != 0 {
		t.Fatalf("approved 后待审应为 0，实际 %d", n)
	}
}
