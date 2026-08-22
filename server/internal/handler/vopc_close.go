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
//   - close：仅 closeable → completed
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

	// pivot 回到 S0 草案重定向：阶段回退、里程碑复位。
	if in.Action == "pivot" {
		res, e := tx.Exec(`UPDATE vopc_projects SET stage='S0',status=?,submitted_at=NULL,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status=?`, next, id, status)
		if e != nil {
			serverError(c, "转向失败")
			return
		}
		if n, _ := res.RowsAffected(); n != 1 {
			c.JSON(409, gin.H{"code": 409, "message": "项目状态已变化，请刷新后重试"})
			return
		}
		if _, e = tx.Exec(`UPDATE vopc_milestones SET status='pending',review_note='' WHERE project_id=? AND stage<>'S0'`, id); e != nil {
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

// stageForStatus 返回动作后的展示阶段；closeable/completed 保持 S9。
func stageForStatus(status, curStage string) string {
	if status == "draft" {
		return "S0"
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
		if from != "completed" && from != "terminated" {
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
