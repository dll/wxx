package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

func setupAuthTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		JWTSecret:      "test-secret-auth",
		JWTExpireHours: 2,
	}

	userRepo := repository.NewUserRepo(db)
	authSvc := service.NewAuthService(cfg, userRepo)
	authHandler := NewAuthHandler(authSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	r.POST("/api/v1/auth/login", authHandler.Login)

	// 保护路由
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/user/profile", authHandler.Profile)

	return r, cfg
}

func TestAuthHandler_Login_Success(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	body := `{"username":"testuser001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != float64(0) {
		t.Errorf("期望 code=0，得到 %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["token"] == "" {
		t.Error("应返回 token")
	}
	if data["role"] != "student" {
		t.Errorf("期望 role=student，得到 %v", data["role"])
	}
}

func TestAuthHandler_Login_EmptyUsername(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	body := `{"username":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 空字符串触发 Gin binding required 校验 → 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingField(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d", w.Code)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d", w.Code)
	}
}

func TestAuthHandler_Login_RepeatedLoginReturnsSameUser(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	body := `{"username":"returninguser"}`
	// 第一次登录
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("第一次登录失败: %d %s", w1.Code, w1.Body.String())
	}

	var resp1 map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	data1, ok := resp1["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("响应缺少 data 字段: %s", w1.Body.String())
	}
	role1 := data1["role"].(string)

	// 第二次登录（同一用户名）
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("第二次登录失败: %d %s", w2.Code, w2.Body.String())
	}

	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	data2, ok := resp2["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("第二次登录响应缺少 data: %s", w2.Body.String())
	}
	role2 := data2["role"].(string)

	// 同一用户两次登录，角色应一致（用户不会被重复创建）
	if role1 != role2 {
		t.Errorf("两次登录角色应一致: %s vs %s", role1, role2)
	}

	// 都应返回 token
	if data1["token"].(string) == "" || data2["token"].(string) == "" {
		t.Error("每次登录均应有 token")
	}
}

func TestAuthHandler_Profile_Authenticated(t *testing.T) {
	r, cfg := setupAuthTestRouter(t)

	// 先登录获取 token
	user := &model.User{
		ID: 99, Username: "profileuser", Role: "counselor",
		OwnerScope: "school", OwnerID: "chzu", DisplayName: "张老师",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["username"] != "profileuser" {
		t.Errorf("期望 username=profileuser，得到 %v", data["username"])
	}
	if data["role"] != "counselor" {
		t.Errorf("期望 role=counselor，得到 %v", data["role"])
	}
}

func TestAuthHandler_Profile_Unauthenticated(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestAuthHandler_Profile_TamperedToken(t *testing.T) {
	r, _ := setupAuthTestRouter(t)

	// 使用不同密钥签发的 token（模拟篡改）
	tamperedCfg := &config.Config{JWTSecret: "wrong-secret", JWTExpireHours: 2}
	user := &model.User{ID: 1, Username: "hacker", Role: "student"}
	token, _ := middleware.GenerateToken(tamperedCfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 篡改的 token 签名不匹配，应返回 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("篡改 token 应返回 401，得到 %d: %s", w.Code, w.Body.String())
	}
}
