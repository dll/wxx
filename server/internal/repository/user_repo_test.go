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

// TestUserRepo_UpsertFromContext_JITCreatesPendingGuest P0-02 回归：
// JIT 冷启动插入的未知用户不得为 active+consented 且角色不得取自 JWT 自述。
// 修复前硬编码 status=active、consented=1、role 直接写 JWT 携带值（可凭空创建高权限账号）。
func TestUserRepo_UpsertFromContext_JITCreatesPendingGuest(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)
	// 模拟未知 JWT：声称自己是 sys_admin（高权限）——角色自述绝不能被信任
	ctx := &model.UserContext{
		Username:     "jit_unknown_sysadmin",
		DisplayName:  "伪造管理员",
		Role:         "sys_admin",
		OwnerScope:   "school",
		OwnerID:      "chzu",
		TokenVersion: 0,
	}

	if err := repo.UpsertFromContext(ctx); err != nil {
		t.Fatalf("UpsertFromContext 失败: %v", err)
	}

	// JIT 创建的账号必须落入 pending 审核态
	user, err := repo.GetByUsername("jit_unknown_sysadmin")
	if err != nil {
		t.Fatalf("查询 JIT 用户失败: %v", err)
	}
	if user == nil {
		t.Fatal("JIT 用户应被创建")
	}
	if user.Status != "pending" {
		t.Errorf("JIT 用户状态应为 pending，得到 %s", user.Status)
	}
	if user.Role != "guest" {
		t.Errorf("JIT 用户角色应强制 guest，得到 %s（不信任 JWT 自述 %s）", user.Role, "sys_admin")
	}
	if user.Consented != 0 {
		t.Errorf("JIT 用户 consented 应为 0（未授权），得到 %d", user.Consented)
	}

	// 上下文同步：role/status/consented 必须与数据库一致
	if ctx.Role != "guest" {
		t.Errorf("上下文 role 应同步为 guest，得到 %s", ctx.Role)
	}
	if ctx.Status != "pending" {
		t.Errorf("上下文 status 应同步为 pending，得到 %s", ctx.Status)
	}
	if ctx.Consented {
		t.Error("上下文 consented 应为 false")
	}
}

// TestUserRepo_UpsertFromContext_PendingBlocked P0-02 回归：
// 已存在但处于 pending/rejected/disabled 的用户，凭业务 JWT 访问一律拒绝。
// 修复前仅拦截 disabled/rejected，pending 游客可畅通访问。
func TestUserRepo_UpsertFromContext_PendingBlocked(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)
	for _, st := range []string{"pending", "rejected", "disabled"} {
		id, err := repo.Create(&model.User{
			Username:    "blocked_" + st,
			DisplayName: "被拦截用户",
			Role:        "guest",
			OwnerScope:  "college",
			OwnerID:     "default",
			Status:      st,
		})
		if err != nil {
			t.Fatalf("创建 %s 用户失败: %v", st, err)
		}
		_ = id

		ctx := &model.UserContext{
			Username:     "blocked_" + st,
			DisplayName:  "被拦截用户",
			Role:         "guest",
			OwnerScope:   "college",
			OwnerID:      "default",
			TokenVersion: 0,
		}
		if err := repo.UpsertFromContext(ctx); err != model.ErrAccountDisabled {
			t.Errorf("状态 %s 应返回 ErrAccountDisabled，得到 %v", st, err)
		}
	}
}

// TestUserRepo_UpsertFromContext_ActiveAllowed P0-02 回归：
// active 状态用户正常放行，权限字段以数据库为权威。
func TestUserRepo_UpsertFromContext_ActiveAllowed(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepo(db)
	id, err := repo.Create(&model.User{
		Username:    "active_user",
		DisplayName: "活跃用户",
		Role:        "student",
		OwnerScope:  "college",
		OwnerID:     "default",
		Status:      "active",
		Consented:   1,
	})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	ctx := &model.UserContext{
		Username:     "active_user",
		DisplayName:  "JWT 自称其它名",
		Role:         "sys_admin", // 伪造高权限角色
		OwnerScope:   "school",
		OwnerID:      "chzu",
		TokenVersion: 0,
	}

	if err := repo.UpsertFromContext(ctx); err != nil {
		t.Fatalf("active 用户 upsert 不应失败: %v", err)
	}

	// 数据库权威值覆盖 JWT 自述
	if ctx.Role != "student" {
		t.Errorf("角色应以数据库为准 student，得到 %s", ctx.Role)
	}
	if ctx.UserID != id {
		t.Errorf("UserID 应以数据库为准 %d，得到 %d", id, ctx.UserID)
	}
	if !ctx.Consented {
		t.Error("consented 应以数据库为准为 true")
	}
}
