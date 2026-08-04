package service

import (
	"os"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestFreshmenGuideSeedAndService(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	migrationPath := "../../migrations/054_freshmen_guide_seed.sql"
	if _, err := os.Stat(migrationPath); err != nil {
		migrationPath = "../migrations/054_freshmen_guide_seed.sql"
	}
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("读取迁移文件失败: %v", err)
	}
	for _, stmt := range testutil.SplitSQL(string(content)) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("执行迁移失败: %v\nSQL: %s", err, truncateTestSQL(stmt))
		}
	}

	svc := NewStudentService(nil, nil, nil, nil, repository.NewKBRepo(db), nil, nil)
	guide, err := svc.GetFreshmenGuide()
	if err != nil {
		t.Fatalf("GetFreshmenGuide 失败: %v", err)
	}
	if guide.Guide == nil {
		t.Fatal("guide-freshmen-2026 应存在")
	}
	if guide.Handbook == nil {
		t.Fatal("policy-student-handbook-2025 应存在")
	}
	if guide.Zzsb == nil {
		t.Fatal("guide-freshmen-2026-zzsb 应存在")
	}
	if guide.Process == nil {
		t.Fatal("process-registration-2026 应存在")
	}
	if len(guide.Steps) != 11 {
		t.Fatalf("期望 11 个报到步骤，得到 %d", len(guide.Steps))
	}
	if len(guide.SourceFiles) != 3 {
		t.Fatalf("期望 3 个官方资料条目，得到 %d", len(guide.SourceFiles))
	}
}

func truncateTestSQL(s string) string {
	if len(s) > 160 {
		return s[:160] + "..."
	}
	return s
}
