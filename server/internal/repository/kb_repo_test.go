package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestKBRepo_Search_Chinese(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	results, err := repo.Search("奖学金", "college", "default", "student", 5)
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("应至少返回 1 条结果")
	}
	if results[0].Resource.Title != "2026年度国家奖学金评选办法" {
		t.Errorf("第一条应为奖学金政策，得到 %s", results[0].Resource.Title)
	}
	// BM25 分数应为负值
	if results[0].Score >= 0 {
		t.Errorf("BM25 rank 应为负值，得到 %f", results[0].Score)
	}
}

func TestKBRepo_Search_English(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	results, err := repo.Search("GPA", "college", "default", "student", 5)
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	// 奖学金和转专业中都提到了 GPA
	if len(results) == 0 {
		t.Error("GPA 应在奖学金和转专业中出现")
	}
}

func TestKBRepo_Search_NoMatch(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	results, err := repo.Search("火星探测器", "college", "default", "student", 5)
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("火星探测器不应有结果，得到 %d 条", len(results))
	}
}

func TestKBRepo_Search_RoleFilter(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// "奖学金" 查询 — student 有权限
	results, err := repo.Search("奖学金", "college", "default", "student", 10)
	if err != nil {
		t.Fatalf("Search 失败: %v", err)
	}
	if len(results) == 0 {
		t.Error("student 应能看到奖学金资源")
	}
}

func TestKBRepo_List(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	list, err := repo.List("", "", "published", "", 0, 10)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("期望 3 条 published 资源，得到 %d", len(list))
	}
}

func TestKBRepo_Count(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	count, err := repo.Count("", "", "published", "")
	if err != nil {
		t.Fatalf("Count 失败: %v", err)
	}
	if count != 3 {
		t.Errorf("期望 3 条 published，得到 %d", count)
	}
}

func TestKBRepo_GetByResourceID(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	kb, err := repo.GetByResourceID("policy-scholarship-2026")
	if err != nil {
		t.Fatalf("GetByResourceID 失败: %v", err)
	}
	if kb == nil {
		t.Fatal("应找到该资源")
	}
	if kb.Title != "2026年度国家奖学金评选办法" {
		t.Errorf("title 不匹配: %s", kb.Title)
	}
}

func TestKBRepo_GetByResourceID_NotFound(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	repo := NewKBRepo(db)

	kb, err := repo.GetByResourceID("nonexistent")
	if err != nil {
		t.Fatalf("GetByResourceID 失败: %v", err)
	}
	if kb != nil {
		t.Error("不存在的资源应返回 nil")
	}
}

func TestKBRepo_CreateAndUpdate(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewKBRepo(db)

	// 创建
	kb := &model.KBResource{
		ResourceID:   "test-policy-1",
		ResourceType: "Policy",
		OwnerScope:   "school",
		RoleScope:    `["student"]`,
		Version:      "1.0",
		Status:       "draft",
		Title:        "测试创建",
		Content:      "测试内容",
	}
	id, err := repo.Create(kb)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("期望有效 id，得到 %d", id)
	}

	// 回查
	created, err := repo.GetByResourceID("test-policy-1")
	if err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	if created.Title != "测试创建" {
		t.Errorf("期望 title=测试创建，得到 %s", created.Title)
	}

	// 更新
	created.Title = "测试更新"
	created.Status = "published"
	if err := repo.Update(created); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	updated, _ := repo.GetByResourceID("test-policy-1")
	if updated.Title != "测试更新" {
		t.Errorf("更新后期望 title=测试更新，得到 %s", updated.Title)
	}
	if updated.Status != "published" {
		t.Errorf("更新后期望 status=published，得到 %s", updated.Status)
	}
}
