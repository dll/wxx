package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// RepairAgentTokenKey 本机修复执行端 token 的环境变量名。
// 仅当该环境变量被显式设定为非空值时，内部修复任务端点才可用；
// 未设定时，内部端点一律返回 404（不暴露端点存在性）。
const RepairAgentTokenKey = "WXX_REPAIR_AGENT_TOKEN"

// repairAgentToken 从环境变量缓存读取执行端 token（进程启动后不变），
// 未配置时返回空串。空串代表"内部端点不可用"。
func repairAgentToken() string {
	return os.Getenv(RepairAgentTokenKey)
}

// RepairAgentTokenAuth 本机修复执行端专用鉴权中间件。
//
// 安全约束：
//  1. Token 来自环境变量 WXX_REPAIR_AGENT_TOKEN，绝不硬编码进源码或日志；
//  2. 使用 crypto/subtle.ConstantTimeCompare 常量时间比较，防时序侧信道；
//  3. Token 未配置（空）时，端点视为"不可用"，直接 404，不暴露路由存在性；
//  4. 仅作用于 /api/v1/internal/repair-tasks 内部路由，与交互式前台用户 JWT 完全隔离，
//     不授予任何业务角色（包括 sys_admin）执行能力。
//
// 调用方（app 装配层）应在注册内部路由前判断 token 是否已配置；中间件自身
// 也做二次兜底，确保即使误注册也会被禁用。
func RepairAgentTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := repairAgentToken()
		if expected == "" {
			// token 未配置：端点不可用，按 404 处理，避免泄露路由结构
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}

		provided := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if len(provided) <= len(prefix) || provided[:len(prefix)] != prefix {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		provided = provided[len(prefix):]

		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			return
		}

		c.Next()
	}
}
