package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// 风险相关动作与状态枚举。
var approvalDecisions = setOf("approve", "reject")
var freezeActions = setOf("freeze", "unfreeze")

// platformGovernanceRoles 是平台治理系统角色（college_admin/school_admin/sys_admin）。
// 与持有 vopc.risk.manage 能力的角色集合一致（见 auth/capabilities.go），供
// GrantGovernanceRole（治理角色授予）以 users.role 为权威判据使用。
var platformGovernanceRoles = setOf("college_admin", "school_admin", "sys_admin")

// isAuthorizedAdmin 判断用户是否为「授权管理员」，即持有 vopc.risk.manage 能力。
//
// PRD §9.1：vopc.risk.manage（冻结、解冻和处置风险项目）默认主体=授权管理员。
// 能力的权威判据是数据库 users.role（经 auth.HasCapability 沿角色继承链解析），
// 而非 JWT 自证或项目内 platform_operator 成员关系——项目运营者平台角色不自动获得
// 风险治理权，其能力必须来自 vopc.risk.manage 授权（college_admin/school_admin/sys_admin）。
func isAuthorizedAdmin(tx *sql.Tx, userID int64) bool {
	var role string
	if err := tx.QueryRow(`SELECT role FROM users WHERE id=?`, userID).Scan(&role); err != nil {
		return false
	}
	return auth.HasCapability(role, auth.VOPCRiskManage)
}

// isProjectAdvisor 判断用户是否为本项目导师或评审者成员（mentor/reviewer 项目角色）。
//
// PRD §13.1 R2「导师/管理员审核与安全检查」中的「导师」即项目内 mentor/reviewer 成员。
// 该判定与平台系统角色解耦：一个系统角色为 teacher/counselor 的用户，仅当被邀请为本
// 项目 mentor/reviewer 成员时，才可作为本项目 R2 风险的审核人。
func isProjectAdvisor(tx *sql.Tx, projectID, userID int64) bool {
	var role string
	if err := tx.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, projectID, userID).Scan(&role); err != nil {
		return false
	}
	return role == "mentor" || role == "reviewer"
}

// isRiskAdvisorOrAdmin 是 R2/R0/R1 风险审批人的统一口径：项目导师/评审者「或」授权管理员。
// 单人有效审核即通过，不再要求两名不同审批人。
func isRiskAdvisorOrAdmin(tx *sql.Tx, projectID, userID int64) bool {
	return isProjectAdvisor(tx, projectID, userID) || isAuthorizedAdmin(tx, userID)
}

