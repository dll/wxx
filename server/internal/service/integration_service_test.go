package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
)

func TestIntegrationService_IsXuegongAvailable_True(t *testing.T) {
	cfg := &config.Config{
		XuegongBaseURL: "https://xuegong.example.com",
		XuegongToken:   "token-123",
	}
	svc := NewIntegrationService(cfg)

	if !svc.IsXuegongAvailable() {
		t.Error("配置了 URL 和 Token 时应返回 true")
	}
}

func TestIntegrationService_IsXuegongAvailable_False(t *testing.T) {
	cfg := &config.Config{
		XuegongBaseURL: "",
		XuegongToken:   "",
	}
	svc := NewIntegrationService(cfg)

	if svc.IsXuegongAvailable() {
		t.Error("未配置时应返回 false")
	}
}

func TestIntegrationService_IsXuegongAvailable_NoToken(t *testing.T) {
	cfg := &config.Config{
		XuegongBaseURL: "https://xuegong.example.com",
		XuegongToken:   "",
	}
	svc := NewIntegrationService(cfg)

	if svc.IsXuegongAvailable() {
		t.Error("有 URL 但无 Token 时应返回 false")
	}
}

func TestIntegrationService_IsYBTAvailable_True(t *testing.T) {
	cfg := &config.Config{
		YBTBaseURL: "https://ybt.example.com",
		YBTToken:   "token-456",
	}
	svc := NewIntegrationService(cfg)

	if !svc.IsYBTAvailable() {
		t.Error("配置了 URL 和 Token 时应返回 true")
	}
}

func TestIntegrationService_IsYBTAvailable_False(t *testing.T) {
	cfg := &config.Config{
		YBTBaseURL: "",
		YBTToken:   "",
	}
	svc := NewIntegrationService(cfg)

	if svc.IsYBTAvailable() {
		t.Error("未配置时应返回 false")
	}
}

// ── Proxy 测试（使用 httptest.Server） ──

func TestIntegrationService_ProxyXuegong_Success(t *testing.T) {
	// 模拟学工系统后端
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 Authorization 头
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-xg-token" {
			t.Errorf("期望 Authorization=Bearer test-xg-token，得到 %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "data": "xuegong"})
	}))
	defer backend.Close()

	cfg := &config.Config{
		XuegongBaseURL: backend.URL,
		XuegongToken:   "test-xg-token",
	}
	svc := NewIntegrationService(cfg)

	result, err := svc.ProxyXuegong("/api/students", map[string]string{"class": "2026"})
	if err != nil {
		t.Fatalf("ProxyXuegong 失败: %v", err)
	}
	var data map[string]string
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if data["data"] != "xuegong" {
		t.Errorf("期望 data=xuegong，得到 %s", data["data"])
	}
}

func TestIntegrationService_ProxyXuegong_NotConfigured(t *testing.T) {
	cfg := &config.Config{
		XuegongBaseURL: "",
		XuegongToken:   "",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyXuegong("/api/students", nil)
	if err == nil {
		t.Fatal("未配置时应返回错误")
	}
}

func TestIntegrationService_ProxyXuegong_BackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		XuegongBaseURL: backend.URL,
		XuegongToken:   "token",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyXuegong("/api/error", nil)
	if err == nil {
		t.Fatal("后端返回 500 时应返回错误")
	}
}

func TestIntegrationService_ProxyXuegong_ConnectionRefused(t *testing.T) {
	cfg := &config.Config{
		XuegongBaseURL: "http://127.0.0.1:19999", // 未监听的端口
		XuegongToken:   "token",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyXuegong("/api/test", nil)
	if err == nil {
		t.Fatal("连接失败时应返回错误")
	}
}

func TestIntegrationService_ProxyYBT_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "data": "ybt"})
	}))
	defer backend.Close()

	cfg := &config.Config{
		YBTBaseURL: backend.URL,
		YBTToken:   "test-ybt-token",
	}
	svc := NewIntegrationService(cfg)

	result, err := svc.ProxyYBT("/api/data", map[string]string{"type": "score"})
	if err != nil {
		t.Fatalf("ProxyYBT 失败: %v", err)
	}
	var data map[string]string
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if data["data"] != "ybt" {
		t.Errorf("期望 data=ybt，得到 %s", data["data"])
	}
}

func TestIntegrationService_ProxyYBT_NotConfigured(t *testing.T) {
	cfg := &config.Config{
		YBTBaseURL: "",
		YBTToken:   "",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyYBT("/api/data", nil)
	if err == nil {
		t.Fatal("未配置时应返回错误")
	}
}

func TestIntegrationService_ProxyYBT_BackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer backend.Close()

	cfg := &config.Config{
		YBTBaseURL: backend.URL,
		YBTToken:   "token",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyYBT("/api/fail", nil)
	if err == nil {
		t.Fatal("后端返回 502 时应返回错误")
	}
}

func TestIntegrationService_ProxyGet_WithQueryParams(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证查询参数被正确传递
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("期望 page=1，得到 %s", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("size") != "10" {
			t.Errorf("期望 size=10，得到 %s", r.URL.Query().Get("size"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		XuegongBaseURL: backend.URL,
		XuegongToken:   "token",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyXuegong("/api/list", map[string]string{
		"page": "1",
		"size": "10",
	})
	if err != nil {
		t.Fatalf("带查询参数代理失败: %v", err)
	}
}
