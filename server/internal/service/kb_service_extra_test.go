package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupKBServiceBrowseTestDB(t *testing.T) *KBService {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 创建 process_steps 表（001_init.sql 可能没有）
	db.Exec(`CREATE TABLE IF NOT EXISTS process_steps (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		resource_id TEXT    NOT NULL REFERENCES kb_resources(resource_id),
		step_order  INTEGER NOT NULL,
		title       TEXT    NOT NULL,
		materials   TEXT    NOT NULL DEFAULT '[]',
		entry_url   TEXT    NOT NULL DEFAULT '',
		deadline    TEXT    NOT NULL DEFAULT '',
		location    TEXT    NOT NULL DEFAULT '',
		notes       TEXT    NOT NULL DEFAULT ''
	)`)

	return NewKBService(repository.NewKBRepo(db))
}

// ── Browse 测试 ──

func TestKBService_Browse_Empty(t *testing.T) {
	svc := setupKBServiceBrowseTestDB(t)

	cards, _, err := svc.Browse(context.Background(), "school", "", "student", "", 1, 0)
	if err != nil {
		t.Fatalf("Browse 失败: %v", err)
	}
	// 空数据库应返回空 map 或没有分组
	if cards == nil {
		t.Error("Browse 应返回非 nil map")
	}
}

func TestKBService_Browse_WithPublishedData(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS process_steps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		resource_id TEXT NOT NULL, step_order INTEGER NOT NULL,
		title TEXT NOT NULL, materials TEXT NOT NULL DEFAULT '[]',
		entry_url TEXT NOT NULL DEFAULT '', deadline TEXT NOT NULL DEFAULT '',
		location TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT ''
	)`)

	repo := repository.NewKBRepo(db)
	svc := NewKBService(repo)

	// 创建已发布和草稿资源
	repo.Create(&model.KBResource{
		ResourceID: "pub-1", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "已发布政策", Summary: "摘要",
	})
	repo.Create(&model.KBResource{
		ResourceID: "draft-1", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "draft",
		Title: "草稿政策", Summary: "不应出现",
	})

	cards, _, err := svc.Browse(context.Background(), "school", "", "student", "", 1, 0)
	if err != nil {
		t.Fatalf("Browse 失败: %v", err)
	}
	// 不应包含草稿
	for _, group := range cards {
		for _, card := range group {
			if card.Title == "草稿政策" {
				t.Error("Browse 不应包含草稿资源")
			}
		}
	}
}

func TestKBService_Browse_WithTypeFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS process_steps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		resource_id TEXT NOT NULL, step_order INTEGER NOT NULL,
		title TEXT NOT NULL, materials TEXT NOT NULL DEFAULT '[]',
		entry_url TEXT NOT NULL DEFAULT '', deadline TEXT NOT NULL DEFAULT '',
		location TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT ''
	)`)

	repo := repository.NewKBRepo(db)
	svc := NewKBService(repo)

	// 使用 repo 直接创建已发布资源
	repo.Create(&model.KBResource{
		ResourceID: "br-policy", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "政策A", Summary: "摘要A",
	})
	repo.Create(&model.KBResource{
		ResourceID: "br-process", ResourceType: "Process", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "流程B", Summary: "摘要B",
	})

	// 测试类型过滤
	cards, _, err := svc.Browse(context.Background(), "school", "", "student", "Policy", 1, 0)
	if err != nil {
		t.Fatalf("Browse 失败: %v", err)
	}
	if _, ok := cards["Process"]; ok {
		t.Error("过滤 Policy 类型后不应包含 Process 分组")
	}
	// Policy 分组应存在
	if _, ok := cards["Policy"]; !ok {
		t.Error("过滤 Policy 类型后应包含 Policy 分组")
	}
}

// ── ImportResources 测试 ──