// CreateRisk 在项目内登记一条风险。创建后风险为 open；R2/R3 风险在审批通过/专项审批前
// 不允许推进里程碑（由 milestone gate 校验）。
func (h *VOPCHandler) CreateRisk(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		RiskLevel   string `json:"risk_level"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.RiskLevel = strings.TrimSpace(in.RiskLevel)
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.RiskLevel == "" {
		in.RiskLevel = "R0"
	}
	if !riskLevels[in.RiskLevel] {
		c.JSON(422, gin.H{"code": 422, "message": "风险等级无效"})
		return
	}
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 200 {
		c.JSON(422, gin.H{"code": 422, "message": "风险标题必填且不超过 200 字"})
		return
	}
	if utf8.RuneCountInString(in.Description) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "风险描述超过 4000 字"})
		return
	}
	u := middleware.GetUserContext(c)
	// R3 风险（禁止或专项审批）是重大风险：仅授权管理员可登记，且之后不进入普通审批，
	// 必须按学校制度专项审批后方可解除推进阻断。R0/R1/R2 维持既有的项目管理者创建语义。
	var tx *sql.Tx
	var acquired bool
	if in.RiskLevel == "R3" {
		// R3 登记是平台治理动作，可用项目成员关系之外的授权管理员路径；自行开启事务并
		// 校验项目存在 + 授权管理员，避免 readableProject 的成员关系 404 约束拦下管理员。
		var err error
		tx, err = h.db.Begin()
		if err != nil {
			serverError(c, "风险登记失败")
			return
		}
		defer tx.Rollback()
		var exist int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM vopc_projects WHERE id=?`, id).Scan(&exist); err != nil || exist != 1 {
			c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
			return
		}
		if !isAuthorizedAdmin(tx, u.UserID) {
			c.JSON(403, gin.H{"code": 403, "message": "R3 风险仅授权管理员可登记"})
			return
		}
		acquired = true
	} else {
		tx, _, acquired = h.manageableProject(c, id)
		if !acquired {
			return
		}
		defer tx.Rollback()
	}
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(409, gin.H{"code": 409, "message": msg})
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_risks(project_id,risk_level,title,description,reported_by) VALUES(?,?,?,?,?)`, id, in.RiskLevel, in.Title, in.Description, u.UserID)
	if err != nil {
		serverError(c, "风险登记失败")
		return
	}
	rid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "risk.created", "", "open", "风险 #"+itoa(rid)+"："+in.Title) != nil {
		serverError(c, "风险审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "风险登记失败")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": gin.H{"id": rid, "status": "open"}})
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// ListRisks 只读返回项目风险列表。
func (h *VOPCHandler) ListRisks(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,risk_level,title,description,status,reported_by,created_at,updated_at FROM vopc_risks WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "风险列表读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var rid, by int64
		var level, title, desc, status, created, updated string
		if rows.Scan(&rid, &level, &title, &desc, &status, &by, &created, &updated) != nil {
			serverError(c, "风险列表读取失败")
			return
		}
		items = append(items, gin.H{"id": rid, "risk_level": level, "title": title, "description": desc, "status": status, "reported_by": by, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// ApproveRisk 对 R0/R1/R2 风险做出 approve/reject。
//
// PRD §13.1 校正：R2「导师/管理员审核与安全检查」为「导师或管理员」单人有效审核即可
// 解除推进阻断，不再要求两名不同审批人。审批人须为项目导师（mentor/reviewer 成员）或
// 持有 vopc.risk.manage 能力的授权管理员。
//
// R3「禁止或专项审批」不走本普通审批通道：平台不内置「双人 approve」伪专项机制，一律
// 拒绝并引导走学校制度专项审批（见 CreateSpecialApproval / ListSpecialApprovals）。
func (h *VOPCHandler) ApproveRisk(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	rid, e := parsePositiveID(c.Param("riskId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "风险 ID 无效"})
		return
	}
	var in struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Decision = strings.TrimSpace(in.Decision)
	in.Reason = strings.TrimSpace(in.Reason)
	if !approvalDecisions[in.Decision] {
		c.JSON(422, gin.H{"code": 422, "message": "审批决策仅支持 approve 或 reject"})
		return
	}
	if in.Reason == "" || utf8.RuneCountInString(in.Reason) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "审批理由必填且不超过 4000 字"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "审批失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var riskProject int64
	var riskStatus, riskLevel string
	if err = tx.QueryRow(`SELECT project_id,status,risk_level FROM vopc_risks WHERE id=?`, rid).Scan(&riskProject, &riskStatus, &riskLevel); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "风险不存在或无权操作"})
		return
	} else if err != nil {
		serverError(c, "审批失败")
		return
	}
	if riskProject != id {
		c.JSON(404, gin.H{"code": 404, "message": "风险不存在或无权操作"})
		return
	}
	// R3 不进入普通审批：默认禁止推进，仅按学校制度专项审批（专项审批记录）放行。
	if riskLevel == "R3" {
		c.JSON(409, gin.H{"code": 409, "message": "R3 风险不通过平台普通审批，须按学校制度专项审批"})
		return
	}
	// 审批人须为项目导师/评审者成员「或」授权管理员。
	if !isRiskAdvisorOrAdmin(tx, id, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅项目导师/评审者或授权管理员可审批风险"})
		return
	}
	if riskStatus != "open" {
		c.JSON(409, gin.H{"code": 409, "message": "仅 open 状态风险可审批"})
		return
	}
	// 已由此审批人审批过则拒绝重复。
	var dup int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM vopc_risk_approvals WHERE risk_id=? AND approver_user_id=?`, rid, u.UserID).Scan(&dup); err != nil {
		serverError(c, "审批失败")
		return
	}
	if dup > 0 {
		c.JSON(409, gin.H{"code": 409, "message": "同一审批人不可重复审批"})
		return
	}
	if _, err = tx.Exec(`INSERT INTO vopc_risk_approvals(risk_id,approver_user_id,decision,reason) VALUES(?,?,?,?)`, rid, u.UserID, in.Decision, in.Reason); err != nil {
		serverError(c, "审批记录写入失败")
		return
	}
	// 单人有效审批：approve 即 approved，reject 即 rejected。
	nextStatus := "approved"
	if in.Decision == "reject" {
		nextStatus = "rejected"
	}
	if _, err = tx.Exec(`UPDATE vopc_risks SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='open'`, nextStatus, rid); err != nil {
		serverError(c, "风险状态更新失败")
		return
	}
	if writeEvent(tx, id, u.UserID, "risk."+in.Decision+"d", riskStatus, nextStatus, "风险 #"+itoa(rid)+"："+in.Reason) != nil {
		serverError(c, "审批审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "审批失败")
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": rid, "status": nextStatus, "level": riskLevel}})
}

// CreateSpecialApproval 登记一条 R3 专项审批记录（按学校制度）。
//
// PRD §13.1：R3「禁止或专项审批 → 默认禁止，按学校制度专项审批」。专项审批是学校制度
// 的外部批准行为，平台仅记录其批准结果（approver=审批主体/机构、reason=批准理由、
// ref=学校制度批文或依据），不伪造、不代学校裁决。仅持有 vopc.risk.manage 能力的授权
// 管理员可登记；存在有效专项审批记录后，R3 阻断（里程碑/发布/文件外发）才解除。
func (h *VOPCHandler) CreateSpecialApproval(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		Reason   string `json:"reason"`
		Approver string `json:"approver"`
		Ref      string `json:"ref"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Approver = strings.TrimSpace(in.Approver)
	in.Ref = strings.TrimSpace(in.Ref)
	if in.Reason == "" || utf8.RuneCountInString(in.Reason) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "专项审批理由必填且不超过 4000 字"})
		return
	}
	if in.Approver == "" || utf8.RuneCountInString(in.Approver) > 200 {
		c.JSON(422, gin.H{"code": 422, "message": "审批主体必填且不超过 200 字"})
		return
	}
	if utf8.RuneCountInString(in.Ref) > 500 {
		c.JSON(422, gin.H{"code": 422, "message": "审批依据/批文编号超过 500 字"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "专项审批登记失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if !isAuthorizedAdmin(tx, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅授权管理员可登记专项审批"})
		return
	}
	var owner int64
	if err = tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在"})
		return
	} else if err != nil {
		serverError(c, "专项审批登记失败")
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_risk_special_approvals(project_id,reason,approver,ref,created_by) VALUES(?,?,?,?,?)`, id, in.Reason, in.Approver, in.Ref, u.UserID)
	if err != nil {
		serverError(c, "专项审批登记失败")
		return
	}
	sid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "risk.special_approval", "", "approved", "专项审批 #"+itoa(sid)+"："+in.Approver) != nil {
		serverError(c, "专项审批审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "专项审批登记失败")
		return
	}
	committed = true
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": gin.H{"id": sid}})
}

