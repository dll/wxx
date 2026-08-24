package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// closeInput 是结项/异常状态机动作的请求体。reason 必填；close/terminate/pivot
// 额外要求人类决策依据，terminate 要求失败证据，close 要求成果包/复盘要点。
type closeInput struct {
	Action          string `json:"action"`
	Reason          string `json:"reason"`
	FailureEvidence string `json:"failure_evidence"`
	OutcomePackage  string `json:"outcome_package"`
	HumanDecision   string `json:"human_decision"`
}

// normalizeAndValidate 返回 (错误文案, 状态码)，并原地 trim。
func (in *closeInput) normalizeAndValidate() (string, int) {
	in.Action = strings.TrimSpace(in.Action)
	in.Reason = strings.TrimSpace(in.Reason)
	in.FailureEvidence = strings.TrimSpace(in.FailureEvidence)
	in.OutcomePackage = strings.TrimSpace(in.OutcomePackage)
	in.HumanDecision = strings.TrimSpace(in.HumanDecision)
	if !closeActions[in.Action] {
		return "动作仅支持 close/pause/resume/pivot/terminate/archive", 422
	}
	if in.Reason == "" || utf8.RuneCountInString(in.Reason) > 4000 {
		return "动作理由必填且不超过 4000 字", 422
	}
	for label, v := range map[string]string{"人类决策依据": in.HumanDecision, "失败证据": in.FailureEvidence, "成果包/复盘要点": in.OutcomePackage} {
		if utf8.RuneCountInString(v) > 4000 {
			return label + "超过 4000 字", 422
		}
	}
	switch in.Action {
	case "terminate":
		if in.FailureEvidence == "" {
			return "终止项目必须填写失败证据", 422
		}
	case "close":
		if in.HumanDecision == "" || in.OutcomePackage == "" {
			return "结项必须填写人类结项决策和成果包/复盘要点", 422
		}
	case "pivot":
		if in.HumanDecision == "" {
			return "转向必须填写人类决策依据", 422
		}
	}
	return "", 0
}

