package auth

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RequireCapability 能力授权中间件
// 用法：secured.GET("/path", auth.RequireCapability(auth.SelfBriefingRead), handler)
//
// 与 middleware.RequireRole 的区别：
//   - RequireRole 是"角色权重 ≥ X"的层级判断
//   - RequireCapability 是"该角色（含继承）拥有该能力"的语义判断
//   - 同时高阶角色因继承自动获得低阶能力，无需在每个端点手动列出
func RequireCapability(cap Capability) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUserContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少用户信息",
			})
			return
		}

		// 多角色：任一角色（含继承）命中即放行；单角色用户 Roles 为空时回退 Role
		if !HasAnyRole(user.Roles, cap) && !HasCapability(user.Role, cap) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足，缺少能力：" + string(cap),
			})
			return
		}

		c.Next()
	}
}

// RequireAnyCapability 任一能力命中即放行（OR 语义）
// 适用于"管理员或本人都可访问"等场景
func RequireAnyCapability(caps ...Capability) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUserContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少用户信息",
			})
			return
		}

		for _, cap := range caps {
			// 多角色：任一角色命中该能力即放行
			if HasAnyRole(user.Roles, cap) || HasCapability(user.Role, cap) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "权限不足",
		})
	}
}
