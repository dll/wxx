package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// createGovTicketTables 为督办工单构造所需真实表（090 迁移不在 NewTestDB 清单内）。
func createGovTicketTables(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, d := range []string{
		`CREATE TABLE IF NOT EXISTS gov_tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_no TEXT NOT NULL, title TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'insight',
			source_type TEXT NOT NULL DEFAULT 'insight',
			source_key TEXT NOT NULL DEFAULT '',
			source_desc TEXT NOT NULL DEFAULT '',
			data_source TEXT NOT NULL DEFAULT 'not_available',
			priority TEXT NOT NULL DEFAULT 'normal',
			status TEXT NOT NULL DEFAULT 'pending',
			college TEXT NOT NULL DEFAULT '',
			assignee_role TEXT NOT NULL DEFAULT '',
			assignee_id BIGINT NOT NULL DEFAULT 0,
			assignee_name TEXT NOT NULL DEFAULT '',
			deadline TEXT NOT NULL DEFAULT '', remark TEXT NOT NULL DEFAULT '',
			created_by BIGINT NOT NULL DEFAULT 0, created_by_role TEXT NOT NULL DEFAULT '',
			closed_by BIGINT NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			closed_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS gov_ticket_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id BIGINT NOT NULL, action TEXT NOT NULL,
			operator_id BIGINT NOT NULL DEFAULT 0, operator_name TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')))`,
	} {
		if _, err := db.Exec(d); err != nil {
			t.Fatalf("建表失败: %v", err)
		}
	}
}

// createNurtureKPITables 为 D5-1 KPI 联动测试构造所需真实表（users 由 NewTestDB 提供）。
func createNurtureKPITables(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, d := range []string{
		`CREATE TABLE IF NOT EXISTS party_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			current_stage TEXT NOT NULL, status TEXT DEFAULT 'applicant',
			created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS party_study_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			study_type TEXT NOT NULL, title TEXT NOT NULL, duration INTEGER,
			study_date TEXT, status TEXT DEFAULT 'completed', created_by BIGINT NULL,
			created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS talk_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT, counselor_id INTEGER NOT NULL,
			student_id INTEGER NOT NULL DEFAULT 0, student_name TEXT NOT NULL DEFAULT '',
			topic TEXT NOT NULL DEFAULT '', emotion TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '', summary TEXT NOT NULL DEFAULT '',
			follow_ups TEXT NOT NULL DEFAULT '[]', status TEXT NOT NULL DEFAULT 'following',
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
			competition_id INTEGER NOT NULL DEFAULT 0, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS student_grades (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			passed INTEGER NOT NULL DEFAULT 0, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS student_profile_snapshot (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL,
			college TEXT NOT NULL DEFAULT '', created_at TEXT DEFAULT (datetime('now')))`,
	} {
		if _, err := db.Exec(d); err != nil {
			t.Fatalf("建表失败: %v", err)
		}
	}
}

