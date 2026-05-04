package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestUserRepo_GetByUsername_Existing(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// 手动插入一个用户
	repo := NewUserRepo(db)
	_, err := repo.Create(&model.User{
		Username:    "testuser",
		DisplayName: "测试用户",
		Role:        "student",
		OwnerScope:  "college",
		OwnerID:     "default",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	user, err := repo.GetByUsername("testuser")
	if err != nil {
		t.Fatalf("GetByUsername 失败: %v", err)
	}
	if user == nil {
		t.Fatal("应找到用户")
	}
	if user.Username != "testuser" {
		t.Errorf("期望 username=testuser，得到 %s", user.Username)
	}
	if user.DisplayName != "测试用户" {
		t.Errorf("期望 display_name=测试用户，得到 %s", user.DisplayName)
	}
	if user.Role != "student" {
		t.Errorf("期望 role=student，得到 %s", user.Role)
	}
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)

	user, err := repo.GetByUsername("nonexistent")
	if err != nil {
		t.Fatalf("GetByUsername 失败: %v", err)
	}
	if user != nil {
		t.Error("不存在的用户应返回 nil")
	}
}

func TestUserRepo_GetByID_Existing(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)
	id, err := repo.Create(&model.User{
		Username:    "iduser",
		DisplayName: "ID用户",
		Role:        "counselor",
		OwnerScope:  "college",
		OwnerID:     "cs",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	user, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if user == nil {
		t.Fatal("应找到用户")
	}
	if user.Username != "iduser" {
		t.Errorf("期望 username=iduser，得到 %s", user.Username)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)

	user, err := repo.GetByID(99999)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if user != nil {
		t.Error("不存在的 ID 应返回 nil")
	}
}

func TestUserRepo_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)

	id, err := repo.Create(&model.User{
		Username:    "newuser",
		DisplayName: "新用户",
		Role:        "teacher",
		OwnerScope:  "school",
		OwnerID:     "",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("期望有效 id，得到 %d", id)
	}

	// 回查验证
	user, _ := repo.GetByID(id)
	if user == nil {
		t.Fatal("回查应找到用户")
	}
	if user.Role != "teacher" {
		t.Errorf("期望 role=teacher，得到 %s", user.Role)
	}
}

func TestUserRepo_Create_Duplicate(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)
	_, err := repo.Create(&model.User{
		Username:    "dup",
		DisplayName: "重复用户1",
		Role:        "student",
		OwnerScope:  "college",
		OwnerID:     "default",
	})
	if err != nil {
		t.Fatalf("首次 Create 失败: %v", err)
	}

	// 重复插入应报错
	_, err = repo.Create(&model.User{
		Username:    "dup",
		DisplayName: "重复用户2",
		Role:        "student",
		OwnerScope:  "college",
		OwnerID:     "default",
	})
	if err == nil {
		t.Error("重复用户名应返回错误")
	}
}
