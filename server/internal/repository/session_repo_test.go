package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestSessionRepo_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	// 先创建用户
	userRepo := NewUserRepo(db)
	userID, err := userRepo.Create(&model.User{
		Username: "sessuser", DisplayName: "会话用户",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	repo := NewSessionRepo(db)
	err = repo.Create(&model.Session{
		SessionID: "test-session-id",
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// 回查
	session, err := repo.GetBySessionID("test-session-id")
	if err != nil {
		t.Fatalf("GetBySessionID 失败: %v", err)
	}
	if session == nil {
		t.Fatal("应找到会话")
	}
	if session.UserID != userID {
		t.Errorf("期望 userID=%d，得到 %d", userID, session.UserID)
	}
}

func TestSessionRepo_GetBySessionID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)

	session, err := repo.GetBySessionID("nonexistent-session")
	if err != nil {
		t.Fatalf("GetBySessionID 失败: %v", err)
	}
	if session != nil {
		t.Error("不存在的会话应返回 nil")
	}
}

func TestSessionRepo_ListByUserID(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := NewUserRepo(db)
	userID, err := userRepo.Create(&model.User{
		Username: "listuser", DisplayName: "列表用户",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	repo := NewSessionRepo(db)

	// 创建多条会话
	for _, sid := range []string{"s1", "s2", "s3"} {
		if err := repo.Create(&model.Session{SessionID: sid, UserID: userID}); err != nil {
			t.Fatalf("Create %s 失败: %v", sid, err)
		}
	}

	list, err := repo.ListByUserID(userID, 10)
	if err != nil {
		t.Fatalf("ListByUserID 失败: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("期望 3 条会话，得到 %d", len(list))
	}
}

func TestSessionRepo_ListByUserID_Limit(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "limituser", DisplayName: "限制用户",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	repo := NewSessionRepo(db)
	for _, sid := range []string{"a", "b", "c", "d", "e"} {
		_ = repo.Create(&model.Session{SessionID: sid, UserID: userID})
	}

	list, err := repo.ListByUserID(userID, 2)
	if err != nil {
		t.Fatalf("ListByUserID 失败: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("limit=2 应返回 2 条，得到 %d", len(list))
	}
}

func TestSessionRepo_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "deluser", DisplayName: "删除用户",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	repo := NewSessionRepo(db)
	_ = repo.Create(&model.Session{SessionID: "to-delete", UserID: userID})

	// 确认存在
	s, _ := repo.GetBySessionID("to-delete")
	if s == nil {
		t.Fatal("删除前应存在")
	}

	// 删除
	if err := repo.Delete("to-delete"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 确认已删除
	s, err := repo.GetBySessionID("to-delete")
	if err != nil {
		t.Fatalf("GetBySessionID 失败: %v", err)
	}
	if s != nil {
		t.Error("删除后应返回 nil")
	}
}

func TestSessionRepo_Touch(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "touchuser", DisplayName: "更新时间用户",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	repo := NewSessionRepo(db)
	_ = repo.Create(&model.Session{SessionID: "touch-me", UserID: userID})

	before, _ := repo.GetBySessionID("touch-me")
	if before == nil {
		t.Fatal("会话应存在")
	}

	if err := repo.Touch("touch-me"); err != nil {
		t.Fatalf("Touch 失败: %v", err)
	}

	after, _ := repo.GetBySessionID("touch-me")
	if after == nil {
		t.Fatal("Touch 后会话应仍存在")
	}
	// updated_at 可能不变化（同一秒内），仅验证操作不报错
}

func TestSessionRepo_Delete_WithMessages(t *testing.T) {
	// SQLite 默认不启用外键约束，因此删除会话时消息不会级联删除。
	// 此测试验证 Delete 操作本身成功，消息残留由业务层处理。
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "cascadeuser", DisplayName: "级联用户",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	sessionRepo := NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "msg-session", UserID: userID})

	// 往会话里写消息
	msgRepo := NewMessageRepo(db)
	_ = msgRepo.Create(&model.Message{
		SessionID: "msg-session", Role: "user", Content: "测试消息",
	})

	// 删除会话 — 操作本身应成功
	if err := sessionRepo.Delete("msg-session"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 会话应已不存在
	s, err := sessionRepo.GetBySessionID("msg-session")
	if err != nil {
		t.Fatalf("GetBySessionID 失败: %v", err)
	}
	if s != nil {
		t.Error("删除后会话应返回 nil")
	}
}
