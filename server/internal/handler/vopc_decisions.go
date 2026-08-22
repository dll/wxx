package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

var decisionActions = setOf("resolve", "cancel")

// ListDecisions 返回当前用户可见项目的决策，不允许通过决策 ID 跨项目读取。
func (h *VOPCHandler) ListDecisions(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "决策列表读取失败")
		return
	}
	defer tx.Rollback()
	var owner int64
	if err = tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		return
	} else if err != nil {
		serverError(c, "决策列表读取失败")
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "read")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		return
	}
	rows, err := tx.Query(`SELECT id,title,background,options,decision,rationale,status,decided_by,created_at,decided_at FROM vopc_decisions WHERE project_id=? ORDER BY created_at DESC,id DESC`, id)
	if err != nil {
		serverError(c, "决策列表读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var did int64
		var title, background, options, decision, rationale, status, created string
		var by sql.NullInt64
		var at sql.NullString
		if err := rows.Scan(&did, &title, &background, &options, &decision, &rationale, &status, &by, &created, &at); err != nil {
			serverError(c, "决策列表读取失败")
			return
		}
		item := gin.H{"id": did, "title": title, "background": background, "options": options, "decision": decision, "rationale": rationale, "status": status, "created_at": created}
		if by.Valid {
			item["decided_by"] = by.Int64
		}
		if at.Valid {
			item["decided_at"] = at.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		serverError(c, "决策列表读取失败")
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}

type decisionInput struct {
	Title      string `json:"title"`
	Background string `json:"background"`
	Options    string `json:"options"`
	Decision   string `json:"decision"`
	Rationale  string `json:"rationale"`
}

func (h *VOPCHandler) CreateDecision(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	var in decisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Background = strings.TrimSpace(in.Background)
	in.Options = strings.TrimSpace(in.Options)
	in.Decision = strings.TrimSpace(in.Decision)
	in.Rationale = strings.TrimSpace(in.Rationale)
	if in.Title == "" || in.Decision == "" {
		c.JSON(422, gin.H{"code": 422, "message": "决策标题和决定内容必填"})
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "决策创建失败")
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
	var stage string
	if err = tx.QueryRow(`SELECT owner_user_id,status,stage FROM vopc_projects WHERE id=?`, id).Scan(&owner, &status, &stage); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	} else if err != nil {
		serverError(c, "决策创建失败")
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "manage")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	if blockedStatuses[status] || completedLike[status] {
		c.JSON(409, gin.H{"code": 409, "message": "当前项目状态禁止创建决策"})
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_decisions(project_id,title,background,options,decision,rationale,status) VALUES(?,?,?,?,?,?,?)`, id, in.Title, in.Background, in.Options, in.Decision, in.Rationale, "pending")
	if err != nil {
		serverError(c, "决策创建失败")
		return
	}
	did, err := res.LastInsertId()
	if err != nil {
		serverError(c, "决策创建失败")
		return
	}
	if err = writeEvent(tx, id, u.UserID, "decision.created", "", "pending", fmt.Sprintf("决策 #%d：%s", did, in.Title)); err != nil {
		serverError(c, "决策审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "决策创建失败")
		return
	}
	committed = true
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": did, "status": "pending", "stage": stage}})
}

func (h *VOPCHandler) ActDecision(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	did, err := parsePositiveID(c.Param("decisionId"))
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "决策 ID 无效"})
		return
	}
	var body struct {
		Action    string `json:"action"`
		Decision  string `json:"decision"`
		Rationale string `json:"rationale"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	body.Action = strings.TrimSpace(body.Action)
	body.Decision = strings.TrimSpace(body.Decision)
	body.Rationale = strings.TrimSpace(body.Rationale)
	if !decisionActions[body.Action] {
		c.JSON(422, gin.H{"code": 422, "message": "决策动作仅支持 resolve 或 cancel"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "决策处理失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var owner int64
	var projectStatus, current, title string
	if err = tx.QueryRow(`SELECT p.owner_user_id,p.status,d.status,d.title FROM vopc_decisions d JOIN vopc_projects p ON p.id=d.project_id WHERE d.id=? AND d.project_id=?`, did, id).Scan(&owner, &projectStatus, &current, &title); errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "决策不存在或无权操作"})
		return
	} else if err != nil {
		serverError(c, "决策处理失败")
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "manage")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "决策不存在或无权操作"})
		return
	}
	if blockedStatuses[projectStatus] || completedLike[projectStatus] {
		c.JSON(409, gin.H{"code": 409, "message": "当前项目状态禁止处理决策"})
		return
	}
	if current != "pending" {
		c.JSON(409, gin.H{"code": 409, "message": "仅 pending 决策可处理"})
		return
	}
	next := "resolved"
	if body.Action == "cancel" {
		next = "cancelled"
	}
	decision := body.Decision
	if decision == "" && next == "resolved" {
		decision = title
	}
	res, err := tx.Exec(`UPDATE vopc_decisions SET status=?,decision=CASE WHEN ?='' THEN decision ELSE ? END,rationale=CASE WHEN ?='' THEN rationale ELSE ? END,decided_by=?,decided_at=CURRENT_TIMESTAMP WHERE id=? AND project_id=? AND status='pending'`, next, decision, decision, body.Rationale, body.Rationale, u.UserID, did, id)
	if err != nil {
		serverError(c, "决策处理失败")
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "决策状态已变化，请刷新后重试"})
		return
	}
	if err = writeEvent(tx, id, u.UserID, "decision.status_changed", current, next, fmt.Sprintf("决策 #%d", did)); err != nil {
		serverError(c, "决策审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "决策处理失败")
		return
	}
	committed = true
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"id": did, "status": next, "decided_by": u.UserID}})
}

func parsePositiveID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscan(s, &id)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid")
	}
	return id, nil
}
