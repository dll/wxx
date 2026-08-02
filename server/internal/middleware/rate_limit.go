package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// timedLimiter 带最后访问时间的限流器
type timedLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// IPRateLimiter IP 维度限流器
type IPRateLimiter struct {
	ips map[string]*timedLimiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter 创建 IP 维度限流器
// rps: 每秒请求数（令牌桶速率）
// burst: 令牌桶容量（突发请求数）
func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	ipl := &IPRateLimiter{
		ips: make(map[string]*timedLimiter),
		r:   rate.Limit(rps),
		b:   burst,
	}
	// 启动后台清理协程（每 10 分钟清理 30 分钟无活动的条目）
	go ipl.cleanupLoop(10*time.Minute, 30*time.Minute)
	return ipl
}

// getLimiter 获取或创建 IP 对应的限流器
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	tl, exists := i.ips[ip]
	if !exists {
		tl = &timedLimiter{
			limiter:    rate.NewLimiter(i.r, i.b),
			lastAccess: time.Now(),
		}
		i.ips[ip] = tl
	} else {
		tl.lastAccess = time.Now()
	}
	return tl.limiter
}

// cleanupLoop 定期清理过期条目（防止内存泄漏）
func (i *IPRateLimiter) cleanupLoop(interval, maxIdle time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		i.mu.Lock()
		now := time.Now()
		for key, tl := range i.ips {
			if now.Sub(tl.lastAccess) > maxIdle {
				delete(i.ips, key)
			}
		}
		i.mu.Unlock()
	}
}

// IPThrottleMiddleware IP 限流中间件构造函数
func IPThrottleMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rps, burst)
	return func(c *gin.Context) {
		// 静态资源不参与限流：Flutter Web 启动时会一次性发起大量
		// 资源请求（main.dart.js / canvaskit / 字体 / 图片等），
		// 全局限流会把它们误伤成 429，导致页面长时间空白。
		if isStaticAssetPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// isStaticAssetPath 判断是否为前端静态资源路径（由 Go 服务直出 Flutter 构建产物）。
// 此类请求走本地静态文件，成本低且可能高频突发，不应计入 IP 限流。
func isStaticAssetPath(path string) bool {
	if strings.HasPrefix(path, "/assets/") {
		return true
	}
	if strings.HasPrefix(path, "/downloads/") {
		return true
	}
	if strings.HasPrefix(path, "/icons/") {
		return true
	}
	if strings.HasPrefix(path, "/canvaskit/") {
		return true
	}
	switch path {
	case "/index.html", "/main.dart.js", "/flutter_bootstrap.js",
		"/flutter_service_worker.js", "/manifest.json", "/version.json",
		"/favicon.png", "/favicon.ico":
		return true
	}
	return false
}

// UserRateLimiter 用户维度限流器
type UserRateLimiter struct {
	users map[int64]*timedLimiter
	mu    sync.Mutex
	r     rate.Limit
	b     int
}

// NewUserRateLimiter 创建用户维度限流器
// rps: 每秒请求数
// burst: 令牌桶容量
func NewUserRateLimiter(rps float64, burst int) *UserRateLimiter {
	ul := &UserRateLimiter{
		users: make(map[int64]*timedLimiter),
		r:     rate.Limit(rps),
		b:     burst,
	}
	go ul.cleanupLoop(10*time.Minute, 30*time.Minute)
	return ul
}

// getLimiter 获取或创建用户对应的限流器
func (u *UserRateLimiter) getLimiter(userID int64) *rate.Limiter {
	u.mu.Lock()
	defer u.mu.Unlock()

	tl, exists := u.users[userID]
	if !exists {
		tl = &timedLimiter{
			limiter:    rate.NewLimiter(u.r, u.b),
			lastAccess: time.Now(),
		}
		u.users[userID] = tl
	} else {
		tl.lastAccess = time.Now()
	}
	return tl.limiter
}

// cleanupLoop 定期清理过期的用户限流器
func (u *UserRateLimiter) cleanupLoop(interval, maxIdle time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		u.mu.Lock()
		now := time.Now()
		for key, tl := range u.users {
			if now.Sub(tl.lastAccess) > maxIdle {
				delete(u.users, key)
			}
		}
		u.mu.Unlock()
	}
}

// UserThrottleMiddleware 用户限流中间件构造函数
// 需在 JWTAuth 之后使用，从用户上下文获取 UserID
func UserThrottleMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := NewUserRateLimiter(rps, burst)
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未登录",
			})
			return
		}
		if !limiter.getLimiter(userCtx.UserID).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// 常用限流配置常量
const (
	// GlobalIPRPS 全局限流：100 req/min/IP ≈ 1.67 rps
	GlobalIPRPS   = 100.0 / 60.0
	GlobalIPBurst = 10

	// LoginIPRPS 登录接口限流：5 req/min/IP ≈ 0.083 rps
	LoginIPRPS   = 5.0 / 60.0
	LoginIPBurst = 3

	// ChatUserRPS 聊天接口限流：60 req/min/用户 = 1 rps
	ChatUserRPS   = 1.0
	ChatUserBurst = 5
)

// GlobalIPRateLimiter 全局限流中间件（IP 维度，100 req/min/IP）
func GlobalIPRateLimiter() gin.HandlerFunc {
	return IPThrottleMiddleware(GlobalIPRPS, GlobalIPBurst)
}

// LoginIPRateLimiter 登录接口限流中间件（IP 维度，5 req/min/IP）
func LoginIPRateLimiter() gin.HandlerFunc {
	return IPThrottleMiddleware(LoginIPRPS, LoginIPBurst)
}

// ChatUserRateLimiter 聊天接口限流中间件（用户维度，60 req/min/用户）
func ChatUserRateLimiter() gin.HandlerFunc {
	return UserThrottleMiddleware(ChatUserRPS, ChatUserBurst)
}

// RateLimitByMinute 快捷函数：按分钟速率创建 IP 限流中间件
func RateLimitByMinute(reqPerMin int) gin.HandlerFunc {
	rps := float64(reqPerMin) / 60.0
	burst := reqPerMin / 10
	if burst < 1 {
		burst = 1
	}
	return IPThrottleMiddleware(rps, burst)
}

// RateLimitByMinuteWithBurst 快捷函数：按分钟速率+突发创建 IP 限流中间件
func RateLimitByMinuteWithBurst(reqPerMin, burst int) gin.HandlerFunc {
	rps := float64(reqPerMin) / 60.0
	return IPThrottleMiddleware(rps, burst)
}

// UserRateLimitByMinute 快捷函数：按分钟速率创建用户限流中间件
func UserRateLimitByMinute(reqPerMin int) gin.HandlerFunc {
	rps := float64(reqPerMin) / 60.0
	burst := reqPerMin / 10
	if burst < 1 {
		burst = 1
	}
	return UserThrottleMiddleware(rps, burst)
}

// Int64FromString 辅助：字符串转 int64，失败返回 0
func Int64FromString(s string) int64 {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return 0
}
