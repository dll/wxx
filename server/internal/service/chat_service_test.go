package service

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/dll/wxx/server/internal/repository"
)

func TestChatService_Ask_NewSession(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	mockLLM := llm.NewMockClient("test")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: "这是回答", FinishReason: "stop",
			PromptTokens: 100, OutputTokens: 20,
		}, nil
	}

	svc := NewChatService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		mockLLM,
	)

	userCtx := &model.UserContext{
		UserID: 1, Username: "test", Role: "student",
		OwnerScope: "college", OwnerID: "default",
	}

	card, sessionID, err := svc.Ask(context.Background(), userCtx, "", "奖学金怎么申请")
	if err != nil {
		t.Fatalf("Ask 失败: %v", err)
	}
	if sessionID == "" {
		t.Error("sessionID 不应为空")
	}
	if card == nil {
		t.Fatal("AnswerCard 不应为空")
	}
	if card.Conclusion == "" {
		t.Error("结论不应为空")
	}
	if card.Fallback {
		t.Error("有知识命中时不应为兜底")
	}
}

func TestChatService_Ask_ExistingSession(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "existing-session", UserID: 1})

	mockLLM := llm.NewMockClient("test")
	svc := NewChatService(
		sessionRepo,
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		mockLLM,
	)

	userCtx := &model.UserContext{UserID: 1, Username: "test", Role: "student"}

	_, returnedSessionID, err := svc.Ask(context.Background(), userCtx, "existing-session", "继续问")
	if err != nil {
		t.Fatalf("Ask 失败: %v", err)
	}
	if returnedSessionID != "existing-session" {
		t.Errorf("期望 session=existing-session，得到 %s", returnedSessionID)
	}
}

func TestChatService_Ask_WrongUserSession(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepo(db)
	_ = sessionRepo.Create(&model.Session{SessionID: "other-user-session", UserID: 99})

	mockLLM := llm.NewMockClient("test")
	svc := NewChatService(
		sessionRepo,
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		mockLLM,
	)

	userCtx := &model.UserContext{UserID: 1, Username: "test", Role: "student"}
	_, _, err := svc.Ask(context.Background(), userCtx, "other-user-session", "测试")
	if err == nil {
		t.Fatal("访问他人会话应返回错误")
	}
}

func TestChatService_Ask_EmptyKnowledge(t *testing.T) {
	db := testutil.NewTestDB(t) // 无种子数据的空白库
	defer db.Close()

	mockLLM := llm.NewMockClient("test")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: "无法回答", FinishReason: "stop",
			PromptTokens: 50, OutputTokens: 5,
		}, nil
	}

	svc := NewChatService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		mockLLM,
	)

	userCtx := &model.UserContext{UserID: 1, Username: "test", Role: "student"}
	card, _, err := svc.Ask(context.Background(), userCtx, "", "随机问题")
	if err != nil {
		t.Fatalf("Ask 失败: %v", err)
	}
	if !card.Fallback {
		t.Error("空知识库时应为兜底回答")
	}
	if card.Confidence != 0.3 {
		t.Errorf("空知识库时 confidence 应为 0.3，得到 %f", card.Confidence)
	}
}

func TestChatService_Ask_LLMFallback(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	mockLLM := llm.NewMockClient("test")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return nil, context.DeadlineExceeded
	}

	svc := NewChatService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		mockLLM,
	)

	userCtx := &model.UserContext{
		UserID: 1, Username: "test", Role: "student",
		OwnerScope: "college", OwnerID: "default",
	}
	card, _, err := svc.Ask(context.Background(), userCtx, "", "奖学金")
	if err != nil {
		t.Fatalf("Ask 失败: %v", err)
	}
	if !card.Fallback {
		t.Error("LLM 错误时应为兜底回答")
	}
	// LLM 降级但仍应保留搜索到的 sources
	if len(card.Sources) == 0 {
		t.Error("LLM 降级时应保留 FTS 搜索到的 sources")
	}
}
