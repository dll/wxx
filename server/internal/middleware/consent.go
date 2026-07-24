package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireConsent 隐私授权中间件
// 要求用户已同意隐私政策与用户协议，否则返回 403 并提示授权
// 仅在 JWT 认证之后使用
func RequireConsent() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少用户信息",
			})
			return
		}

		// 检查授权状态
		if !user.Consented {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "请先阅读并同意隐私政策与用户协议",
				"data": gin.H{
					"required":   true,
					"action":     "show_consent",
					"privacyUrl": "/docs/privacy-policy",
					"termsUrl":   "/docs/user-agreement",
				},
			})
			return
		}

		c.Next()
	}
}
