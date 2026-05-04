package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// roleHierarchy 角色层级权重，数值越大权限越高
var roleHierarchy = map[string]int{
	"student":       10,
	"student_union": 20,
	"counselor":     30,
	"college_admin": 40,
	"school_admin":  50,
	"sys_admin":     60,
	// 扩展角色
	"assistant": 25,
	"teacher":   35,
}

// RequireRole RBAC 角色权限中间件
// minRole 指定允许访问的最低角色级别
func RequireRole(minRole string) gin.HandlerFunc {
	minLevel, ok := roleHierarchy[minRole]
	if !ok {
		// 配置错误：未知角色，拒绝所有请求
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "系统配置错误：未知角色要求",
			})
		}
	}

	return func(c *gin.Context) {
		user := GetUserContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少用户信息",
			})
			return
		}

		userLevel, exists := roleHierarchy[user.Role]
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "未知用户角色",
			})
			return
		}

		if userLevel < minLevel {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足，需要 " + minRole + " 及以上角色",
			})
			return
		}

		c.Next()
	}
}

// RequireRoles 精确角色列表匹配中间件
// 仅允许指定角色列表中的用户访问
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		user := GetUserContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少用户信息",
			})
			return
		}

		if !allowed[user.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "该操作不允许当前角色访问",
			})
			return
		}

		c.Next()
	}
}
