package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsStaticAssetPath(t *testing.T) {
	cases := map[string]bool{
		"/assets/assets/fonts/roboto-regular.ttf": true,
		"/assets/baidu_campus_map.html":          true,
		"/assets/Canvaskit/wasm":                 true,
		"/canvaskit/canvaskit.wasm":              true,
		"/downloads/蔚小芯-v1.0.0.apk":             true,
		"/icons/favicon.png":                     true,
		"/index.html":                            true,
		"/main.dart.js":                          true,
		"/flutter_bootstrap.js":                  true,
		"/manifest.json":                         true,
		"/api/v1/chat":                           false,
		"/api/v1/login":                          false,
		"/health":                                false,
		"/":                                      false,
		"/assets":                                false,
	}
	for path, want := range cases {
		if got := isStaticAssetPath(path); got != want {
			t.Errorf("isStaticAssetPath(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestIPThrottleSkipsStatic 静态资源请求不受 IP 限流影响，即使突发大量也不会被 429。
func TestIPThrottleSkipsStatic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rps := 1.0 / 60.0 // 1 req/min，严格到正常请求都会被拦
	burst := 1
	handler := IPThrottleMiddleware(rps, burst)

	for _, path := range []string{
		"/assets/assets/fonts/roboto-regular.ttf",
		"/main.dart.js",
		"/flutter_bootstrap.js",
		"/canvaskit/canvaskit.wasm",
	} {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodGet, path, nil)
		handler(ctx)
		if ctx.IsAborted() {
			t.Fatalf("静态资源 %s 不应被限流中间件拦截", path)
		}
	}
}

// TestIPThrottleRejectsApi 非静态 API 请求仍受 IP 限流约束。
func TestIPThrottleRejectsApi(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rps := 1.0 / 60.0 // 1 req/min
	burst := 1
	handler := IPThrottleMiddleware(rps, burst)

	// 令牌桶初始 1 个令牌：第一次放行，第二次即被限流。
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)
		handler(ctx)
		if i == 0 {
			if ctx.IsAborted() {
				t.Fatal("首次请求应放行")
			}
		} else if !ctx.IsAborted() {
			t.Fatal("超过令牌桶容量后的 API 请求应被拦截(429)")
		}
	}
}
