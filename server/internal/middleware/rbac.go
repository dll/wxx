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

// RequireRole RBAC 角色权限中间件（按角色权重比较）
// minRole 指定允许访问的最低角色级别
//
// Deprecated: 新代码建议使用 auth.RequireCapability 进行能力级授权。
// RequireRole 仅保留以兼容存量端点；新增端点请基于 capability 设计。
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

		userLevel := 0
		// 多角色：取所有角色中的最高层级；单角色用户 Roles 为空时回退 Role
		for _, r := range append([]string{user.Role}, user.Roles...) {
			if lv, ok := roleHierarchy[r]; ok && lv > userLevel {
				userLevel = lv
			}
		}
		if userLevel == 0 {
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

		// 多角色：任一角色命中即放行；单角色用户 Roles 为空时回退 Role
		ok := allowed[user.Role]
		if !ok {
			for _, r := range user.Roles {
				if allowed[r] {
					ok = true
					break
				}
			}
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "该操作不允许当前角色访问",
			})
			return
		}

		c.Next()
	}
}
