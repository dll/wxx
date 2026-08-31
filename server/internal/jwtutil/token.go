package jwtutil

import (
	"errors"
	"strconv"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims JWT 载荷
type CustomClaims struct {
	UserID       int64    `json:"user_id"`
	Username     string   `json:"username"`
	Role         string   `json:"role"`
	Roles        []string `json:"roles,omitempty"` // 全部角色（多角色用户非空；旧 token 无此字段）
	OwnerScope   string   `json:"owner_scope"`
	OwnerID      string   `json:"owner_id"`
	DisplayName  string   `json:"display_name"`
	Consented    bool     `json:"consented"`
	TokenVersion int      `json:"tv"`
	Status       string   `json:"status"` // 账号状态：active/pending/rejected/disabled，用于中间件即时校验
	Grade        int      `json:"grade"`  // 学生年级（1~4，按入学年份推导；非学生/未知为 0）
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
		Roles:        user.Roles, // 多角色：全量写入 JWT；单角色用户为 nil（JSON omitempty）
		OwnerScope:   user.OwnerScope,
		OwnerID:      user.OwnerID,
		DisplayName:  user.DisplayName,
		Consented:    user.Consented == 1,
		TokenVersion: user.TokenVersion,
		Status:       user.Status, // 写入状态，中间件据此拦截 pending/rejected
		Grade:        ResolveGrade(user.EnrollmentYear),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWTExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "wxx",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ResolveGrade 由入学年份推导当前年级（1~4，越界按 4 处理；解析失败/未提供返回 0）。
// 例：2026 入学，2026 学年为大一(1)，2029 为大四(4)。
func ResolveGrade(enrollmentYear string) int {
	y, err := strconv.Atoi(enrollmentYear)
	if err != nil || y <= 0 {
		return 0
	}
	g := time.Now().Year() - y + 1
	if g < 1 {
		return 1 // 未来入学年份保守视为大一
	}
	if g > 4 {
		return 4
	}
	return g
}