// ListSpecialApprovals 只读返回项目专项审批记录。
func (h *VOPCHandler) ListSpecialApprovals(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,reason,approver,ref,created_by,created_at FROM vopc_risk_special_approvals WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "专项审批列表读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var sid, createdBy int64
		var reason, approver, ref, createdAt string
		if rows.Scan(&sid, &reason, &approver, &ref, &createdBy, &createdAt) != nil {
			serverError(c, "专项审批列表读取失败")
			return
		}
		items = append(items, gin.H{"id": sid, "reason": reason, "approver": approver, "ref": ref, "created_by": createdBy, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// FreezeProject 授权管理员冻结或解冻项目。冻结后项目进入 risk_frozen，所有写操作被
// blockedStatuses 拦截；解冻恢复。必须填写理由并审计。
//
// PRD §9.1 / §13.2：冻结/解冻权归 vopc.risk.manage（授权管理员），不依赖项目内
// platform_operator 成员关系。
func (h *VOPCHandler) FreezeProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Action = strings.TrimSpace(in.Action)
	in.Reason = strings.TrimSpace(in.Reason)
	if !freezeActions[in.Action] {
		c.JSON(422, gin.H{"code": 422, "message": "冻结动作仅支持 freeze 或 unfreeze"})
		return
	}
	if in.Reason == "" || utf8.RuneCountInString(in.Reason) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "冻结理由必填且不超过 4000 字"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "冻结操作失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var status string
	if err = tx.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, id).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在"})
		return
	} else if err != nil {
		serverError(c, "冻结操作失败")
		return
	}
	if !isAuthorizedAdmin(tx, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅授权管理员可冻结/解冻项目"})
		return
	}
	if in.Action == "freeze" {
		if status == "risk_frozen" {
			c.JSON(409, gin.H{"code": 409, "message": "项目已冻结"})
			return
		}
		if status == "completed" || status == "terminated" || status == "archived" {
			c.JSON(409, gin.H{"code": 409, "message": "当前状态不可冻结"})
			return
		}
	} else {
		if status != "risk_frozen" {
			c.JSON(409, gin.H{"code": 409, "message": "项目未处于冻结状态"})
			return
		}
	}
	var next string
	if in.Action == "freeze" {
		next = "risk_frozen"
	} else {
		next = "pending_review"
	}
	res, err := tx.Exec(`UPDATE vopc_projects SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status=?`, next, id, status)
	if err != nil {
		serverError(c, "冻结操作失败")
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "项目状态已变化，请刷新后重试"})
		return
	}
	if _, err = tx.Exec(`INSERT INTO vopc_freeze_records(project_id,action,reason,acted_by) VALUES(?,?,?,?)`, id, in.Action, in.Reason, u.UserID); err != nil {
		serverError(c, "冻结记录写入失败")
		return
	}
	if writeEvent(tx, id, u.UserID, "project."+in.Action, status, next, in.Reason) != nil {
		serverError(c, "冻结审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "冻结操作失败")
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": id, "status": next}})
}

// CreateRiskAppeal 项目主理人针对风险冻结/处置发起申诉。
func (h *VOPCHandler) CreateRiskAppeal(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Reason == "" || utf8.RuneCountInString(in.Reason) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "申诉理由必填且不超过 4000 字"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, owner, ok := h.manageableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	_ = owner
	res, err := tx.Exec(`INSERT INTO vopc_risk_appeals(project_id,reason,submitted_by) VALUES(?,?,?)`, id, in.Reason, u.UserID)
	if err != nil {
		serverError(c, "申诉提交失败")
		return
	}
	aid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "risk.appeal_submitted", "", "pending", "申诉 #"+itoa(aid)) != nil {
		serverError(c, "申诉审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "申诉提交失败")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": gin.H{"id": aid, "status": "pending"}})
}

