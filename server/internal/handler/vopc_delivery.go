package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// platform_operator is a platform-governance role and must never be granted by
// a normal project invitation, including invitations created by owners.
var projectRoles = setOf("co_owner", "member", "mentor", "reviewer")
var artifactTypes = setOf("document", "image", "archive", "repository", "dataset", "link", "file", "other")
var sourceKinds = setOf("link", "repository", "storage_ref", "dataset_ref")

// G 阶段要求的成果类型（v2.0 精简主线）。G2 产出为主（文档/文件/仓库/数据），G3 反馈与验证，G4 复盘归档。
var milestoneArtifactTypes = map[string]map[string]bool{
	"G0": setOf("document", "file"), "G1": setOf("document", "file"), "G2": setOf("document", "file", "repository", "dataset"),
	"G3": setOf("document", "file", "dataset"), "G4": setOf("document", "file", "archive", "repository"),
}

func (h *VOPCHandler) ListMembers(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, owner, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	_ = owner
	rows, err := tx.Query(`SELECT m.user_id,m.project_role,m.status,m.created_at,u.username,u.display_name FROM vopc_project_members m JOIN users u ON u.id=m.user_id WHERE m.project_id=? ORDER BY m.id`, id)
	if err != nil {
		serverError(c, "成员列表读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var uid int64
		var role, status, created, username, name string
		if rows.Scan(&uid, &role, &status, &created, &username, &name) != nil {
			serverError(c, "成员列表读取失败")
			return
		}
		items = append(items, gin.H{"user_id": uid, "project_role": role, "status": status, "username": username, "display_name": name, "created_at": created})
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}

func (h *VOPCHandler) InviteMember(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		UserID  int64  `json:"user_id"`
		Role    string `json:"project_role"`
		Message string `json:"message"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Role = strings.TrimSpace(in.Role)
	in.Message = strings.TrimSpace(in.Message)
	if in.UserID <= 0 || !projectRoles[in.Role] || utf8.RuneCountInString(in.Message) > 500 {
		c.JSON(422, gin.H{"code": 422, "message": "受邀用户、项目角色或邀请说明无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "邀请失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var owner int64
	if tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner) != nil {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	allowed, e := projectPolicy(tx, id, u.UserID, owner, "manage")
	if e != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	var status, scope, college, role string
	if err = tx.QueryRow(`SELECT status,owner_scope,owner_id,role FROM users WHERE id=?`, in.UserID).Scan(&status, &scope, &college, &role); errors.Is(err, sql.ErrNoRows) {
		c.JSON(422, gin.H{"code": 422, "message": "受邀用户不存在"})
		return
	} else if err != nil {
		serverError(c, "邀请失败")
		return
	}
	if status != "active" || role == "guest" || scope != "college" || !strings.EqualFold(college, h.collegeID) {
		c.JSON(422, gin.H{"code": 422, "message": "仅可邀请计算机学院已授权且状态正常的用户"})
		return
	}
	var n int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, id, in.UserID).Scan(&n); err != nil {
		serverError(c, "邀请失败")
		return
	}
	if n > 0 {
		c.JSON(409, gin.H{"code": 409, "message": "用户已是项目成员"})
		return
	}
	if err = tx.QueryRow(`SELECT COUNT(*) FROM vopc_invitations WHERE project_id=? AND invitee_user_id=? AND status='pending'`, id, in.UserID).Scan(&n); err != nil {
		serverError(c, "邀请失败")
		return
	}
	if n > 0 {
		c.JSON(409, gin.H{"code": 409, "message": "已有待处理邀请"})
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_invitations(project_id,invitee_user_id,project_role,message,invited_by) VALUES(?,?,?,?,?)`, id, in.UserID, in.Role, in.Message, u.UserID)
	if err != nil {
		serverError(c, "邀请失败")
		return
	}
	iid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "member.invited", "", "pending", fmt.Sprintf("邀请 #%d 用户 #%d", iid, in.UserID)) != nil {
		serverError(c, "邀请审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "邀请失败")
		return
	}
	committed = true
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": iid, "status": "pending"}})
}

func (h *VOPCHandler) ListMyInvitations(c *gin.Context) {
	u := middleware.GetUserContext(c)
	rows, err := h.db.Query(`SELECT i.id,i.project_id,p.name,i.project_role,i.message,i.status,i.created_at FROM vopc_invitations i JOIN vopc_projects p ON p.id=i.project_id WHERE i.invitee_user_id=? ORDER BY i.created_at DESC`, u.UserID)
	if err != nil {
		serverError(c, "邀请列表读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var id, pid int64
		var name, role, msg, status, created string
		if rows.Scan(&id, &pid, &name, &role, &msg, &status, &created) != nil {
			serverError(c, "邀请列表读取失败")
			return
		}
		items = append(items, gin.H{"id": id, "project_id": pid, "project_name": name, "project_role": role, "message": msg, "status": status, "created_at": created})
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}
func (h *VOPCHandler) RespondInvitation(c *gin.Context) {
	iid, e := parsePositiveID(c.Param("invitationId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "邀请 ID 无效"})
		return
	}
	var in struct {
		Action string `json:"action"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	if in.Action != "accept" && in.Action != "decline" {
		c.JSON(422, gin.H{"code": 422, "message": "邀请动作仅支持 accept 或 decline"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "邀请处理失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var pid, uid int64
	var role, status string
	if tx.QueryRow(`SELECT project_id,invitee_user_id,project_role,status FROM vopc_invitations WHERE id=? AND invitee_user_id=?`, iid, u.UserID).Scan(&pid, &uid, &role, &status) != nil {
		c.JSON(404, gin.H{"code": 404, "message": "邀请不存在或无权操作"})
		return
	}
	if status != "pending" {
		c.JSON(409, gin.H{"code": 409, "message": "邀请已处理"})
		return
	}
	next := "declined"
	if in.Action == "accept" {
		var userStatus, scope, college, userRole string
		if tx.QueryRow(`SELECT status,owner_scope,owner_id,role FROM users WHERE id=?`, uid).Scan(&userStatus, &scope, &college, &userRole) != nil || userStatus != "active" || userRole == "guest" || scope != "college" || !strings.EqualFold(college, h.collegeID) {
			c.JSON(403, gin.H{"code": 403, "message": "当前账号已不具备 vOPC 学院准入资格"})
			return
		}
		next = "accepted"
	}
	res, err := tx.Exec(`UPDATE vopc_invitations SET status=?,responded_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'`, next, iid)
	if err != nil {
		serverError(c, "邀请处理失败")
		return
	}
	if n, rowsErr := res.RowsAffected(); rowsErr != nil {
		serverError(c, "邀请处理失败")
		return
	} else if n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "邀请状态已变化"})
		return
	}
	if next == "accepted" {
		if _, err = tx.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role,status) VALUES(?,?,?,'active')`, pid, uid, role); err != nil {
			serverError(c, "加入项目失败")
			return
		}
	}
	if writeEvent(tx, pid, u.UserID, "member.invitation_responded", "pending", next, fmt.Sprintf("邀请 #%d", iid)) != nil {
		serverError(c, "邀请审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "邀请处理失败")
		return
	}
	committed = true
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"id": iid, "status": next}})
}

func (h *VOPCHandler) ListArtifacts(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT a.id,a.name,a.artifact_type,a.description,a.visibility,a.license,a.created_by,a.created_at,COUNT(v.id) FROM vopc_artifacts a LEFT JOIN vopc_artifact_versions v ON v.artifact_id=a.id WHERE a.project_id=? GROUP BY a.id ORDER BY a.updated_at DESC`, id)
	if err != nil {
		serverError(c, "成果读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var aid, by, count int64
		var name, typ, desc, vis, license, created string
		if rows.Scan(&aid, &name, &typ, &desc, &vis, &license, &by, &created, &count) != nil {
			serverError(c, "成果读取失败")
			return
		}
		items = append(items, gin.H{"id": aid, "name": name, "artifact_type": typ, "description": desc, "visibility": vis, "license": license, "created_by": by, "created_at": created, "version_count": count})
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}
func (h *VOPCHandler) CreateArtifact(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		Name        string `json:"name"`
		Type        string `json:"artifact_type"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
		License     string `json:"license"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Type = strings.TrimSpace(in.Type)
	if in.Visibility == "" {
		in.Visibility = "private"
	}
	if in.Name == "" || !artifactTypes[in.Type] || (in.Visibility != "private" && in.Visibility != "project") {
		c.JSON(422, gin.H{"code": 422, "message": "成果名称、类型或可见性无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, owner, ok := h.manageableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	_ = owner
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(409, gin.H{"code": 409, "message": msg})
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_artifacts(project_id,name,artifact_type,description,visibility,license,created_by) VALUES(?,?,?,?,?,?,?)`, id, in.Name, in.Type, strings.TrimSpace(in.Description), in.Visibility, strings.TrimSpace(in.License), u.UserID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			c.JSON(409, gin.H{"code": 409, "message": "同名成果已存在"})
		} else {
			serverError(c, "成果创建失败")
		}
		return
	}
	aid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "artifact.created", "", "created", fmt.Sprintf("成果 #%d", aid)) != nil {
		serverError(c, "成果审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "成果创建失败")
		return
	}
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": aid}})
}
func (h *VOPCHandler) ListArtifactVersions(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	aid, e := parsePositiveID(c.Param("artifactId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "成果 ID 无效"})
		return
	}
	tx, _, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT v.id,v.version,v.source_kind,v.source_ref,v.checksum,v.release_notes,v.status,v.intended_stage,v.created_by,v.created_at FROM vopc_artifact_versions v JOIN vopc_artifacts a ON a.id=v.artifact_id WHERE v.artifact_id=? AND a.project_id=? ORDER BY v.id DESC`, aid, id)
	if err != nil {
		serverError(c, "版本读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var vid, by int64
		var version, kind, ref, sum, notes, status, intendedStage, created string
		if rows.Scan(&vid, &version, &kind, &ref, &sum, &notes, &status, &intendedStage, &by, &created) != nil {
			serverError(c, "版本读取失败")
			return
		}
		items = append(items, gin.H{"id": vid, "version": version, "source_kind": kind, "source_ref": ref, "checksum": sum, "release_notes": notes, "status": status, "intended_stage": intendedStage, "created_by": by, "created_at": created})
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}
func (h *VOPCHandler) CreateArtifactVersion(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	aid, e := parsePositiveID(c.Param("artifactId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "成果 ID 无效"})
		return
	}
	var in struct {
		Version    string `json:"version"`
		SourceKind string `json:"source_kind"`
		SourceRef  string `json:"source_ref"`
		Checksum   string `json:"checksum"`
		Notes      string `json:"release_notes"`
		Stage      string `json:"intended_stage"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Version = strings.TrimSpace(in.Version)
	in.SourceKind = strings.TrimSpace(in.SourceKind)
	in.SourceRef = strings.TrimSpace(in.SourceRef)
	in.Checksum = strings.ToLower(strings.TrimSpace(in.Checksum))
	in.Stage = strings.ToUpper(strings.TrimSpace(in.Stage))
	if in.Version == "" || !sourceKinds[in.SourceKind] || in.SourceRef == "" || utf8.RuneCountInString(in.SourceRef) > 2000 || strings.ContainsAny(in.SourceRef, "\r\n") || !validSHA256(in.Checksum) || milestoneArtifactTypes[in.Stage] == nil {
		c.JSON(422, gin.H{"code": 422, "message": "版本或安全来源引用无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, _, ok := h.manageableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(409, gin.H{"code": 409, "message": msg})
		return
	}
	var n int
	if tx.QueryRow(`SELECT COUNT(*) FROM vopc_artifacts WHERE id=? AND project_id=?`, aid, id).Scan(&n) != nil || n != 1 {
		c.JSON(404, gin.H{"code": 404, "message": "成果不存在或无权操作"})
		return
	}
	// 受控文件引用：source_kind=storage_ref 时，source_ref 必须是本项目的受控私有文件 object_key，
	// 且文件已落盘（storage_status 不能为 scan_failed），否则拒绝，确保里程碑证据指向真实受控文件。
	if in.SourceKind == "storage_ref" {
		if !validObjectKey(in.SourceRef) {
			c.JSON(422, gin.H{"code": 422, "message": "受控文件引用 key 无效"})
			return
		}
		var fstatus string
		err := tx.QueryRow(`SELECT storage_status FROM vopc_files WHERE project_id=? AND object_key=?`, id, in.SourceRef).Scan(&fstatus)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && fstatus == "scan_failed") {
			c.JSON(422, gin.H{"code": 422, "message": "受控文件引用不存在或不可用"})
			return
		}
		if err != nil {
			serverError(c, "版本创建失败")
			return
		}
	}
	res, err := tx.Exec(`INSERT INTO vopc_artifact_versions(artifact_id,version,source_kind,source_ref,checksum,release_notes,status,intended_stage,created_by) VALUES(?,?,?,?,?,?,'active',?,?)`, aid, in.Version, in.SourceKind, in.SourceRef, in.Checksum, strings.TrimSpace(in.Notes), in.Stage, u.UserID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			c.JSON(409, gin.H{"code": 409, "message": "版本号已存在"})
		} else {
			serverError(c, "版本创建失败")
		}
		return
	}
	vid, _ := res.LastInsertId()
	updated, err := tx.Exec(`UPDATE vopc_artifacts SET updated_at=CURRENT_TIMESTAMP WHERE id=?`, aid)
	if err != nil {
		serverError(c, "版本创建失败")
		return
	}
	if affected, rowsErr := updated.RowsAffected(); rowsErr != nil || affected != 1 {
		serverError(c, "版本创建失败")
		return
	}
	if writeEvent(tx, id, u.UserID, "artifact.version_created", "", "created", fmt.Sprintf("成果 #%d 版本 #%d", aid, vid)) != nil {
		serverError(c, "版本审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "版本创建失败")
		return
	}
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": vid}})
}

func (h *VOPCHandler) ListMilestoneSubmissions(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,stage,evidence,artifact_version_ids,reviewer_user_id,status,submitted_by,submitted_at,reviewed_at FROM vopc_milestone_submissions WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "里程碑材料读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var sid, by int64
		var stage, evidence, versions, status, submitted string
		var reviewer sql.NullInt64
		var reviewed sql.NullString
		if rows.Scan(&sid, &stage, &evidence, &versions, &reviewer, &status, &by, &submitted, &reviewed) != nil {
			serverError(c, "里程碑材料读取失败")
			return
		}
		item := gin.H{"id": sid, "stage": stage, "evidence": evidence, "artifact_version_ids": json.RawMessage(versions), "status": status, "submitted_by": by, "submitted_at": submitted}
		if reviewer.Valid {
			item["reviewer_user_id"] = reviewer.Int64
		}
		if reviewed.Valid {
			item["reviewed_at"] = reviewed.String
		}
		items = append(items, item)
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}
func (h *VOPCHandler) SubmitMilestone(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in struct {
		Stage      string  `json:"stage"`
		Evidence   string  `json:"evidence"`
		VersionIDs []int64 `json:"artifact_version_ids"`
		ReviewerID *int64  `json:"reviewer_user_id"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Stage = strings.ToUpper(strings.TrimSpace(in.Stage))
	in.Evidence = strings.TrimSpace(in.Evidence)
	n := stageIndexOf(in.Stage)
	if n < 1 || n > 4 || in.Evidence == "" {
		c.JSON(422, gin.H{"code": 422, "message": "目标阶段或证据无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, _, ok := h.manageableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	var current, status string
	if tx.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, id).Scan(&current, &status) != nil {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在"})
		return
	}
	cur := stageIndexOf(current)
	if n != cur+1 || blockedStatuses[status] || completedLike[status] {
		c.JSON(409, gin.H{"code": 409, "message": "只能提交当前阶段的下一里程碑"})
		return
	}
	if allowed, msg, code := milestoneAdvanceAllowed(tx, id); !allowed {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}
	var pending int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_milestone_submissions WHERE project_id=? AND stage=? AND status='pending'`, id, in.Stage).Scan(&pending); err != nil {
		serverError(c, "里程碑提交失败")
		return
	}
	if pending > 0 {
		c.JSON(409, gin.H{"code": 409, "message": "该阶段已有待评审材料"})
		return
	}
	if len(in.VersionIDs) == 0 || len(in.VersionIDs) > 20 {
		c.JSON(422, gin.H{"code": 422, "message": "正式里程碑必须绑定至少一个本项目成果版本"})
		return
	}
	if in.ReviewerID != nil {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active' AND project_role IN ('mentor','reviewer','platform_operator')`, id, *in.ReviewerID).Scan(&count); err != nil {
			serverError(c, "里程碑提交失败")
			return
		}
		if count != 1 {
			c.JSON(422, gin.H{"code": 422, "message": "指定评审必须是项目内导师、评审者或平台运营"})
			return
		}
	}
	seen := make(map[int64]bool, len(in.VersionIDs))
	matchedRequiredType := false
	for _, vid := range in.VersionIDs {
		if vid <= 0 || seen[vid] {
			c.JSON(422, gin.H{"code": 422, "message": "证据版本不得重复且 ID 必须有效"})
			return
		}
		seen[vid] = true
		var artifactType, versionStatus, intendedStage, checksum string
		err := tx.QueryRow(`SELECT a.artifact_type,v.status,v.intended_stage,v.checksum FROM vopc_artifact_versions v JOIN vopc_artifacts a ON a.id=v.artifact_id WHERE v.id=? AND a.project_id=?`, vid, id).Scan(&artifactType, &versionStatus, &intendedStage, &checksum)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(422, gin.H{"code": 422, "message": "证据版本不属于该项目"})
			return
		}
		if err != nil {
			serverError(c, "里程碑提交失败")
			return
		}
		if versionStatus != "active" || intendedStage != in.Stage || !validSHA256(checksum) {
			c.JSON(422, gin.H{"code": 422, "message": "证据版本已失效或不适用于目标阶段"})
			return
		}
		matchedRequiredType = matchedRequiredType || milestoneArtifactTypes[in.Stage][artifactType]
	}
	if !matchedRequiredType {
		c.JSON(422, gin.H{"code": 422, "message": "缺少目标阶段要求的成果类型"})
		return
	}
	raw, _ := json.Marshal(in.VersionIDs)
	res, err := tx.Exec(`INSERT INTO vopc_milestone_submissions(project_id,stage,evidence,artifact_version_ids,reviewer_user_id,submitted_by) VALUES(?,?,?,?,?,?)`, id, in.Stage, in.Evidence, string(raw), in.ReviewerID, u.UserID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			c.JSON(409, gin.H{"code": 409, "message": "该阶段已有待评审材料"})
		} else {
			serverError(c, "里程碑提交失败")
		}
		return
	}
	sid, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "milestone.submitted", current, in.Stage, fmt.Sprintf("提交 #%d", sid)) != nil {
		serverError(c, "里程碑审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "里程碑提交失败")
		return
	}
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": sid, "status": "pending"}})
}

// reviewScoreIn 是评审请求体中可选的评分维度得分（A4 评分量表）。
// dimension_key 必须命中该提交 stage 的量表维度；score ∈ [0, max_score]。
type reviewScoreIn struct {
	DimensionKey string `json:"dimension_key"`
	Score        int64  `json:"score"`
	Comment      string `json:"comment"`
}

// reviewCondIn 是评审请求体中可选的条件通过条目（A4 conditional pass）。
type reviewCondIn struct {
	Description string `json:"description"`
	DueAt       string `json:"due_at"`
}

func (h *VOPCHandler) ReviewMilestone(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	sid, e := parsePositiveID(c.Param("submissionId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "提交 ID 无效"})
		return
	}
	var in struct {
		Result     string         `json:"result"`
		Note       string         `json:"note"`
		Scores     []reviewScoreIn `json:"scores"`
		Conditions []reviewCondIn `json:"conditions"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	if in.Result != "pass" && in.Result != "return" && in.Result != "conditional_pass" {
		c.JSON(422, gin.H{"code": 422, "message": "评审结果和意见必填"})
		return
	}
	if in.Note == "" {
		c.JSON(422, gin.H{"code": 422, "message": "评审意见必填"})
		return
	}
	// A4 conditional pass：必须携带至少一条待闭环条件。
	if in.Result == "conditional_pass" && len(in.Conditions) == 0 {
		c.JSON(422, gin.H{"code": 422, "message": "条件通过必须登记至少一条待闭环条件"})
		return
	}

	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "评审失败")
		return
	}
	defer tx.Rollback()
	var owner, reviewer int64
	var reviewerID sql.NullInt64
	var stage, status, current string
	if tx.QueryRow(`SELECT p.owner_user_id,s.reviewer_user_id,s.stage,s.status,p.stage FROM vopc_milestone_submissions s JOIN vopc_projects p ON p.id=s.project_id WHERE s.id=? AND s.project_id=?`, sid, id).Scan(&owner, &reviewerID, &stage, &status, &current) != nil {
		c.JSON(404, gin.H{"code": 404, "message": "里程碑提交不存在或无权访问"})
		return
	}
	reviewer = u.UserID
	allowed := reviewerID.Valid && reviewerID.Int64 == u.UserID
	if !allowed {
		var role string
		err := tx.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, id, u.UserID).Scan(&role)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			serverError(c, "评审失败")
			return
		}
		allowed = role == "platform_operator"
	}
	if !allowed {
		c.JSON(403, gin.H{"code": 403, "message": "仅指定评审或平台运营可评审"})
		return
	}
	if status != "pending" {
		c.JSON(409, gin.H{"code": 409, "message": "该提交已评审"})
		return
	}
	next := "returned"
	switch in.Result {
	case "pass":
		next = "passed"
	case "conditional_pass":
		next = "condition_pending"
	}
	res, err := tx.Exec(`INSERT INTO vopc_milestone_reviews(submission_id,reviewer_user_id,result,note) VALUES(?,?,?,?)`, sid, reviewer, in.Result, in.Note)
	if err != nil {
		serverError(c, "评审保存失败")
		return
	}
	reviewID, _ := res.LastInsertId()
	// A4 评分量表：可选附带各维度得分，落库到 vopc_review_scores；不传即回退原行为。
	if code, msg := recordReviewScores(tx, id, stage, reviewID, in.Scores); code != 0 {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}
	// A4 conditional pass：登记待闭环条件。
	if in.Result == "conditional_pass" {
		for _, cond := range in.Conditions {
			desc := strings.TrimSpace(cond.Description)
			if desc == "" || utf8.RuneCountInString(desc) > 1000 {
				c.JSON(422, gin.H{"code": 422, "message": "待闭环条件内容必填且不超过 1000 字"})
				return
			}
			if _, err = tx.Exec(`INSERT INTO vopc_milestone_conditions(submission_id,description,due_at) VALUES(?,?,?)`, sid, desc, nullableString(strings.TrimSpace(cond.DueAt))); err != nil {
				serverError(c, "条件登记失败")
				return
			}
		}
	}
	res, err = tx.Exec(`UPDATE vopc_milestone_submissions SET status=?,reviewed_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'`, next, sid)
	if err != nil {
		serverError(c, "评审保存失败")
		return
	}
	if affected, rowsErr := res.RowsAffected(); rowsErr != nil {
		serverError(c, "评审保存失败")
		return
	} else if affected != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "该提交状态已变化"})
		return
	}
	if next == "passed" {
		if code, msg := advanceMilestoneAsPass(tx, id, sid, current, stage, in.Note); code != 0 {
			c.JSON(code, gin.H{"code": code, "message": msg})
			return
		}
	}
	if writeEvent(tx, id, u.UserID, "milestone.reviewed", "pending", next, fmt.Sprintf("提交 #%d：%s", sid, in.Note)) != nil {
		serverError(c, "评审审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "评审失败")
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"id": sid, "status": next}})
}

// advanceMilestoneAsPass 执行 pass 分支的完整推进逻辑（风险门禁复检 + 阶段 CAS + 里程碑状态）。
// finalize（conditional pass 闭环）复用同一推进路径，确保不绕过 R2/R3 与 TOCTOU 门禁。
// 返回 (状态码 0=成功, 错误文案)。
func advanceMilestoneAsPass(tx *sql.Tx, projectID, submissionID int64, current, stage, note string) (int, string) {
	// H-B1 修复：提交后、评审前可能新登记了 R2/R3 风险或项目被升档，
	// 推进落地前必须复检风险门禁，封死 SubmitMilestone 通过后的 TOCTOU 绕过。
	if allowed, msg, code := milestoneAdvanceAllowed(tx, projectID); !allowed {
		return code, msg
	}
	cur := stageIndexOf(current)
	target := stageIndexOf(stage)
	if target != cur+1 {
		return 409, "项目阶段已变化，无法通过"
	}
	nextStatus := stageStatuses[target]
	res, err := tx.Exec(`UPDATE vopc_projects SET stage=?,status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND stage=?`, stage, nextStatus, projectID, current)
	if err != nil {
		return 500, "项目阶段更新失败"
	}
	if n, rowsErr := res.RowsAffected(); rowsErr != nil {
		return 500, "项目阶段更新失败"
	} else if n != 1 {
		return 409, "项目阶段已变化"
	}
	res, err = tx.Exec(`UPDATE vopc_milestones SET status='passed',review_note=? WHERE project_id=? AND stage=?`, note, projectID, current)
	if err != nil {
		return 500, "里程碑状态更新失败"
	}
	if n, rowsErr := res.RowsAffected(); rowsErr != nil || n != 1 {
		return 500, "里程碑状态更新失败"
	}
	return 0, ""
}

// recordReviewScores 校验并写入评审维度得分。scores 为空即直接返回成功（回退原行为）。
// 约束：dimension_key 必须命中该 stage 量表；score ∈ [0, max_score]。
func recordReviewScores(tx *sql.Tx, projectID int64, stage string, reviewID int64, scores []reviewScoreIn) (int, string) {
	if len(scores) == 0 {
		return 0, ""
	}
	_ = projectID
	for _, s := range scores {
		key := strings.TrimSpace(s.DimensionKey)
		if key == "" {
			return 422, "评分维度键不能为空"
		}
		var rubricID int64
		var maxScore int64
		err := tx.QueryRow(`SELECT id,max_score FROM vopc_rubrics WHERE stage=? AND dimension_key=?`, stage, key).Scan(&rubricID, &maxScore)
		if errors.Is(err, sql.ErrNoRows) {
			return 422, "评分维度不存在于该阶段量表：" + key
		}
		if err != nil {
			return 500, "评分量表读取失败"
		}
		if s.Score < 0 || s.Score > maxScore {
			return 422, "维度得分越界：" + key
		}
		if _, err = tx.Exec(`INSERT INTO vopc_review_scores(review_id,rubric_id,score,comment) VALUES(?,?,?,?)`, reviewID, rubricID, s.Score, strings.TrimSpace(s.Comment)); err != nil {
			return 500, "维度得分写入失败"
		}
	}
	return 0, ""
}

func (h *VOPCHandler) readableProject(c *gin.Context, id int64) (*sql.Tx, int64, bool) {
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "项目读取失败")
		return nil, 0, false
	}
	var owner int64
	if tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner) != nil {
		tx.Rollback()
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		return nil, 0, false
	}
	allowed, e := projectPolicy(tx, id, u.UserID, owner, "read")
	if e != nil || !allowed {
		tx.Rollback()
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		return nil, 0, false
	}
	return tx, owner, true
}
func (h *VOPCHandler) manageableProject(c *gin.Context, id int64) (*sql.Tx, int64, bool) {
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "项目读取失败")
		return nil, 0, false
	}
	var owner int64
	if tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner) != nil {
		tx.Rollback()
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return nil, 0, false
	}
	allowed, e := projectPolicy(tx, id, u.UserID, owner, "manage")
	if e != nil || !allowed {
		tx.Rollback()
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return nil, 0, false
	}
	return tx, owner, true
}

var _ = http.StatusOK

// projectBlockedForWrite 统一拦截治理冻结及不可写状态下的业务写操作，返回 (是否拦截, 文案)。
// blockedStatuses = {paused, risk_frozen, terminated, archived}：这些状态下不应产生新的成果/版本/风险等业务写。
// 注：申诉（CreateRiskAppeal）是冻结后的 remedy 路径，有意不加拦截。
func projectBlockedForWrite(tx *sql.Tx, id int64) (bool, string) {
	var status string
	if err := tx.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, id).Scan(&status); err != nil {
		return true, "项目状态读取失败"
	}
	if blockedStatuses[status] || completedLike[status] {
		return true, "当前项目治理状态禁止写操作"
	}
	return false, ""
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
