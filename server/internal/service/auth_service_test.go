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

// TestProfileDetail_StudentFindsCounselor 校验学生个人信息详情能聚合出辅导员联系人
func TestProfileDetail_StudentFindsCounselor(t *testing.T) {
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := repository.NewUserRepo(db)

	// 学生（cs 学院）
	studentHash, _ := bcrypt.GenerateFromPassword([]byte("p1"), bcrypt.MinCost)
	_, _ = repo.Create(&model.User{
		Username: "stu1", DisplayName: "张同学", Role: "student",
		OwnerScope: "college", OwnerID: "cs", PasswordHash: string(studentHash), Status: "active",
		College: "计算机学院", Major: "软件工程", ClassName: "软工1班",
	})
	// 辅导员（cs 学院，应被聚合）
	counselorHash, _ := bcrypt.GenerateFromPassword([]byte("p2"), bcrypt.MinCost)
	counID, _ := repo.Create(&model.User{
		Username: "coun1", DisplayName: "李辅导员", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs", PasswordHash: string(counselorHash), Status: "active",
		College: "计算机学院",
	})
	// Create 不写入联系方式，用 SQL 补 phone
	if _, err := db.Exec("UPDATE users SET phone = ? WHERE id = ?", "13800000000", counID); err != nil {
		t.Fatalf("补 phone 失败: %v", err)
	}

	svc := NewAuthService(&config.Config{}, repo)
	stu, _ := repo.GetByUsername("stu1")
	detail, err := svc.GetProfileDetail(stu.ID)
	if err != nil {
		t.Fatalf("GetProfileDetail 失败: %v", err)
	}
	if detail.College != "计算机学院" || detail.Major != "软件工程" {
		t.Fatalf("基本信息不符: %+v", detail)
	}
	if len(detail.Supervisors) != 1 || detail.Supervisors[0].Name != "李辅导员" {
		t.Fatalf("学生应聚合到辅导员: %+v", detail.Supervisors)
	}
	if detail.Supervisors[0].Phone != "13800000000" {
		t.Fatalf("辅导员联系方式未带回: %+v", detail.Supervisors[0])
	}
}
