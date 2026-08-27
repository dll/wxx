package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupRepairTokenRouter 构造一个仅挂载 RepairAgentTokenAuth 的极小 gin 路由。
func setupRepairTokenRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RepairAgentTokenAuth())
	r.POST("/internal/repair-tasks/next", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// TestRepairAgentToken_Unconfigured_Returns404 token 未配置时，内部端点应返回 404
// （而非 401），避免泄露路由存在性。
func TestRepairAgentToken_Unconfigured_Returns404(t *testing.T) {
	// 确保环境变量未设置
	os.Unsetenv(RepairAgentTokenKey)
	t.Cleanup(func() { os.Unsetenv(RepairAgentTokenKey) })

	r := setupRepairTokenRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/repair-tasks/next", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("token 未配置时应返回 404，得到 %d (body=%s)", w.Code, w.Body.String())
	}
}

// TestRepairAgentToken_MissingHeader_401 已配置 token 但请求缺失 Authorization 头 → 401。
func TestRepairAgentToken_MissingHeader_401(t *testing.T) {
	os.Setenv(RepairAgentTokenKey, "secret-token-123")
	t.Cleanup(func() { os.Unsetenv(RepairAgentTokenKey) })

	r := setupRepairTokenRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/repair-tasks/next", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("缺失 token 时应返回 401，得到 %d (body=%s)", w.Code, w.Body.String())
	}
}

// TestRepairAgentToken_WrongToken_401 错误 token → 401。
func TestRepairAgentToken_WrongToken_401(t *testing.T) {
	os.Setenv(RepairAgentTokenKey, "secret-token-123")
	t.Cleanup(func() { os.Unsetenv(RepairAgentTokenKey) })

	r := setupRepairTokenRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/repair-tasks/next", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("错误 token 时应返回 401，得到 %d (body=%s)", w.Code, w.Body.String())
	}
}

// TestRepairAgentToken_NoBearerPrefix_401 Authorization 头缺 Bearer 前缀 → 401。
func TestRepairAgentToken_NoBearerPrefix_401(t *testing.T) {
	os.Setenv(RepairAgentTokenKey, "secret-token-123")
	t.Cleanup(func() { os.Unsetenv(RepairAgentTokenKey) })

	r := setupRepairTokenRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/repair-tasks/next", nil)
	req.Header.Set("Authorization", "secret-token-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("缺 Bearer 前缀时应返回 401，得到 %d (body=%s)", w.Code, w.Body.String())
	}
}

// TestRepairAgentToken_CorrectToken_200 正确 token 放行。
func TestRepairAgentToken_CorrectToken_200(t *testing.T) {
	os.Setenv(RepairAgentTokenKey, "secret-token-123")
	t.Cleanup(func() { os.Unsetenv(RepairAgentTokenKey) })

	r := setupRepairTokenRouter()
	req := httptest.NewRequest(http.MethodPost, "/internal/repair-tasks/next", nil)
	req.Header.Set("Authorization", "Bearer secret-token-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("正确 token 时应返回 200，得到 %d (body=%s)", w.Code, w.Body.String())
	}
}
