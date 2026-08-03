package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

func setupDataImportTestDB(t *testing.T) *DataImportRepo {
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
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, course_id, semester, grade_type)
		);
		CREATE TABLE IF NOT EXISTS course_schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			course_id TEXT NOT NULL,
			course_name TEXT NOT NULL,
			semester_code TEXT NOT NULL,
			weekday INTEGER NOT NULL,
			start_period INTEGER NOT NULL,
			end_period INTEGER NOT NULL,
			weeks_pattern TEXT NOT NULL DEFAULT '1-20',
			location TEXT,
			teacher TEXT,
			color TEXT DEFAULT '#1565C0',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(user_id, course_id, weekday, start_period, semester_code)
		);
		CREATE TABLE IF NOT EXISTS exam_schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			exam_id TEXT NOT NULL UNIQUE,
			course_id TEXT NOT NULL DEFAULT '',
			course_name TEXT NOT NULL,
			exam_type TEXT NOT NULL DEFAULT 'final',
			date TEXT NOT NULL,
			time_start TEXT NOT NULL,
			time_end TEXT NOT NULL,
			location TEXT NOT NULL,
			seat TEXT DEFAULT '',
			semester TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(course_id, exam_type, semester)
		);
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			role TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT ''
		);
	`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO users (username, role, display_name) VALUES ('stu_g1','student','张三')`)
	return NewDataImportRepo(db)
}

func TestDataImportRepo_Grades(t *testing.T) {
	r := setupDataImportTestDB(t)

	g := &GradeRow{UserID: "1", CourseID: "CS101", CourseName: "数据结构", Semester: "2025-2026-2", Score: 88, GPA: 3.5, Passed: true, Credits: 4}
	created, err := r.UpsertGrade(g)
	if err != nil || !created {
		t.Fatalf("首次写入应为新增 created=%v err=%v", created, err)
	}
	// 幂等更新
	g.Score = 92
	created, err = r.UpsertGrade(g)
	if err != nil || created {
		t.Fatalf("二次写入应为更新 created=%v err=%v", created, err)
	}

	summaries, err := r.ListGradeSummaries()
	if err != nil {
		t.Fatalf("ListGradeSummaries 失败: %v", err)
	}
	if len(summaries) != 1 || summaries[0].Credits != 4 || summaries[0].AvgScore != 92 {
		t.Fatalf("成绩聚合不符: %+v", summaries)
	}
}

func TestDataImportRepo_Schedules(t *testing.T) {
	r := setupDataImportTestDB(t)

	_ = r.UpsertSchedule(&ScheduleRow{UserID: 1, CourseID: "CS101", CourseName: "数据结构", SemesterCode: "2025-2026-2", Weekday: 1, StartPeriod: 1, EndPeriod: 2, Location: "信息楼301", Teacher: "张老师"})
	_ = r.UpsertSchedule(&ScheduleRow{UserID: 1, CourseID: "CS102", CourseName: "操作系统", SemesterCode: "2025-2026-2", Weekday: 1, StartPeriod: 1, EndPeriod: 2, Location: "信息楼301", Teacher: "张老师"})

	list, err := r.ListSchedules("")
	if err != nil {
		t.Fatalf("ListSchedules 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("课表应为 2 条，实际 %d", len(list))
	}
}

func TestDataImportRepo_Exams(t *testing.T) {
	r := setupDataImportTestDB(t)

	_, _ = r.db.Exec(`INSERT INTO exam_schedules (exam_id, course_id, course_name, date, time_start, time_end, location, semester)
		VALUES ('e1','CS101','数据结构','2026-06-15','08:30','10:30','信息楼301','2025-2026-2')`)

	list, err := r.ListExams("")
	if err != nil {
		t.Fatalf("ListExams 失败: %v", err)
	}
	if len(list) != 1 || list[0]["course_name"] != "数据结构" || list[0]["location"] != "信息楼301" {
		t.Fatalf("考试安排不符: %+v", list)
	}
}
