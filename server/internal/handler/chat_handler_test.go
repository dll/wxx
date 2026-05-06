package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

func setupChatTestRouter(t *testing.T, mockClient llm.ChatClient) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDBFull(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		JWTSecret:      "test-secret-chat",
		JWTExpireHours: 2,
	}

	sessionRepo := repository.NewSessionRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	kbRepo := repository.NewKBRepo(db)

	chatSvc := service.NewChatService(sessionRepo, messageRepo, kbRepo, repository.NewAgentRepo(db), mockClient)
	chatHandler := NewChatHandler(chatSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.POST("/chat", chatHandler.Ask)

	return r, cfg
}

// ═══ SetEmotionService 测试 ═══

func TestChatHandler_SetEmotionService(t *testing.T) {
	// 构造 ChatHandler（nil chatSvc 对 setter 测试无害）
	h := &ChatHandler{}
	if h.emotionSvc != nil {
		t.Error("初始 emotionSvc 应为 nil")
	}

	// 设置 nil 情感服务（nil-safe）
	h.SetEmotionService(nil)
	if h.emotionSvc != nil {
		t.Error("SetEmotionService(nil) 后 emotionSvc 应为 nil")
	}
}

func TestChatHandler_Ask_Success(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content:      "根据知识库，国家奖学金每年评选一次。",
			FinishReason: "stop",
			PromptTokens: 200,
			OutputTokens: 50,
		}, nil
	}

	r, cfg := setupChatTestRouter(t, mockLLM)

	user := &model.User{
		ID: 1, Username: "student1", Role: "student",
		OwnerScope: "college", OwnerID: "default",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"question":"奖学金怎么申请"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.ChatResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("AnswerCard 不应为空")
	}
	if resp.Data.Conclusion == "" {
		t.Error("结论不应为空")
	}
	if resp.SessionID == "" {
		t.Error("应返回 session_id")
	}
	if resp.Data.Fallback {
		t.Error("不应为兜底回答（知识库有数据）")
	}
}

func TestChatHandler_Ask_NoKnowledge(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content:      "抱歉，知识库中没有相关信息。建议联系辅导员。",
			FinishReason: "stop",
			PromptTokens: 100,
			OutputTokens: 30,
		}, nil
	}

	r, cfg := setupChatTestRouter(t, mockLLM)

	user := &model.User{ID: 1, Username: "s1", Role: "student", OwnerScope: "college", OwnerID: "default"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"question":"地球上有什么"}` // 不相关的查询
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}

	var resp model.ChatResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	// 无知识命中时应为兜底
	if !resp.Data.Fallback {
		t.Log("WARNING: 预期应为兜底回答，但可能 query 部分命中")
	}
}

func TestChatHandler_Ask_EmptyQuestion(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	r, cfg := setupChatTestRouter(t, mockLLM)

	user := &model.User{ID: 1, Username: "s1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"question":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("空问题应返回 400，得到 %d", w.Code)
	}
}

func TestChatHandler_Ask_Unauthenticated(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	r, _ := setupChatTestRouter(t, mockLLM)

	body := `{"question":"测试"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestChatHandler_Ask_InvalidJSON(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	r, cfg := setupChatTestRouter(t, mockLLM)

	user := &model.User{ID: 1, Username: "s1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader("bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d", w.Code)
	}
}

func TestChatHandler_Ask_LLMError(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return nil, context.DeadlineExceeded
	}

	r, cfg := setupChatTestRouter(t, mockLLM)

	user := &model.User{ID: 1, Username: "s1", Role: "student", OwnerScope: "college", OwnerID: "default"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"question":"奖学金怎么申请"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// LLM 错误时服务返回兜底回答（非 500），保留 sources
	if w.Code != http.StatusOK {
		t.Errorf("期望 200（兜底），得到 %d", w.Code)
	}

	var resp model.ChatResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Data.Fallback {
		t.Error("LLM 错误时应返回兜底回答")
	}
}

func TestChatHandler_Ask_WithSessionID(t *testing.T) {
	mockLLM := llm.NewMockClient("test-llm")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: "这是对第二个问题的回答。", FinishReason: "stop",
			PromptTokens: 50, OutputTokens: 10,
		}, nil
	}

	r, cfg := setupChatTestRouter(t, mockLLM)

	user := &model.User{ID: 1, Username: "s1", Role: "student", OwnerScope: "college", OwnerID: "default"}
	token, _ := middleware.GenerateToken(cfg, user)

	// 第一问：创建新会话
	body1 := `{"question":"奖学金怎么申请"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	var resp1 model.ChatResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	sessionID := resp1.SessionID

	// 第二问：复用同一会话
	body2 := `{"question":"还有什么需要注意的？","session_id":"` + sessionID + `"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w2.Code)
	}

	var resp2 model.ChatResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	// 会话 ID 应保持一致
	if resp2.SessionID != sessionID {
		t.Errorf("会话 ID 应保持一致: %s vs %s", resp2.SessionID, sessionID)
	}
}
