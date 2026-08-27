package app

import (
	"database/sql"
	"testing"

	dbutil "github.com/dll/wxx/server/internal/db"
	_ "modernc.org/sqlite"
)

// TestMigration110_RestoreCampusSteps 用生产 runMigrations 全量跑迁移，
// 验证 110_restore_campus_steps.sql 在真实 splitSQL 下是否插入 12 条已发布节点。
func TestMigration110_RestoreCampusSteps(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db, dbutil.DriverSQLite); err != nil {
		t.Fatalf("runMigrations 失败: %v", err)
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM campus_checkin_steps").Scan(&total); err != nil {
		t.Fatalf("count total: %v", err)
	}
	t.Logf("campus_checkin_steps 总数 = %d", total)

	var published int
	if err := db.QueryRow("SELECT COUNT(*) FROM campus_checkin_steps WHERE status='published'").Scan(&published); err != nil {
		t.Fatalf("count published: %v", err)
	}
	t.Logf("published 数 = %d", published)

	var huifeng, langya int
	db.QueryRow("SELECT COUNT(*) FROM campus_checkin_steps WHERE campus_id='huifeng'").Scan(&huifeng)
	db.QueryRow("SELECT COUNT(*) FROM campus_checkin_steps WHERE campus_id='langya'").Scan(&langya)
	t.Logf("会峰=%d 琅琊=%d", huifeng, langya)

	if published < 12 {
		t.Errorf("期望至少 12 条 published 节点，实际 %d 条 —— 迁移 110 未正确恢复数据", published)
	}
	if huifeng != 6 || langya != 6 {
		t.Errorf("期望会峰6+琅琊6，实际 huifeng=%d langya=%d", huifeng, langya)
	}
}
