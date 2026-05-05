package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ── RBAC RequireRole 性能基准 ──

func BenchmarkRequireRole_Allowed(b *testing.B) {
	cfg := &config.Config{JWTSecret: "bench-secret-key-32chars", JWTExpireHours: 2}
	user := &model.User{
		ID: 1, Username: "admin", Role: "sys_admin",
		OwnerScope: "school", DisplayName: "管理员",
	}
	token, _ := GenerateToken(cfg, user)

	gin.SetMode(gin.ReleaseMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		// 模拟 JWTAuth 注入用户上下文
		userCtx := &model.UserContext{
			UserID: 1, Username: "admin", Role: "sys_admin",
			OwnerScope: "school", DisplayName: "管理员",
		}
		c.Set(contextKeyUser, userCtx)

		RequireRole("school_admin")(c)
	}
}

func BenchmarkRequireRole_Denied(b *testing.B) {
	cfg := &config.Config{JWTSecret: "bench-secret-key-32chars", JWTExpireHours: 2}
	user := &model.User{
		ID: 1, Username: "student", Role: "student",
		OwnerScope: "school", DisplayName: "学生",
	}
	token, _ := GenerateToken(cfg, user)

	gin.SetMode(gin.ReleaseMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		userCtx := &model.UserContext{
			UserID: 1, Username: "student", Role: "student",
			OwnerScope: "school", DisplayName: "学生",
		}
		c.Set(contextKeyUser, userCtx)

		RequireRole("school_admin")(c)
	}
}

func BenchmarkRequireRoles_Allowed(b *testing.B) {
	cfg := &config.Config{JWTSecret: "bench-secret-key-32chars", JWTExpireHours: 2}
	user := &model.User{
		ID: 1, Username: "counselor", Role: "counselor",
		OwnerScope: "school", DisplayName: "辅导员",
	}
	token, _ := GenerateToken(cfg, user)

	gin.SetMode(gin.ReleaseMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		userCtx := &model.UserContext{
			UserID: 1, Username: "counselor", Role: "counselor",
			OwnerScope: "school", DisplayName: "辅导员",
		}
		c.Set(contextKeyUser, userCtx)

		RequireRoles("counselor", "school_admin", "sys_admin")(c)
	}
}
