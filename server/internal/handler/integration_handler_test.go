package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

func setupIntegrationTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "test-secret-integration",
		JWTExpireHours: 2,
		// 不配置外部系统 URL，测试不可用场景
		XuegongBaseURL: "",
		XuegongToken:   "",
		YBTBaseURL:     "",
		YBTToken:       "",
	}

	integrationSvc := service.NewIntegrationService(cfg)
	integrationHandler := NewIntegrationHandler(integrationSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/integration/status", integrationHandler.Status)
	protected.GET("/integration/xuegong/*path", integrationHandler.ProxyXuegong)
	protected.GET("/integration/ybt/*path", integrationHandler.ProxyYBT)

	return r, cfg
}

func TestIntegrationHandler_Status(t *testing.T) {
	r, cfg := setupIntegrationTestRouter(t)

	user := &model.User{
		ID: 1, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integration/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["code"].(float64) != 0 {
		t.Errorf("期望 code=0，得到 %v", resp["code"])
	}
	data := resp["data"].(map[string]interface{})
	xg := data["xuegong"].(map[string]interface{})
	if xg["available"].(bool) != false {
		t.Error("学工系统未配置，available 应为 false")
	}
	ybt := data["ybt"].(map[string]interface{})
	if ybt["available"].(bool) != false {
		t.Error("一表通未配置，available 应为 false")
	}
}

func TestIntegrationHandler_ProxyXuegong_Unconfigured(t *testing.T) {
	r, cfg := setupIntegrationTestRouter(t)

	user := &model.User{
		ID: 1, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integration/xuegong/students", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("未配置时期望 502，得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegrationHandler_ProxyYBT_Unconfigured(t *testing.T) {
	r, cfg := setupIntegrationTestRouter(t)

	user := &model.User{
		ID: 1, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integration/ybt/departments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("未配置时期望 502，得到 %d: %s", w.Code, w.Body.String())
	}
}
