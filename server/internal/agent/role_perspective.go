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
	base := roleBasePerspective(userCtx.Role)
	// 学生按年级追加针对性关注点，让回答更贴合其当前阶段需求
	if gradeHint := gradePerspective(userCtx); gradeHint != "" {
		if base != "" {
			base += "\n"
		}
		base += gradeHint
	}
	return base
}

// roleBasePerspective 各角色的基础视角约束
func roleBasePerspective(role string) string {
	switch role {
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

// gradePerspective 学生按年级（1~4）返回针对性关注点提示。
// 让同一学生问题因所处年级不同，回答侧重点有所差异。
func gradePerspective(userCtx *model.UserContext) string {
	if userCtx == nil {
		return ""
	}
	role := userCtx.Role
	if role != "student" {
		return "" // 仅学生按年级细化
	}
	switch userCtx.Grade {
	case 1:
		return "该学生为大一新生：回答宜侧重报到入学、军训、选课、适应校园、入团入党启蒙、专业认知方向。"
	case 2:
		return "该学生为大二：回答宜侧重学业基础巩固、学科竞赛/大创、入党积极分子培养、社团与能力发展方向。"
	case 3:
		return "该学生为大三：回答宜侧重专业深化、考研/考公/就业方向选择、科研项目与实习实践、发展预备党员方向。"
	case 4:
		return "该学生为大四：回答宜侧重毕业资格、毕业论文/设计、就业求职/升学、离校手续、档案去向方向。"
	default:
		return ""
	}
}
