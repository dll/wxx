package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// 自定义 claims（JWT 载荷）
type CustomClaims struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	OwnerScope  string `json:"owner_scope"`
	OwnerID     string `json:"owner_id"`
	DisplayName string `json:"display_name"`
	Consented   bool   `json:"consented"`
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
	claims := CustomClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		OwnerScope:  user.OwnerScope,
		OwnerID:     user.OwnerID,
		DisplayName: user.DisplayName,
		Consented:   true, // 登录即视为已同意（首次使用由前端弹窗处理）
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
			log.Printf("[JWTAuth] 缺少认证信息 path=%s method=%s ip=%s", c.Request.URL.Path, c.Request.Method, c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少认证信息，请重新登录",
			})
			return
		}

		// 格式：Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("[JWTAuth] 认证格式错误 path=%s method=%s", c.Request.URL.Path, c.Request.Method)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "认证格式错误，需要 Bearer token",
			})
			return
		}

		// 解析并验证 token
		tokenStr := parts[1]
		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			// 确保签名算法是 HMAC
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("不支持的签名算法")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			log.Printf("[JWTAuth] token 验证失败 path=%s method=%s err=%v", c.Request.URL.Path, c.Request.Method, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "token 无效或已过期，请重新登录",
			})
			return
		}

		// nbf (Not Before) 验证
		if claims.NotBefore != nil && time.Now().Unix() < claims.NotBefore.Unix() {
			log.Printf("[JWTAuth] token 尚未生效 path=%s method=%s", c.Request.URL.Path, c.Request.Method)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "token 尚未生效",
			})
			return
		}

		// iss (Issuer) 验证
		if claims.Issuer != "" && claims.Issuer != "wxx" {
			log.Printf("[JWTAuth] 非预期的 token 签发者 path=%s method=%s iss=%s", c.Request.URL.Path, c.Request.Method, claims.Issuer)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "token 签发者无效",
			})
			return
		}

		// 注入用户上下文
		userCtx := &model.UserContext{
			Consented:   claims.Consented,
			UserID:      claims.UserID,
			Username:    claims.Username,
			Role:        claims.Role,
			OwnerScope:  claims.OwnerScope,
			OwnerID:     claims.OwnerID,
			DisplayName: claims.DisplayName,
		}
		c.Set(contextKeyUser, userCtx)

		log.Printf("[JWTAuth] 认证成功 user=%s role=%s path=%s", maskName(claims.Username), claims.Role, c.Request.URL.Path)

		c.Next()
	}
}

// maskName 对用户姓名脱敏：张* 或 张**
func maskName(name string) string {
	if len(name) == 0 {
		return name
	}
	r := []rune(name)
	if len(r) == 1 {
		return string(r[0]) + "*"
	}
	return string(r[0]) + "**"
}

// maskStudentID 对学号脱敏：12****34
func maskStudentID(id string) string {
	if len(id) <= 4 {
		return id[:1] + "***"
	}
	return id[:2] + "****" + id[len(id)-2:]
}

// maskPhone 对手机号脱敏：138****1234
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
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
