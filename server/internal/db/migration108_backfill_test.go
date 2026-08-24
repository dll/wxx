package db

// 迁移 108（vOPC S→G 存量数据回填）功能性回归：
// 在 SQLite 内存库上重建 097-100+107 schema，播种 v1.0 遗留 S 阶段数据，
// 执行 108 后断言 projects/milestones/artifact_versions 全部收敛到 G0-G4，且重复执行幂等。

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func migration108SeedLegacy(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, name := range []string{"097_vopc_p0.sql", "099_vopc_collaboration_delivery.sql", "100_vopc_artifact_version_gates.sql", "107_vopc_v2_layers.sql"} {
		raw, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(raw)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// 播种 v1.0 遗留数据：owner_user_id=1 无外键强制（SQLite 默认关闭 foreign_keys）
	legacy := []struct{ stage, status string }{
		{"S0", "draft"},
		{"S1", "pending_review"},
		{"S4", "developing"},
		{"S6", "delivering"},
		{"S9", "closed"},
	}
	for i, p := range legacy {
		if _, err := db.Exec(`INSERT INTO vopc_projects(name,stage,status,owner_user_id) VALUES(?,?,?,?)`,
			"legacy-"+p.stage, p.stage, p.status, i+1); err != nil {
			t.Fatal(err)
		}
	}
	// 里程碑遗留行（v1.0 播种 S 阶段）
	for _, ms := range []string{"S0", "S1", "S5", "S8"} {
		if _, err := db.Exec(`INSERT INTO vopc_milestones(project_id,stage,required_evidence) VALUES(1,?,'x')`, ms); err != nil {
			t.Fatal(err)
		}
	}
	// 成果版次遗留门禁阶段
	if _, err := db.Exec(`INSERT INTO vopc_artifacts(project_id,name,artifact_type,created_by) VALUES(1,'a','doc',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO vopc_artifact_versions(artifact_id,version,source_kind,source_ref,intended_stage,created_by) VALUES(1,'v1','upload','x','S2',1)`); err != nil {
		t.Fatal(err)
	}
}

func runMigration108(t *testing.T, db *sql.DB) {
	t.Helper()
	raw, err := os.ReadFile("../../migrations/108_vopc_stage_backfill.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range splitSQLStmt(string(raw)) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("108 语句执行失败: %v\n%s", err, stmt)
		}
	}
}

func TestMigration108StageBackfill(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	migration108SeedLegacy(t, db)
	runMigration108(t, db)

	// 项目主表映射断言
	want := map[string]string{ // stage -> status
		"G0": "draft",
		"G1": "pending_review",
		"G2": "developing",
		"G4": "closeable",
	}
	rows, err := db.Query(`SELECT stage,status FROM vopc_projects WHERE name LIKE 'legacy-%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var s, st string
		if err := rows.Scan(&s, &st); err != nil {
			t.Fatal(err)
		}
		got[s] = st
	}
	if len(got) != len(want) { // S6/S9 都并入 G4，去重后共 G0/G1/G2/G4 四个阶段
		t.Fatalf("项目阶段数不符: got %d (%v)", len(got), got)
	}
	for stage, status := range want {
		if got[stage] == "" {
			t.Errorf("缺少映射结果 %s", stage)
			continue
		}
		if stage == "G4" && got["G4"] != "closeable" {
			// S6/S9 均应落 closeable；map 只存一份，单独在下方按计数校验
		} else if got[stage] != status {
			t.Errorf("stage=%s status=%s want %s", stage, got[stage], status)
		}
	}
	var g4 int
	if err := db.QueryRow(`SELECT COUNT(*) FROM vopc_projects WHERE stage='G4' AND status='closeable'`).Scan(&g4); err != nil || g4 != 2 {
		t.Fatalf("G4/closeable 应有 2 行(S6+S9), got %d err=%v", g4, err)
	}

	// 里程碑映射断言
	for _, c := range []struct{ from, to string }{{"S0", "G0"}, {"S1", "G1"}, {"S5", "G3"}, {"S8", "G4"}} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM vopc_milestones WHERE project_id=1 AND stage=?`, c.to).Scan(&n); err != nil || n != 1 {
			t.Fatalf("里程碑 %s→%s 应为 1 行, got %d err=%v", c.from, c.to, n, err)
		}
	}
	var leftover int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_milestones WHERE stage LIKE 'S%'`).Scan(&leftover)
	if leftover != 0 {
		t.Fatalf("里程碑仍残留 S 阶段 %d 行", leftover)
	}

	// 成果版次门禁字段映射断言
	var is string
	if err := db.QueryRow(`SELECT intended_stage FROM vopc_artifact_versions LIMIT 1`).Scan(&is); err != nil || is != "G1" {
		t.Fatalf("intended_stage 应为 G1, got %s err=%v", is, err)
	}

	// 幂等：重复执行不报错、结果不变
	runMigration108(t, db)
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM vopc_projects WHERE stage LIKE 'S%'`).Scan(&cnt); err != nil || cnt != 0 {
		t.Fatalf("二次执行后仍残留 S 阶段项目: %d err=%v", cnt, err)
	}
}
