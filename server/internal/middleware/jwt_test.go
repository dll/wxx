package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/jwtutil"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:      "test-secret-key",
		JWTExpireHours: 2,
	}

	user := &model.User{
		ID:          1,
		Username:    "testuser",
		Role:        "student",
		OwnerScope:  "college",
		OwnerID:     "default",
		DisplayName: "测试用户",
	}

	token, err := GenerateToken(cfg, user)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	if token == "" {
		t.Fatal("token 不应为空")
	}
}

func TestGenerateToken_EmptySecret(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:      "",
		JWTExpireHours: 2,
	}

	user := &model.User{Username: "test"}
	_, err := GenerateToken(cfg, user)
	if err == nil {
		t.Fatal("空白密钥应返回错误")
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWTSecret:      "test-secret",
		JWTExpireHours: 2,
	}

	user := &model.User{
		ID: 1, Username: "testuser", Role: "student",
		OwnerScope: "college", OwnerID: "default",
	}
	token, _ := GenerateToken(cfg, user)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestJWTAuth_BadFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Basic xxx")

	JWTAuth(cfg)(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer this-is-not-a-valid-jwt")

	JWTAuth(cfg)(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWTSecret:      "test-secret",
		JWTExpireHours: -1, // 立即过期
	}

	user := &model.User{ID: 1, Username: "test", Role: "student"}
	token, _ := GenerateToken(cfg, user)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401（已过期），得到 %d", w.Code)
	}
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgSign := &config.Config{JWTSecret: "secret-a", JWTExpireHours: 2}
	cfgVerify := &config.Config{JWTSecret: "secret-b"}

	user := &model.User{ID: 1, Username: "test", Role: "student"}
	token, _ := GenerateToken(cfgSign, user)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfgVerify)(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401（签名不匹配），得到 %d", w.Code)
	}
}

// TestJWTAuth_PendingStatusBlocked P0-01 回归：待审核游客不得访问业务接口
// 修复前 GuestRegister 对 Status="pending" 的用户无条件签发业务 JWT，且中间件
// 只验签名不验状态，pending 可访问对话/知识库等能力。修复后签发 token 携带
// status 字段，中间件对非 active 状态一律 403。
func TestJWTAuth_PendingStatusBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 2}

	user := &model.User{
		ID: 1, Username: "guest_138", Role: "guest",
		OwnerScope: "college", OwnerID: "default",
		Status: "pending", // 待审核游客
	}
	token, err := GenerateToken(cfg, user)
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403（pending 拒绝业务访问），得到 %d", w.Code)
	}
	if !c.IsAborted() {
		t.Error("pending 请求应被中止")
	}
}

// TestJWTAuth_RejectedStatusBlocked P0-01 回归：已拒绝游客同样被拦截
func TestJWTAuth_RejectedStatusBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 2}

	user := &model.User{ID: 2, Username: "guest_139", Role: "guest", Status: "rejected"}
	token, _ := GenerateToken(cfg, user)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403（rejected 拒绝访问），得到 %d", w.Code)
	}
}

// TestJWTAuth_ActiveStatusAllowed 回归：active 用户正常放行且注入 Status
func TestJWTAuth_ActiveStatusAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 2}

	user := &model.User{ID: 3, Username: "2021001", Role: "student", Status: "active"}
	token, _ := GenerateToken(cfg, user)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200（active 放行），得到 %d", w.Code)
	}
	if uc := GetUserContext(c); uc == nil || uc.Status != "active" {
		t.Error("active 用户上下文应注入 Status=active")
	}
}

// TestJWTAuth_EmptyStatusAllowed 兼容：旧 token（无 status 字段）不误伤
func TestJWTAuth_EmptyStatusAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 2}

	user := &model.User{ID: 4, Username: "legacy", Role: "student"}
	token, _ := GenerateToken(cfg, user)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuth(cfg)(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200（旧 token 兼容放行），得到 %d", w.Code)
	}
}

func TestGetUserContext_AfterAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWTSecret:      "test-secret",
		JWTExpireHours: 2,
	}

	user := &model.User{
		ID: 42, Username: "u42", Role: "counselor",
		OwnerScope: "school", OwnerID: "chzu",
	}
	token, _ := GenerateToken(cfg, user)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	var capturedUser *model.UserContext
	handler := JWTAuth(cfg)
	handler(c)

	if !c.IsAborted() {
		capturedUser = GetUserContext(c)
	}

	if capturedUser == nil {
		t.Fatal("应能从上下文中获取用户信息")
	}
	if capturedUser.UserID != 42 {
		t.Errorf("期望 UserID=42，得到 %d", capturedUser.UserID)
	}
	if capturedUser.Role != "counselor" {
		t.Errorf("期望 role=counselor，得到 %s", capturedUser.Role)
	}
}

func TestClaims_Expiry(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:      "test-secret",
		JWTExpireHours: 1,
	}

	user := &model.User{ID: 1, Username: "test", Role: "student"}
	token, _ := GenerateToken(cfg, user)

	// 验证 token 的过期时间大致正确
	claims := &jwtutil.CustomClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		t.Fatalf("解析 token 失败: %v", err)
	}

	expectedExpiry := time.Now().Add(time.Hour * time.Duration(cfg.JWTExpireHours))
	diff := claims.ExpiresAt.Time.Sub(expectedExpiry).Abs()
	if diff > 2*time.Second {
		t.Errorf("过期时间偏差过大: %v", diff)
	}
}
