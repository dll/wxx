package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter IP 维度限流器
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter 创建 IP 维度限流器
// rps: 每秒请求数（令牌桶速率）
// burst: 令牌桶容量（突发请求数）
func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   rate.Limit(rps),
		b:   burst,
	}
}

// getLimiter 获取或创建 IP 对应的限流器
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}
	return limiter
}

// IPThrottleMiddleware IP 限流中间件构造函数
func IPThrottleMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rps, burst)
	return func(c *gin.Context) {
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

// UserRateLimiter 用户维度限流器
type UserRateLimiter struct {
	users map[int64]*rate.Limiter
	mu    sync.Mutex
	r     rate.Limit
	b     int
}

// NewUserRateLimiter 创建用户维度限流器
// rps: 每秒请求数
// burst: 令牌桶容量
func NewUserRateLimiter(rps float64, burst int) *UserRateLimiter {
	return &UserRateLimiter{
		users: make(map[int64]*rate.Limiter),
		r:     rate.Limit(rps),
		b:     burst,
	}
}

// getLimiter 获取或创建用户对应的限流器
func (u *UserRateLimiter) getLimiter(userID int64) *rate.Limiter {
	u.mu.Lock()
	defer u.mu.Unlock()

	limiter, exists := u.users[userID]
	if !exists {
		limiter = rate.NewLimiter(u.r, u.b)
		u.users[userID] = limiter
	}
	return limiter
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

// CleanupStaleLimiters 定期清理过期的限流器（可选，防止内存泄漏）
// 长时间不活跃的 IP/用户 限流器会被清理
func (i *IPRateLimiter) CleanupStaleLimiters(maxAge time.Duration) {
	// 简单实现：基于访问时间的清理需要额外结构
	// 生产环境建议封装带最后访问时间的 limiter
	i.mu.Lock()
	defer i.mu.Unlock()
	// 对于中小规模服务，map 增长缓慢可接受
	// 如需严格控制内存，可引入 lastAccessed 字段
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
