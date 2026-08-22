package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// GrantGovernanceRole 提供 platform_operator 项目角色的受控授予/撤销端点。
//
// 背景（audit-report.md M-B1 / C.5 残留）：platform_operator 是平台治理项目角色，
// 既不被普通邀请白名单（projectRoles）授予，历史上也无 in-system 授予路径，只能靠
// 测试 db.Exec 直插 vopc_project_members 自证。这让 R2/R3 治理通道在真实产品中不可达。
//
// 本端点落地最小可用的治理侧 provisioning：
//   - 仅治理系统角色（college_admin/school_admin/sys_admin）可调用，且以数据库 users.role
//     为权威判据（非 JWT 自证），fail-closed：非治理角色一律 403。
//   - 只允许授予/撤销 platform_operator 这一治理角色，其余任何 project_role 一律拒绝
//     （防借本端点写入 owner/co_owner 等提权）。
//   - 授予/撤销均在单事务内写入 vopc_events 审计，失败整体回滚，无伪成功。
//   - 普通 manager/owner 无 vopc.audit / vopc.risk.manage 能力，路由层即被 403 拦截。
//
// 语义：grant 仅当目标用户尚无 active 成员关系时写入 platform_operator（不覆盖既有
// 角色，防把 owner/co_owner 降级为平台治理或把普通成员被动提权）；revoke 仅当当前角色
// 恰为 platform_operator 时移除该成员关系，其余一律 fail-closed 拒绝。
func (h *VOPCHandler) GrantGovernanceRole(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		UserID      int64  `json:"user_id"`
		Action      string `json:"action"`
		ProjectRole string `json:"project_role"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Action = strings.TrimSpace(in.Action)
	in.ProjectRole = strings.TrimSpace(in.ProjectRole)
	if in.Action != "grant" && in.Action != "revoke" {
		c.JSON(422, gin.H{"code": 422, "message": "治理角色动作仅支持 grant 或 revoke"})
		return
	}
	// fail-closed：仅允许 platform_operator 这一治理角色，其余一律拒绝。
	if in.ProjectRole != "platform_operator" {
		c.JSON(422, gin.H{"code": 422, "message": "仅可授予/撤销 platform_operator 治理角色"})
		return
	}
	if in.UserID <= 0 {
		c.JSON(422, gin.H{"code": 422, "message": "目标用户无效"})
		return
	}

	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "治理角色操作失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// 权威判据：调用者必须是治理系统角色（DB users.role，非 JWT 自证）。
	var callerRole string
	if err = tx.QueryRow(`SELECT role FROM users WHERE id=?`, u.UserID).Scan(&callerRole); errors.Is(err, sql.ErrNoRows) {
		c.JSON(403, gin.H{"code": 403, "message": "调用者不存在"})
		return
	} else if err != nil {
		serverError(c, "治理角色操作失败")
		return
	}
	if !platformGovernanceRoles[callerRole] {
		c.JSON(403, gin.H{"code": 403, "message": "仅平台治理系统角色可授予治理角色"})
		return
	}

	// 项目必须存在。
	var owner int64
	if err = tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在"})
		return
	} else if err != nil {
		serverError(c, "治理角色操作失败")
		return
	}

	// 目标用户必须是计算机学院已授权且状态正常的用户。
	var tStatus, tScope, tCollege, tRole string
	if err = tx.QueryRow(`SELECT status,owner_scope,owner_id,role FROM users WHERE id=?`, in.UserID).Scan(&tStatus, &tScope, &tCollege, &tRole); errors.Is(err, sql.ErrNoRows) {
		c.JSON(422, gin.H{"code": 422, "message": "目标用户不存在"})
		return
	} else if err != nil {
		serverError(c, "治理角色操作失败")
		return
	}
	if tStatus != "active" || tRole == "guest" || tScope != "college" || !strings.EqualFold(tCollege, h.collegeID) {
		c.JSON(422, gin.H{"code": 422, "message": "仅可授予计算机学院已授权且状态正常的用户"})
		return
	}

	// 当前成员关系（若存在）。
	var curRole string
	var curStatus string
	err = tx.QueryRow(`SELECT project_role,status FROM vopc_project_members WHERE project_id=? AND user_id=?`, id, in.UserID).Scan(&curRole, &curStatus)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		serverError(c, "治理角色操作失败")
		return
	}

	switch in.Action {
	case "grant":
		if exists && curStatus == "active" {
			c.JSON(409, gin.H{"code": 409, "message": "用户已是项目成员，不可直接授予治理角色"})
			return
		}
		if in.UserID == owner {
			c.JSON(409, gin.H{"code": 409, "message": "项目主理人不可被授予治理角色"})
			return
		}
		if exists {
			// 历史关系为非 active，复位为 platform_operator + active。
			if _, err = tx.Exec(`UPDATE vopc_project_members SET project_role='platform_operator',status='active',created_at=CURRENT_TIMESTAMP WHERE project_id=? AND user_id=?`, id, in.UserID); err != nil {
				serverError(c, "治理角色授予失败")
				return
			}
		} else {
			if _, err = tx.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role,status) VALUES(?,?,'platform_operator','active')`, id, in.UserID); err != nil {
				serverError(c, "治理角色授予失败")
				return
			}
		}
		if writeEvent(tx, id, u.UserID, "governance_role.granted", "", "platform_operator", "授予用户 #"+itoa(in.UserID)+" platform_operator") != nil {
			serverError(c, "治理角色审计写入失败")
			return
		}
	case "revoke":
		if !exists || curStatus != "active" || curRole != "platform_operator" {
			c.JSON(409, gin.H{"code": 409, "message": "该用户当前不是 platform_operator 成员，不可撤销"})
			return
		}
		if in.UserID == owner {
			c.JSON(409, gin.H{"code": 409, "message": "不可撤销项目主理人"})
			return
		}
		if _, err = tx.Exec(`DELETE FROM vopc_project_members WHERE project_id=? AND user_id=? AND project_role='platform_operator'`, id, in.UserID); err != nil {
			serverError(c, "治理角色撤销失败")
			return
		}
		if writeEvent(tx, id, u.UserID, "governance_role.revoked", "platform_operator", "", "撤销用户 #"+itoa(in.UserID)+" platform_operator") != nil {
			serverError(c, "治理角色审计写入失败")
			return
		}
	}

	if tx.Commit() != nil {
		serverError(c, "治理角色操作失败")
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"project_id": id, "user_id": in.UserID, "project_role": "platform_operator", "action": in.Action}})
}
