// Package testutil 提供测试辅助函数（内存 SQLite、迁移执行等）
package testutil

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite" // SQLite 驱动（含 FTS5）
)

// NewTestDB 创建内存 SQLite 数据库并执行结构迁移脚本
// 返回 *sql.DB，测试结束后调用方负责 defer db.Close()
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}

	// 执行 001_init.sql 迁移（基础 schema）
	migrationPath := resolveMigrationPath(t, "001_init.sql")
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("读取迁移文件失败: %v", err)
	}
	execMigrationSQL(t, db, string(sqlContent))

	return db
}

// NewTestDBFull 创建内存 SQLite 数据库并执行全部迁移（含种子数据）
func NewTestDBFull(t *testing.T) *sql.DB {
	t.Helper()

	db := NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 执行种子数据迁移
	seedPath := resolveMigrationPath(t, "002_test_data.sql")
	seedContent, err := os.ReadFile(seedPath)
	if err != nil {
		t.Logf("跳过种子数据: %v", err)
		return db
	}
	execMigrationSQL(t, db, string(seedContent))

	return db
}

// resolveMigrationPath 在候选路径中查找迁移文件
func resolveMigrationPath(t *testing.T, name string) string {
	t.Helper()
	candidates := []string{
		"../../migrations/" + name,
		"../../../migrations/" + name,
		"migrations/" + name,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Fatalf("找不到迁移文件: %s (已尝试 ../../migrations/ 和 migrations/)", name)
	return ""
}

// execMigrationSQL 解析并执行迁移 SQL（按分号分割，处理触发器复合语句）
func execMigrationSQL(t *testing.T, db *sql.DB, content string) {
	t.Helper()

	for _, stmt := range SplitSQL(content) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("执行 SQL 失败: %v\nSQL: %s", err, truncateSQL(stmt, 200))
		}
	}
}

// SplitSQL 按分号分割 SQL 语句，正确处理触发器复合语句
func SplitSQL(content string) []string {
	var statements []string
	var current strings.Builder
	inTrigger := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// 跳过纯注释行
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		// 检测触发器开始/结束
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "CREATE TRIGGER") {
			inTrigger = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		// 触发器以 END; 结束
		if inTrigger && strings.HasSuffix(trimmed, "END;") {
			statements = append(statements, current.String())
			current.Reset()
			inTrigger = false
			continue
		}

		// 非触发器上下文中，分号是语句终结符
		if !inTrigger && strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	// 处理末尾无分号的语句
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		statements = append(statements, remaining)
	}

	return statements
}

// truncateSQL 截断 SQL 用于错误日志
func truncateSQL(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
