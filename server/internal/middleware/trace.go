package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceID 注入 TraceID 中间件
// 为每个请求生成唯一追踪 ID，写入 gin.Context 和响应头
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用请求头中已有的 TraceID（支持链路追踪透传）
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 存入上下文 + 响应头
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}

// GetTraceID 从 gin.Context 中获取 TraceID
func GetTraceID(c *gin.Context) string {
	if v, exists := c.Get("trace_id"); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
