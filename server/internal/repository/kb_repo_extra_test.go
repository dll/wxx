package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestKBRepo_GetProcessSteps(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// 创建资源和流程步骤
	kb := &model.KBResource{
		ResourceID: "proc-test-1", ResourceType: "Process", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "1.0", Status: "published",
		Title: "入学流程", Content: "入学流程详细内容",
	}
	repo.Create(kb)

	// 直接插入流程步骤
	db.Exec(`INSERT INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
		VALUES ('proc-test-1', 1, '提交申请', '["身份证","录取通知书"]', 'https://example.com', '2026-09-01', '行政楼101', '需本人到场')`)
	db.Exec(`INSERT INTO process_steps (resource_id, step_order, title, materials, entry_url, deadline, location, notes)
		VALUES ('proc-test-1', 2, '缴费注册', '[]', '', '', '', '')`)

	steps, err := repo.GetProcessSteps("proc-test-1")
	if err != nil {
		t.Fatalf("GetProcessSteps 失败: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("期望 2 个步骤，得到 %d", len(steps))
	}
	if steps[0].Title != "提交申请" {
		t.Errorf("期望第一步 Title=提交申请，得到 %s", steps[0].Title)
	}
	if steps[0].StepOrder != 1 {
		t.Errorf("期望 StepOrder=1，得到 %d", steps[0].StepOrder)
	}
	if steps[1].Title != "缴费注册" {
		t.Errorf("期望第二步 Title=缴费注册，得到 %s", steps[1].Title)
	}
}

func TestKBRepo_GetProcessSteps_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	steps, err := repo.GetProcessSteps("nonexistent")
	if err != nil {
		t.Fatalf("GetProcessSteps 失败: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("期望 0 个步骤，得到 %d", len(steps))
	}
}

func TestKBRepo_GetPublishedCards(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// 创建已发布资源
	repo.Create(&model.KBResource{
		ResourceID: "card-1", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student,counselor", Version: "1.0", Status: "published",
		Title: "奖学金办法", Summary: "2026年度奖学金", Tags: `["奖学金"]`,
	})
	repo.Create(&model.KBResource{
		ResourceID: "card-2", ResourceType: "Process", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "入学流程", Summary: "新生入学", Tags: `["入学"]`,
	})
	// 草稿状态不应出现
	repo.Create(&model.KBResource{
		ResourceID: "card-3", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "draft",
		Title: "草稿政策", Summary: "不应出现",
	})

	cards, err := repo.GetPublishedCards("school", "", "student", "")
	if err != nil {
		t.Fatalf("GetPublishedCards 失败: %v", err)
	}
	if len(cards) < 2 {
		t.Errorf("期望至少 2 个分组，得到 %d: %v", len(cards), cards)
	}
	// 不应包含草稿
	for _, group := range cards {
		for _, card := range group {
			if card.Title == "草稿政策" {
				t.Error("GetPublishedCards 不应包含草稿资源")
			}
		}
	}
}

func TestKBRepo_GetPublishedCards_TypeFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	repo.Create(&model.KBResource{
		ResourceID: "cf-1", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "政策A", Summary: "摘要A",
	})
	repo.Create(&model.KBResource{
		ResourceID: "cf-2", ResourceType: "Process", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "流程B", Summary: "摘要B",
	})

	cards, err := repo.GetPublishedCards("school", "", "student", "Policy")
	if err != nil {
		t.Fatalf("GetPublishedCards 失败: %v", err)
	}
	if _, ok := cards["Process"]; ok {
		t.Error("过滤 Policy 类型后不应包含 Process 分组")
	}
}

func TestKBRepo_Upsert_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	kb := &model.KBResource{
		ResourceID: "upsert-new", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "新资源", Content: "正文内容",
	}
	id, action, err := repo.Upsert(kb)
	if err != nil {
		t.Fatalf("Upsert 失败: %v", err)
	}
	if action != "created" {
		t.Errorf("期望 action=created，得到 %s", action)
	}
	if id <= 0 {
		t.Errorf("期望有效 id，得到 %d", id)
	}
}

func TestKBRepo_Upsert_UpdateWithHigherVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// 先创建 v1.0
	kb := &model.KBResource{
		ResourceID: "upsert-ver", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "原始版本", Content: "v1正文",
	}
	repo.Upsert(kb)

	// 再导入 v2.0（应更新）
	kbV2 := &model.KBResource{
		ResourceID: "upsert-ver", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "2.0", Status: "published",
		Title: "更新版本", Content: "v2正文",
	}
	_, action, err := repo.Upsert(kbV2)
	if err != nil {
		t.Fatalf("Upsert v2 失败: %v", err)
	}
	if action != "updated" {
		t.Errorf("高版本应触发更新，得到 %s", action)
	}

	// 验证内容已更新
	updated, _ := repo.GetByResourceID("upsert-ver")
	if updated.Title != "更新版本" {
		t.Errorf("期望 Title=更新版本，得到 %s", updated.Title)
	}
}

func TestKBRepo_Upsert_SkipLowerVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// 先创建 v3.0
	kb := &model.KBResource{
		ResourceID: "upsert-skip", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "3.0", Status: "published",
		Title: "高版本", Content: "v3正文",
	}
	repo.Upsert(kb)

	// 再导入 v1.0（应跳过）
	kbOld := &model.KBResource{
		ResourceID: "upsert-skip", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "低版本", Content: "v1正文",
	}
	_, action, err := repo.Upsert(kbOld)
	if err != nil {
		t.Fatalf("Upsert v1 失败: %v", err)
	}
	if action != "skipped" {
		t.Errorf("低版本应跳过，得到 %s", action)
	}
}

func TestKBRepo_ListSince(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// 创建资源（created_at 由数据库自动生成）
	repo.Create(&model.KBResource{
		ResourceID: "since-1", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "新资源", Content: "正文",
	})

	// 用很早的时间作为游标，应返回所有已发布资源
	resources, err := repo.ListSince("", "2020-01-01T00:00:00Z", 100)
	if err != nil {
		t.Fatalf("ListSince 失败: %v", err)
	}
	if len(resources) < 1 {
		t.Error("应至少返回 1 条已发布资源")
	}

	// 用未来时间作为游标，应返回空
	resources, err = repo.ListSince("", "2099-01-01T00:00:00Z", 100)
	if err != nil {
		t.Fatalf("ListSince 失败: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("未来游标应返回空，得到 %d 条", len(resources))
	}
}

func TestKBRepo_ListSince_TypeFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	repo.Create(&model.KBResource{
		ResourceID: "ls-p", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "政策", Content: "正文",
	})
	repo.Create(&model.KBResource{
		ResourceID: "ls-f", ResourceType: "FAQ", OwnerScope: "school",
		RoleScope: "student", Version: "1.0", Status: "published",
		Title: "问答", Content: "正文",
	})

	resources, err := repo.ListSince("FAQ", "2020-01-01T00:00:00Z", 100)
	if err != nil {
		t.Fatalf("ListSince 失败: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("过滤 FAQ 后期望 1 条，得到 %d", len(resources))
	}
	if resources[0].Title != "问答" {
		t.Errorf("期望 Title=问答，得到 %s", resources[0].Title)
	}
}
