package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// 自定义 claims（JWT 载荷）
type Claims struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	OwnerScope  string `json:"owner_scope"`
	OwnerID     string `json:"owner_id"`
	DisplayName string `json:"display_name"`
	jwt.RegisteredClaims
}

// contextKey 用于 gin.Context 存取用户信息的键名
const contextKeyUser = "user_ctx"

// GenerateToken 签发 JWT token
func GenerateToken(cfg *config.Config, user *model.User) (string, error) {
	if cfg.JWTSecret == "" {
		return "", errors.New("JWT_SECRET 未配置")
	}

	now := time.Now()
	claims := Claims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		OwnerScope:  user.OwnerScope,
		OwnerID:     user.OwnerID,
		DisplayName: user.DisplayName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWTExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "wxx",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// JWTAuth JWT 认证中间件
// 从 Authorization 头解析 Bearer token，验证后将用户上下文注入 gin.Context
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization 头提取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少认证信息",
			})
			return
		}

		// 格式：Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证格式错误，需要 Bearer token",
			})
			return
		}

		// 解析并验证 token
		tokenStr := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			// 确保签名算法是 HMAC
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("不支持的签名算法")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "token 无效或已过期",
			})
			return
		}

		// 注入用户上下文
		userCtx := &model.UserContext{
			UserID:      claims.UserID,
			Username:    claims.Username,
			Role:        claims.Role,
			OwnerScope:  claims.OwnerScope,
			OwnerID:     claims.OwnerID,
			DisplayName: claims.DisplayName,
		}
		c.Set(contextKeyUser, userCtx)

		c.Next()
	}
}

// GetUserContext 从 gin.Context 提取用户上下文（中间件注入）
func GetUserContext(c *gin.Context) *model.UserContext {
	if v, exists := c.Get(contextKeyUser); exists {
		if ctx, ok := v.(*model.UserContext); ok {
			return ctx
		}
	}
	return nil
}
