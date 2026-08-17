package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

// setupHomeworkTestDB 教师作业仓库测试库（含 users/teacher_courses/homework/student_grades），
// 对齐 teacher_course_repo_test / phase3_teacher_grades_test 的多表 CREATE 范式。
func setupHomeworkTestDB(t *testing.T) *HomeworkRepo {
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
		CREATE TABLE IF NOT EXISTS homework (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			teacher_id    INTEGER NOT NULL,
			course_id     TEXT    NOT NULL,
			course_name   TEXT    NOT NULL DEFAULT '',
			semester      TEXT    NOT NULL,
			title         TEXT    NOT NULL,
			description   TEXT    NOT NULL DEFAULT '',
			publish_at    TEXT,
			due_at        TEXT,
			status        TEXT    NOT NULL DEFAULT 'active',
			created_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at    TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
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
	// 教师 3（王老师）+ 3 名学生（用于统计 join + passed 计数）
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (3,'tea_a','teacher','王老师')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (10,'stu1','student','张三')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (11,'stu2','student','李四')`)
	_, _ = db.Exec(`INSERT INTO users (id, username, role, display_name) VALUES (12,'stu3','student','王五')`)
	return NewHomeworkRepo(db)
}

// TestHomeworkRepo_Publish_Idempotent 发布幂等：同组合仅一条，不覆盖内容，不同标题可再发
func TestHomeworkRepo_Publish_Idempotent(t *testing.T) {
	r := setupHomeworkTestDB(t)
	id, existed, err := r.Publish(&Homework{
		TeacherID: 3, CourseID: "cs101", CourseName: "数据结构", Semester: "2025-2026-2",
		Title: "第一次作业", Description: "完成练习", Status: HomeworkStatusActive,
	})
	if err != nil || existed || id <= 0 {
		t.Fatalf("首次发布应新增: id=%d existed=%v err=%v", id, existed, err)
	}
	// 幂等：course_id 大小写差异应命中同一条（normalize 大写），existed=true 且不覆盖内容
	_, existed2, err := r.Publish(&Homework{
		TeacherID: 3, CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2",
		Title: "第一次作业", Description: "不应覆盖", Status: HomeworkStatusActive,
	})
	if err != nil || !existed2 {
		t.Fatalf("同组合重复发布应 existed=true: existed=%v err=%v", existed2, err)
	}
	list, _ := r.ListByTeacher(3)
	if len(list) != 1 {
		t.Fatalf("应有 1 条作业，实际 %d", len(list))
	}
	if list[0].Description != "完成练习" {
		t.Fatalf("幂等不应覆盖内容: %s", list[0].Description)
	}
	// 不同标题可再发
	if _, existed3, _ := r.Publish(&Homework{
		TeacherID: 3, CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2",
		Title: "第二次作业", Status: HomeworkStatusActive,
	}); existed3 {
		t.Fatalf("不同标题应新增而非幂等")
	}
	if l, _ := r.ListByTeacher(3); len(l) != 2 {
		t.Fatalf("应有 2 条作业，实际 %d", len(l))
	}
}

// TestHomeworkRepo_Archive_SoftDelete 下架软删 + 本人可见 + 他人不可见
func TestHomeworkRepo_Archive_SoftDelete(t *testing.T) {
	r := setupHomeworkTestDB(t)
	id, _, _ := r.Publish(&Homework{
		TeacherID: 3, CourseID: "CS200", CourseName: "算法", Semester: "2025-2026-2",
		Title: "作业A", Status: HomeworkStatusActive,
	})
	if err := r.Archive(id); err != nil {
		t.Fatalf("下架失败: %v", err)
	}
	hw, _ := r.GetOwnHomework(id, 3)
	if hw == nil || hw.Status != HomeworkStatusArchived {
		t.Fatalf("下架后应置 archived: %+v", hw)
	}
	// 他人不可查（读取边界=teacher_id）
	if hw2, _ := r.GetOwnHomework(id, 4); hw2 != nil {
		t.Fatalf("他人不应看到本人作业: %+v", hw2)
	}
}

// TestHomeworkRepo_Update 编辑（归属课程不变）
func TestHomeworkRepo_Update(t *testing.T) {
	r := setupHomeworkTestDB(t)
	id, _, _ := r.Publish(&Homework{
		TeacherID: 3, CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2",
		Title: "原题", Status: HomeworkStatusActive,
	})
	if err := r.Update(id, &Homework{
		CourseName: "数据结构", Title: "新题", Description: "说明", Status: HomeworkStatusPublished,
	}); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	hw, _ := r.GetOwnHomework(id, 3)
	if hw == nil || hw.Title != "新题" || hw.Description != "说明" || hw.Status != HomeworkStatusPublished {
		t.Fatalf("更新后字段不符: %+v", hw)
	}
	// 归属课程不可经 Update 改变
	if hw.CourseID != "CS101" {
		t.Fatalf("归属课程不应随编辑改变: %s", hw.CourseID)
	}
}

// TestHomeworkRepo_GradeStats_Empty_Honest 0 行 -> total=0 + not_available，不报错不补造
func TestHomeworkRepo_GradeStats_Empty_Honest(t *testing.T) {
	r := setupHomeworkTestDB(t)
	stats, err := r.GradeStatsByCourse("CS101", "2025-2026-2")
	if err != nil {
		t.Fatalf("0 行统计不应报错: %v", err)
	}
	if stats == nil || stats.Total != 0 || !stats.NotAvailable {
		t.Fatalf("0 行应 total=0 + not_available: %+v", stats)
	}
	if stats.Levels["优秀"] != 0 || stats.Levels["良好"] != 0 || stats.Levels["及格"] != 0 || stats.Levels["不及格"] != 0 {
		t.Fatalf("0 行各档应为 0: %+v", stats.Levels)
	}
}

// TestHomeworkRepo_GradeStats_Real 真实成绩聚合：人数/均分/及格率/四档；非 final 不计入
func TestHomeworkRepo_GradeStats_Real(t *testing.T) {
	r := setupHomeworkTestDB(t)
	// 3 条真实 final 成绩：92(优秀) / 55(不及格) / 68(及格)。注意：student_grades 由迁移 036 建表，无 created_by/updated_by 列。
	_, _ = r.db.Exec(`INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, grade_level, passed, credits_earned)
		VALUES ('10','CS101','数据结构','2025-2026-2','final',92,'优秀',1,4)`)
	_, _ = r.db.Exec(`INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, grade_level, passed, credits_earned)
		VALUES ('11','CS101','数据结构','2025-2026-2','final',55,'不及格',0,0)`)
	_, _ = r.db.Exec(`INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, grade_level, passed, credits_earned)
		VALUES ('12','CS101','数据结构','2025-2026-2','final',68,'及格',1,3)`)
	// 非 final 类型不计入统计
	_, _ = r.db.Exec(`INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, grade_level, passed, credits_earned)
		VALUES ('10','CS101','数据结构','2025-2026-2','midterm',100,'优秀',1,4)`)
	// 其它课程不计入
	_, _ = r.db.Exec(`INSERT INTO student_grades (user_id, course_id, course_name, semester, grade_type, score, grade_level, passed, credits_earned)
		VALUES ('10','CS102','操作系统','2025-2026-2','final',90,'优秀',1,4)`)

	stats, err := r.GradeStatsByCourse("CS101", "2025-2026-2")
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if stats.Total != 3 {
		t.Fatalf("人数应为 3，实际 %d", stats.Total)
	}
	if stats.PassedCount != 2 {
		t.Fatalf("及格人数应为 2，实际 %d", stats.PassedCount)
	}
	// 均分 (92+55+68)/3 ≈ 71.67
	if stats.AvgScore < 71.5 || stats.AvgScore > 71.8 {
		t.Fatalf("均分应约 71.67: %f", stats.AvgScore)
	}
	if stats.PassRate < 0.66 || stats.PassRate > 0.67 {
		t.Fatalf("及格率应约 2/3: %f", stats.PassRate)
	}
	if stats.Levels["优秀"] != 1 || stats.Levels["及格"] != 1 || stats.Levels["不及格"] != 1 || stats.Levels["良好"] != 0 {
		t.Fatalf("分档分布不符: %+v", stats.Levels)
	}
	if stats.CourseName != "数据结构" {
		t.Fatalf("应取到课程名: %s", stats.CourseName)
	}
	// 其它课程独立诚实空
	other, _ := r.GradeStatsByCourse("CS102", "2025-2026-2")
	if other.Total != 1 || other.NotAvailable {
		t.Fatalf("CS102 应统计到 1 条: %+v", other)
	}
	zero, _ := r.GradeStatsByCourse("CS999", "2025-2026-2")
	if zero.Total != 0 || !zero.NotAvailable {
		t.Fatalf("无记录课程应诚实为空: %+v", zero)
	}
}
