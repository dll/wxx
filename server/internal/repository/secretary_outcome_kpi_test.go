package repository

import (
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/testutil"
)

// createNurtureKPITables 为 GetNurtureKPI 构造所需真实表（NewTestDB 已含 users 基础 schema）。
// 这些表在 NewTestDB 的全量迁移之外，故在测试内显式建表。
func createNurtureKPITables(t *testing.T, db *sql.DB) {
	t.Helper()
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS party_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, current_stage TEXT NOT NULL,
			status TEXT DEFAULT 'applicant', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS party_study_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, study_type TEXT NOT NULL, title TEXT NOT NULL,
			duration INTEGER, study_date TEXT, status TEXT DEFAULT 'completed',
			created_by BIGINT NULL, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS talk_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			counselor_id INTEGER NOT NULL, student_id INTEGER NOT NULL DEFAULT 0,
			student_name TEXT NOT NULL DEFAULT '', topic TEXT NOT NULL DEFAULT '',
			emotion TEXT NOT NULL DEFAULT '', content TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '', follow_ups TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'following',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')))`,
		`CREATE TABLE IF NOT EXISTS facility_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT, role TEXT NOT NULL, title TEXT NOT NULL,
			operator_id INTEGER NOT NULL, student_id INTEGER NOT NULL DEFAULT 0,
			occurred_at TEXT NOT NULL, created_at TEXT DEFAULT (CURRENT_TIMESTAMP))`,
		`CREATE TABLE IF NOT EXISTS course_schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT, teacher_id INTEGER NOT NULL DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS competitions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, level TEXT NOT NULL DEFAULT '',
			created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS health_activity_signups (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			activity_id TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT '',
			attended INTEGER NOT NULL DEFAULT 0, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS student_points (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			points INTEGER NOT NULL DEFAULT 0, reason TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '', created_at TEXT DEFAULT (datetime('now','localtime')))`,
		`CREATE TABLE IF NOT EXISTS graduation_outcome (
			id INTEGER PRIMARY KEY AUTOINCREMENT, student_id INTEGER NOT NULL,
			student_name TEXT NOT NULL DEFAULT '', college TEXT NOT NULL DEFAULT '',
			major TEXT NOT NULL DEFAULT '', graduate_year INTEGER NOT NULL DEFAULT 0,
			outcome_type TEXT NOT NULL, employer_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending', submitted_by INTEGER NOT NULL DEFAULT 0,
			submitted_role TEXT NOT NULL DEFAULT '', data_source TEXT NOT NULL DEFAULT 'real',
			created_at TEXT DEFAULT (CURRENT_TIMESTAMP))`,
		`CREATE TABLE IF NOT EXISTS competition_registrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'registered', award_level TEXT,
			college TEXT NOT NULL DEFAULT '', advisor_name TEXT NOT NULL DEFAULT '',
			competition_id INTEGER NOT NULL DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS student_grades (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			passed INTEGER NOT NULL DEFAULT 0, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS student_profile_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			college TEXT NOT NULL DEFAULT '', created_at TEXT DEFAULT (datetime('now')))`,
	}
	for _, d := range ddl {
		if _, err := db.Exec(d); err != nil {
			t.Fatalf("建表失败: %v\nSQL: %s", err, d)
		}
	}
}

// findKPI 按 key 从 KPI 列表中取出一项
func findKPI(kpis []map[string]interface{}, key string) map[string]interface{} {
	for _, k := range kpis {
		if k["key"] == key {
			return k
		}
	}
	return nil
}

