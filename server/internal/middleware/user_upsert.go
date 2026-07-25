package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// UserUpserter 用户 upsert 接口（避免直接依赖 repository 包）
type UserUpserter interface {
	UpsertFromContext(userCtx *model.UserContext) error
}

func maskNameForLog(name string) string {
	if len(name) == 0 {
		return name
	}
	r := []rune(name)
	if len(r) == 1 {
		return string(r[0]) + "*"
	}
	return string(r[0]) + "**"
}

// EnsureUserExists 确保 JWT 中的用户存在于数据库（JIT 创建）
// 用于 Vercel 等无服务器环境，冷启动时数据库为空但 JWT 仍然有效。
func EnsureUserExists(upserter UserUpserter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			c.Next()
			return
		}

		if err := upserter.UpsertFromContext(userCtx); err != nil {
			// 账户被停用/拒绝 → 403 禁止访问
			if errors.Is(err, model.ErrAccountDisabled) {
				log.Printf("[EnsureUserExists] 账户已停用，拒绝访问 user=%s", maskNameForLog(userCtx.Username))
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "账户已被停用，请联系管理员",
				})
				return
			}
			// 令牌已被吊销（版本过旧）→ 401 要求重新登录
			if errors.Is(err, model.ErrTokenRevoked) {
				log.Printf("[EnsureUserExists] 令牌已吊销 user=%s", maskNameForLog(userCtx.Username))
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "登录已失效，请重新登录",
				})
				return
			}
			// 其它错误 → 500
			log.Printf("[EnsureUserExists] 用户 upsert 失败 user=%s err=%v", maskNameForLog(userCtx.Username), err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "用户状态异常，请重新登录",
			})
			return
		}

		c.Next()
	}
}
