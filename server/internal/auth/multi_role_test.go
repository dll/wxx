package auth

import "testing"

// 多角色能力并集判定（2026-09-01，user_roles + 权限并集）

func TestHasAnyRole(t *testing.T) {
	// 教师+学院管理员：应同时拥有教师与学生能力，以及管理员能力
	roles := []string{"college_admin", "teacher"}

	if !HasAnyRole(roles, CollegeMetricsRead) {
		t.Error("college_admin 角色应拥有 CollegeMetricsRead")
	}
	if !HasAnyRole(roles, TeacherLessonPrep) {
		t.Error("teacher 角色应拥有 TeacherLessonPrep")
	}
	if !HasAnyRole(roles, SelfChat) {
		t.Error("继承链应拥有 SelfChat（student 基线）")
	}
	if HasAnyRole(roles, SystemAuditAll) {
		t.Error("college_admin 不应拥有 SystemAuditAll（sys_admin 专属）")
	}
	if HasAnyRole(nil, SelfChat) {
		t.Error("空角色列表应返回 false")
	}
}

func TestCapabilitiesOfAny(t *testing.T) {
	caps := CapabilitiesOfAny([]string{"teacher", "counselor"})
	set := make(map[Capability]bool)
	for _, c := range caps {
		set[c] = true
	}
	if !set[TeacherLessonPrep] {
		t.Error("并集应包含 teacher 的 TeacherLessonPrep")
	}
	if !set[CounselorClassRead] {
		t.Error("并集应包含 counselor 的 CounselorClassRead")
	}
	if len(CapabilitiesOfAny(nil)) != 0 {
		t.Error("空角色列表应返回空能力集")
	}
}

func TestPrimaryRole(t *testing.T) {
	cases := []struct {
		roles []string
		want  string
	}{
		{[]string{"teacher", "college_admin"}, "college_admin"},
		{[]string{"college_admin", "teacher"}, "college_admin"},
		{[]string{"student", "teacher"}, "teacher"},
		{[]string{"student_union"}, "student_union"},
		{nil, ""},
	}
	for _, c := range cases {
		if got := PrimaryRole(c.roles); got != c.want {
			t.Errorf("PrimaryRole(%v) = %q, want %q", c.roles, got, c.want)
		}
	}
}

func TestRoleMatchesAny(t *testing.T) {
	roles := []string{"teacher", "assistant"}
	if !RoleMatchesAny(roles, "teacher") {
		t.Error("应命中 teacher")
	}
	if !RoleMatchesAny(roles, "student_union") {
		t.Error("应命中继承链上的 student_union")
	}
	if RoleMatchesAny(roles, "counselor") {
		t.Error("不应命中平级角色 counselor")
	}
}
