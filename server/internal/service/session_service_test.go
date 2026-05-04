package service

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestSessionService_DeleteSession_Success(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "owner", DisplayName: "会话拥有者",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	sessionRepo := repository.NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "my-session", UserID: userID})

	svc := NewSessionService(sessionRepo, repository.NewMessageRepo(db))

	if err := svc.DeleteSession(userID, "my-session"); err != nil {
		t.Fatalf("DeleteSession 失败: %v", err)
	}

	// 验证已删除
	s, _ := sessionRepo.GetBySessionID("my-session")
	if s != nil {
		t.Error("删除后应返回 nil")
	}
}

func TestSessionService_DeleteSession_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	svc := NewSessionService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
	)

	err := svc.DeleteSession(1, "nonexistent")
	if err == nil {
		t.Fatal("不存在的会话应返回错误")
	}
}

func TestSessionService_DeleteSession_WrongUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	ownerID, _ := userRepo.Create(&model.User{
		Username: "owner", DisplayName: "拥有者",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})
	otherID, _ := userRepo.Create(&model.User{
		Username: "other", DisplayName: "其他人",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	sessionRepo := repository.NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "owner-session", UserID: ownerID})

	svc := NewSessionService(sessionRepo, repository.NewMessageRepo(db))

	// 其他人试图删除
	err := svc.DeleteSession(otherID, "owner-session")
	if err == nil {
		t.Fatal("删除他人会话应返回错误")
	}
}

func TestSessionService_ListSessions(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "lister", DisplayName: "列表者",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	sessionRepo := repository.NewSessionRepo(db)
	for _, sid := range []string{"a", "b"} {
		_ = sessionRepo.Create(&model.Session{SessionID: sid, UserID: userID})
	}

	svc := NewSessionService(sessionRepo, repository.NewMessageRepo(db))

	list, err := svc.ListSessions(userID, 10)
	if err != nil {
		t.Fatalf("ListSessions 失败: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("期望 2 条会话，得到 %d", len(list))
	}
}

func TestSessionService_GetSessionMessages(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	userID, _ := userRepo.Create(&model.User{
		Username: "reader", DisplayName: "读者",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	sessionRepo := repository.NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "read-session", UserID: userID})

	msgRepo := repository.NewMessageRepo(db)
	_ = msgRepo.Create(&model.Message{
		SessionID: "read-session", Role: "user", Content: "你好",
	})
	_ = msgRepo.Create(&model.Message{
		SessionID: "read-session", Role: "assistant", Content: "你好！有什么可以帮你的？",
	})

	svc := NewSessionService(sessionRepo, msgRepo)

	msgs, err := svc.GetSessionMessages(userID, "read-session", 50)
	if err != nil {
		t.Fatalf("GetSessionMessages 失败: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("期望 2 条消息，得到 %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("第 1 条应为 user，得到 %s", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("第 2 条应为 assistant，得到 %s", msgs[1].Role)
	}
}

func TestSessionService_GetSessionMessages_WrongUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	ownerID, _ := userRepo.Create(&model.User{
		Username: "msgowner", DisplayName: "消息拥有者",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})
	otherID, _ := userRepo.Create(&model.User{
		Username: "intruder", DisplayName: "入侵者",
		Role: "student", OwnerScope: "college", OwnerID: "default",
	})

	sessionRepo := repository.NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "private-session", UserID: ownerID})

	svc := NewSessionService(sessionRepo, repository.NewMessageRepo(db))

	_, err := svc.GetSessionMessages(otherID, "private-session", 50)
	if err == nil {
		t.Fatal("访问他人会话消息应返回错误")
	}
}
