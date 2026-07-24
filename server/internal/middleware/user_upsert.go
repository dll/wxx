package middleware

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// UserUpserter 用户 upsert 接口（避免直接依赖 repository 包）
type UserUpserter interface {
	UpsertFromContext(userCtx *model.UserContext) error
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
			log.Printf("[EnsureUserExists] 用户 upsert 失败 user=%s err=%v", userCtx.Username, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "用户状态异常，请重新登录",
			})
			return
		}

		c.Next()
	}
}
