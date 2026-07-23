package service

import (
	"errors"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func newAuthServiceForTest(t *testing.T, username, password, status string) *AuthService {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { _ = db.Close() })

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("生成测试密码哈希失败: %v", err)
	}
	repo := repository.NewUserRepo(db)
	_, err = repo.Create(&model.User{
		Username:     username,
		DisplayName:  "测试用户",
		Role:         "student",
		OwnerScope:   "college",
		OwnerID:      "cs",
		PasswordHash: string(hash),
		Status:       status,
	})
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return NewAuthService(&config.Config{
		JWTSecret:      "test-secret-key",
		JWTExpireHours: 2,
	}, repo)
}

func TestAuthService_LoginByUsername_WithPassword(t *testing.T) {
	svc := newAuthServiceForTest(t, "2023211981", "2023211981", "active")
	result, err := svc.LoginByUsername("2023211981", "", "2023211981")
	if err != nil {
		t.Fatalf("账号密码登录失败: %v", err)
	}
	if result.Token == "" || result.ExpiresIn != 7200 {
		t.Fatalf("登录结果异常: %+v", result)
	}
}

func TestAuthService_LoginByUsername_RejectsPasswordless(t *testing.T) {
	svc := newAuthServiceForTest(t, "passwordless-case", "correct-password", "active")
	_, err := svc.LoginByUsername("passwordless-case", "", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("空密码应被拒绝，得到: %v", err)
	}
}

func TestAuthService_LoginByUsername_RejectsWrongPassword(t *testing.T) {
	svc := newAuthServiceForTest(t, "wrong-password-case", "correct-password", "active")
	_, err := svc.LoginByUsername("wrong-password-case", "", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("错误密码应被拒绝，得到: %v", err)
	}
}

func TestAuthService_LoginByUsername_DoesNotAutoCreate(t *testing.T) {
	svc := newAuthServiceForTest(t, "existing", "password123", "active")
	_, err := svc.LoginByUsername("not-exists", "student", "password123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("不存在的账号应被拒绝，得到: %v", err)
	}
}

func TestAuthService_LoginByUsername_RejectsDisabledAccount(t *testing.T) {
	svc := newAuthServiceForTest(t, "disabled", "password123", "disabled")
	_, err := svc.LoginByUsername("disabled", "", "password123")
	if !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("停用账号应被拒绝，得到: %v", err)
	}
}
