package jwtutil

import (
	"errors"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims JWT 载荷
type CustomClaims struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	OwnerScope   string `json:"owner_scope"`
	OwnerID      string `json:"owner_id"`
	DisplayName  string `json:"display_name"`
	Consented    bool   `json:"consented"`
	TokenVersion int    `json:"tv"`
	Status       string `json:"status"` // 账号状态：active/pending/rejected/disabled，用于中间件即时校验
	jwt.RegisteredClaims
}

// GenerateToken 签发 JWT token
func GenerateToken(cfg *config.Config, user *model.User) (string, error) {
	if cfg.JWTSecret == "" {
		return "", errors.New("JWT_SECRET 未配置")
	}

	now := time.Now()
	claims := CustomClaims{
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
		OwnerScope:   user.OwnerScope,
		OwnerID:      user.OwnerID,
		DisplayName:  user.DisplayName,
		Consented:    user.Consented == 1,
		TokenVersion: user.TokenVersion,
		Status:       user.Status, // 写入状态，中间件据此拦截 pending/rejected
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWTExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "wxx",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}
