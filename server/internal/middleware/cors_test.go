package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Origin", "http://localhost:3000")

	CORS()(c)

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Errorf("应允许 localhost:3000 来源，得到 %s", allowOrigin)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Origin", "https://evil.com")

	CORSWithConfig("http://localhost:3000,http://localhost:8080", false)(c)

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("不允许的来源不应返回 Allow-Origin，得到 %s", allowOrigin)
	}
}

func TestCORS_CommonHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:8080",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:8080",
	}

	for _, origin := range allowedOrigins {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Origin", origin)

		CORS()(c)

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("应设置 Access-Control-Allow-Methods 头")
		}
		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("应设置 Access-Control-Allow-Headers 头")
		}
		if w.Header().Get("Access-Control-Max-Age") == "" {
			t.Error("应设置 Access-Control-Max-Age 头")
		}
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodOptions, "/test", nil)
	c.Request.Header.Set("Origin", "http://localhost:3000")

	CORS()(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS 预检请求应返回 204，得到 %d", w.Code)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	// 不设置 Origin 头

	CORS()(c)

	if w.Code == http.StatusNoContent {
		t.Error("非 OPTIONS 请求不应被拦截")
	}
	// 没有 Origin 头时不应设置 Allow-Origin
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("无 Origin 头时不应设置 Access-Control-Allow-Origin")
	}
}
