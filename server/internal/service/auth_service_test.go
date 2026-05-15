package service

import (
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestAuthService_LoginByUsername_NewUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		JWTSecret:      "test-secret-key",
		JWTExpireHours: 2,
	}

	svc := NewAuthService(cfg, repository.NewUserRepo(db))

	result, err := svc.LoginByUsername("张三", "", "")
	if err != nil {
		t.Fatalf("LoginByUsername 失败: %v", err)
	}
	if result.Token == "" {
		t.Error("token 不应为空")
	}
	if result.ExpiresIn != 7200 {
		t.Errorf("expires_in 应为 7200，得到 %d", result.ExpiresIn)
	}
	if result.DisplayName != "张三" {
		t.Errorf("display_name 应为张三，得到 %s", result.DisplayName)
	}
	if result.Role != "student" {
		t.Errorf("新用户 role 应为 student，得到 %s", result.Role)
	}
}

func TestAuthService_LoginByUsername_ExistingUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// 先手动创建用户
	userRepo := repository.NewUserRepo(db)
	_, err := userRepo.Create(&model.User{
		Username:    "existing",
		DisplayName: "老用户",
		Role:        "counselor",
		OwnerScope:  "college",
		OwnerID:     "default",
	})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	cfg := &config.Config{
		JWTSecret:      "test-secret-key",
		JWTExpireHours: 4,
	}

	svc := NewAuthService(cfg, userRepo)

	result, err := svc.LoginByUsername("existing", "", "")
	if err != nil {
		t.Fatalf("LoginByUsername 失败: %v", err)
	}
	if result.Role != "counselor" {
		t.Errorf("老用户 role 应为 counselor，得到 %s", result.Role)
	}
	if result.DisplayName != "老用户" {
		t.Errorf("display_name 应为老用户，得到 %s", result.DisplayName)
	}
	if result.ExpiresIn != 14400 {
		t.Errorf("expires_in 应为 14400，得到 %d", result.ExpiresIn)
	}
}

func TestAuthService_LoginByUsername_EmptyUsername(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 2}
	svc := NewAuthService(cfg, repository.NewUserRepo(db))

	_, err := svc.LoginByUsername("", "", "")
	if err == nil {
		t.Fatal("空用户名应返回错误")
	}
}

func TestAuthService_LoginByUsername_RepeatedLogin(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 2}
	svc := NewAuthService(cfg, repository.NewUserRepo(db))

	// 首次登录（创建用户）
	result1, err := svc.LoginByUsername("repeat", "", "")
	if err != nil {
		t.Fatalf("首次登录失败: %v", err)
	}

	// 二次登录（使用已有用户）
	result2, err := svc.LoginByUsername("repeat", "", "")
	if err != nil {
		t.Fatalf("二次登录失败: %v", err)
	}

	// 两次登录 token 可能不同（时间戳变化），但用户信息应一致
	if result2.DisplayName != result1.DisplayName {
		t.Errorf("display_name 应一致: %s vs %s", result1.DisplayName, result2.DisplayName)
	}
	if result2.Role != result1.Role {
		t.Errorf("role 应一致: %s vs %s", result1.Role, result2.Role)
	}
}
