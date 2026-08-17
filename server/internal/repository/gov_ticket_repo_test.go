package repository

import (
	"database/sql"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

// createGovTicketTables 为督办工单构造所需真实表（090 迁移不在 NewTestDB 的全量清单内，故显式建表）。
func createGovTicketTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS gov_tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_no TEXT NOT NULL,
			title TEXT NOT NULL,
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
			deadline TEXT NOT NULL DEFAULT '',
			remark TEXT NOT NULL DEFAULT '',
			created_by BIGINT NOT NULL DEFAULT 0,
			created_by_role TEXT NOT NULL DEFAULT '',
			closed_by BIGINT NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			closed_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS gov_ticket_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id BIGINT NOT NULL,
			action TEXT NOT NULL,
			operator_id BIGINT NOT NULL DEFAULT 0,
			operator_name TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')))`,
	}
	for _, d := range ddl {
		if _, err := db.Exec(d); err != nil {
			t.Fatalf("建表失败: %v\nSQL: %s", err, d)
		}
	}
}

// TestGovTicketRepo_CreateGet 验证创建→查询闭环：字段持久化正确，状态缺省 pending。
func TestGovTicketRepo_CreateGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createGovTicketTables(t, db)

	repo := NewGovTicketRepo(db)
	id, err := repo.Create(&model.GovTicket{
		Title: "督办二课积分补录", Category: "supplement", SourceType: "kpi",
		SourceKey: "nurture.second_class_points", SourceDesc: "二课积分真实存在但需核验",
		DataSource: "not_available", Priority: "high", Status: "pending",
		College: "cs", CreatedBy: 1, CreatedByRole: "college_admin",
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if id <= 0 {
		t.Fatalf("id 应为正数，得到 %d", id)
	}
	got, err := repo.GetByID(id)
	if err != nil || got == nil {
		t.Fatalf("查询失败: err=%v got=%v", err, got)
	}
	if got.Title != "督办二课积分补录" || got.SourceKey != "nurture.second_class_points" {
		t.Fatalf("字段持久化不符: %+v", got)
	}
	if got.Status != "pending" || got.Category != "supplement" {
		t.Fatalf("状态/类别缺省不符: status=%s category=%s", got.Status, got.Category)
	}
	if got.TicketNo == "" {
		t.Fatalf("ticket_no 未自动生成")
	}
}

// TestGovTicketRepo_StatusFlow 验证状态流转 + 关闭记录 closed_by/closed_at。
func TestGovTicketRepo_StatusFlow(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createGovTicketTables(t, db)

	repo := NewGovTicketRepo(db)
	id, _ := repo.Create(&model.GovTicket{Title: "流转测试", CreatedBy: 1, CreatedByRole: "college_admin"})

	// pending -> processing
	if err := repo.UpdateStatus(id, 2, "王辅导员", "processing", "开始处理"); err != nil {
		t.Fatalf("转 processing 失败: %v", err)
	}
	// processing -> completed
	if err := repo.UpdateStatus(id, 2, "王辅导员", "completed", "已完成处理"); err != nil {
		t.Fatalf("转 completed 失败: %v", err)
	}
	got, _ := repo.GetByID(id)
	if got.Status != "completed" {
		t.Fatalf("状态应为 completed，得到 %s", got.Status)
	}
	if got.ClosedBy == 0 {
		t.Fatalf("completed 应记录 closed_by，得到 0")
	}
	if got.ClosedAt == nil || *got.ClosedAt == "" {
		t.Fatalf("completed 应记录 closed_at")
	}

	logs, err := repo.ListLogs(id)
	if err != nil {
		t.Fatalf("查询日志失败: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("应至少 2 条操作日志，得到 %d", len(logs))
	}
	// 日志按时间升序：processing 先于 completed
	if logs[len(logs)-1].Action != "completed" {
		t.Fatalf("最后一条日志应为 completed，得到 %s", logs[len(logs)-1].Action)
	}
}

// TestGovTicketRepo_ListAndAssign 验证分派与责任人过滤。
func TestGovTicketRepo_ListAndAssign(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createGovTicketTables(t, db)

	repo := NewGovTicketRepo(db)
	id1, _ := repo.Create(&model.GovTicket{Title: "A", College: "cs", CreatedBy: 1, CreatedByRole: "college_admin"})
	id2, _ := repo.Create(&model.GovTicket{Title: "B", College: "cs", CreatedBy: 1, CreatedByRole: "college_admin"})

	if err := repo.Assign(id1, 10, "counselor", "王辅导员", "2026-09-01", 1, "李书记"); err != nil {
		t.Fatalf("分派失败: %v", err)
	}
	got, _ := repo.GetByID(id1)
	if got.AssigneeID != 10 || got.AssigneeName != "王辅导员" {
		t.Fatalf("分派未持久化: %+v", got)
	}

	// 责任人视角：仅看到分派给 id=10 的
	mine, total, err := repo.List("", "", "", 10, 0, 20)
	if err != nil {
		t.Fatalf("责任人列表失败: %v", err)
	}
	if total != 1 || len(mine) != 1 || mine[0].ID != id1 {
		t.Fatalf("责任人应只看到 1 张工单，total=%d len=%d", total, len(mine))
	}

	// 管理端：按学院看到全部
	all, at, err := repo.List("", "cs", "", 0, 0, 20)
	if err != nil || at != 2 || len(all) != 2 {
		t.Fatalf("管理端应按学院看到 2 张，total=%d len=%d err=%v", at, len(all), err)
	}

	// 状态统计
	stats, _ := repo.CountByStatus("cs")
	if stats["pending"] != 2 {
		t.Fatalf("pending 应为 2，得到 %d", stats["pending"])
	}
	_ = id2
}
