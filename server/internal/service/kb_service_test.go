package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestKBService_List(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	list, total, err := svc.List(context.Background(), "", "", "published", "", 1, 10)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total != 3 {
		t.Errorf("期望 total=3，得到 %d", total)
	}
	if len(list) != 3 {
		t.Errorf("期望 3 条，得到 %d", len(list))
	}
}

func TestKBService_List_Pagination(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	// 每页 1 条
	list, total, err := svc.List(context.Background(), "", "", "published", "", 1, 1)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if total != 3 {
		t.Errorf("total 应为 3，得到 %d", total)
	}
	if len(list) != 1 {
		t.Errorf("pageSize=1 应返回 1 条，得到 %d", len(list))
	}
}

func TestKBService_Get(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	kb, err := svc.Get(context.Background(), "policy-scholarship-2026")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if kb.Title != "2026年度国家奖学金评选办法" {
		t.Errorf("title 不匹配: %s", kb.Title)
	}
}

func TestKBService_Get_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("不存在的资源应返回错误")
	}
}

func TestKBService_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	req := &model.KBCreateRequest{
		ResourceType: "Policy",
		OwnerScope:   "school",
		RoleScope:    `["student"]`,
		Title:        "测试政策",
		Content:      "这是测试内容",
	}

	kb, err := svc.Create(context.Background(), req, "admin")
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if kb.ResourceID == "" {
		t.Error("resource_id 不应为空")
	}
	if kb.Title != "测试政策" {
		t.Errorf("title 应为测试政策，得到 %s", kb.Title)
	}
	if kb.Status != "draft" {
		t.Errorf("新资源 status 应为 draft，得到 %s", kb.Status)
	}
	if kb.UpdatedBy != "admin" {
		t.Errorf("updated_by 应为 admin，得到 %s", kb.UpdatedBy)
	}
}

func TestKBService_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	// 先创建
	created, _ := svc.Create(context.Background(), &model.KBCreateRequest{
		ResourceType: "FAQ",
		OwnerScope:   "college",
		RoleScope:    `["student"]`,
		Title:        "原始标题",
		Content:      "原始内容",
	}, "admin")

	// 更新
	req := &model.KBUpdateRequest{
		Title:   "更新后的标题",
		Content: "更新后的内容",
		Status:  "published",
	}
	updated, err := svc.Update(context.Background(), created.ResourceID, req, "editor")
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Title != "更新后的标题" {
		t.Errorf("title 应为更新后的标题，得到 %s", updated.Title)
	}
	if updated.Status != "published" {
		t.Errorf("status 应为 published，得到 %s", updated.Status)
	}
	if updated.UpdatedBy != "editor" {
		t.Errorf("updated_by 应为 editor，得到 %s", updated.UpdatedBy)
	}
}

func TestKBService_Update_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewKBService(repository.NewKBRepo(db), db)

	_, err := svc.Update(context.Background(), "nonexistent", &model.KBUpdateRequest{Title: "x"}, "admin")
	if err == nil {
		t.Fatal("更新不存在的资源应返回错误")
	}
}
