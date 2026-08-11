package service

import (
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

// TestRestoreSnapshot_UserStatus 校验用户状态操作恢复
func TestRestoreSnapshot_UserStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	auditRepo := repository.NewAuditRepo(db)
	svc := NewAdminService(userRepo, auditRepo, repository.NewSettingsRepo(db))

	// 建一个用户（active）
	hash, _ := bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost)
	uid, err := userRepo.Create(&model.User{
		Username: "stu_x", DisplayName: "X同学", Role: "student",
		OwnerScope: "college", OwnerID: "cs",
		PasswordHash: string(hash), Status: "active",
	})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	// 批量改为 disabled（触发快照）
	if _, err := svc.BatchUpdateStatus([]int64{uid}, "disabled", "admin"); err != nil {
		t.Fatalf("BatchUpdateStatus 失败: %v", err)
	}
	u1, _ := userRepo.GetByID(uid)
	if u1.Status != "disabled" {
		t.Fatalf("状态应为 disabled，实际 %s", u1.Status)
	}

	// 快照应已生成
	snaps, err := svc.ListSnapshots(10)
	if err != nil || len(snaps) != 1 {
		t.Fatalf("应生成 1 条快照: len=%d err=%v", len(snaps), err)
	}
	snap := snaps[0]
	if snap.Operation != "user.status" || snap.BeforeJSON != "active" {
		t.Fatalf("快照内容不符: %+v", snap)
	}

	// 恢复 → active
	n, err := svc.RestoreSnapshot(snap.ID, "admin")
	if err != nil || n != 1 {
		t.Fatalf("RestoreSnapshot 失败: n=%d err=%v", n, err)
	}
	u2, _ := userRepo.GetByID(uid)
	if u2.Status != "active" {
		t.Fatalf("恢复后状态应为 active，实际 %s", u2.Status)
	}

	// 二次恢复应被拒绝（已恢复）
	if _, err := svc.RestoreSnapshot(snap.ID, "admin"); err == nil {
		t.Fatal("二次恢复应被拒绝")
	}

	_ = config.Config{}
}
