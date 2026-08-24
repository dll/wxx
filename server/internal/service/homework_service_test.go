package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// setupHomeworkSvcTestDB 教师作业服务测试库（含 users/courses/teacher_courses/homework/student_grades）。
// teacher_course repo 提供 approved 归属判据；HomeworkService 注入两者，返回 raw db 供统计插入成绩断言。
func setupHomeworkSvcTestDB(t *testing.T) (*HomeworkService, *sql.DB) {
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
		CREATE TABLE IF NOT EXISTS homework (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			teacher_id INTEGER NOT NULL,
			course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '',
			semester TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			publish_at TEXT,
			due_at TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			UNIQUE(teacher_id, course_id, semester, title)
		);
		CREATE TABLE IF NOT EXISTS student_grades (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '',
			semester TEXT NOT NULL,
			grade_type TEXT NOT NULL DEFAULT 'final',
			score REAL NOT NULL DEFAULT 0,
			gpa REAL NOT NULL DEFAULT 0,
			grade_level TEXT NOT NULL DEFAULT '',
			passed INTEGER NOT NULL DEFAULT 0,
			credits_earned REAL NOT NULL DEFAULT 0,
			created_by INTEGER NOT NULL DEFAULT 0,
			updated_by INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
			updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
			UNIQUE(user_id, course_id, semester, grade_type)
		)
	`)
	// 主数据 courses：供 CourseExists 校验（CS101/CS102/CS999 均存在，ZZZ999 不在表内用于虚构课程测试）
	if _, err := db.Exec(`DELETE FROM courses; INSERT INTO courses (course_id, course_name) VALUES ('CS101','数据结构'),('CS102','操作系统'),('CS999','虚构课')`); err != nil {
		t.Fatalf("插入 courses 失败: %v", err)
	}
	// users 表：迁移 001 建表，但 NewTestDB 不跑 007 种子，故需显式插入教师/学生
	if _, err := db.Exec(`DELETE FROM users`); err != nil {
		t.Fatalf("清空 users 失败: %v", err)
	}
	for _, u := range []struct {
		id               int
		name, role, disp string
	}{
		{3, "tea_a", "teacher", "王老师"},
		{10, "stu1", "student", "张三"},
	} {
		if _, err := db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (?,?,?,?)`, u.id, u.name, u.role, u.disp); err != nil {
			t.Fatalf("插入用户 %d 失败: %v", u.id, err)
		}
	}
	tcRepo := repository.NewTeacherCourseRepo(db)
	tcSvc := NewTeacherCourseService(tcRepo)
	hwRepo := repository.NewHomeworkRepo(db)
	svc := NewHomeworkService(hwRepo, tcSvc)
	return svc, db
}

// TestHomeworkService_Publish_ApprovedOnly 归属强校验：仅 approved 授课课程可发布
func TestHomeworkService_Publish_ApprovedOnly(t *testing.T) {
	svc, _ := setupHomeworkSvcTestDB(t)
	ctx := context.Background()

	// 申报 CS101 并通过（approved 唯一来源为教辅真实审核）
	// 通过 teacherCourseSvc 走完整申报→审核，验证 approved 后放行
	id, _, err := svc.teacherCourseSvc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil {
		t.Fatalf("申报失败: %v", err)
	}
	if err := svc.teacherCourseSvc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusApproved, "确认"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}

	// approved → 放行
	idHw, existed, err := svc.PublishHomework(ctx, 3, "CS101", "数据结构", "2025-2026-2", "作业1", "说明", "", "")
	if err != nil || idHw <= 0 || existed {
		t.Fatalf("approved 课程应放行发布: id=%d existed=%v err=%v", idHw, existed, err)
	}
	// 幂等：同键再发布不重复
	_, existed2, err := svc.PublishHomework(ctx, 3, "CS101", "数据结构", "2025-2026-2", "作业1", "说明", "", "")
	if err != nil || !existed2 {
		t.Fatalf("同键再发布应 existed=true: existed=%v err=%v", existed2, err)
	}
	// 课程存在但未申报(CS102) → 拒绝并提示先申报
	_, _, err = svc.PublishHomework(ctx, 3, "CS102", "操作系统", "2025-2026-2", "作业2", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "申报") {
		t.Fatalf("未申报课程应拒绝并提示先申报, err=%v", err)
	}
	// 虚构课程(不在 courses 主数据) → CourseExists 拒绝
	_, _, err = svc.PublishHomework(ctx, 3, "ZZZ999", "虚构", "2025-2026-2", "作业3", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "课程不存在") {
		t.Fatalf("不在 courses 主数据的虚构课程应被拒绝, err=%v", err)
	}
}

