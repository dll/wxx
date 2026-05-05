package repository

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupMessageTestDB(t *testing.T) (*MessageRepo, *SessionRepo) {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	return NewMessageRepo(db), NewSessionRepo(db)
}

func TestMessageRepo_ListBySessionID(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	sessRepo.Create(&model.Session{SessionID: "sess-msg-test", UserID: 1})
	msgRepo.Create(&model.Message{SessionID: "sess-msg-test", Role: "user", Content: "你好"})
	msgRepo.Create(&model.Message{SessionID: "sess-msg-test", Role: "assistant", Content: "你好！有什么可以帮助你的？"})

	messages, err := msgRepo.ListBySessionID("sess-msg-test", 50)
	if err != nil {
		t.Fatalf("ListBySessionID 失败: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("期望 2 条消息，得到 %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Errorf("期望第一条是 user，得到 %s", messages[0].Role)
	}
	if messages[1].Role != "assistant" {
		t.Errorf("期望第二条是 assistant，得到 %s", messages[1].Role)
	}
}

func TestMessageRepo_ListBySessionID_RespectsLimit(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	sessRepo.Create(&model.Session{SessionID: "sess-limit", UserID: 1})
	for i := 0; i < 5; i++ {
		msgRepo.Create(&model.Message{SessionID: "sess-limit", Role: "user", Content: "消息"})
	}

	messages, err := msgRepo.ListBySessionID("sess-limit", 3)
	if err != nil {
		t.Fatalf("ListBySessionID 失败: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("limit=3 应返回 3 条，得到 %d", len(messages))
	}
}

func TestMessageRepo_ListBySessionID_Empty(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	sessRepo.Create(&model.Session{SessionID: "sess-empty", UserID: 1})

	messages, err := msgRepo.ListBySessionID("sess-empty", 50)
	if err != nil {
		t.Fatalf("ListBySessionID 失败: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("期望 0 条消息，得到 %d", len(messages))
	}
}

func TestMessageRepo_GetRecentContext(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	sessRepo.Create(&model.Session{SessionID: "sess-ctx", UserID: 1})
	for i := 0; i < 10; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		msgRepo.Create(&model.Message{SessionID: "sess-ctx", Role: role, Content: "消息"})
	}

	ctx, err := msgRepo.GetRecentContext("sess-ctx", 5)
	if err != nil {
		t.Fatalf("GetRecentContext 失败: %v", err)
	}
	if len(ctx) != 5 {
		t.Errorf("期望取最近 5 条，得到 %d", len(ctx))
	}
	// 取最新 5 条后反转正序：最后 5 条是 a,u,a,u,a → 反转后 a,u,a,u,a
	if ctx[0].Role != "assistant" {
		t.Errorf("期望第一条是 assistant，得到 %s", ctx[0].Role)
	}
}
