package repository

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

// ── escapeQuery 性能基准 ──

func BenchmarkEscapeQuery_Chinese(b *testing.B) {
	query := "奖学金申请条件和流程说明"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = escapeQuery(query)
	}
}

func BenchmarkEscapeQuery_English(b *testing.B) {
	query := "scholarship application process"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = escapeQuery(query)
	}
}

func BenchmarkEscapeQuery_Short(b *testing.B) {
	query := "入学"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = escapeQuery(query)
	}
}

func BenchmarkEscapeQuery_Empty(b *testing.B) {
	query := ""
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = escapeQuery(query)
	}
}

// ── FTS5 Search 性能基准 ──

// setupSearchBenchDB 创建含大量数据的测试库
func setupSearchBenchDB(b *testing.B, count int) *KBRepo {
	b.Helper()

	// 直接在内存中创建 SQLite 并执行迁移（避免依赖 NewTestDB 需要 *testing.T）
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("打开内存数据库失败: %v", err)
	}
	b.Cleanup(func() { db.Close() })

	// 执行迁移
	migrationPath := "../../migrations/001_init.sql"
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		migrationPath = "migrations/001_init.sql"
		sqlContent, err = os.ReadFile(migrationPath)
		if err != nil {
			b.Fatalf("读取迁移文件失败: %v", err)
		}
	}
	for _, stmt := range testutil.SplitSQL(string(sqlContent)) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			b.Fatalf("执行迁移失败: %v\nSQL: %s", err, stmt[:min(len(stmt), 200)])
		}
	}

	repo := NewKBRepo(db)

	// 批量插入测试数据
	for i := 0; i < count; i++ {
		_, err := repo.Create(&model.KBResource{
			ResourceID:   fmt.Sprintf("bench-%d", i),
			ResourceType: "Policy",
			OwnerScope:   "school",
			RoleScope:    "student",
			Version:      "1.0",
			Status:       "published",
			Title:        fmt.Sprintf("测试政策文档 %d", i),
			Summary:      fmt.Sprintf("这是第 %d 条政策文档的摘要", i),
			Content: fmt.Sprintf(
				"第 %d 条政策文档的正文内容。包含奖学金、助学金、入学、离校等常见学工关键词。",
				i,
			),
			Tags:      "政策,学工",
			UpdatedBy: "bench",
		})
		if err != nil {
			b.Fatalf("插入基准数据失败: %v", err)
		}
	}

	return repo
}

func BenchmarkFTS5Search_100Docs(b *testing.B) {
	repo := setupSearchBenchDB(b, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := repo.Search("奖学金", "school", "", "student", 5)
		if err != nil {
			b.Fatalf("搜索失败: %v", err)
		}
	}
}

func BenchmarkFTS5Search_1000Docs(b *testing.B) {
	repo := setupSearchBenchDB(b, 1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := repo.Search("入学流程", "school", "", "student", 5)
		if err != nil {
			b.Fatalf("搜索失败: %v", err)
		}
	}
}

func BenchmarkFTS5Search_ChineseMultiChar(b *testing.B) {
	repo := setupSearchBenchDB(b, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := repo.Search("助学贷款申请", "school", "", "student", 10)
		if err != nil {
			b.Fatalf("搜索失败: %v", err)
		}
	}
}

// ── KBRepo CRUD 性能基准 ──

func BenchmarkKBCreate(b *testing.B) {
	repo := setupSearchBenchDB(b, 0) // 空库

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.Create(&model.KBResource{
			ResourceID:   fmt.Sprintf("cr-%d", i),
			ResourceType: "FAQ",
			OwnerScope:   "school",
			RoleScope:    "student",
			Version:      "1.0",
			Status:       "published",
			Title:        fmt.Sprintf("常见问题 %d", i),
			Summary:      "摘要",
			Content:      "正文内容",
			Tags:         "FAQ",
			UpdatedBy:    "bench",
		})
		if err != nil {
			b.Fatalf("Create 失败: %v", err)
		}
	}
}

func BenchmarkKBGetByResourceID(b *testing.B) {
	repo := setupSearchBenchDB(b, 10)
	// 确保目标记录存在
	repo.Create(&model.KBResource{
		ResourceID:   "bench-get-target",
		ResourceType: "FAQ",
		OwnerScope:   "school",
		RoleScope:    "student",
		Version:      "1.0",
		Status:       "published",
		Title:        "获取目标",
		Summary:      "摘要",
		Content:      "正文",
		UpdatedBy:    "bench",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetByResourceID("bench-get-target")
		if err != nil {
			b.Fatalf("GetByResourceID 失败: %v", err)
		}
	}
}

func BenchmarkKBList(b *testing.B) {
	repo := setupSearchBenchDB(b, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := repo.List("school", "", "published", "", 0, 20)
		if err != nil {
			b.Fatalf("List 失败: %v", err)
		}
	}
}

// ── compareVersion 性能基准 ──

func BenchmarkCompareVersion(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = compareVersion("3.14.159", "2.718.281")
	}
}
