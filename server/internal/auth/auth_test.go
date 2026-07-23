package auth

import (
	"testing"
)

// TestHasCapability_DirectMatch 直接拥有的 capability
func TestHasCapability_DirectMatch(t *testing.T) {
	if !HasCapability("student", SelfBriefingRead) {
		t.Error("学生应拥有 self.briefing.read")
	}
	if !HasCapability("counselor", CounselorAlertRead) {
		t.Error("辅导员应拥有 counselor.alert.read")
	}
}

// TestHasCapability_Inherited 继承的 capability
func TestHasCapability_Inherited(t *testing.T) {
	// 高阶角色继承低阶 self.* 能力
	if !HasCapability("counselor", SelfBriefingRead) {
		t.Error("辅导员应继承 self.briefing.read")
	}
	if !HasCapability("sys_admin", SelfBriefingRead) {
		t.Error("系统管理员应继承 self.briefing.read")
	}
	// college_admin 同时继承三条线
	if !HasCapability("college_admin", CounselorAlertRead) {
		t.Error("学院管理员应继承 counselor.alert.read")
	}
	if !HasCapability("college_admin", TeacherLessonPrep) {
		t.Error("学院管理员应继承 teacher.lesson.prep")
	}
	if !HasCapability("college_admin", AssistantScheduleCheck) {
		t.Error("学院管理员应继承 assistant.schedule.check")
	}
}

// TestHasCapability_Denied 无权限场景
func TestHasCapability_Denied(t *testing.T) {
	if HasCapability("student", CounselorAlertRead) {
		t.Error("学生不应拥有 counselor.alert.read")
	}
	// teacher 与 counselor 平级，互不继承
	if HasCapability("teacher", CounselorAlertRead) {
		t.Error("教师不应继承 counselor.alert.read（平级关系）")
	}
	if HasCapability("counselor", TeacherLessonPrep) {
		t.Error("辅导员不应继承 teacher.lesson.prep（平级关系）")
	}
}

func TestImportStudentCapability_AllNonStudentRoles(t *testing.T) {
	for _, role := range []string{
		"student_union", "counselor", "teacher", "assistant",
		"college_admin", "school_admin", "sys_admin",
	} {
		if !HasCapability(role, CounselorImportStudent) {
			t.Errorf("%s 应拥有导入学生能力", role)
		}
	}
	for _, role := range []string{"student", "guest"} {
		if HasCapability(role, CounselorImportStudent) {
			t.Errorf("%s 不应拥有导入学生能力", role)
		}
	}
}

// TestHasCapability_UnknownRole 未知角色
func TestHasCapability_UnknownRole(t *testing.T) {
	if HasCapability("hacker", SelfBriefingRead) {
		t.Error("未知角色不应拥有任何 capability")
	}
	if HasCapability("", SelfBriefingRead) {
		t.Error("空角色不应拥有任何 capability")
	}
}

// TestCapabilitiesOf 列出所有能力
func TestCapabilitiesOf(t *testing.T) {
	studentCaps := CapabilitiesOf("student")
	if len(studentCaps) == 0 {
		t.Error("学生应至少有一组 self.* 能力")
	}

	sysAdminCaps := CapabilitiesOf("sys_admin")
	if len(sysAdminCaps) <= len(studentCaps) {
		t.Errorf("系统管理员能力数应多于学生（继承），实际 %d vs %d",
			len(sysAdminCaps), len(studentCaps))
	}
}

// TestCapabilitiesOf_NoDuplicates 去重
func TestCapabilitiesOf_NoDuplicates(t *testing.T) {
	caps := CapabilitiesOf("college_admin")
	seen := make(map[Capability]bool)
	for _, c := range caps {
		if seen[c] {
			t.Errorf("发现重复 capability：%s", c)
		}
		seen[c] = true
	}
}

// TestIsKnownRole 角色注册检查
func TestIsKnownRole(t *testing.T) {
	for _, r := range []string{
		"student", "student_union", "counselor", "teacher",
		"assistant", "college_admin", "school_admin", "sys_admin",
	} {
		if !IsKnownRole(r) {
			t.Errorf("%s 应为已知角色", r)
		}
	}
	if IsKnownRole("hacker") {
		t.Error("hacker 不应为已知角色")
	}
}