// ResolveRiskAppeal 授权管理员裁定申诉（upheld 维持原处置 / dismissed 驳回）。
//
// PRD §9.1 / §13.2：申诉裁定权归 vopc.risk.manage（授权管理员）。
func (h *VOPCHandler) ResolveRiskAppeal(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	aid, e := parsePositiveID(c.Param("appealId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "申诉 ID 无效"})
		return
	}
	var in struct {
		Decision   string `json:"decision"`
		Resolution string `json:"resolution"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Decision = strings.TrimSpace(in.Decision)
	in.Resolution = strings.TrimSpace(in.Resolution)
	if in.Decision != "upheld" && in.Decision != "dismissed" {
		c.JSON(422, gin.H{"code": 422, "message": "申诉裁定仅支持 upheld 或 dismissed"})
		return
	}
	if in.Resolution == "" || utf8.RuneCountInString(in.Resolution) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "裁定说明必填且不超过 4000 字"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "申诉裁定失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var appealProject int64
	var status string
	if err = tx.QueryRow(`SELECT project_id,status FROM vopc_risk_appeals WHERE id=?`, aid).Scan(&appealProject, &status); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "申诉不存在"})
		return
	} else if err != nil {
		serverError(c, "申诉裁定失败")
		return
	}
	if appealProject != id {
		c.JSON(404, gin.H{"code": 404, "message": "申诉不存在"})
		return
	}
	if !isAuthorizedAdmin(tx, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅授权管理员可裁定申诉"})
		return
	}
	if status != "pending" {
		c.JSON(409, gin.H{"code": 409, "message": "申诉已裁定"})
		return
	}
	res, err := tx.Exec(`UPDATE vopc_risk_appeals SET status=?,resolved_by=?,resolution=?,resolved_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'`, in.Decision, u.UserID, in.Resolution, aid)
	if err != nil {
		serverError(c, "申诉裁定失败")
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "申诉状态已变化"})
		return
	}
	// upheld 意味着维持原处置（通常为解除冻结之外的处置），此处仅记录裁定；
	// 若为 dismissed 且项目曾被冻结，不在此最小闭环内自动解冻。
	if writeEvent(tx, id, u.UserID, "risk.appeal_resolved", "pending", in.Decision, "申诉 #"+itoa(aid)+"："+in.Resolution) != nil {
		serverError(c, "申诉审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "申诉裁定失败")
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": aid, "status": in.Decision}})
}

// milestoneAdvanceAllowed 是里程碑推进门禁。
//
// 按 PRD §13.1 分级口径：
//   - R2（较高风险）：需存在至少一条 status=approved 且 risk_level=R2 的风险，即已由
//     「导师或管理员」单人有效审核通过，方可推进里程碑。
//   - R3（禁止或专项审批）：一旦项目为 R3，或项目上登记了 R3 风险，即进入 R3-tier——
//     默认禁止推进，仅当存在至少一条学校制度专项审批记录
//     （vopc_risk_special_approvals）时才放行。R3 与 R2 不可混同：普通 approve 不计入
//     R3 的放行条件。
//   - R0/R1：不设风险门禁（R0 自动进入孵化，R1 用户告知与基础审核）。
//
// 返回 (是否允许, 错误文案, 状态码)。
func milestoneAdvanceAllowed(tx *sql.Tx, projectID int64) (bool, string, int) {
	var riskLevel string
	if err := tx.QueryRow(`SELECT risk_level FROM vopc_projects WHERE id=?`, projectID).Scan(&riskLevel); err != nil {
		return false, "项目风险门禁读取失败", 500
	}
	// 项目上是否存在 R3 风险项（无论该项的普通状态如何，R3 默认禁止）。
	var r3RiskCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND risk_level='R3'`, projectID).Scan(&r3RiskCount); err != nil {
		return false, "项目风险门禁读取失败", 500
	}
	isR3 := riskLevel == "R3" || r3RiskCount > 0
	if isR3 {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risk_special_approvals WHERE project_id=?`, projectID).Scan(&n); err != nil {
			return false, "项目风险门禁读取失败", 500
		}
		if n == 0 {
			return false, "R3 项目须按学校制度专项审批后方可推进里程碑", 409
		}
		return true, "", 0
	}
	if riskLevel != "R2" {
		return true, "", 0
	}
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND risk_level='R2' AND status='approved'`, projectID).Scan(&n); err != nil {
		return false, "项目风险门禁读取失败", 500
	}
	if n == 0 {
		return false, "R2 项目须经导师或管理员审核通过后方可推进里程碑", 409
	}
	return true, "", 0
}
