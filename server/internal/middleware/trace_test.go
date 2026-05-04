package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTraceID_GeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	TraceID()(c)

	traceID := GetTraceID(c)
	if traceID == "" {
		t.Fatal("应生成非空 TraceID")
	}

	// 验证响应头
	respTraceID := w.Header().Get("X-Trace-ID")
	if respTraceID != traceID {
		t.Errorf("响应头 X-Trace-ID 应与上下文一致: %s vs %s", respTraceID, traceID)
	}
}

func TestTraceID_PassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("X-Trace-ID", "custom-trace-123")

	TraceID()(c)

	traceID := GetTraceID(c)
	if traceID != "custom-trace-123" {
		t.Errorf("应透传已有的 TraceID: 期望 custom-trace-123 得到 %s", traceID)
	}
}

func TestTraceID_UniquePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

		TraceID()(c)

		id := GetTraceID(c)
		if ids[id] {
			t.Errorf("TraceID 重复: %s", id)
		}
		ids[id] = true
	}
}

func TestGetTraceID_NoMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// 不调用 TraceID 中间件，直接获取
	id := GetTraceID(c)
	if id != "" {
		t.Errorf("未注入时应返回空字符串，得到 %s", id)
	}
}
