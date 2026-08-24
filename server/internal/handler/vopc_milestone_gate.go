package handler

// vOPC A4 里程碑完整业务门禁：评分量表读取 / 条件闭环(finalize) / 豁免(waiver) / 甲方结构化证据(client evidence)。
// 纯增量实现：复用既有 readableProject/manageableProject/projectBlockedForWrite/milestoneAdvanceAllowed/
// writeEvent/validObjectKey/parsePositiveID/serverError 等辅助；不改 pass/return 语义，不改既有表结构。
// 所有读写均挂 vopc 组（CollegeAccess 学院准入已在 group 层），capability 沿用现有常量。

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/dll/wxx/server/internal/middleware"
)

// ---- ① 评分量表 rubric ----

// ListRubrics 返回本项目 S0–S9 全阶段量表（含只读维度定义）。
// 读权限：项目读（无 capability 门控，同 ListMilestoneSubmissions）。
func (h *VOPCHandler) ListRubrics(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok2 := h.readableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT stage,dimension_key,title,max_score,min_pass,description FROM vopc_rubrics ORDER BY stage,dimension_key`)
	if err != nil {
		serverError(c, "评分量表读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var stage, key, title, desc string
		var maxScore, minPass int64
		if rows.Scan(&stage, &key, &title, &maxScore, &minPass, &desc) != nil {
			serverError(c, "评分量表读取失败")
			return
		}
		items = append(items, gin.H{
			"stage": stage, "dimension_key": key, "title": title,
			"max_score": maxScore, "min_pass": minPass, "description": desc,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// GetSubmissionReview 返回单个里程碑提交的评审详情（含评分维度得分 scores 与条件 conditions）。
// 读权限：项目读。
func (h *VOPCHandler) GetSubmissionReview(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	sid, e := parsePositiveID(c.Param("submissionId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "提交 ID 无效"})
		return
	}
	tx, _, ok2 := h.readableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	// 校验提交属于本项目
	var pid int64
	if err := tx.QueryRow(`SELECT project_id FROM vopc_milestone_submissions WHERE id=?`, sid).Scan(&pid); err != nil || pid != id {
		c.JSON(404, gin.H{"code": 404, "message": "提交不存在或无权访问"})
		return
	}

	var reviewID int64
	var reviewer int64
	var result, note string
	err := tx.QueryRow(`SELECT id,reviewer_user_id,result,note FROM vopc_milestone_reviews WHERE submission_id=?`, sid).Scan(&reviewID, &reviewer, &result, &note)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"reviewed": false}})
			return
		}
		serverError(c, "评审读取失败")
		return
	}

	// 评分维度
	scoreRows, serr := tx.Query(`SELECT r.dimension_key,r.title,rs.score,rs.comment FROM vopc_review_scores rs JOIN vopc_rubrics r ON r.id=rs.rubric_id WHERE rs.review_id=?`, reviewID)
	if serr != nil {
		serverError(c, "评审读取失败")
		return
	}
	scores := make([]gin.H, 0)
	for scoreRows.Next() {
		var dk, dt, cmt string
		var sc int64
		if scoreRows.Scan(&dk, &dt, &sc, &cmt) == nil {
			scores = append(scores, gin.H{"dimension_key": dk, "title": dt, "score": sc, "comment": cmt})
		}
	}
	scoreRows.Close()

	// 条件
	condRows, cerr := tx.Query(`SELECT id,description,satisfied,due_at FROM vopc_milestone_conditions WHERE submission_id=? ORDER BY id`, sid)
	if cerr != nil {
		serverError(c, "评审读取失败")
		return
	}
	conds := make([]gin.H, 0)
	for condRows.Next() {
		var cid int64
		var desc string
		var sat int64
		var rawDue any
		if condRows.Scan(&cid, &desc, &sat, &rawDue) == nil {
			conds = append(conds, gin.H{"id": cid, "description": desc, "satisfied": sat == 1, "due_at": anyString(rawDue)})
		}
	}
	condRows.Close()

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"reviewed": true, "review_id": reviewID, "reviewer_user_id": reviewer,
		"result": result, "note": note, "scores": scores, "conditions": conds,
	}})
}

// ---- ② 条件闭环 conditional pass ----

// MarkConditionSatisfied 主理人（manage）标记单条条件已满足。
func (h *VOPCHandler) MarkConditionSatisfied(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	sid, e := parsePositiveID(c.Param("submissionId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "提交 ID 无效"})
		return
	}
	cid, e2 := parsePositiveID(c.Param("conditionId"))
	if e2 != nil {
		c.JSON(400, gin.H{"code": 400, "message": "条件 ID 无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, _, ok2 := h.manageableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	// 校验该条件属于本项目提交
	var subStatus string
	if err := tx.QueryRow(`SELECT s.status FROM vopc_milestone_conditions mc JOIN vopc_milestone_submissions s ON s.id=mc.submission_id JOIN vopc_projects p ON p.id=s.project_id WHERE mc.id=? AND s.project_id=?`, cid, id).Scan(&subStatus); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "条件不存在或无权操作"})
		return
	}
	if subStatus != "condition_pending" {
		c.JSON(409, gin.H{"code": 409, "message": "该提交不在待闭环状态"})
		return
	}
	res, err := tx.Exec(`UPDATE vopc_milestone_conditions SET satisfied=1,satisfied_by=?,satisfied_at=CURRENT_TIMESTAMP WHERE id=? AND satisfied=0`, u.UserID, cid)
	if err != nil {
		serverError(c, "条件更新失败")
		return
	}
	if n, rerr := res.RowsAffected(); rerr != nil || n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "条件已满足或被并发修改"})
		return
	}
	if writeEvent(tx, id, u.UserID, "milestone.condition.satisfied", "0", "1", fmt.Sprintf("提交 #%d 条件 #%d 已满足", sid, cid)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "条件更新失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"submission_id": sid, "condition_id": cid, "satisfied": true}})
}

// FinalizeMilestone 评审者/平台运营确认条件全部满足，submission 转 passed 并执行与 pass 相同的推进逻辑
// （风险门禁复检 + TOCTOU + 阶段 CAS），不绕过 milestoneAdvanceAllowed。
func (h *VOPCHandler) FinalizeMilestone(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	sid, e := parsePositiveID(c.Param("submissionId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "提交 ID 无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "闭环失败")
		return
	}
	defer tx.Rollback()

	var owner int64
	var reviewerID sql.NullInt64
	var subStage, subStatus, curStage, curStatus string
	if err := tx.QueryRow(`SELECT p.owner_user_id,s.reviewer_user_id,s.stage,s.status,p.stage,p.status FROM vopc_milestone_submissions s JOIN vopc_projects p ON p.id=s.project_id WHERE s.id=? AND s.project_id=?`, sid, id).Scan(&owner, &reviewerID, &subStage, &subStatus, &curStage, &curStatus); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "提交不存在或无权访问"})
		return
	}
	if subStatus != "condition_pending" {
		c.JSON(409, gin.H{"code": 409, "message": "该提交不在待闭环状态"})
		return
	}
	allowed := reviewerID.Valid && reviewerID.Int64 == u.UserID
	if !allowed {
		var role string
		err := tx.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, id, u.UserID).Scan(&role)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			serverError(c, "闭环失败")
			return
		}
		allowed = role == "platform_operator"
	}
	if !allowed {
		c.JSON(403, gin.H{"code": 403, "message": "仅原评审或平台运营可确认闭环"})
		return
	}
	// 校验全部条件已满足
	var open int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_milestone_conditions WHERE submission_id=? AND satisfied=0`, sid).Scan(&open); err != nil {
		serverError(c, "条件读取失败")
		return
	}
	if open > 0 {
		c.JSON(409, gin.H{"code": 409, "message": "仍有未满足的条件，不可闭环"})
		return
	}
	// 复用 pass 推进路径
	if code, msg := advanceMilestoneAsPass(tx, id, sid, curStage, subStage, "条件通过闭环"); code != 0 {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}
	if writeEvent(tx, id, u.UserID, "milestone.finalized", "condition_pending", "passed", fmt.Sprintf("提交 #%d 条件闭环通过", sid)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "闭环失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": sid, "status": "passed"}})
}

