package model

import "testing"

func TestSanitizePrivate(t *testing.T) {
	u := &User{
		Username:        "20260001",
		DisplayName:     "测试",
		Phone:           "13800000000",
		Wechat:          "wx_test",
		Email:           "t@t.com",
		BirthDate:       "2008-01-01",
		College:         "计算机学院",
		Campus:          "会峰校区",
		PoliticalStatus: "共青团员",
	}

	// 非授权角色（student/student_union/guest）→ 脱敏
	for _, role := range []string{"student", "student_union", "guest"} {
		cp := *u
		if !cp.SanitizePrivate(role) {
			t.Fatalf("角色 %s 应触发脱敏", role)
		}
		if cp.Phone != "" || cp.Wechat != "" || cp.Email != "" || cp.BirthDate != "" {
			t.Fatalf("角色 %s 脱敏后仍泄露私密字段: %+v", role, cp)
		}
		// 公共字段应保留
		if cp.Campus != "会峰校区" || cp.PoliticalStatus != "共青团员" || cp.College != "计算机学院" {
			t.Fatalf("角色 %s 脱敏误伤公共字段: %+v", role, cp)
		}
	}

	// 授权角色（counselor/admin/teacher）→ 保留
	for _, role := range []string{"counselor", "teacher", "college_admin", "school_admin", "sys_admin"} {
		cp := *u
		if cp.SanitizePrivate(role) {
			t.Fatalf("角色 %s 不应触发脱敏", role)
		}
		if cp.Phone != "13800000000" || cp.BirthDate != "2008-01-01" {
			t.Fatalf("角色 %s 私密字段被误删: %+v", role, cp)
		}
	}
}
