package db

// 临时校验：确认 091 迁移在 ToMySQL 转换后不产生破坏性输出（方言兼容）。
// 迁移核心是 CREATE TABLE + CREATE INDEX，均应被 ToMySQL 干净转换。

import (
	"os"
	"strings"
	"testing"
)

func TestToMySQL_Migration091(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/091_student_profile_snapshot_history.sql")
	if err != nil {
		// 从 repository 测试目录上下文也尝试一次
		raw, err = os.ReadFile("../../../migrations/091_student_profile_snapshot_history.sql")
	}
	if err != nil {
		t.Skipf("找不到 091 迁移文件: %v", err)
	}
	content := string(raw)

	// 逐条转换（与 migrate runner 同逻辑：按分号分割，FTS5 跳过）
	stmts := splitSQLStmt(content)
	converted := []string{}
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(strings.ToUpper(s), "PRAGMA") {
			continue
		}
		m := ToMySQL(s)
		if m == "" {
			continue
		}
		converted = append(converted, m)
	}
	if len(converted) == 0 {
		t.Fatalf("091 转换后无任何语句输出")
	}

	joined := strings.Join(converted, ";\n")
	// 不应残留 SQLite 专有语法
	for _, banned := range []string{"AUTOINCREMENT", "datetime('now'", "ON CONFLICT"} {
		if strings.Contains(strings.ToLower(joined), strings.ToLower(banned)) {
			t.Errorf("091 MySQL 转换后仍含 SQLite 专有语法 %q:\n%s", banned, joined)
		}
	}
	// MySQL 应转成 BIGINT 自增主键 + DATETIME + 去重唯一索引名（IF NOT EXISTS 被剥离）
	if !strings.Contains(joined, "BIGINT PRIMARY KEY AUTO_INCREMENT") {
		t.Errorf("MySQL 转换应产出 BIGINT 自增主键，实际:\n%s", joined)
	}
	if !strings.Contains(joined, "DATETIME") {
		t.Errorf("MySQL 转换应产出 DATETIME 列，实际:\n%s", joined)
	}
	t.Logf("091 → MySQL 转换语句:\n%s", joined)
}

// splitSQLStmt 简易按分号分割（091 无触发器，简单实现即可）
func splitSQLStmt(content string) []string {
	var out []string
	cur := strings.Builder{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cur.WriteString(line)
		cur.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}
