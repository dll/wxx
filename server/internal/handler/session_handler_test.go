package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

func setupSessionTestRouter(t *testing.T) (*gin.Engine, *config.Config, *repository.SessionRepo, *repository.MessageRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		JWTSecret:      "test-secret-session",
		JWTExpireHours: 2,
	}

	sessionRepo := repository.NewSessionRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	sessionSvc := service.NewSessionService(sessionRepo, messageRepo)
	sessionHandler := NewSessionHandler(sessionSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/sessions", sessionHandler.ListSessions)
	protected.GET("/sessions/:id/messages", sessionHandler.GetMessages)

	return r, cfg, sessionRepo, messageRepo
}

func TestSessionHandler_ListSessions_Empty(t *testing.T) {
	r, cfg, _, _ := setupSessionTestRouter(t)

	user := &model.User{ID: 1, Username: "newuser", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	// 空列表时 data 字段为 null
	if resp["data"] != nil {
		data := resp["data"].([]interface{})
		if len(data) != 0 {
			t.Errorf("期望空列表，得到 %d 条", len(data))
		}
	}
}

func TestSessionHandler_ListSessions_HasData(t *testing.T) {
	r, cfg, sessionRepo, _ := setupSessionTestRouter(t)

	// 创建几条会话
	sessionRepo.Create(&model.Session{SessionID: "sess-1", UserID: 1})
	sessionRepo.Create(&model.Session{SessionID: "sess-2", UserID: 1})
	sessionRepo.Create(&model.Session{SessionID: "sess-3", UserID: 2}) // 其他用户的

	user := &model.User{ID: 1, Username: "user1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("期望 2 条（仅用户 1 的），得到 %d 条", len(data))
	}
}

func TestSessionHandler_ListSessions_Unauthenticated(t *testing.T) {
	r, _, _, _ := setupSessionTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestSessionHandler_ListSessions_RespectsLimit(t *testing.T) {
	r, cfg, sessionRepo, _ := setupSessionTestRouter(t)

	for i := 0; i < 10; i++ {
		sessionRepo.Create(&model.Session{
			SessionID: "sess-limit-" + string(rune('0'+i)),
			UserID:    1,
		})
	}

	user := &model.User{ID: 1, Username: "user1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions?limit=3", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 3 {
		t.Errorf("limit=3 应返回 3 条，得到 %d", len(data))
	}
}

func TestSessionHandler_GetMessages_Success(t *testing.T) {
	r, cfg, sessionRepo, messageRepo := setupSessionTestRouter(t)

	sessionRepo.Create(&model.Session{SessionID: "sess-msg", UserID: 1})
	messageRepo.Create(&model.Message{SessionID: "sess-msg", Role: "user", Content: "你好"})
	messageRepo.Create(&model.Message{SessionID: "sess-msg", Role: "assistant", Content: "你好！"})

	user := &model.User{ID: 1, Username: "user1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/sess-msg/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) < 2 {
		t.Errorf("期望至少 2 条消息，得到 %d", len(data))
	}
}

func TestSessionHandler_GetMessages_CrossUser(t *testing.T) {
	r, cfg, sessionRepo, _ := setupSessionTestRouter(t)

	sessionRepo.Create(&model.Session{SessionID: "sess-cross", UserID: 2}) // 属于用户 2

	user := &model.User{ID: 1, Username: "user1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/sess-cross/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("跨用户访问应返回 403，得到 %d", w.Code)
	}
}

func TestSessionHandler_GetMessages_NotFound(t *testing.T) {
	r, cfg, _, _ := setupSessionTestRouter(t)

	user := &model.User{ID: 1, Username: "user1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/nonexistent/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("不存在的会话应返回 403，得到 %d", w.Code)
	}
}
