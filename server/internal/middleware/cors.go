package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 跨域资源共享中间件（向后兼容版本，默认允许所有来源）
// 建议使用 CORSWithConfig 传入配置以获得更精细的控制
func CORS() gin.HandlerFunc {
	return CORSWithConfig("*", false)
}

// CORSWithConfig 带配置的 CORS 跨域中间件
// allowedOrigins: 允许的来源列表，逗号分隔，支持通配符子域名如 *.vercel.app
// isRelease: 是否为生产模式
func CORSWithConfig(allowedOrigins string, isRelease bool) gin.HandlerFunc {
	origins := parseAllowedOrigins(allowedOrigins)
	allowAll := len(origins) == 1 && origins[0] == "*"

	if isRelease && allowAll {
		log.Printf("[WARN] CORS 配置为允许所有来源 (*)，生产环境建议配置具体白名单")
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			allowed := allowAll || isOriginAllowed(origin, origins)
			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Trace-ID")
		c.Header("Access-Control-Expose-Headers", "X-Trace-ID")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// parseAllowedOrigins 解析逗号分隔的允许来源列表
func parseAllowedOrigins(allowedOrigins string) []string {
	if allowedOrigins == "" {
		return []string{"*"}
	}
	parts := strings.Split(allowedOrigins, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"*"}
	}
	return result
}

// isOriginAllowed 检查 origin 是否在允许列表中
// 支持精确匹配和通配符子域名匹配（如 *.example.com）
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	origin = strings.ToLower(origin)
	originHost := extractHost(origin)

	for _, allowed := range allowedOrigins {
		allowed = strings.ToLower(allowed)
		if allowed == "*" {
			return true
		}

		// 精确匹配（完整 origin）
		if origin == allowed {
			return true
		}

		// 通配符子域名匹配（仅针对 host 部分）
		if strings.HasPrefix(allowed, "*.") {
			allowedHost := allowed[2:]
			// 检查是否为子域名：originHost 以 .allowedHost 结尾
			if strings.HasSuffix(originHost, "."+allowedHost) {
				return true
			}
		}
	}
	return false
}

// extractHost 从 origin 中提取 host（去掉协议和端口）
func extractHost(origin string) string {
	host := origin
	// 去掉协议
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}
	// 去掉端口
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}