// TestGetNurtureKPI_Real 验证「真实源返回 real 数值」：注入真实记录后，各 KPI 应返回真实聚合值。
func TestGetNurtureKPI_Real(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createNurtureKPITables(t, db)

	// 本院范围（owner_id=cs）——先清空基础迁移可能预置的学生，保证计数确定
	_, _ = db.Exec(`DELETE FROM users WHERE role='student'`)
	_, err := db.Exec(`INSERT INTO users (username, role, display_name, owner_scope, owner_id)
		VALUES ('cs_stu1','student','张一','college','cs'),
		       ('cs_stu2','student','李二','college','cs'),
		       ('cs_stu3','student','王三','college','cs'),
		       ('cs_teacher','teacher','赵老师','college','cs'),
		       ('cs_assistant','assistant','钱老师','college','cs')`)
	if err != nil {
		t.Fatalf("插入学生失败: %v", err)
	}
	var uid1, uid2, uid3 int64
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_stu1'`).Scan(&uid1)
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_stu2'`).Scan(&uid2)
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_stu3'`).Scan(&uid3)
	var teacherID, assistantID int64
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_teacher'`).Scan(&teacherID)
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_assistant'`).Scan(&assistantID)

	// 党建：2 名有入党进度（1 正式党员 + 1 预备），另 1 人未在 party_progress
	_, _ = db.Exec(`INSERT INTO party_progress (user_id, current_stage, status) VALUES (?,?,?), (?,?,?)`,
		uid1, "member", "member", uid2, "probation", "probation")
	// 党课学习：3 人次
	_, _ = db.Exec(`INSERT INTO party_study_records (user_id, study_type, title, duration, study_date) VALUES
		(?,?,?,120,'2026-03-01'), (?,?,?,60,'2026-03-02'), (?,?,?,60,'2026-03-03')`,
		uid1, "theory", "党课一", uid2, "practice", "实践", uid3, "meeting", "组织生活")
	// 谈心：2 条（覆盖 2 名学生）
	_, _ = db.Exec(`INSERT INTO talk_records (counselor_id, student_id, student_name) VALUES (100,?,?), (100,?,?)`,
		uid1, "张一", uid2, "李二")
	// 后勤：1 条（由本院教辅登记）
	_, _ = db.Exec(`INSERT INTO facility_records (role, title, operator_id, student_id, occurred_at) VALUES ('lab','开门',?,0,'2026-03-01T08:00:00')`, assistantID)
	// 排课：2 节（由本院教师登记）
	_, _ = db.Exec(`INSERT INTO course_schedules (teacher_id) VALUES (?), (?)`, teacherID, teacherID)
	// 二课：2 人次到场 + 50 分
	_, _ = db.Exec(`INSERT INTO health_activity_signups (user_id, activity_id, status, attended) VALUES (?,?,?,1), (?,?,?,1)`,
		uid1, "a1", "done", uid2, "a2", "done")
	_, _ = db.Exec(`INSERT INTO student_points (user_id, points, reason, source) VALUES (?,30,'活动','other'), (?,20,'活动','other')`, uid1, uid2)
	// 竞赛：1 项获奖 + 竞赛归属（relation 表明级别）
	_, _ = db.Exec(`INSERT INTO competitions (level) VALUES ('national')`)
	var compID int64
	_ = db.QueryRow(`SELECT id FROM competitions LIMIT 1`).Scan(&compID)
	_, _ = db.Exec(`INSERT INTO competition_registrations (user_id, status, award_level, college, competition_id) VALUES (?, 'awarded','first','cs', ?)`, uid1, compID)
	// 毕业去向：3 条 approved（全部就业 → 就业率 100%）：就业+就业+灵活
	_, _ = db.Exec(`INSERT INTO graduation_outcome (student_id, student_name, college, outcome_type, status)
		VALUES (?,?,?,?,'approved'), (?,?,?,?,'approved'), (?,?,?,?,'approved')`,
		uid1, "张一", "cs", "employment", uid2, "李二", "cs", "employment", uid3, "王三", "cs", "flexible")
	// 成绩：2 条全通过 → 100%（需快照表归属本院才计入）
	_, _ = db.Exec(`INSERT INTO student_profile_snapshot (user_id, college) VALUES (?, 'cs'), (?, 'cs'), (?, 'cs')`, uid1, uid2, uid3)
	_, _ = db.Exec(`INSERT INTO student_grades (user_id, passed) VALUES (?,1),(?,1)`, uid1, uid2)

	repo := NewSecretaryOutcomeRepo(db)
	kpis := repo.GetNurtureKPI("cs")
	if kpis == nil {
		t.Fatalf("GetNurtureKPI 返回 nil")
	}

	assertKPI(t, kpis, "nurture.student_total", 3, "real")
	assertKPI(t, kpis, "nurture.party_member", 2, "real")
	assertKPI(t, kpis, "nurture.party_applicant", 2, "real")
	assertKPI(t, kpis, "nurture.party_study", 3, "real")
	assertKPI(t, kpis, "nurture.talk_total", 2, "real")
	assertKPI(t, kpis, "nurture.facility", 1, "real")
	assertKPI(t, kpis, "nurture.course", 2, "real")
	assertKPI(t, kpis, "nurture.second_class", 2, "real")
	assertKPI(t, kpis, "nurture.second_class_points", 50, "real")
	assertKPI(t, kpis, "nurture.award", 1, "real")

	// 毕业去向落实率：3/3 = 100%
	emp := findKPI(kpis, "nurture.employment_rate")
	if emp == nil {
		t.Fatalf("缺少 nurture.employment_rate")
	}
	if emp["data_source"] != "real" {
		t.Fatalf("employment_rate 应为 real，得到 %v", emp["data_source"])
	}
	if v, ok := emp["value"].(float64); !ok || v != 100.0 {
		t.Fatalf("employment_rate 应为 100.0，得到 %v", emp["value"])
	}
	// 升学率：0/3 = 0%
	postgrad := findKPI(kpis, "nurture.postgrad_rate")
	if postgrad == nil || postgrad["value"].(float64) != 0 {
		t.Fatalf("postgrad_rate 应为 0.0，得到 %v", postgrad)
	}
	// 学业通过率：100%
	acad := findKPI(kpis, "nurture.academic_pass")
	if acad == nil || acad["data_source"] != "real" || numOf(acad, "value") != 100.0 {
		t.Fatalf("academic_pass 应为 real 100%%，得到 data_source=%v value=%v", acad["data_source"], acad["value"])
	}

	// 固定「无数据源 → not_available」卡片必须存在，且 value 恒为 nil（不伪造）
	for _, key := range []string{"nurture.intervention_total", "nurture.second_class_pass_rate", "nurture.growth_trend", "nurture.employment_target"} {
		c := findKPI(kpis, key)
		if c == nil {
			t.Fatalf("缺少 not_available 卡片 %s", key)
		}
		if c["data_source"] != "not_available" {
			t.Fatalf("%s data_source 应为 not_available", key)
		}
		if c["value"] != nil {
			t.Fatalf("%s value 应为 nil（绝不伪造），得到 %v", key, c["value"])
		}
		if c["upload_target"] != "kb" {
			t.Fatalf("%s upload_target 应为 kb，得到 %v", key, c["upload_target"])
		}
	}
}

