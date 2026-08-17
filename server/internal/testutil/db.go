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
	db.SetMaxOpenConns(1)

	// 执行 001_init.sql 迁移（基础 schema）
	migrationPath := resolveMigrationPath(t, "001_init.sql")
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("读取迁移文件失败: %v", err)
	}
	execMigrationSQL(t, db, string(sqlContent))

	// 执行后续增量迁移（仅 schema，不含种子数据）
	for _, m := range []string{
		"004_emotion_enhance.sql",
		"005_agents.sql",
		"006_fix_emotion_risk_level.sql",
		"009_feedback_and_settings.sql",
		"010_add_password_hash.sql",
		"011_feedback_enhance.sql",
		"012_voice_config.sql",
		"013_user_model_config.sql",
		"014_update_model_defaults.sql",
		"015_token_usage.sql",
		"016_fix_seed_users.sql",
		"017_session_title.sql",
		"018_process_records.sql",
		"019_feedback_screenshot_blob.sql",
		"021_add_step_contact_fields.sql",
		"022_issue_forecasts.sql",
		"023_graduation_topics.sql",
		"024_student_features.sql",
		"025_add_user_status.sql",
		"027_add_guest_role.sql",
		"029_student_user_import.sql",
		"040_add_user_token_version.sql",
		"041_add_user_consented.sql",
		"042_add_step_geo_media_fields.sql",
		"043_student_profile_snapshot.sql",
		"044_student_checkins.sql",
		"045_student_personality.sql",
		"046_chat_metrics.sql",
		"047_add_kb_remark.sql",
		"048_campus_map_steps.sql",
		"049_fts_tags.sql",
		"039_feedback_closed_loop.sql",
		"063_feedback_module.sql",
		"064_feedback_repair_jobs.sql",
		"065_external_apps.sql",
		"066_ai_briefings.sql",
		"067_ai_briefings_seed.sql",
		"068_twin_portraits.sql",
		"069_user_contact_fields.sql",
		"070_ai_briefings_home_highlight.sql",
		"071_user_portal_credentials.sql",
		"072_audit_snapshots.sql",
		"073_ai_briefings_refactor.sql",
		"074_student_profile_fields.sql",
		"075_must_change_password.sql",
		"076_user_ai_key.sql",
		"091_student_profile_snapshot_history.sql",
	} {
		p := resolveMigrationPath(t, m)
		c, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读取迁移文件失败 %s: %v", m, err)
		}
		execMigrationSQL(t, db, string(c))
	}

	return db
}

// NewTestDBFull 创建内存 SQLite 数据库并执行全部迁移（含种子数据）
func NewTestDBFull(t *testing.T) *sql.DB {
	t.Helper()

	db := NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 测试种子只服务内存库；生产迁移不携带测试数据。
	execMigrationSQL(t, db, embeddedTestSeedSQL)

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
			// 与生产迁移 runner 保持一致：ALTER TABLE ADD COLUMN 重复列名视为已达目标状态
			if isDuplicateColumnError(err) && strings.Contains(strings.ToUpper(stmt), "ALTER TABLE") {
				t.Logf("跳过重复列: %v", err)
				continue
			}
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

// isDuplicateColumnError 检测 SQLite "duplicate column name" 错误
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "duplicate column")
}

// truncateSQL 截断 SQL 用于错误日志
func truncateSQL(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

const embeddedTestSeedSQL = `
INSERT INTO kb_resources (resource_id, resource_type, owner_scope, owner_id, role_scope, version, status, title, summary, content, source_link, effective_at, tags, updated_by)
VALUES
('policy-scholarship-2026', 'Policy', 'school', '', '["student","counselor","teacher"]', 'test', 'published', '2026年度国家奖学金评选办法', '国家奖学金用于奖励特别优秀的全日制本专科学生，奖励标准为每人每年8000元。', '申请条件包括热爱祖国、遵守校规、诚实守信、学习成绩优异，上一学年平均学分绩点 GPA 排名在本专业前10%且无不及格科目。评选流程为学生申请、班级评议、学院审核、学校评审、公示。', '', '2026-09-01 00:00:00', '["test"]', 'test'),
('process-major-transfer-2026', 'Process', 'school', '', '["student","counselor"]', 'test', 'published', '本科生转专业办理流程', '符合条件的学生可按学校通知申请转专业，平均学分绩点 GPA 应达到转入专业要求。', '流程包括在线申请、转出学院审核、转入学院考核、教务处审批、公示、办理学籍异动。', '', '2026-06-01 00:00:00', '["test"]', 'test'),
('faq-graduation-2026', 'FAQ', 'school', '', '["student"]', 'test', 'published', '毕业生离校手续常见问题', '毕业生离校需办理图书馆、宿舍、财务、学院等环节。', '离校手续通常包括归还图书、宿舍退宿、财务结算、档案确认和证书领取。具体要求以学校当年通知为准。', '', '2026-06-01 00:00:00', '["test"]', 'test');

INSERT INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
VALUES
('process-major-transfer-2026', 1, '在线申请', '[]', '', '学校通知时间内', '教务系统', '填写转专业申请'),
('process-major-transfer-2026', 2, '学院审核', '["成绩单"]', '', '提交后审核', '学院教学办公室', '完成转出与转入学院审核'),
('process-major-transfer-2026', 3, '教务处审批', '[]', '', '学院审核后', '教务处', '完成公示和学籍异动');
`
