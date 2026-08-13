package middleware

import (
	"database/sql"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var auditWg sync.WaitGroup

// WaitFlush 等待所有未完成的审计日志写入完成（在服务优雅关闭时调用）
func WaitFlush() {
	auditWg.Wait()
}

// AuditLog 审计日志中间件
// 在请求处理完成后异步记录审计日志到 audit_logs 表
func AuditLog(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 获取 TraceID（由 TraceID 中间件提前注入）
		traceID, _ := c.Get("trace_id")
		traceStr, _ := traceID.(string)

		// 继续处理请求
		c.Next()

		// 请求处理完成后记录审计日志
		duration := time.Since(start).Milliseconds()
		user := GetUserContext(c)

		var userID *string
		username := ""
		role := ""
		if user != nil {
			s := strconv.FormatInt(user.UserID, 10)
			userID = &s
			username = user.Username
			role = user.Role
		}

		// ── 安全：gin.Context 在请求结束（c.Next 返回）后会被 sync.Pool 复用，
		// 异步 goroutine 再读 c.Request/c.Writer 属于数据竞争（可能读到下个请求的数据）。
		// 因此先把所有需要写入审计的值拷贝为局部变量，再传入闭包。
		method := c.Request.Method
		fullPath := c.FullPath()
		urlPath := c.Request.URL.Path
		clientIP := c.ClientIP()
		statusCode := c.Writer.Status()

		// 异步写入审计日志（不阻塞响应）
		auditWg.Add(1)
		go func() {
			defer auditWg.Done()
			_, err := db.Exec(
				`INSERT INTO audit_logs (user_id, username, role, action, resource, detail, trace_id, ip, duration_ms, result_code)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				userID,
				username,
				role,
				method,
				fullPath,
				urlPath,
				traceStr,
				clientIP,
				duration,
				statusCode,
			)
			if err != nil {
				log.Printf("审计日志写入失败: %v [trace=%s]", err, traceStr)
			}
		}()
	}
}