// TestGetNurtureKPI_NotAvailable 验证「无源 → not_available 且 value 空、绝不伪造」：
// 毕业去向表空（无 approved 记录）时，就业率应返回 not_available + value=nil。
func TestGetNurtureKPI_NotAvailable(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createNurtureKPITables(t, db)

	// 注入学生但无任何毕业去向 approved 记录；先清空预置学生保证计数确定
	_, _ = db.Exec(`DELETE FROM users WHERE role='student'`)
	_, err := db.Exec(`INSERT INTO users (username, role, display_name, owner_scope, owner_id)
		VALUES ('cs_stu1','student','张一','college','cs')`)
	if err != nil {
		t.Fatalf("插入学生失败: %v", err)
	}

	repo := NewSecretaryOutcomeRepo(db)
	kpis := repo.GetNurtureKPI("cs")

	// 真实可算的学生数 → real
	assertKPI(t, kpis, "nurture.student_total", 1, "real")

	// 毕业去向无 approved → not_available 且 value=nil
	emp := findKPI(kpis, "nurture.employment_rate")
	if emp == nil {
		t.Fatalf("缺少 nurture.employment_rate")
	}
	if emp["data_source"] != "not_available" {
		t.Fatalf("空去向表 employment_rate 应为 not_available，得到 %v", emp["data_source"])
	}
	if emp["value"] != nil {
		t.Fatalf("空去向表 employment_rate value 应为 nil，得到 %v", emp["value"])
	}
	if emp["upload_target"] != "kb" {
		t.Fatalf("employment_rate upload_target 应为 kb，得到 %v", emp["upload_target"])
	}
}

// assertKPI 断言某 KPI 的 data_source=real 且 value==期望数值。
func assertKPI(t *testing.T, kpis []map[string]interface{}, key string, wantVal int, wantSrc string) {
	t.Helper()
	k := findKPI(kpis, key)
	if k == nil {
		t.Fatalf("缺少 KPI: %s", key)
	}
	if k["data_source"] != wantSrc {
		t.Fatalf("%s data_source 应为 %s，得到 %v", key, wantSrc, k["data_source"])
	}
	got, ok := k["value"].(float64)
	if !ok {
		t.Fatalf("%s value 应为数值，得到 %v", key, k["value"])
	}
	if int(got) != wantVal {
		t.Fatalf("%s value 应为 %d，得到 %v", key, wantVal, got)
	}
}
