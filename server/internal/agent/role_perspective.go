package agent

import (
	"github.com/dll/wxx/server/internal/model"
)

// rolePerspective 根据用户角色返回针对该角色的回答视角约束
// 让同一个问题被不同角色提问时，返回视角不同的内容
func rolePerspective(userCtx *model.UserContext) string {
	if userCtx == nil {
		return ""
	}
	switch userCtx.Role {
	case "student":
		return "用户身份：学生。回答需站在学生视角：\n- 重点说明「我作为学生该怎么做、找谁、何时办」\n- 用第二人称称呼（你/您）\n- 强调对学生的影响而非对管理流程的描述"
	case "counselor":
		return "用户身份：辅导员。回答需站在辅导员视角：\n- 突出「如何指导学生、风险点、谈话切入」\n- 提供可操作的工作建议（关注名单、谈心要点、转介路径）\n- 适当引用上级文件和管理依据"
	case "teacher":
		return "用户身份：教师。回答需站在教学视角：\n- 关注课堂教学、备课、考试、学情如何与该问题关联\n- 提供可融入教学的角度"
	case "assistant":
		return "用户身份：教辅。回答需站在事务办理视角：\n- 突出排课、考试、毕业等事务的具体规则与材料"
	case "student_union":
		return "用户身份：学生会成员。回答需结合活动组织视角：\n- 关注活动审批、宣传、参与人群"
	case "college_admin", "school_admin", "sys_admin":
		return "用户身份：管理员。回答需站在管理者视角：\n- 关注政策依据、统计口径、决策风险\n- 适当提供数据视角和跨学院/跨年级的对比建议"
	default:
		return ""
	}
}
