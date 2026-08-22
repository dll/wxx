package db

import (
	"os"
	"strings"
	"testing"
)

// vOPC migrations are executed through the SQLite-to-MySQL translator in
// production. Keep a focused regression gate so newly added P0 tables do not
// silently reintroduce SQLite-only DDL.
func TestToMySQLVOPCMigrations(t *testing.T) {
	for _, name := range []string{"097_vopc_p0.sql", "098_vopc_decisions.sql", "099_vopc_collaboration_delivery.sql", "100_vopc_artifact_version_gates.sql", "101_vopc_close_state_machine.sql", "102_vopc_risk_governance.sql", "103_vopc_private_files.sql"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("../../migrations/" + name)
			if err != nil {
				t.Fatal(err)
			}
			converted := make([]string, 0)
			for _, stmt := range splitSQLStmt(string(raw)) {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if out := strings.TrimSpace(ToMySQL(stmt)); out != "" {
					converted = append(converted, out)
				}
			}
			if len(converted) == 0 {
				t.Fatal("conversion produced no statements")
			}
			joined := strings.Join(converted, ";\n")
			for _, banned := range []string{"AUTOINCREMENT", "datetime('now'", "ON CONFLICT", "INDEX IF NOT EXISTS"} {
				if strings.Contains(strings.ToUpper(joined), strings.ToUpper(banned)) {
					t.Errorf("converted migration retains SQLite-only syntax %q:\n%s", banned, joined)
				}
			}
			if strings.Contains(string(raw), "INTEGER PRIMARY KEY AUTOINCREMENT") && !strings.Contains(joined, "BIGINT PRIMARY KEY AUTO_INCREMENT") {
				t.Errorf("converted migration is missing MySQL auto-increment primary key:\n%s", joined)
			}
		})
	}
}
