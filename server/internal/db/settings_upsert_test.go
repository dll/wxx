package db

import (
	"strings"
	"testing"
)

// TestAdaptSettingsUpsert_MySQL 回归测试（2026-09-01 用户反馈④「管理开关无效」）：
// settings_repo.Upsert 的 SQL 使用 excluded.value 与 ON CONFLICT 方言，
// 必须经 AdaptForDriver 正确转换为 MySQL 的 ON DUPLICATE KEY UPDATE 与 VALUES()，
// 否则 MySQL 下写入静默失败，开关回读永远为默认「开」，无法显示「关」。
func TestAdaptSettingsUpsert_MySQL(t *testing.T) {
	// 与 repository.SettingsRepo.Upsert 相同的语句（去掉反引号包裹 value）
	stmt := "INSERT INTO system_settings (`key`, value, updated_by, updated_at) " +
		"VALUES (?, ?, ?, CURRENT_TIMESTAMP) " +
		"ON CONFLICT(`key`) DO UPDATE SET value=excluded.value, updated_by=excluded.updated_by, updated_at=CURRENT_TIMESTAMP"

	out := AdaptForDriver(stmt, DriverMySQL)
	upper := strings.ToUpper(out)

	for _, must := range []string{"ON DUPLICATE KEY UPDATE", "VALUES(VALUE)", "VALUES(UPDATED_BY)"} {
		if !strings.Contains(upper, must) {
			t.Errorf("MySQL 转换后缺少 %q:\n%s", must, out)
		}
	}
	if strings.Contains(upper, "ON CONFLICT") {
		t.Errorf("MySQL 转换后残留 ON CONFLICT（应转为 ON DUPLICATE KEY UPDATE）:\n%s", out)
	}
	if strings.Contains(upper, "EXCLUDED.") {
		t.Errorf("MySQL 转换后残留 excluded. 别名（应转为 VALUES()）:\n%s", out)
	}
}

// TestToMySQL_OnConflictDoNothing 迁移文件中的 ON CONFLICT(...) DO NOTHING
// 需整句改写为 INSERT IGNORE，保证 114_user_roles.sql 等迁移在全新 MySQL 安装可执行。
func TestToMySQL_OnConflictDoNothing(t *testing.T) {
	sql := `INSERT INTO user_roles (user_id, role, granted_by)
SELECT id, 'college_admin', 'migration_114'
FROM users WHERE username = '120001'
ON CONFLICT(user_id, role) DO NOTHING`

	out := ToMySQL(sql)
	t.Logf("转换结果:\n%s", out)

	if out == "" {
		t.Fatal("ToMySQL 返回空串")
	}
	upper := strings.ToUpper(out)
	if !strings.Contains(upper, "INSERT IGNORE INTO") {
		t.Errorf("应改写为 INSERT IGNORE INTO:\n%s", out)
	}
	if strings.Contains(upper, "ON CONFLICT") {
		t.Errorf("应移除 ON CONFLICT ... DO NOTHING:\n%s", out)
	}
	// SELECT 体必须保留
	if !strings.Contains(upper, "SELECT ID") || !strings.Contains(upper, "FROM USERS") {
		t.Errorf("SELECT 体丢失:\n%s", out)
	}
}

// TestToMySQL_OnConflictDoNothing_WithSemicolon 回归测试（2026-09-07 生产 114 迁移 MySQL 1064）：
// splitSQL 保留语句尾部分号，onConflictNothingRe 原用 $ 锚定导致带 ';' 的语句不匹配、
// 转换被跳过，生产报 "near 'CONFLICT(user_id, role) DO NOTHING'" 语法错误。
func TestToMySQL_OnConflictDoNothing_WithSemicolon(t *testing.T) {
	sql := `INSERT INTO user_roles (user_id, role, granted_by)
SELECT id, 'teacher', 'migration_114'
FROM users WHERE username = '120001'
ON CONFLICT(user_id, role) DO NOTHING;`

	out := ToMySQL(sql)
	t.Logf("转换结果:\n%s", out)

	upper := strings.ToUpper(out)
	if !strings.Contains(upper, "INSERT IGNORE INTO") {
		t.Errorf("带尾部分号的语句应改写为 INSERT IGNORE INTO:\n%s", out)
	}
	if strings.Contains(upper, "ON CONFLICT") {
		t.Errorf("应移除 ON CONFLICT ... DO NOTHING:\n%s", out)
	}
	if !strings.Contains(upper, "FROM USERS") {
		t.Errorf("SELECT 体丢失:\n%s", out)
	}
}

// TestToMySQL_OnConflictDoNothing_NoMatch 普通 INSERT（无 ON CONFLICT）不应被改写
func TestToMySQL_OnConflictDoNothing_NoMatch(t *testing.T) {
	sql := `INSERT INTO kb_resources (resource_id) VALUES ('x')`
	out := ToMySQL(sql)
	upper := strings.ToUpper(out)
	if strings.Contains(upper, "INSERT IGNORE") {
		t.Errorf("普通 INSERT 不应被改写为 INSERT IGNORE:\n%s", out)
	}
	if !strings.Contains(upper, "INSERT INTO") {
		t.Errorf("普通 INSERT 内容丢失:\n%s", out)
	}
}
