package db

// 临时校验：确认 095 迁移（P2 教师作业信息发布 homework 表）在 ToMySQL 转换后不产生破坏性输出（方言兼容）。
// 迁移核心是 CREATE TABLE + CREATE INDEX，均应被 ToMySQL 干净转换，SQLite/MySQL 双跑。

import (
	"os"
	"strings"
	"testing"
)

func TestToMySQL_Migration095(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/095_homework.sql")
	if err != nil {
		raw, err = os.ReadFile("../../../migrations/095_homework.sql")
	}
	if err != nil {
		t.Skipf("找不到 095 迁移文件: %v", err)
	}
	content := string(raw)

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
		t.Fatalf("095 转换后无任何语句输出")
	}

	joined := strings.Join(converted, ";\n")
	// 不应残留 SQLite 专有语法
	for _, banned := range []string{"AUTOINCREMENT", "datetime('now'", "ON CONFLICT"} {
		if strings.Contains(strings.ToLower(joined), strings.ToLower(banned)) {
			t.Errorf("095 MySQL 转换后仍含 SQLite 专有语法 %q:\n%s", banned, joined)
		}
	}
	// MySQL 应产出 BIGINT 自增主键 + DATETIME + teacher_id/course_id/semester/title 列 + 唯一约束
	if !strings.Contains(joined, "BIGINT PRIMARY KEY AUTO_INCREMENT") {
		t.Errorf("MySQL 转换应产出 BIGINT 自增主键，实际:\n%s", joined)
	}
	if !strings.Contains(joined, "DATETIME") {
		t.Errorf("MySQL 转换应产出 DATETIME 列，实际:\n%s", joined)
	}
	for _, col := range []string{"teacher_id", "course_id", "semester", "title", "status", "description"} {
		if !strings.Contains(strings.ToLower(joined), strings.ToLower(col)) {
			t.Errorf("MySQL 转换应含列 %q，实际:\n%s", col, joined)
		}
	}
	if !strings.Contains(strings.ToUpper(joined), "UNIQUE") {
		t.Errorf("MySQL 转换应含唯一约束(幂等键)，实际:\n%s", joined)
	}
	t.Logf("095 → MySQL 转换语句:\n%s", joined)
}
