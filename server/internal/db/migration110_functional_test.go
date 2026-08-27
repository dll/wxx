package db

// 迁移 110（恢复 campus_checkin_steps 12 条 published 种子）功能回归（reviewer 回修）：
// 在 SQLite 内存库上重建 048 schema，验证：
//   - H1：表内已存在同 (campus_id, step_order) 的多条（含 status 不同）数据时，
//     110 不建索引、不报 duplicate entry、不丢数据，幂等执行；
//   - H2：审核流 (draft→pending_review→published) 同 step_order 多 status 共存不被破坏。

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func migration110Load(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../migrations/110_restore_campus_steps.sql")
	if err != nil {
		raw, err = os.ReadFile("../../../migrations/110_restore_campus_steps.sql")
	}
	if err != nil {
		t.Fatalf("找不到 110 迁移文件: %v", err)
	}
	return string(raw)
}

func migration110ExecAll(t *testing.T, db *sql.DB, sql string) {
	t.Helper()
	for _, stmt := range splitSQLStmt(sql) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("110 执行失败: %v\nSQL: %s", err, stmt)
		}
	}
}

// TestMigration110_Functional_IdempotentAndWorkflowSafe 覆盖 H1 + H2：
// 1. 先播种 048 表结构（含 12 条 published 种子）。
// 2. 模拟 079 清空后再建部分节点（含同 step_order 的 draft 与 published 多版本）。
// 3. 执行 110，断言：不报错、published 种子恢复、既有 draft 节点未被删除/覆盖。
// 4. 重复执行 110，断言幂等（不产生重复 published 种子）。
func TestMigration110_Functional_IdempotentAndWorkflowSafe(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	// 1. 建表（048 的 campus_checkin_steps，不含唯一索引）
	raw048, err := os.ReadFile("../../migrations/048_campus_map_steps.sql")
	if err != nil {
		raw048, err = os.ReadFile("../../../migrations/048_campus_map_steps.sql")
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range splitSQLStmt(string(raw048)) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("048 建表失败: %v\nSQL: %s", err, stmt)
		}
	}

	// 2. 模拟 079 清空
	if _, err := db.Exec(`DELETE FROM campus_checkin_steps`); err != nil {
		t.Fatal(err)
	}

	// 3. 模拟管理员用重复 step_order 重建部分节点：
	//    - 已发布 step_order=1（重建了）
	//    - 同 step_order=1 的一个 draft（H2：审核流多版本并存）
	//    - 一个 pending_review 的 step_order=2（H2）
	if _, err := db.Exec(`INSERT INTO campus_checkin_steps
		(campus_id,step_order,title,location,lat,lng,status)
		VALUES('huifeng',1,'重建节点','会峰南门',32.0,118.0,'published')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO campus_checkin_steps
		(campus_id,step_order,title,location,lat,lng,status)
		VALUES('huifeng',1,'新建草稿','会峰南门草稿',32.0,118.0,'draft')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO campus_checkin_steps
		(campus_id,step_order,title,location,lat,lng,status)
		VALUES('huifeng',2,'待审核节点','待审',32.0,118.0,'pending_review')`); err != nil {
		t.Fatal(err)
	}

	// 4. 执行 110（第一次）
	migration110ExecAll(t, db, migration110Load(t))

	// 5. 验证 published 种子恢复：会峰 step_order=1 的 published 现有 1 条（重建的那条）
	//    注意 110 用 WHERE NOT EXISTS 守卫：会峰 step_order=1 已有 published → 不插入第 13 条
	var huifengPublished1 int
	if err := db.QueryRow(`SELECT COUNT(*) FROM campus_checkin_steps
		WHERE campus_id='huifeng' AND step_order=1 AND status='published'`).Scan(&huifengPublished1); err != nil {
		t.Fatal(err)
	}
	if huifengPublished1 != 1 {
		t.Errorf("H1/H2: 会峰 step_order=1 已有 1 条 published，110 不应重复插入（实际 %d 条）", huifengPublished1)
	}

	// 6. 验证既有 draft/pending_review 未被删除（不丢数据）
	var draftCount, pendingCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM campus_checkin_steps WHERE status='draft'`).Scan(&draftCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM campus_checkin_steps WHERE status='pending_review'`).Scan(&pendingCount); err != nil {
		t.Fatal(err)
	}
	if draftCount != 1 || pendingCount != 1 {
		t.Errorf("H1: 既有 draft/pending_review 节点被误删（draft=%d pending=%d）", draftCount, pendingCount)
	}

	// 7. 其余 11 条 published 种子（会峰 2-6 共 5 条 + 琅琊 6 条）全部恢复
	var publishedTotal int
	if err := db.QueryRow(`SELECT COUNT(*) FROM campus_checkin_steps WHERE status='published'`).Scan(&publishedTotal); err != nil {
		t.Fatal(err)
	}
	// 会峰 published：step1 已存在(1) + step2 待审已占位不冲突(0 published) + step3-6(4) = 5 条会峰 published？
	// 精确核对：会峰 1(重建已存在)+2(无 published，110 插入)+3+4+5+6 = 6 条会峰 published
	//          琅琊 1-6 = 6 条 published
	//          合计 12 条 published
	if publishedTotal != 12 {
		t.Errorf("H1: published 种子应恢复 12 条（实际 %d 条）", publishedTotal)
	}

	// 8. 重复执行 110，验证幂等（published 仍 12 条，无新增）
	migration110ExecAll(t, db, migration110Load(t))
	if err := db.QueryRow(`SELECT COUNT(*) FROM campus_checkin_steps WHERE status='published'`).Scan(&publishedTotal); err != nil {
		t.Fatal(err)
	}
	if publishedTotal != 12 {
		t.Errorf("H1 幂等: 重复执行后 published 应为 12 条（实际 %d 条）", publishedTotal)
	}
	t.Logf("[OK] 110 功能回归通过：不建索引、幂等、不丢数据、不阻碍审核流。")
}
