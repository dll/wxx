package middleware

import (
	"testing"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
)

// ── JWT GenerateToken 性能基准 ──

func BenchmarkGenerateToken(b *testing.B) {
	cfg := &config.Config{
		JWTSecret:      "bench-secret-key-32chars-minimum",
		JWTExpireHours: 2,
	}
	user := &model.User{
		ID:          1,
		Username:    "benchuser",
		Role:        "student",
		OwnerScope:  "school",
		OwnerID:     "school-1",
		DisplayName: "基准测试用户",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateToken(cfg, user)
		if err != nil {
			b.Fatalf("GenerateToken 失败: %v", err)
		}
	}
}

// ── JWT ParseWithClaims 性能基准 ──

func BenchmarkParseToken(b *testing.B) {
	cfg := &config.Config{
		JWTSecret:      "bench-secret-key-32chars-minimum",
		JWTExpireHours: 2,
	}
	user := &model.User{
		ID: 1, Username: "benchuser", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "用户",
	}
	tokenStr, err := GenerateToken(cfg, user)
	if err != nil {
		b.Fatalf("预处理 token 失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		claims := &CustomClaims{}
		_, err := jwtlib.ParseWithClaims(tokenStr, claims, func(t *jwtlib.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil {
			b.Fatalf("解析 token 失败: %v", err)
		}
	}
}