func TestKBService_ImportResources_SingleValid(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	ndjson := `{"resource_id":"imp-1","resource_type":"Policy","title":"导入测试","content":"导入正文","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
`
	resp, err := svc.ImportResources(context.Background(),ndjson, "importer")
	if err != nil {
		t.Fatalf("ImportResources 失败: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("期望 total=1，得到 %d", resp.Total)
	}
	if resp.Created != 1 {
		t.Errorf("期望 created=1，得到 %d", resp.Created)
	}

	// 验证已入库
	kb, err := svc.Get(context.Background(),"imp-1")
	if err != nil {
		t.Fatalf("导入后 Get 失败: %v", err)
	}
	if kb.Title != "导入测试" {
		t.Errorf("期望 Title=导入测试，得到 %s", kb.Title)
	}
	if kb.UpdatedBy != "importer" {
		t.Errorf("期望 UpdatedBy=importer，得到 %s", kb.UpdatedBy)
	}
}

func TestKBService_ImportResources_MultipleLines(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	ndjson := `{"resource_id":"multi-1","resource_type":"FAQ","title":"FAQ1","content":"正文1","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
{"resource_id":"multi-2","resource_type":"FAQ","title":"FAQ2","content":"正文2","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
`
	resp, err := svc.ImportResources(context.Background(),ndjson, "importer")
	if err != nil {
		t.Fatalf("ImportResources 失败: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("期望 total=2，得到 %d", resp.Total)
	}
	if resp.Created != 2 {
		t.Errorf("期望 created=2，得到 %d", resp.Created)
	}
}

func TestKBService_ImportResources_InvalidJSON(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	ndjson := `这不是JSON
{"resource_id":"valid","resource_type":"Policy","title":"有效","content":"正文","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
`
	resp, err := svc.ImportResources(context.Background(),ndjson, "importer")
	if err != nil {
		t.Fatalf("ImportResources 失败: %v", err)
	}
	if resp.Created != 1 {
		t.Errorf("有效行应被创建，得到 created=%d", resp.Created)
	}
	if resp.Skipped < 1 {
		t.Errorf("无效 JSON 行应被跳过，得到 skipped=%d", resp.Skipped)
	}
}

func TestKBService_ImportResources_MissingRequiredFields(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	// 缺少 title 和 content
	ndjson := `{"resource_id":"no-title","resource_type":"Policy","content":"","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
`
	resp, err := svc.ImportResources(context.Background(),ndjson, "importer")
	if err != nil {
		t.Fatalf("ImportResources 失败: %v", err)
	}
	if resp.Skipped != 1 {
		t.Errorf("缺少必填字段应跳过，得到 skipped=%d", resp.Skipped)
	}
}

func TestKBService_ImportResources_InvalidResourceType(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	ndjson := `{"resource_id":"bad-type","resource_type":"InvalidType","title":"坏类型","content":"正文","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
`
	resp, err := svc.ImportResources(context.Background(),ndjson, "importer")
	if err != nil {
		t.Fatalf("ImportResources 失败: %v", err)
	}
	if resp.Skipped != 1 {
		t.Errorf("无效类型应跳过，得到 skipped=%d", resp.Skipped)
	}
}

func TestKBService_ImportResources_VersionUpsert(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	// 先导入 v1.0
	ndjson := `{"resource_id":"ver-test","resource_type":"Policy","title":"v1标题","content":"v1正文","owner_scope":"school","role_scope":"student","version":"1.0","status":"published"}
`
	resp, err := svc.ImportResources(context.Background(),ndjson, "importer")
	if err != nil {
		t.Fatalf("首次导入失败: %v", err)
	}
	if resp.Created != 1 {
		t.Errorf("首次导入应 created=1，得到 %d", resp.Created)
	}

	// 再导入 v2.0（应更新）
	ndjson2 := `{"resource_id":"ver-test","resource_type":"Policy","title":"v2标题","content":"v2正文","owner_scope":"school","role_scope":"student","version":"2.0","status":"published"}
`
	resp2, err := svc.ImportResources(context.Background(), ndjson2, "importer")
	if err != nil {
		t.Fatalf("二次导入失败: %v", err)
	}
	if resp2.Updated != 1 {
		t.Errorf("高版本应更新，得到 updated=%d", resp2.Updated)
	}

	// 验证内容已更新
	kb, _ := svc.Get(context.Background(),"ver-test")
	if kb.Title != "v2标题" {
		t.Errorf("期望 Title=v2标题，得到 %s", kb.Title)
	}
}

// ── ExportResources 测试 ──

func TestKBService_ExportResources(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := repository.NewKBRepo(db)
	svc := NewKBService(repo)

	// 创建已发布资源
	repo.Create(&model.KBResource{
		ResourceID: "exp-1", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "导出资源", Content: "导出正文",
	})

	resources, err := svc.ExportResources(context.Background(),"", "2020-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("ExportResources 失败: %v", err)
	}
	if len(resources) < 1 {
		t.Errorf("应至少导出 1 条，得到 %d", len(resources))
	}
}

func TestKBService_ExportResources_EmptyWithFutureCursor(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db))

	// 用未来时间作为游标，应返回空
	resources, err := svc.ExportResources(context.Background(),"", "2099-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("ExportResources 失败: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("未来游标应返回空，得到 %d 条", len(resources))
	}
}

func TestKBService_ExportResources_TypeFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := repository.NewKBRepo(db)
	svc := NewKBService(repo)

	repo.Create(&model.KBResource{
		ResourceID: "exp-p", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "政策", Content: "政策正文",
	})
	repo.Create(&model.KBResource{
		ResourceID: "exp-f", ResourceType: "FAQ", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "问答", Content: "问答正文",
	})

	resources, err := svc.ExportResources(context.Background(),"FAQ", "2020-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("ExportResources 失败: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("过滤 FAQ 后期望 1 条，得到 %d", len(resources))
	}
	if resources[0].Title != "问答" {
		t.Errorf("期望 Title=问答，得到 %s", resources[0].Title)
	}
}

// ── 辅助 ──

func strPtr(s string) *string {
	return &s
}
