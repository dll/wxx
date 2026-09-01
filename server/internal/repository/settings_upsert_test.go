package repository

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestSettingsUpsert_Persist 验证 Upsert 在 SQLite 下可重复写入且值持久化。
// 回归（2026-09-01 用户反馈④「管理开关无效」）：修复前 settings_repo.Upsert 的
// SQL 带 `excluded."value"` 反引号，MySQL 方言适配正则匹配不到，导致生产 MySQL
// 下所有开关写入静默失败，回读永远为默认「开」，无法显示「关」。
// 本用例确保能写入 false（显示「关」）并在 feature.* 前缀读取中取回。
func TestSettingsUpsert_Persist(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE system_settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		updated_by TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
		updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
	)`); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	repo := NewSettingsRepo(db)

	if err := repo.Upsert("feature.guest_register", "true", "test"); err != nil {
		t.Fatalf("首次 Upsert(true) 失败: %v", err)
	}
	if v, _ := repo.Get("feature.guest_register"); v != "true" {
		t.Fatalf("读回应为 true，得到 %q", v)
	}

	// 关键：重复写入 false —— 修复前会失败/回读 true
	if err := repo.Upsert("feature.guest_register", "false", "test"); err != nil {
		t.Fatalf("二次 Upsert(false) 失败: %v", err)
	}
	if v, _ := repo.Get("feature.guest_register"); v != "false" {
		t.Fatalf("关闭后读回应为 false（显示「关」），得到 %q", v)
	}

	// 多键批量：feature 前缀能取回
	if err := repo.Upsert("feature.other", "1", "test"); err != nil {
		t.Fatalf("其他开关写入失败: %v", err)
	}
	m, err := repo.GetByPrefix("feature.")
	if err != nil {
		t.Fatalf("GetByPrefix 失败: %v", err)
	}
	if m["feature.guest_register"] != "false" || m["feature.other"] != "1" {
		t.Fatalf("GetByPrefix 结果异常: %v", m)
	}
}
