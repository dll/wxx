package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

// setupTeacherCourseTestDB 教师授课关系测试库（含 users + teacher_courses）
func setupTeacherCourseTestDB(t *testing.T) *TeacherCourseRepo {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	_, _ = db.Exec(`
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
	_, _ = db.Exec(`DELETE FROM users; INSERT INTO users (id, username, role, display_name) VALUES (3,'tea_a','teacher','王老师')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (4,'tea_b','teacher','李老师')`)
	return NewTeacherCourseRepo(db)
}

// TestTeacherCourseRepo_Submit_Idempotent 申报幂等：pending 拒绝重复、approved 不重复、rejected 可重申报
func TestTeacherCourseRepo_Submit_Idempotent(t *testing.T) {
	r := setupTeacherCourseTestDB(t)

	// 首次申报 → pending
	id, status, err := r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "cs101", CourseName: "数据结构", Semester: "2025-2026-2", CreatedBy: 3})
	if err != nil || status != CourseStatusPending || id <= 0 {
		t.Fatalf("首次申报应返回 pending: id=%d status=%s err=%v", id, status, err)
	}

	// 幂等：course_id 大小写差异应命中同一条（normalize 大写）
	_, _, err = r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", CreatedBy: 3})
	if err == nil {
		t.Fatalf("同组合 pending 时重复申报应报错")
	}

	// 审核通过后：approved，再申报应返回 approved 不重复
	if err := r.ReviewTeacherCourse(id, 9, "教务", CourseStatusApproved, "确认授课"); err != nil {
		t.Fatalf("审核通过失败: %v", err)
	}
	_, status, err = r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "cs101", CourseName: "数据结构", Semester: "2025-2026-2", CreatedBy: 3})
	if err != nil || status != CourseStatusApproved {
		t.Fatalf("approved 后重复申报应返回 approved: status=%s err=%v", status, err)
	}

	// 用另一门课验证：驳回后可重新申报（置回 pending）
	id2, _, _ := r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "CS150", CourseName: "离散", Semester: "2025-2026-2", CreatedBy: 3})
	if err := r.ReviewTeacherCourse(id2, 9, "教务", CourseStatusRejected, "信息不符"); err != nil {
		t.Fatalf("驳回失败: %v", err)
	}
	_, status, err = r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "cs150", CourseName: "离散", Semester: "2025-2026-2", CreatedBy: 3})
	if err != nil || status != CourseStatusPending {
		t.Fatalf("驳回后重申报应置回 pending: status=%s err=%v", status, err)
	}
}

// TestTeacherCourseRepo_Review_StateMachine 审核流：pending→approved / pending→rejected + 留痕
func TestTeacherCourseRepo_Review_StateMachine(t *testing.T) {
	r := setupTeacherCourseTestDB(t)

	id, _, _ := r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "CS200", CourseName: "算法", Semester: "2025-2026-2", CreatedBy: 3})

	// 非法状态拒绝
	if err := r.ReviewTeacherCourse(id, 9, "教务", "weird", ""); err == nil {
		t.Fatalf("非法审核状态应报错")
	}
	// 不存在记录
	if err := r.ReviewTeacherCourse(9999, 9, "教务", CourseStatusApproved, ""); err == nil {
		t.Fatalf("不存在申报应报错")
	}

	// approved + 留痕
	if err := r.ReviewTeacherCourse(id, 9, "张教务", CourseStatusApproved, "确认"); err != nil {
		t.Fatalf("审核通过失败: %v", err)
	}
	mine, err := r.ListTeacherCourses(3, "", 100)
	if err != nil || len(mine) != 1 {
		t.Fatalf("应查到 1 条申报: %+v err=%v", mine, err)
	}
	tc := mine[0]
	if tc.Status != CourseStatusApproved || tc.ReviewedBy != 9 || tc.ReviewedName != "张教务" ||
		tc.ReviewNote != "确认" || tc.ReviewedAt == "" {
		t.Fatalf("审核留痕不符: %+v", tc)
	}

	// approved 后再审核应拒绝（仅 pending 可审）
	if err := r.ReviewTeacherCourse(id, 9, "张教务", CourseStatusRejected, ""); err == nil {
		t.Fatalf("approved 后不可再审核")
	}
}

// TestTeacherCourseRepo_GetStatus 成绩强校验判据：无/pending/approved/rejected
func TestTeacherCourseRepo_GetStatus(t *testing.T) {
	r := setupTeacherCourseTestDB(t)

	// 无申报
	exists, status, err := r.GetTeacherCourseStatus(3, "CS301", "2025-2026-2")
	if err != nil || exists || status != "" {
		t.Fatalf("无申报应返回 exists=false: %v %q err=%v", exists, status, err)
	}

	// pending
	id, _, _ := r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "CS301", CourseName: "考研英语", Semester: "2025-2026-2", CreatedBy: 3})
	exists, status, _ = r.GetTeacherCourseStatus(3, "cs301", "2025-2026-2")
	if !exists || status != CourseStatusPending {
		t.Fatalf("pending 判据不符: %v %q", exists, status)
	}

	// approved
	_ = r.ReviewTeacherCourse(id, 9, "教务", CourseStatusApproved, "")
	exists, status, _ = r.GetTeacherCourseStatus(3, "CS301", "2025-2026-2")
	if !exists || status != CourseStatusApproved {
		t.Fatalf("approved 判据不符: %v %q", exists, status)
	}

	// rejected
	id2, _, _ := r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 4, CourseID: "CS302", CourseName: "高数", Semester: "2025-2026-2", CreatedBy: 4})
	_ = r.ReviewTeacherCourse(id2, 9, "教务", CourseStatusRejected, "")
	exists, status, _ = r.GetTeacherCourseStatus(4, "cs302", "2025-2026-2")
	if !exists || status != CourseStatusRejected {
		t.Fatalf("rejected 判据不符: %v %q", exists, status)
	}

	// 教师不匹配（教师 3 查教师 4 的课）→ 无申报
	exists, _, _ = r.GetTeacherCourseStatus(3, "CS302", "2025-2026-2")
	if exists {
		t.Fatalf("教师 3 不应看到教师 4 的申报")
	}
}

// TestTeacherCourseRepo_CountPending 待审角标
func TestTeacherCourseRepo_CountPending(t *testing.T) {
	r := setupTeacherCourseTestDB(t)
	n, _ := r.CountPendingTeacherCourses()
	if n != 0 {
		t.Fatalf("初始待审应为 0，实际 %d", n)
	}
	r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 3, CourseID: "C1", Semester: "S1", CreatedBy: 3})
	r.SubmitTeacherCourse(&TeacherCourse{TeacherID: 4, CourseID: "C2", Semester: "S1", CreatedBy: 4})
	n, _ = r.CountPendingTeacherCourses()
	if n != 2 {
		t.Fatalf("应有 2 条待审，实际 %d", n)
	}
}
