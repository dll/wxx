package handler

import (
	"bytes"
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

func setupFeedbackTestRouter(t *testing.T) (*gin.Engine, *config.Config, *repository.UserRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		JWTSecret:      "test-secret-feedback",
		JWTExpireHours: 2,
	}

	userRepo := repository.NewUserRepo(db)
	feedbackRepo := repository.NewFeedbackRepo(db)
	screenshotRepo := repository.NewFeedbackScreenshotRepo(db)
	feedbackSvc := service.NewFeedbackService(feedbackRepo, userRepo)
	feedbackHandler := NewFeedbackHandler(feedbackSvc, screenshotRepo)

	r := gin.New()
	r.Use(middleware.TraceID())

	secured := r.Group("/api/v1")
	secured.Use(middleware.JWTAuth(cfg))
	secured.POST("/feedback", feedbackHandler.Submit)

	return r, cfg, userRepo
}

func TestFeedbackHandler_Submit_Success(t *testing.T) {
	r, cfg, userRepo := setupFeedbackTestRouter(t)

	// 创建测试用户
	user := &model.User{ID: 1, Username: "feedbackuser", Role: "student", OwnerScope: "school"}
	if _, err := userRepo.Create(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	token, _ := middleware.GenerateToken(cfg, user)

	body := map[string]string{
		"category":       "answer_error",
		"content":        "入党介绍人，写思想汇报",
		"message_id":     "msg-123",
		"resource_id":    "",
		"screenshot_url": "data:image/png;base64,iVBORw0KGgo=",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["code"] != float64(0) {
		t.Errorf("期望 code=0，得到 %v", resp["code"])
	}
	if resp["message"] != "反馈已提交" {
		t.Errorf("期望 message=反馈已提交，得到 %v", resp["message"])
	}
}

// TestFeedbackHandler_Submit_UserNotInDB 验证 JWT 中的用户不存在于数据库时（例如 Vercel
// /tmp 数据库冷启动后旧 token 仍有效），外键约束会导致 500，便于确认线上错误来源。
func TestFeedbackHandler_Submit_UserNotInDB(t *testing.T) {
	r, cfg, _ := setupFeedbackTestRouter(t)

	// 构造一个数据库中没有对应记录的用户的 token
	user := &model.User{ID: 99999, Username: "ghostuser", Role: "student", OwnerScope: "school"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"category":"answer_error","content":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 用户不存在时应返回 401，提示重新登录，而不是 500
	t.Logf("UserNotInDB 响应: %d %s", w.Code, w.Body.String())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestFeedbackHandler_Submit_Unauthenticated(t *testing.T) {
	r, _, _ := setupFeedbackTestRouter(t)

	body := `{"category":"answer_error","content":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestFeedbackHandler_Submit_InvalidCategory(t *testing.T) {
	r, cfg, userRepo := setupFeedbackTestRouter(t)

	user := &model.User{ID: 1, Username: "feedbackuser2", Role: "student", OwnerScope: "school"}
	if _, err := userRepo.Create(user); err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"category":"invalid","content":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", w.Code, w.Body.String())
	}
}
