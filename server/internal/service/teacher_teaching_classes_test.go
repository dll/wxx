package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestTeachingClassesOnlyApprovedCoursesForTeacher(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS teacher_courses (id INTEGER PRIMARY KEY AUTOINCREMENT, teacher_id INTEGER NOT NULL, course_id TEXT NOT NULL, course_name TEXT NOT NULL DEFAULT '', semester TEXT NOT NULL, status TEXT NOT NULL, created_by INTEGER NOT NULL DEFAULT 0, reviewed_by INTEGER NOT NULL DEFAULT 0, reviewed_name TEXT NOT NULL DEFAULT '', review_note TEXT NOT NULL DEFAULT '', reviewed_at TEXT, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '', UNIQUE(teacher_id, course_id, semester))`); err != nil {
		t.Fatalf("创建测试授课关系表失败: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS courses (id INTEGER PRIMARY KEY AUTOINCREMENT, course_id TEXT NOT NULL UNIQUE, course_name TEXT NOT NULL)`); err != nil {
		t.Fatalf("创建测试课程目录失败: %v", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO courses (course_id, course_name) VALUES ('CS-101', '课程目录中的数据结构')`); err != nil {
		t.Fatalf("写入课程目录失败: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO teacher_courses (teacher_id, course_id, course_name, semester, status, created_by) VALUES (1, 'CS-101', '数据结构', '2026-秋', 'approved', 1), (1, 'CS-999', '待审核课程', '2026-秋', 'pending', 1), (2, 'CS-202', '他人课程', '2026-秋', 'approved', 2)`); err != nil {
		t.Fatalf("写入测试授课关系失败: %v", err)
	}
	svc := NewTeacherService(nil, repository.NewTeacherCourseRepo(db))
	classes, err := svc.TeachingClasses(context.Background(), 1)
	if err != nil {
		t.Fatalf("查询教师课程失败: %v", err)
	}
	if len(classes) != 1 || classes[0]["course_id"] != "CS-101" || classes[0]["course"] != "课程目录中的数据结构" {
		t.Fatalf("课程白名单错误: %#v", classes)
	}
}

func TestTeachingClassesFallsBackToImportedSchedule(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS teacher_courses (id INTEGER PRIMARY KEY AUTOINCREMENT, teacher_id INTEGER NOT NULL, course_id TEXT NOT NULL, course_name TEXT NOT NULL DEFAULT '', semester TEXT NOT NULL, status TEXT NOT NULL, created_by INTEGER NOT NULL DEFAULT 0, reviewed_by INTEGER NOT NULL DEFAULT 0, reviewed_name TEXT NOT NULL DEFAULT '', review_note TEXT NOT NULL DEFAULT '', reviewed_at TEXT, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '', UNIQUE(teacher_id, course_id, semester))`); err != nil {
		t.Fatalf("创建测试授课关系表失败: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS courses (id INTEGER PRIMARY KEY AUTOINCREMENT, course_id TEXT NOT NULL UNIQUE, course_name TEXT NOT NULL)`); err != nil {
		t.Fatalf("创建测试课程目录失败: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS course_schedules (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, course_id TEXT NOT NULL, course_name TEXT NOT NULL, semester_code TEXT NOT NULL, weekday INTEGER NOT NULL, start_period INTEGER NOT NULL, end_period INTEGER NOT NULL, weeks_pattern TEXT NOT NULL DEFAULT '1-20', location TEXT, teacher TEXT)`); err != nil {
		t.Fatalf("创建测试课表失败: %v", err)
	}
	if _, err := db.Exec(`UPDATE users SET display_name='刘东良', role='teacher' WHERE id=1`); err != nil {
		t.Fatalf("设置测试教师失败: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO course_schedules (user_id, course_id, course_name, semester_code, weekday, start_period, end_period, location, teacher) VALUES (99, 'CS-206', '网络安全基础', '2026-秋', 2, 3, 4, '明理楼101', ' 刘东良 '), (98, 'CS-206', '网络安全基础', '2026-秋', 4, 5, 6, '明理楼101', '刘东良'), (97, 'CS-999', '他人课程', '2026-秋', 1, 1, 2, '明理楼102', '其他教师')`); err != nil {
		t.Fatalf("写入测试课表失败: %v", err)
	}
	svc := NewTeacherService(nil, repository.NewTeacherCourseRepo(db))
	classes, err := svc.TeachingClasses(context.Background(), 1)
	if err != nil {
		t.Fatalf("查询课表回退失败: %v", err)
	}
	if len(classes) != 1 || classes[0]["course_id"] != "CS-206" || classes[0]["data_source"] != "course_schedules.teacher" {
		t.Fatalf("课表回退结果错误: %#v", classes)
	}
}