// CloseProject 是项目结项与异常状态机的统一入口。所有动作在同一事务内写
// vopc_close_records 与 vopc_events，任一步失败整体回滚。
//
// 合法流转：
//   - close：仅 closeable → completed（G4 复盘后结项）
//   - pause：任何非 draft/completed/closeable/terminated/archived 状态 → paused
//   - resume：仅 paused → 恢复到此前的活跃状态
//   - pivot：活跃或 paused → draft（回到 S0 重新定向）
//   - terminate：非 completed/archived → terminated
//   - archive：completed 或 terminated → archived
//
// 权限：仅项目 owner/co_owner/platform_operator（projectPolicy manage）。
func (h *VOPCHandler) CloseProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in closeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	if msg, code := in.normalizeAndValidate(); code != 0 {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "状态流转失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var owner int64
	var stage, status string
	if err = tx.QueryRow(`SELECT owner_user_id,stage,status FROM vopc_projects WHERE id=?`, id).Scan(&owner, &stage, &status); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	} else if err != nil {
		serverError(c, "状态读取失败")
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "manage")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}

	next, ok := closeTransition(status, in.Action)
	if !ok {
		c.JSON(409, gin.H{"code": 409, "message": "当前状态不允许该动作"})
		return
	}

	// H-B2 修复：risk_frozen 是治理冻结态，只能由平台治理角色经 unfreeze 解除，
	// 项目主理人不得趁机 pivot/terminate/pause 绕过治理冻结。
	if status == "risk_frozen" {
		c.JSON(409, gin.H{"code": 409, "message": "项目处于治理冻结状态，须先由平台治理角色解冻"})
		return
	}

	// resume 恢复到此前 pause 记录的 previous_status；若历史丢失则回退到 pending_review。
	if in.Action == "resume" {
		var prev sql.NullString
		if err = tx.QueryRow(`SELECT previous_status FROM vopc_close_records WHERE project_id=? AND action='pause' ORDER BY id DESC LIMIT 1`, id).Scan(&prev); err != nil && !errors.Is(err, sql.ErrNoRows) {
			serverError(c, "继续失败")
			return
		}
		next = "pending_review"
		if prev.Valid && prev.String != "" {
			next = prev.String
		}
	}

	// pivot 回到 G0 草案重定向：阶段回退、里程碑复位。
	if in.Action == "pivot" {
		res, e := tx.Exec(`UPDATE vopc_projects SET stage='G0',status=?,submitted_at=NULL,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status=?`, next, id, status)
		if e != nil {
			serverError(c, "转向失败")
			return
		}
		if n, _ := res.RowsAffected(); n != 1 {
			c.JSON(409, gin.H{"code": 409, "message": "项目状态已变化，请刷新后重试"})
			return
		}
		if _, e = tx.Exec(`UPDATE vopc_milestones SET status='pending',review_note='' WHERE project_id=? AND stage<>'G0'`, id); e != nil {
			serverError(c, "转向失败")
			return
		}
	} else if in.Action == "close" || in.Action == "terminate" || in.Action == "archive" {
		res, e := tx.Exec(`UPDATE vopc_projects SET status=?,closed_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status=?`, next, id, status)
		if e != nil {
			serverError(c, "状态流转失败")
			return
		}
		if n, _ := res.RowsAffected(); n != 1 {
			c.JSON(409, gin.H{"code": 409, "message": "项目状态已变化，请刷新后重试"})
			return
		}
		if in.Action == "close" {
			if _, e = tx.Exec(`UPDATE vopc_projects SET completed_at=CURRENT_TIMESTAMP WHERE id=?`, id); e != nil {
				serverError(c, "结项失败")
				return
			}
		}
	} else if in.Action == "pause" || in.Action == "resume" {
		res, e := tx.Exec(`UPDATE vopc_projects SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status=?`, next, id, status)
		if e != nil {
			serverError(c, "状态流转失败")
			return
		}
		if n, _ := res.RowsAffected(); n != 1 {
			c.JSON(409, gin.H{"code": 409, "message": "项目状态已变化，请刷新后重试"})
			return
		}
	}

	if _, err = tx.Exec(`INSERT INTO vopc_close_records(project_id,action,reason,failure_evidence,outcome_package,human_decision,decided_by,previous_status,new_status) VALUES(?,?,?,?,?,?,?,?,?)`, id, in.Action, in.Reason, in.FailureEvidence, in.OutcomePackage, in.HumanDecision, u.UserID, status, next); err != nil {
		serverError(c, "状态记录写入失败")
		return
	}
	if err = writeEvent(tx, id, u.UserID, "project."+in.Action, status, next, in.Reason); err != nil {
		serverError(c, "状态审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "状态流转失败")
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": id, "status": next, "stage": stageForStatus(next, stage)}})
}

// stageForStatus 返回动作后的展示阶段；draft 回到 G0，closeable/completed 保持 G4。
func stageForStatus(status, curStage string) string {
	if status == "draft" {
		return "G0"
	}
	if status == "closeable" || status == "completed" {
		return "G4"
	}
	return curStage
}

// closeTransition 返回 目标状态 与 是否合法。
func closeTransition(from, action string) (string, bool) {
	switch action {
	case "close":
		if from != statusCloseable {
			return "", false
		}
		return "completed", true
	case "pause":
		if from == "draft" || completedLike[from] || from == "terminated" || from == "archived" || from == "paused" {
			return "", false
		}
		return "paused", true
	case "resume":
		if from != "paused" {
			return "", false
		}
		// 实际目标状态由调用方读取 pause 记录的 previous_status 决定。
		return "pending_review", true
	case "pivot":
		if from == "completed" || from == "terminated" || from == "archived" {
			return "", false
		}
		return "draft", true
	case "terminate":
		if from == "completed" || from == "archived" || from == "terminated" {
			return "", false
		}
		return "terminated", true
	case "archive":
		// 允许归档：completed / terminated（原有）以及 draft 之外的常规可操作状态（收尾留存项目）。
		// 排除：archived（已归档）、completed/terminated 之外的所有终态由上面分支命中；
		// draft 草稿不入归档（草稿可删除）——但若用户选择归档也放行，便于统一入口。
		if from == "archived" {
			return "", false
		}
		return "archived", true
	}
	return "", false
}

// ListCloseRecords 只读返回项目结项/异常状态记录。
func (h *VOPCHandler) ListCloseRecords(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok := h.readableProject(c, id)
	if !ok {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,action,reason,failure_evidence,outcome_package,human_decision,decided_by,previous_status,new_status,created_at FROM vopc_close_records WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "状态记录读取失败")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var rid, by int64
		var action, reason, fe, op, hd, prev, next, created string
		if rows.Scan(&rid, &action, &reason, &fe, &op, &hd, &by, &prev, &next, &created) != nil {
			serverError(c, "状态记录读取失败")
			return
		}
		items = append(items, gin.H{"id": rid, "action": action, "reason": reason, "failure_evidence": fe, "outcome_package": op, "human_decision": hd, "decided_by": by, "previous_status": prev, "new_status": next, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// DeleteProject 硬删除一个 S0 草稿项目并级联清理全部关联数据（数据链路贯通）。
// 权限：VOPCProjectManage（主理人/联合主理人，projectPolicy manage）。
// 边界：仅允许删除 draft 草稿；已提交/进行/终止/归档项目禁止删除（数据沉淀，请走 archive 归档）。
// 依赖 ON DELETE CASCADE：生产连接已启用 foreign_keys(on)，子表全链路级联。
func (h *VOPCHandler) DeleteProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "项目删除失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var owner int64
	var status string
	if err := tx.QueryRow(`SELECT owner_user_id,status FROM vopc_projects WHERE id=?`, id).Scan(&owner, &status); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	} else if err != nil {
		serverError(c, "项目读取失败")
		return
	}
	allowed, e := projectPolicy(tx, id, u.UserID, owner, "manage")
	if e != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	if status != "draft" {
		c.JSON(409, gin.H{"code": 409, "message": "仅 G0 草稿项目可删除；已提交或已归档项目请使用归档或结项"})
		return
	}
	// 删除前留痕（写全局概念：draft 项目删除本身即终态，先写一条 close_records 不可行——
	// 该表随项目级联；此处直接清理。如需删除审计，应接入全局 audit 通道，本期记录日志即可。）
	if _, err := tx.Exec(`DELETE FROM vopc_projects WHERE id=? AND status='draft'`, id); err != nil {
		serverError(c, "项目删除失败")
		return
	}
	// 显式清理磁盘私有文件目录（DB 行走级联；磁盘对象按 uploadDir/<projectID> 清理）。
	// 注意：不能用 resolveUploadDir（它会重新 mkdir）；这里纯路径拼接后删除。
	if dir, derr := h.projectUploadDirPath(id); derr == nil {
		_ = os.RemoveAll(dir)
	}
	if err := tx.Commit(); err != nil {
		serverError(c, "项目删除失败")
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": id, "deleted": true}})
}

// projectUploadDirPath 返回项目私有文件磁盘目录路径（纯拼接，不创建）。
func (h *VOPCHandler) projectUploadDirPath(projectID int64) (string, error) {
	root := h.uploadDir
	if root == "" {
		root = ".uploads/vopc"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(abs, strconv.FormatInt(projectID, 10)), nil
}