// TestGovTicketService_FullFlow 验证创建→分派→状态流转闭环（纯新增后端逻辑）。
func TestGovTicketService_FullFlow(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createGovTicketTables(t, db) // repository 包测试辅助（同包可见）

	svc := NewGovTicketService(repository.NewGovTicketRepo(db), repository.NewSecretaryOutcomeRepo(db))
	ctx := context.Background()

	// 1. 创建（治理洞察类）
	id, err := svc.CreateTicket(ctx, &repository.GovTicketCreateReq{
		Title: "督办课程质量风险整改", Category: "insight", SourceType: "insight",
		SourceKey: "course_quality", SourceDesc: "课程通过率偏低，课程质量分析识别到风险课程",
		Priority: "high", College: "cs", CreatedBy: 1, CreatedByRole: "college_admin", CreatedByName: "李书记",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if id <= 0 {
		t.Fatalf("id 应为正数: %d", id)
	}

	// 2. 分派
	if err := svc.Assign(ctx, id, 10, "counselor", "王辅导员", "2026-09-01", 1, "李书记"); err != nil {
		t.Fatalf("分派失败: %v", err)
	}
	got, _, err := svc.Get(ctx, id, 1, true) // 管理端查看
	if err != nil {
		t.Fatalf("管理端查看失败: %v", err)
	}
	if got.AssigneeName != "王辅导员" || got.AssigneeRole != "counselor" {
		t.Fatalf("分派未生效: %+v", got)
	}

	// 3. 责任人推进：pending -> processing -> completed
	if err := svc.UpdateStatus(ctx, id, 10, "王辅导员", "processing", "已制定整改"); err != nil {
		t.Fatalf("转 processing 失败: %v", err)
	}
	if err := svc.UpdateStatus(ctx, id, 10, "王辅导员", "completed", "整改完成"); err != nil {
		t.Fatalf("转 completed 失败: %v", err)
	}
	got2, _, _ := svc.Get(ctx, id, 1, true)
	if got2.Status != "completed" {
		t.Fatalf("状态应为 completed: %s", got2.Status)
	}
	if got2.ClosedAt == nil || *got2.ClosedAt == "" {
		t.Fatalf("completed 应记录 closed_at")
	}

	// 4. 完结态不可回退
	if err := svc.UpdateStatus(ctx, id, 10, "王辅导员", "processing", "尝试回退"); err == nil {
		t.Fatalf("完结工单回退应报错")
	}

	// 5. 责任人视角：本人看到，他人看不到
	mine, mTotal, _ := svc.ListMine(ctx, 10, "", 0, 20)
	if mTotal != 1 || len(mine) != 1 {
		t.Fatalf("责任人应看到 1 张工单: total=%d len=%d", mTotal, len(mine))
	}
	if _, _, err := svc.Get(ctx, id, 999, false); err == nil {
		t.Fatalf("非责任人查看应报错")
	}

	// 6. 管理端列表 + 统计
	all, aTotal, _ := svc.List(ctx, "completed", "cs", "", 0, 20)
	if aTotal != 1 || len(all) != 1 {
		t.Fatalf("管理端应看到 1 张 completed: total=%d len=%d", aTotal, len(all))
	}
	stats, _ := svc.Stats(ctx, "cs")
	if stats["completed"] != 1 {
		t.Fatalf("completed 统计应为 1: %v", stats)
	}
}

// TestGovTicketService_CreateFromKPI 验证 D5-1 联动：not_available 补料 KPI 一键生成补料工单，
// 且 real 数据源指标拒绝生成（诚实边界，不伪造）。
func TestGovTicketService_CreateFromKPI(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createGovTicketTables(t, db)
	createNurtureKPITables(t, db) // 复用 D5-1 聚合表

	// 无毕业去向 approved 记录 → employment_rate 为 not_available + upload_target=kb
	_, _ = db.Exec(`DELETE FROM users WHERE role='student'`)
	_, err := db.Exec(`INSERT INTO users (username, role, display_name, owner_scope, owner_id)
		VALUES ('cs_stu1','student','张一','college','cs')`)
	if err != nil {
		t.Fatalf("插入学生失败: %v", err)
	}

	svc := NewGovTicketService(repository.NewGovTicketRepo(db), repository.NewSecretaryOutcomeRepo(db))
	ctx := context.Background()

	// 1. 从 not_available 补料 KPI 生成工单（本院 cs 范围）
	id, err := svc.CreateFromKPI(ctx, "nurture.employment_rate", "cs", &repository.GovTicketCreateReq{
		CreatedBy: 1, CreatedByRole: "college_admin", CreatedByName: "李书记",
		AssigneeID: 10, AssigneeName: "王辅导员", AssigneeRole: "counselor",
	})
	if err != nil {
		t.Fatalf("从 KPI 生成工单失败: %v", err)
	}
	got, _, err := svc.Get(ctx, id, 1, true)
	if err != nil {
		t.Fatalf("查询工单失败: %v", err)
	}
	if got.Category != "supplement" {
		t.Fatalf("类别应为 supplement: %s", got.Category)
	}
	if got.SourceType != "kpi" || got.SourceKey != "nurture.employment_rate" {
		t.Fatalf("来源应记录 KPI: src=%s key=%s", got.SourceType, got.SourceKey)
	}
	if got.DataSource != "not_available" {
		t.Fatalf("DataSource 应沿用 not_available: %s", got.DataSource)
	}
	if got.AssigneeName != "王辅导员" {
		t.Fatalf("分派信息应生效: %s", got.AssigneeName)
	}
	if got.College != "cs" {
		t.Fatalf("工单应落本院: %s", got.College)
	}

	// 2. 诚实边界：real 指标（已 available）拒绝生成补料工单
	// nurture.student_total 需有真实学生（已有 1 名）→ data_source=real，不应生成补料工单
	if _, err := svc.CreateFromKPI(ctx, "nurture.student_total", "cs", &repository.GovTicketCreateReq{
		CreatedBy: 1, CreatedByRole: "college_admin", CreatedByName: "李书记",
	}); err == nil {
		t.Fatalf("real 指标应拒绝生成补料工单")
	}

	// 3. 不存在的 KPI 拒绝
	if _, err := svc.CreateFromKPI(ctx, "nurture.not_exist", "cs", &repository.GovTicketCreateReq{
		CreatedBy: 1, CreatedByRole: "college_admin", CreatedByName: "李书记",
	}); err == nil {
		t.Fatalf("未知 KPI 应拒绝")
	}
}
