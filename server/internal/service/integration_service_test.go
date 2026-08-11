package service

import (
	"encoding/json"
	"net"
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

// ── P0-03 SSRF 防护回归 ──

// TestIsBlockedIP P0-03 回归：私网/环回/链路本地地址必须被拒绝
func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "localhost", "::1",
		"10.0.0.1", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", // 云元数据服务（经典 SSRF 目标）
		"0.0.0.0",
	}
	for _, h := range blocked {
		if !isBlockedIP(h) {
			t.Errorf("地址 %s 应被判定为禁止访问", h)
		}
	}

	allowed := []string{"8.8.8.8", "114.114.114.114", "xuegong.chzu.edu.cn"}
	for _, h := range allowed {
		if isBlockedIP(h) {
			t.Errorf("地址 %s 不应被误判为禁止", h)
		}
	}
}

// TestIsPrivateIP 校验网段判断函数本身
func TestIsPrivateIP(t *testing.T) {
	if !isPrivateIP(net.ParseIP("192.168.5.5")) {
		t.Error("192.168.5.5 应为私网")
	}
	if !isPrivateIP(net.ParseIP("127.0.0.1")) {
		t.Error("127.0.0.1 应为环回")
	}
	if !isPrivateIP(net.ParseIP("169.254.169.254")) {
		t.Error("169.254.169.254 应为链路本地")
	}
	if isPrivateIP(net.ParseIP("8.8.8.8")) {
		t.Error("8.8.8.8 不应为私网")
	}
}

// TestIsAllowedHost 主机白名单判断（子域放行、后缀伪造拒绝）
func TestIsAllowedHost(t *testing.T) {
	s := &IntegrationService{allowedHosts: map[string]string{"学工系统": "xuegong.chzu.edu.cn"}}

	if !s.isAllowedHost("xuegong.chzu.edu.cn", "xuegong.chzu.edu.cn") {
		t.Error("同主机应放行")
	}
	if !s.isAllowedHost("api.xuegong.chzu.edu.cn", "xuegong.chzu.edu.cn") {
		t.Error("子域应放行")
	}
	if s.isAllowedHost("evil.com", "xuegong.chzu.edu.cn") {
		t.Error("无关域名应拒绝")
	}
	if s.isAllowedHost("xuegong.chzu.edu.cn.evil.com", "xuegong.chzu.edu.cn") {
		t.Error("后缀伪造域名应拒绝")
	}
	if s.isAllowedHost("", "xuegong.chzu.edu.cn") {
		t.Error("空主机应拒绝")
	}
}

// TestIsAllowedRedirect 重定向目标校验（精确 host:port 匹配）
func TestIsAllowedRedirect(t *testing.T) {
	s := &IntegrationService{}
	if !s.isAllowedRedirect("xuegong.chzu.edu.cn", "xuegong.chzu.edu.cn") {
		t.Error("同主机应允许")
	}
	// 同主机不同端口（如 80→8080 跳内网）应拒绝
	if s.isAllowedRedirect("xuegong.chzu.edu.cn:80", "xuegong.chzu.edu.cn:8080") {
		t.Error("同主机跨端口重定向应拒绝")
	}
	if s.isAllowedRedirect("xuegong.chzu.edu.cn", "evil.com") {
		t.Error("跨主机重定向应拒绝")
	}
}

// TestHostOfBaseURL baseURL 主机提取
func TestHostOfBaseURL(t *testing.T) {
	if h := hostOfBaseURL("https://xuegong.chzu.edu.cn/base"); h != "xuegong.chzu.edu.cn" {
		t.Errorf("期望提取主机 xuegong.chzu.edu.cn，得到 %s", h)
	}
	if h := hostOfBaseURL("not a url"); h != "" {
		t.Errorf("非法 URL 应返回空，得到 %s", h)
	}
	if h := hostOfBaseURL(""); h != "" {
		t.Errorf("空 URL 应返回空，得到 %s", h)
	}
}

// TestProxyGet_PathSmugglingBlocked P0-03 回归：
// path 内嵌绝对 URL（协议走私）必须被拒绝，不得请求白名单以外主机。
func TestProxyGet_PathSmugglingBlocked(t *testing.T) {
	goodSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"good":true}`))
	}))
	defer goodSrv.Close()

	evilSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"evil":true}`))
	}))
	defer evilSrv.Close()

	cfg := &config.Config{
		XuegongBaseURL: goodSrv.URL,
		XuegongToken:   "token",
	}
	svc := NewIntegrationService(cfg)

	// 尝试把 evilSrv 作为绝对 URL 拼进 path
	_, err := svc.ProxyXuegong(evilSrv.URL, nil)
	if err == nil {
		t.Fatal("path 内嵌绝对 URL（协议走私）应被拒绝")
	}

	// path 以 / 开头的正常路径应成功
	if _, err := svc.ProxyXuegong("/api/hello", nil); err != nil {
		t.Fatalf("正常路径代理应成功: %v", err)
	}
}

// TestProxyGet_RedirectCrossHostBlocked P0-03 回归：
// 302 跳转到非白名单主机必须被拒绝（防止 Authorization 头外泄）。
func TestProxyGet_RedirectCrossHostBlocked(t *testing.T) {
	evilSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"evil":true}`))
	}))
	defer evilSrv.Close()

	goodSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 重定向到 evilSrv（跨主机）
		http.Redirect(w, r, evilSrv.URL+"/steal", http.StatusFound)
	}))
	defer goodSrv.Close()

	cfg := &config.Config{
		XuegongBaseURL: goodSrv.URL,
		XuegongToken:   "token",
	}
	svc := NewIntegrationService(cfg)

	_, err := svc.ProxyXuegong("/redirect", nil)
	if err == nil {
		t.Fatal("跨主机 302 跟随应被拒绝")
	}
}