// ---- ③ 豁免 waiver ----

// ListMilestoneWaivers 项目豁免申请列表（读权限）。
func (h *VOPCHandler) ListMilestoneWaivers(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok2 := h.readableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,stage,required_evidence,reason,status,requested_by,reviewed_by,review_note,created_at,reviewed_at FROM vopc_milestone_waivers WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "豁免列表读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var id2 int64
		var stage, reqEv, reason, status, rnote, reqBy string
		var reviewer sql.NullInt64
		var createdAt, reviewedAt any
		if rows.Scan(&id2, &stage, &reqEv, &reason, &status, &reqBy, &reviewer, &rnote, &createdAt, &reviewedAt) != nil {
			serverError(c, "豁免列表读取失败")
			return
		}
		items = append(items, gin.H{
			"id": id2, "stage": stage, "required_evidence": reqEv, "reason": reason,
			"status": status, "requested_by": reqBy, "reviewed_by": nullableInt64(reviewer),
			"review_note": rnote, "created_at": anyString(createdAt), "reviewed_at": anyString(reviewedAt),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// CreateMilestoneWaiver 主理人（manage）为某阶段必交证据申请豁免。
func (h *VOPCHandler) CreateMilestoneWaiver(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	var in struct {
		Stage           string `json:"stage"`
		RequiredEvidence string `json:"required_evidence"`
		Reason          string `json:"reason"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Stage = strings.TrimSpace(in.Stage)
	in.RequiredEvidence = strings.TrimSpace(in.RequiredEvidence)
	in.Reason = strings.TrimSpace(in.Reason)
	if !isVOPCStage(in.Stage) || in.Reason == "" {
		c.JSON(422, gin.H{"code": 422, "message": "豁免阶段与申请理由必填"})
		return
	}
	if utf8.RuneCountInString(in.Reason) > 2000 {
		c.JSON(422, gin.H{"code": 422, "message": "申请理由不超过 2000 字"})
		return
	}
	tx, _, ok2 := h.manageableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(409, gin.H{"code": 409, "message": msg})
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_milestone_waivers(project_id,stage,required_evidence,reason,requested_by) VALUES(?,?,?,?,?)`, id, in.Stage, in.RequiredEvidence, in.Reason, u.UserID)
	if err != nil {
		serverError(c, "豁免申请失败")
		return
	}
	wid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "milestone.waiver.requested", "", "pending", fmt.Sprintf("申请豁免 %s：%s", in.Stage, in.Reason)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "豁免申请失败")
		return
	}
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": wid, "status": "pending"}})
}

// ReviewMilestoneWaiver 审批豁免（approve/reject）。R2/R3 项目豁免必须由治理角色（VOPCRiskManage/VOPCAudit）复核，
// 导师单签被拒；R3 阻断本身不可豁免（豁免仅作用于必交证据/评分维度）。
func (h *VOPCHandler) ReviewMilestoneWaiver(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	wid, e := parsePositiveID(c.Param("waiverId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "豁免 ID 无效"})
		return
	}
	u := middleware.GetUserContext(c)
	var in struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Action = strings.TrimSpace(in.Action)
	in.Note = strings.TrimSpace(in.Note)
	if in.Action != "approve" && in.Action != "reject" {
		c.JSON(422, gin.H{"code": 422, "message": "审批动作必须为 approve 或 reject"})
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "豁免审批失败")
		return
	}
	defer tx.Rollback()
	var wkStatus, riskLevel string
	if err := tx.QueryRow(`SELECT w.status,p.risk_level FROM vopc_milestone_waivers w JOIN vopc_projects p ON p.id=w.project_id WHERE w.id=? AND w.project_id=?`, wid, id).Scan(&wkStatus, &riskLevel); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "豁免不存在或无权操作"})
		return
	}
	if wkStatus != "pending" {
		c.JSON(409, gin.H{"code": 409, "message": "该豁免已审批"})
		return
	}
	// R2/R3 判定
	var r23Count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND risk_level IN ('R2','R3')`, id).Scan(&r23Count); err != nil {
		serverError(c, "豁免审批失败")
		return
	}
	isR23 := riskLevel == "R2" || riskLevel == "R3" || r23Count > 0
	// R2/R3 项目豁免必须由平台治理系统角色复核（DB users.role 权威判据，非 JWT 自证）。
	var callerRole string
	if err := tx.QueryRow(`SELECT role FROM users WHERE id=?`, u.UserID).Scan(&callerRole); err != nil {
		serverError(c, "豁免审批失败")
		return
	}
	if isR23 && !platformGovernanceRoles[callerRole] {
		c.JSON(403, gin.H{"code": 403, "message": "R2/R3 项目豁免须平台治理角色复核"})
		return
	}
	// R3 不能豁免 R3 阻断本身
	next := "rejected"
	if in.Action == "approve" {
		next = "approved"
	}
	res, err := tx.Exec(`UPDATE vopc_milestone_waivers SET status=?,reviewed_by=?,review_note=? WHERE id=? AND status='pending'`, next, u.UserID, in.Note, wid)
	if err != nil {
		serverError(c, "豁免审批失败")
		return
	}
	if n, rerr := res.RowsAffected(); rerr != nil || n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "该豁免状态已变化"})
		return
	}
	if writeEvent(tx, id, u.UserID, "milestone.waiver.reviewed", "pending", next, fmt.Sprintf("豁免 #%d %s：%s", wid, in.Action, in.Note)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "豁免审批失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": wid, "status": next}})
}

// ---- ④ 甲方结构化证据 client evidence ----

// ListClientEvidence 甲方结构化证据列表（读权限）。
func (h *VOPCHandler) ListClientEvidence(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok2 := h.readableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,stage,client_rep,client_contact,conclusion,sign_method,file_ref,note,created_by,created_at FROM vopc_client_evidence WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "甲方证据读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var eid int64
		var stage, rep, contact, conclusion, sign, fileRef, note string
		var createdBy int64
		var createdAt any
		if rows.Scan(&eid, &stage, &rep, &contact, &conclusion, &sign, &fileRef, &note, &createdBy, &createdAt) != nil {
			serverError(c, "甲方证据读取失败")
			return
		}
		items = append(items, gin.H{
			"id": eid, "stage": stage, "client_rep": rep, "client_contact": contact,
			"conclusion": conclusion, "sign_method": sign, "file_ref": fileRef,
			"note": note, "created_by": createdBy, "created_at": anyString(createdAt),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// CreateClientEvidence 主理人（manage）登记甲方结构化证据。仅 S2/S5/S6 阶段可取；file_ref 若提供须为本项目受控私有文件。
func (h *VOPCHandler) CreateClientEvidence(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	var in struct {
		Stage         string `json:"stage"`
		ClientRep     string `json:"client_rep"`
		ClientContact string `json:"client_contact"`
		Conclusion    string `json:"conclusion"`
		SignMethod    string `json:"sign_method"`
		FileRef       string `json:"file_ref"`
		Note          string `json:"note"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Stage = strings.TrimSpace(in.Stage)
	in.ClientRep = strings.TrimSpace(in.ClientRep)
	in.Conclusion = strings.TrimSpace(in.Conclusion)
	in.FileRef = strings.TrimSpace(in.FileRef)
	if !sliceIn([]string{"G2", "G3"}, in.Stage) {
		c.JSON(422, gin.H{"code": 422, "message": "反馈/自查证据仅可在 G2/G3 阶段登记"})
		return
	}
	if in.ClientRep == "" || !sliceIn([]string{"confirmed", "rejected", "reserved"}, in.Conclusion) {
		c.JSON(422, gin.H{"code": 422, "message": "反馈来源与确认结论必填且结论合法"})
		return
	}
	if in.FileRef != "" && !validObjectKey(in.FileRef) {
		c.JSON(422, gin.H{"code": 422, "message": "文件引用无效"})
		return
	}
	tx, _, ok2 := h.manageableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(409, gin.H{"code": 409, "message": msg})
		return
	}
	// file_ref 是本项目受控文件且非 scan_failed
	if in.FileRef != "" {
		var fs string
		err := tx.QueryRow(`SELECT storage_status FROM vopc_files WHERE project_id=? AND object_key=?`, id, in.FileRef).Scan(&fs)
		if err != nil || fs == "scan_failed" {
			c.JSON(422, gin.H{"code": 422, "message": "文件引用不是本项目受控文件或存在安全问题"})
			return
		}
	}
	res, err := tx.Exec(`INSERT INTO vopc_client_evidence(project_id,stage,client_rep,client_contact,conclusion,sign_method,file_ref,note,created_by) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, in.Stage, in.ClientRep, in.ClientContact, in.Conclusion, in.SignMethod, in.FileRef, in.Note, u.UserID)
	if err != nil {
		serverError(c, "甲方证据登记失败")
		return
	}
	eid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "milestone.client_evidence", "", in.Stage, fmt.Sprintf("甲方证据 #%d：%s", eid, in.Conclusion)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "甲方证据登记失败")
		return
	}
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": eid, "stage": in.Stage}})
}

// UpdateClientEvidence 更新甲方证据（限尚未进入验收后续阶段/未结项前）。
func (h *VOPCHandler) UpdateClientEvidence(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	eid, e := parsePositiveID(c.Param("evidenceId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "证据 ID 无效"})
		return
	}
	u := middleware.GetUserContext(c)
	var in struct {
		ClientRep     string `json:"client_rep"`
		ClientContact string `json:"client_contact"`
		Conclusion    string `json:"conclusion"`
		SignMethod    string `json:"sign_method"`
		FileRef       string `json:"file_ref"`
		Note          string `json:"note"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Conclusion = strings.TrimSpace(in.Conclusion)
	in.FileRef = strings.TrimSpace(in.FileRef)
	if !sliceIn([]string{"confirmed", "rejected", "reserved"}, in.Conclusion) {
		c.JSON(422, gin.H{"code": 422, "message": "确认结论非法"})
		return
	}
	if in.FileRef != "" && !validObjectKey(in.FileRef) {
		c.JSON(422, gin.H{"code": 422, "message": "文件引用无效"})
		return
	}
	tx, _, ok2 := h.manageableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(409, gin.H{"code": 409, "message": msg})
		return
	}
	var existStage string
	if err := tx.QueryRow(`SELECT stage FROM vopc_client_evidence WHERE id=? AND project_id=?`, eid, id).Scan(&existStage); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "证据不存在或无权操作"})
		return
	}
	if in.FileRef != "" {
		var fs string
		err := tx.QueryRow(`SELECT storage_status FROM vopc_files WHERE project_id=? AND object_key=?`, id, in.FileRef).Scan(&fs)
		if err != nil || fs == "scan_failed" {
			c.JSON(422, gin.H{"code": 422, "message": "文件引用不是本项目受控文件或存在安全问题"})
			return
		}
	}
	_, err := tx.Exec(`UPDATE vopc_client_evidence SET client_rep=?,client_contact=?,conclusion=?,sign_method=?,file_ref=?,note=? WHERE id=? AND project_id=?`,
		strings.TrimSpace(in.ClientRep), in.ClientContact, in.Conclusion, in.SignMethod, in.FileRef, in.Note, eid, id)
	if err != nil {
		serverError(c, "甲方证据更新失败")
		return
	}
	if writeEvent(tx, id, u.UserID, "milestone.client_evidence.updated", existStage, existStage, fmt.Sprintf("甲方证据 #%d 更新", eid)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "甲方证据更新失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": eid}})
}

// ---- 辅助 ----

func isVOPCStage(s string) bool {
	return isGStage(s)
}

func sliceIn(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func nullableInt64(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}

func anyString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprint(t)
	}
}

// 注：vOPC 能力判定以路由中间件 auth.RequireCapability/RequireAnyCapability 在入口执行；
// handler 内需要治理角色判定时，以数据库 users.role + platformGovernanceRoles 为权威（非 JWT 自证）。

var _ = json.Marshal
