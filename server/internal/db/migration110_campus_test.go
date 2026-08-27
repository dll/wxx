package db

// 回归测试：校验 110_restore_campus_steps.sql 迁移的方言转换与幂等性风险。
//
// reviewer 回修后验证目标：
//   1. 不再使用 (campus_id, step_order) 唯一索引（会阻碍审核流 H2）；
//   2. 幂等去重改用 `INSERT ... SELECT ... WHERE NOT EXISTS` 守卫（H1 不建索引、
//      不 DELETE 数据、容忍同 step_order 多 status 共存）;
//   3. MySQL 方言不改写该 INSERT...SELECT 语法，双方言均通用；
//   4. 仍恢复 12 条（会峰6+琅琊6）的权威坐标 published 种子。

import (
	"os"
	"strings"
	"testing"
)

func read110SQL(t *testing.T) string {
	raw, err := os.ReadFile("../../migrations/110_restore_campus_steps.sql")
	if err != nil {
		raw, err = os.ReadFile("../../../migrations/110_restore_campus_steps.sql")
	}
	if err != nil {
		t.Skipf("找不到 110 迁移文件: %v", err)
	}
	return string(raw)
}

func TestToMySQL_Migration110_RestoreCampusSteps(t *testing.T) {
	content := read110SQL(t)

	stmts := splitSQLStmt(content)
	var converted []string
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
		t.Fatalf("110 转换后无任何语句输出")
	}

	joined := strings.Join(converted, ";\n")
	upper := strings.ToUpper(joined)

	// 1. 不应残留 SQLite 专有语法 INSERT OR IGNORE
	if strings.Contains(upper, "INSERT OR IGNORE") {
		t.Errorf("110 MySQL 转换后仍残留 SQLite 的 INSERT OR IGNORE:\n%s", joined)
	}
	// 2. 不应建唯一索引（H2 回修：唯一索引阻碍审核流）
	if strings.Contains(upper, "CREATE UNIQUE INDEX") || strings.Contains(upper, "CREATE INDEX") {
		t.Errorf("110 不应再建唯一/普通索引，应改用 WHERE NOT EXISTS 幂等守卫:\n%s", joined)
	}
	// 3. 应使用 WHERE NOT EXISTS 幂等守卫（12 条各一条）
	if strings.Count(upper, "WHERE NOT EXISTS") != 12 {
		t.Errorf("110 应含 12 条 WHERE NOT EXISTS 守卫（实际 %d 条）:\n%s",
			strings.Count(upper, "WHERE NOT EXISTS"), joined)
	}
	// 4. 不应残留其它 SQLite 专有语法
	for _, banned := range []string{"AUTOINCREMENT", "datetime('now'", "ON CONFLICT"} {
		if strings.Contains(strings.ToLower(joined), strings.ToLower(banned)) {
			t.Errorf("110 MySQL 转换后仍含 SQLite 专有语法 %q:\n%s", banned, joined)
		}
	}
	// 5. 应包含 12 行 INSERT（会峰6 + 琅琊6）
	if strings.Count(upper, "INSERT INTO CAMPUS_CHECKIN_STEPS") != 12 {
		t.Errorf("110 应含 12 条 INSERT INTO campus_checkin_steps（实际 %d 条）",
			strings.Count(upper, "INSERT INTO CAMPUS_CHECKIN_STEPS"))
	}
	if !strings.Contains(joined, "'huifeng'") || !strings.Contains(joined, "'langya'") {
		t.Errorf("110 转换后应含 huifeng 与 langya 两类 campus_id:\n%s", joined)
	}
	// 6. 坐标应采用 050 权威纠正值（会峰 step1 = 32.2705/118.3055，琅琊 step1 = 32.2921/118.2988）
	if !strings.Contains(joined, "32.2705") || !strings.Contains(joined, "118.3055") {
		t.Errorf("110 应含会峰校区 authority 坐标 32.2705/118.3055:\n%s", joined)
	}
	if !strings.Contains(joined, "32.2921") || !strings.Contains(joined, "118.2988") {
		t.Errorf("110 应含琅琊校区 authority 坐标 32.2921/118.2988:\n%s", joined)
	}
	t.Logf("110 → MySQL 转换语句:\n%s", joined)
}

// TestMigration110_NoUniqueIndex_GuardUsesNotExists 回修验证：
//   - H2：不得再建 (campus_id, step_order) 唯一索引（step_order 非唯一业务键，
//     审核流 draft/pending_review/published 多状态共存）。
//   - H1：幂等去重改用 WHERE NOT EXISTS 守卫，不 DELETE 数据、不建索引，
//     容忍表内已存在同 step_order 的多状态行。
func TestMigration110_NoUniqueIndex_GuardUsesNotExists(t *testing.T) {
	content := read110SQL(t)
	// 剥离注释行后再校验，避免注释中引用 "CREATE UNIQUE INDEX" 等字样被误判
	upper := strings.ToUpper(stripSQLComments(content))

	if strings.Contains(upper, "CREATE UNIQUE INDEX") {
		t.Errorf("[缺陷] 110 仍含唯一索引，会阻碍审核流（H2 未修复）。")
	}
	if strings.Contains(upper, "CREATE INDEX") {
		t.Errorf("[缺陷] 110 不应含任何索引，改用 WHERE NOT EXISTS 守卫。")
	}
	if strings.Contains(upper, "DELETE FROM") {
		t.Errorf("[缺陷] 110 不应含 DELETE，避免丢数据（H1 要求不丢数据）。")
	}
	if cnt := strings.Count(upper, "WHERE NOT EXISTS"); cnt != 12 {
		t.Errorf("[缺陷] 110 应含 12 条 WHERE NOT EXISTS 幂等守卫（实际 %d 条）。", cnt)
	}
	t.Logf("[OK] 110 已废弃唯一索引，改用 WHERE NOT EXISTS 守卫实现幂等且不阻碍审核流。")
}

// stripSQLComments 移除 SQL 行内/整行的 `--` 注释，返回仅含可执行语句的文本。
func stripSQLComments(sql string) string {
	var out strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		// 去掉行内注释（字符串内无 `--`，迁移文件注释均为整行或行首 `--`）
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// TestMigration110_PublishesSeed 确认 110 恢复的均为 published 状态种子。
func TestMigration110_PublishesSeed(t *testing.T) {
	content := read110SQL(t)
	// 12 条种子全部 status='published'
	if strings.Count(content, "'published'") < 12 {
		t.Errorf("110 应含至少 12 条 status='published' 的种子（实际 %d 条）。",
			strings.Count(content, "'published'"))
	}
}
