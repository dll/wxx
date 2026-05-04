package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

func TestRequireRole_StudentAccessStudentResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	// 注入 student 用户上下文
	c.Set(contextKeyUser, &model.UserContext{Role: "student"})

	RequireRole("student")(c)

	if w.Code != http.StatusOK {
		t.Errorf("student 应有权限访问 student 级资源，得到 %d", w.Code)
	}
}

func TestRequireRole_StudentAccessCounselorResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(contextKeyUser, &model.UserContext{Role: "student"})

	RequireRole("counselor")(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("student 无权访问 counselor 级资源，期望 403 得到 %d", w.Code)
	}
}

func TestRequireRole_CounselorAccessStudentResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(contextKeyUser, &model.UserContext{Role: "counselor"})

	RequireRole("student")(c)

	if w.Code != http.StatusOK {
		t.Errorf("counselor 应有权限访问 student 级资源（层级更高），得到 %d", w.Code)
	}
}

func TestRequireRole_SysAdminAccessAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, minRole := range []string{"student", "student_union", "counselor", "college_admin", "school_admin", "sys_admin"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		c.Set(contextKeyUser, &model.UserContext{Role: "sys_admin"})

		RequireRole(minRole)(c)

		if w.Code != http.StatusOK {
			t.Errorf("sys_admin 应能访问 %s 级资源，得到 %d", minRole, w.Code)
		}
	}
}

func TestRequireRole_UnknownRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(contextKeyUser, &model.UserContext{Role: "visitor"})

	RequireRole("student")(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("未知角色应返回 403，得到 %d", w.Code)
	}
}

func TestRequireRole_NoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	// 不注入 user context

	RequireRole("student")(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("缺少用户信息应返回 401，得到 %d", w.Code)
	}
}

func TestRequireRoles_ExactMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(contextKeyUser, &model.UserContext{Role: "counselor"})

	RequireRoles("counselor", "college_admin")(c)

	if w.Code != http.StatusOK {
		t.Errorf("counselor 应在白名单中，得到 %d", w.Code)
	}
}

func TestRequireRoles_NotInWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set(contextKeyUser, &model.UserContext{Role: "student"})

	RequireRoles("counselor", "college_admin")(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("student 不在白名单中，期望 403 得到 %d", w.Code)
	}
}

func TestRequireRoles_NoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	RequireRoles("counselor")(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("缺少用户信息应返回 401，得到 %d", w.Code)
	}
}

func TestRoleHierarchy_Order(t *testing.T) {
	// 验证角色层级严格递增
	roles := []string{"student", "student_union", "counselor", "college_admin", "school_admin", "sys_admin"}
	for i := 0; i < len(roles)-1; i++ {
		if roleHierarchy[roles[i]] >= roleHierarchy[roles[i+1]] {
			t.Errorf("角色层级错误: %s(%d) 应小于 %s(%d)",
				roles[i], roleHierarchy[roles[i]], roles[i+1], roleHierarchy[roles[i+1]])
		}
	}
}

func TestRoleHierarchy_Extensions(t *testing.T) {
	// assistant 在 student_union 和 counselor 之间
	if roleHierarchy["assistant"] <= roleHierarchy["student_union"] {
		t.Error("assistant 层级应高于 student_union")
	}
	if roleHierarchy["assistant"] >= roleHierarchy["counselor"] {
		t.Error("assistant 层级应低于 counselor")
	}
	// teacher 在 counselor 和 college_admin 之间
	if roleHierarchy["teacher"] <= roleHierarchy["counselor"] {
		t.Error("teacher 层级应高于 counselor")
	}
	if roleHierarchy["teacher"] >= roleHierarchy["college_admin"] {
		t.Error("teacher 层级应低于 college_admin")
	}
}
