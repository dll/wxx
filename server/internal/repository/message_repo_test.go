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

// ═══ GetRecentQuestionsByUserID 测试 ═══

func TestMessageRepo_GetRecentQuestionsByUserID_WithData(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	// 创建用户 1 的会话和消息
	sessRepo.Create(&model.Session{SessionID: "sess-q1", UserID: 1})
	msgRepo.Create(&model.Message{SessionID: "sess-q1", Role: "user", Content: "奖学金怎么申请？"})
	msgRepo.Create(&model.Message{SessionID: "sess-q1", Role: "assistant", Content: "奖学金每年9月申请..."})
	msgRepo.Create(&model.Message{SessionID: "sess-q1", Role: "user", Content: "需要什么材料？"})

	// 创建用户 2 的会话（用于隔离验证）
	sessRepo.Create(&model.Session{SessionID: "sess-q2", UserID: 2})
	msgRepo.Create(&model.Message{SessionID: "sess-q2", Role: "user", Content: "其他用户的问题"})

	questions, err := msgRepo.GetRecentQuestionsByUserID(1, 10)
	if err != nil {
		t.Fatalf("GetRecentQuestionsByUserID 失败: %v", err)
	}
	if len(questions) != 2 {
		t.Errorf("用户 1 应有 2 条提问，得到 %d", len(questions))
	}
	// 最新问题在前（ORDER BY m.id DESC）
	if questions[0] != "需要什么材料？" {
		t.Errorf("最新问题应为需要什么材料？，得到 %s", questions[0])
	}
}

func TestMessageRepo_GetRecentQuestionsByUserID_EmptyDB(t *testing.T) {
	msgRepo, _ := setupMessageTestDB(t)

	questions, err := msgRepo.GetRecentQuestionsByUserID(999, 10)
	if err != nil {
		t.Fatalf("空数据库查询不应报错: %v", err)
	}
	if len(questions) != 0 {
		t.Errorf("空数据库应返回空列表，得到 %d 条", len(questions))
	}
}

func TestMessageRepo_GetRecentQuestionsByUserID_LimitRespected(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	sessRepo.Create(&model.Session{SessionID: "sess-limit-q", UserID: 1})
	for i := 0; i < 5; i++ {
		msgRepo.Create(&model.Message{SessionID: "sess-limit-q", Role: "user", Content: "问题"})
	}

	questions, err := msgRepo.GetRecentQuestionsByUserID(1, 3)
	if err != nil {
		t.Fatalf("GetRecentQuestionsByUserID 失败: %v", err)
	}
	if len(questions) != 3 {
		t.Errorf("limit=3 应返回 3 条，得到 %d", len(questions))
	}
}

func TestMessageRepo_GetRecentQuestionsByUserID_OnlyUserMessages(t *testing.T) {
	msgRepo, sessRepo := setupMessageTestDB(t)

	sessRepo.Create(&model.Session{SessionID: "sess-role", UserID: 1})
	msgRepo.Create(&model.Message{SessionID: "sess-role", Role: "user", Content: "用户问题"})
	msgRepo.Create(&model.Message{SessionID: "sess-role", Role: "assistant", Content: "助手回复"})
	msgRepo.Create(&model.Message{SessionID: "sess-role", Role: "system", Content: "系统消息"})

	questions, err := msgRepo.GetRecentQuestionsByUserID(1, 10)
	if err != nil {
		t.Fatalf("GetRecentQuestionsByUserID 失败: %v", err)
	}
	// 应该只返回 user 角色的消息，过滤掉 assistant 和 system
	if len(questions) != 1 {
		t.Errorf("应只返回 user 消息 1 条，得到 %d", len(questions))
	}
	if questions[0] != "用户问题" {
		t.Errorf("期望用户问题，得到 %s", questions[0])
	}
}