// TestHomeworkService_Publish_NonApprovedStates 对称成绩强校验三态：无申报/pending/rejected
func TestHomeworkService_Publish_NonApprovedStates(t *testing.T) {
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS courses (id INTEGER PRIMARY KEY AUTOINCREMENT, course_id TEXT NOT NULL UNIQUE, course_name TEXT NOT NULL, semester TEXT DEFAULT '');
		CREATE TABLE IF NOT EXISTS teacher_courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT, teacher_id INTEGER NOT NULL, course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '', semester TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
			created_by INTEGER NOT NULL, reviewed_by INTEGER NOT NULL DEFAULT 0, reviewed_name TEXT NOT NULL DEFAULT '',
			review_note TEXT NOT NULL DEFAULT '', reviewed_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')), updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			UNIQUE(teacher_id, course_id, semester));
		CREATE TABLE IF NOT EXISTS homework (
			id INTEGER PRIMARY KEY AUTOINCREMENT, teacher_id INTEGER NOT NULL, course_id TEXT NOT NULL,
			course_name TEXT NOT NULL DEFAULT '', semester TEXT NOT NULL, title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '', publish_at TEXT, due_at TEXT, status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')), updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			UNIQUE(teacher_id, course_id, semester, title));
	`)
	_, _ = db.Exec(`INSERT INTO courses (course_id, course_name) VALUES ('CS101','数据结构'),('CS102','操作系统');`)
	tcRepo := repository.NewTeacherCourseRepo(db)
	tcSvc := NewTeacherCourseService(tcRepo)
	svc := NewHomeworkService(repository.NewHomeworkRepo(db), tcSvc)
	ctx := context.Background()

	// 无申报 → 拒绝
	_, _, err := svc.PublishHomework(ctx, 3, "CS101", "数据结构", "2025-2026-2", "作业", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "申报") {
		t.Fatalf("无申报应拒绝并提示先申报, err=%v", err)
	}
	// pending → 拒绝
	idp, _, _ := tcSvc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	_, _, err = svc.PublishHomework(ctx, 3, "CS101", "数据结构", "2025-2026-2", "作业", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "待审核") {
		t.Fatalf("pending 应拒绝并提示待审核, err=%v", err)
	}
	// rejected → 拒绝
	_ = tcSvc.ReviewTeacherCourse(ctx, idp, 9, "张教务", repository.CourseStatusRejected, "不符")
	_, _, err = svc.PublishHomework(ctx, 3, "CS101", "数据结构", "2025-2026-2", "作业", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "驳回") {
		t.Fatalf("rejected 应拒绝并提示驳回, err=%v", err)
	}
}

// TestHomeworkService_Update_OwnerOnly 编辑仅本人且已下架不可编辑
func TestHomeworkService_Update_OwnerOnly(t *testing.T) {
	svc, _ := setupHomeworkSvcTestDB(t)
	ctx := context.Background()
	// 走完整 approved 申报
	id, _, err := svc.teacherCourseSvc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil {
		t.Fatalf("申报失败: %v", err)
	}
	_ = svc.teacherCourseSvc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusApproved, "确认")

	idHw, _, err := svc.PublishHomework(ctx, 3, "CS101", "数据结构", "2025-2026-2", "作业1", "", "", "")
	if err != nil {
		t.Fatalf("发布失败: %v", err)
	}
	// 他人(999)编辑 → 拒绝
	if err := svc.UpdateHomework(ctx, idHw, 999, "新题", "", "", ""); err == nil {
		t.Fatalf("非本人编辑应拒绝")
	}
	// 本人编辑 → 成功
	if err := svc.UpdateHomework(ctx, idHw, 3, "新题", "新说明", "2026-01-01", "2026-01-10"); err != nil {
		t.Fatalf("本人编辑应成功: %v", err)
	}
	// 下架后再编辑 → 拒绝
	if err := svc.ArchiveHomework(ctx, idHw, 3); err != nil {
		t.Fatalf("下架失败: %v", err)
	}
	if err := svc.UpdateHomework(ctx, idHw, 3, "又改", "", "", ""); err == nil || !strings.Contains(err.Error(), "下架") {
		t.Fatalf("已下架作业不可编辑, err=%v", err)
	}
}

// TestHomeworkService_GradeStats_ApprovedOnly 统计仅 approved 课程返回真实；未确认返回空 not_available
func TestHomeworkService_GradeStats_ApprovedOnly(t *testing.T) {
	svc, env := setupHomeworkSvcTestDB(t)
	ctx := context.Background()
	id, _, err := svc.teacherCourseSvc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil {
		t.Fatalf("申报失败: %v", err)
	}
	_ = svc.teacherCourseSvc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusApproved, "确认")
	// 录入 CS101 真实成绩（approved 课程）。注意：student_grades 由迁移 036 建表，无 created_by/updated_by 列，故不写这两列。
	if _, err := env.Exec(`
		INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, gpa, grade_level, passed, credits_earned) VALUES
			('10','CS101','数据结构','2025-2026-2','final', 90, 3.7, '优秀', 1, 4);
	`); err != nil {
		t.Fatalf("插入测试成绩失败: %v", err)
	}
	// approved → 真实统计
	stats, err := svc.GradeStatsByCourse(ctx, 3, "CS101", "2025-2026-2")
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if stats == nil || stats.Total != 1 || stats.NotAvailable {
		t.Fatalf("approved 课程应返回真实统计: %+v", stats)
	}
	// 未确认课程(CS102 未申报) → 返回空 not_available（诚实口径，不报错）
	stats2, err := svc.GradeStatsByCourse(ctx, 3, "CS102", "2025-2026-2")
	if err != nil {
		t.Fatalf("未确认课程统计不应报错: %v", err)
	}
	if stats2 == nil || stats2.Total != 0 || !stats2.NotAvailable {
		t.Fatalf("未确认课程应返回空 not_available: %+v", stats2)
	}
}

// TestHomeworkService_ListApprovedCourses approved 白名单（前端课程下拉数据源）
func TestHomeworkService_ListApprovedCourses(t *testing.T) {
	svc, _ := setupHomeworkSvcTestDB(t)
	ctx := context.Background()
	id, _, err := svc.teacherCourseSvc.SubmitTeacherCourse(ctx, 3, "CS101", "数据结构", "2025-2026-2")
	if err != nil {
		t.Fatalf("申报失败: %v", err)
	}
	_ = svc.teacherCourseSvc.ReviewTeacherCourse(ctx, id, 9, "张教务", repository.CourseStatusApproved, "确认")
	list, err := svc.ListApprovedCourses(ctx, 3)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(list) != 1 || list[0].CourseID != "CS101" {
		t.Fatalf("approved 白名单应仅含 CS101: %+v", list)
	}
}
