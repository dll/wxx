package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// 风险相关动作与状态枚举。
var approvalDecisions = setOf("approve", "reject")
var freezeActions = setOf("freeze", "unfreeze")

// isRiskManager 判断当前用户是否具备平台风险治理权。风险冻结/解冻/审批是
// 平台治理动作，只允许项目内 platform_operator 角色执行；双重审批在本最小闭环
// 中落地为“两名不同审批人对同一条风险均 approve”。
func isRiskManager(tx *sql.Tx, projectID, userID int64) bool {
	var role string
	if err := tx.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, projectID, userID).Scan(&role); err != nil {
		return false
	}
	return role == "platform_operator"
}

// isSpecialRiskGovernance 判断用户是否持有 R3 专项审批角色。
//
// R3 与 R2 治理口径分离：R2 走一般平台治理角色（platform_operator）的双人审批；
// R3 按 PRD 13.1「禁止或专项审批 → 默认禁止，按学校制度专项审批」落地为独立专项
// 通道——R3 风险只能由项目内 risk_governance 专项角色创建与审批，普通 manager、
// 甚至 platform_operator 都不得越权。risk_governance 由平台在治理侧授予，项目侧
// 不开放自助授予（与 platform_operator 一致）。
func isSpecialRiskGovernance(tx *sql.Tx, projectID, userID int64) bool {
	var role string
	if err := tx.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, projectID, userID).Scan(&role); err != nil {
		return false
	}
	return role == "risk_governance"
}

// CreateRisk 在项目内登记一条风险。创建后风险为 open；R2/R3 风险在审批通过前
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
	// R3 专项口径先于普通 manage 边界判断：专项治理角色（risk_governance）作为项目成员
	// 经 readableProject 通过，再以 isSpecialRiskGovernance 二次校验；普通 manager 仍走
	// manageableProject。避免把 risk_governance 直接放大为完整项目管理权。
	var tx *sql.Tx
	var owner int64
	var acquired bool
	if in.RiskLevel == "R3" {
		tx, owner, acquired = h.readableProject(c, id)
	} else {
		tx, owner, acquired = h.manageableProject(c, id)
	}
	if !acquired {
		return
	}
	defer tx.Rollback()
	_ = owner
	// R3 专项口径：仅平台专项治理角色可登记 R3 风险，普通 manager（owner/co_owner）
	// 不得替项目擅自创建“禁止或专项审批”级别风险。R0/R1/R2 维持既有创建语义。
	if in.RiskLevel == "R3" && !isSpecialRiskGovernance(tx, id, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "R3 风险仅平台专项治理角色可创建"})
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

// ApproveRisk 平台治理角色对风险做出 approve/reject。R2 解除推进阻断需“双人审批”：
// 同一条风险未被两个不同审批人都 approve 前，R2/R3 项目里程碑仍被拦截。
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
	// 审批权限按风险等级分流：R3 走独立专项通道（risk_governance 专项角色），
	// R2 及以下走一般平台治理（platform_operator）。两者不可互相越权。
	if riskLevel == "R3" {
		if !isSpecialRiskGovernance(tx, id, u.UserID) {
			c.JSON(403, gin.H{"code": 403, "message": "R3 风险仅专项审批角色可审批"})
			return
		}
	} else if !isRiskManager(tx, id, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅平台治理角色可审批风险"})
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
	nextStatus := riskStatus
	if in.Decision == "reject" {
		nextStatus = "rejected"
	} else {
		// approve：累计两名不同审批人 approve 才转为 approved。
		var approveCount int
		if err = tx.QueryRow(`SELECT COUNT(DISTINCT approver_user_id) FROM vopc_risk_approvals WHERE risk_id=? AND decision='approve'`, rid).Scan(&approveCount); err != nil {
			serverError(c, "审批失败")
			return
		}
		if approveCount >= 2 {
			nextStatus = "approved"
		}
	}
	if nextStatus != riskStatus {
		if _, err = tx.Exec(`UPDATE vopc_risks SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='open'`, nextStatus, rid); err != nil {
			serverError(c, "风险状态更新失败")
			return
		}
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

// FreezeProject 平台治理角色冻结或解冻项目。冻结后项目进入 risk_frozen，所有
// 写操作被 blockedStatuses 拦截；解冻恢复。必须填写理由并审计。
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
	if !isRiskManager(tx, id, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅平台治理角色可冻结/解冻项目"})
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

// ResolveRiskAppeal 平台治理角色裁定申诉（upheld 维持原处置 / dismissed 驳回）。
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
	if !isRiskManager(tx, id, u.UserID) {
		c.JSON(403, gin.H{"code": 403, "message": "仅平台治理角色可裁定申诉"})
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
// R2（一般风险）：项目 risk_level 为 R2 时，必须存在至少一条 status=approved 且
// risk_level=R2 的风险（即已经平台双人审批）方可推进里程碑。
//
// R3（禁止或专项审批）：一旦项目为 R3，或项目上登记了任意未通过专项审批的 R3
// 风险，即视为 R3-tier，在“两名不同专项审批人（risk_governance）均 approve”之前
// 一律禁止推进里程碑。R3 较 R2 更严格：即使项目本体是 R0/R1/R2，只要挂有否决级
// R3 风险，也必须先走专项审批。
// 返回 (是否允许, 错误文案, 状态码)。
func milestoneAdvanceAllowed(tx *sql.Tx, projectID int64) (bool, string, int) {
	var riskLevel string
	if err := tx.QueryRow(`SELECT risk_level FROM vopc_projects WHERE id=?`, projectID).Scan(&riskLevel); err != nil {
		return false, "项目风险门禁读取失败", 500
	}
	// 是否存在未通过专项审批的 R3 风险。
	var r3Outstanding int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND risk_level='R3' AND status<>'approved'`, projectID).Scan(&r3Outstanding); err != nil {
		return false, "项目风险门禁读取失败", 500
	}
	// 项目级别 R3：需专项审批通过的 R3 风险。
	if riskLevel == "R3" {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND risk_level='R3' AND status='approved'`, projectID).Scan(&n); err != nil {
			return false, "项目风险门禁读取失败", 500
		}
		if n == 0 {
			return false, "R3 项目须经平台专项审批后方可推进里程碑", 409
		}
		return true, "", 0
	}
	// 即使项目本体非 R3，存在未专项审批的 R3 风险也阻断（禁止推进）。
	if r3Outstanding > 0 {
		return false, "存在未通过专项审批的 R3 风险，禁止推进里程碑", 409
	}
	if riskLevel != "R2" {
		return true, "", 0
	}
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND risk_level='R2' AND status='approved'`, projectID).Scan(&n); err != nil {
		return false, "项目风险门禁读取失败", 500
	}
	if n == 0 {
		return false, "R2 项目须经平台双人审批后方可推进里程碑", 409
	}
	return true, "", 0
}
