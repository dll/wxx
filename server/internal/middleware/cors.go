package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 跨域资源共享中间件
// 开发环境允许所有来源，生产环境应配置白名单
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 允许的来源（开发 + Vercel 生产环境）
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"http://localhost:9091",
			"http://localhost:9092",
			"http://localhost:9093",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
			"http://127.0.0.1:9091",
			"http://127.0.0.1:9092",
			"http://127.0.0.1:9093",
			"https://wxx-server.vercel.app",
			"https://wxx-server-czldl.vercel.app",
			"https://wxx-server-osgisone-czldl.vercel.app",
			"https://web-czldl.vercel.app",
			"https://wxx-frontend-czldl.vercel.app",
			"https://wxx-frontend.vercel.app",
			"https://wxx.pydaydayup.xyz",
			"https://api.pydaydayup.xyz",
			"https://wxx-frontend-lnmddge3y-czldl.vercel.app",
		}

		allowed := false
		for _, o := range allowedOrigins {
			if strings.EqualFold(origin, o) {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Trace-ID")
		c.Header("Access-Control-Expose-Headers", "X-Trace-ID")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		// 预检请求直接返回
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
