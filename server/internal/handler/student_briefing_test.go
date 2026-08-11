package handler

import (
	"database/sql"
	"testing"
	"time"

	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
)

func mustDBExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec fail: %v\n%s", err, query)
	}
}

// TestEnrichBriefingWithRealData 校验今日速览真实数据覆盖逻辑：
// 真实课表/任务/校历事件存在时替换兜底数据；无数据时保留兜底
func TestEnrichBriefingWithRealData(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// 课程/计划/校历相关表（037 迁移未纳入 testutil，此处按 037 结构内联建表）
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS academic_calendars (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			academic_year INTEGER NOT NULL,
			semester INTEGER NOT NULL,
			semester_code TEXT NOT NULL UNIQUE,
			semester_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			register_date TEXT,
			total_weeks INTEGER NOT NULL DEFAULT 20,
			week_start_day TEXT DEFAULT 'monday',
			status TEXT DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(academic_year, semester)
		);
		CREATE TABLE IF NOT EXISTS academic_calendar_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			semester_code TEXT NOT NULL,
			event_name TEXT NOT NULL,
			event_type TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT,
			week_no INTEGER,
			affects_classes INTEGER DEFAULT 0,
			description TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
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
		CREATE TABLE IF NOT EXISTS study_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			plan_type TEXT NOT NULL DEFAULT 'weekly',
			semester_code TEXT,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			goals_json TEXT NOT NULL DEFAULT '[]',
			progress REAL DEFAULT 0,
			ai_generated INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active',
			linked_plan_id INTEGER,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS study_plan_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id INTEGER NOT NULL,
			course_id TEXT,
			course_name TEXT,
			title TEXT NOT NULL,
			description TEXT,
			scheduled_date TEXT,
			scheduled_duration INTEGER DEFAULT 0,
			actual_duration INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			evidence TEXT,
			reflection TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (plan_id) REFERENCES study_plans(id) ON DELETE CASCADE
		);
	`)

	h := NewStudentHandler(nil, db)

	userID := int64(1001)
	mustDBExec(t, db, `INSERT INTO users (id, username, display_name, role, password_hash, consented, created_at, updated_at)
		VALUES (?, 's1001', '测试同学', 'student', 'x', 1, '2026-01-01', '2026-01-01')`, userID)

	// 当前学期校历
	today := time.Now()
	semStart := today.AddDate(0, -1, 0).Format("2006-01-02")
	semEnd := today.AddDate(0, 6, 0).Format("2006-01-02")
	mustDBExec(t, db, `INSERT INTO academic_calendars (academic_year, semester, semester_code, semester_name, start_date, end_date, register_date, total_weeks, status)
		VALUES (2027, 1, '2099-2100-1', '测试学期', ?, ?, ?, 20, 'completed')`, semStart, semEnd, semStart)

	// 今日课表（真实课程 → 覆盖 courses）
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	mustDBExec(t, db, `INSERT INTO course_schedules (user_id, course_id, course_name, semester_code, weekday, start_period, end_period, weeks_pattern, location, teacher)
		VALUES (?, 'cs1', '数据结构', '2099-2100-1', ?, 1, 2, '1-20', 'A101', '王老师')`, userID, weekday)

	// 今日待办 + 已完成任务
	todayStr := today.Format("2006-01-02")
	mustDBExec(t, db, `INSERT INTO study_plans (id, user_id, title, plan_type, start_date, end_date, status, created_at, updated_at)
		VALUES (1, ?, '周计划', 'weekly', ?, ?, 'active', '2026-01-01', '2026-01-01')`, userID, todayStr, todayStr)
	mustDBExec(t, db, `INSERT INTO study_plan_tasks (plan_id, title, scheduled_date, status) VALUES (1, '数据结构作业', ?, 'pending')`, todayStr)
	mustDBExec(t, db, `INSERT INTO study_plan_tasks (plan_id, title, scheduled_date, status) VALUES (1, '已完成任务', ?, 'done')`, todayStr)

	// 未来 3 天考试事件
	futureDate := today.AddDate(0, 0, 3).Format("2006-01-02")
	mustDBExec(t, db, `INSERT INTO academic_calendar_events (semester_code, event_name, event_type, start_date, end_date)
		VALUES ('2099-2100-1', '期中考试', 'exam', ?, ?)`, futureDate, futureDate)

	b := &service.DailyBriefing{
		Courses:    []map[string]interface{}{{"title": "兜底课程"}},
		Deadlines:  []map[string]interface{}{{"title": "兜底待办"}},
		Activities: []map[string]interface{}{{"title": "兜底活动"}},
	}
	h.enrichBriefingWithRealData(b, userID)

	// courses 必须被真实课表替换
	if len(b.Courses) == 0 {
		t.Fatal("enrich 后 courses 不应为空")
	}
	if b.Courses[0]["title"] != "数据结构" {
		t.Errorf("courses 未覆盖为真实课表: %#v", b.Courses)
	}

	// deadlines 保留 pending，过滤 done
	foundPending := false
	for _, d := range b.Deadlines {
		if d["title"] == "数据结构作业" {
			foundPending = true
		}
		if d["title"] == "已完成任务" {
			t.Errorf("已完成任务不应出现在待办中: %#v", b.Deadlines)
		}
	}
	if !foundPending {
		t.Errorf("未找到待办 '数据结构作业': %#v", b.Deadlines)
	}

	// activities 出现真实校历事件
	foundEvent := false
	for _, a := range b.Activities {
		if a["title"] == "期中考试" {
			foundEvent = true
		}
	}
	if !foundEvent {
		t.Errorf("activities 未包含真实校历事件: %#v", b.Activities)
	}

	// 另一用户无任何数据 → 保留兜底
	emptyUser := int64(2002)
	mustDBExec(t, db, `INSERT INTO users (id, username, display_name, role, password_hash, consented, created_at, updated_at)
		VALUES (?, 's2002', '空同学', 'student', 'x', 1, '2026-01-01', '2026-01-01')`, emptyUser)
	b2 := &service.DailyBriefing{
		Courses:    []map[string]interface{}{{"title": "兜底课程"}},
		Deadlines:  []map[string]interface{}{{"title": "兜底待办"}},
		Activities: []map[string]interface{}{{"title": "兜底活动"}},
	}
	h.enrichBriefingWithRealData(b2, emptyUser)
	if len(b2.Courses) == 0 || b2.Courses[0]["title"] != "兜底课程" {
		t.Errorf("无数据用户应保留兜底 courses: %#v", b2.Courses)
	}
	if len(b2.Deadlines) == 0 || b2.Deadlines[0]["title"] != "兜底待办" {
		t.Errorf("无数据用户应保留兜底 deadlines: %#v", b2.Deadlines)
	}
}
