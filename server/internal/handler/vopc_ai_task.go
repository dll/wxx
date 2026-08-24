package handler

// vOPC 虚拟向导（v2.0 重构，替代 v1.0 B1 AI 任务真实执行）。
//
// v2.0 核心：虚拟向导以「流程角色 + 模板 + 清单 + 角色扮演提示」的模板化形式，基于项目类型与
// 用户指令在本地渲染出一份结构化草稿，**不调用任何真实 LLM**。草稿由主理人审阅（accept/revise/
// reject/overrule 四态），并保留版次（revision）与修改率统计，供教学统计与可追溯。
//
// 与 v1.0 的关系：
//   - 保留 vopc_ai_tasks / vopc_ai_quotas 表结构（历史迁移 106 不回退）：output_content 承载模板
//     渲染的草稿；prompt_tokens/output_tokens 不再真实累计（保持 0）；provider 固定 "template"；
//     model 固定 "virtual_guide"。
//   - 审阅四态（accept/revise/reject/overrule）+ revision + 修改率 沿用原语义，仅生成端从 LLM 换为模板。

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// aiDecisions 人工审阅合法动作（沿用：接受/修改/退回/否决）。
var aiDecisions = setOf("accept", "revise", "reject", "overrule")

// virtualGuideModel 虚拟向导的固定 model/provider 标记（非真实模型）。
const (
	virtualGuideModel    = "virtual_guide"
	virtualGuideProvider = "template"
)

// virtualGuideTemplates 按岗位 key 提供的模板化引导：职责、清单、角色扮演提示、草稿骨架。
// 仅本地规则/模板渲染，无外部模型依赖。
var virtualGuideTemplates = map[string]struct {
	Name           string
	Responsibility string
	Checklist      []string
	Roleplay       string
}{
	"project_manager": {
		Name:           "产品经理向导",
		Responsibility: "负责把想法拆成可验证的目标与范围，确保每一步都有清晰交付口径。",
		Checklist:      []string{"一句话说清要解决的问题", "明确目标用户与价值假设", "列出本阶段要做与不做的事"},
		Roleplay:       "你是独立产品负责人：追问「为什么做、为谁做、做到什么程度算完成」。",
	},
	"market_user": {
		Name:           "市场与用户向导",
		Responsibility: "负责用户调研、需求验证与反馈采集，帮主理人判断想法是否成立。",
		Checklist:      []string{"识别目标用户与使用场景", "设计最小验证方式（访谈/问卷/模拟反馈）", "记录反馈并归纳关键假设"},
		Roleplay:       "你是假想的目标用户与市场观察者：站在用户角度挑战价值假设。",
	},
	"product_solution": {
		Name:           "产品与方案向导",
		Responsibility: "负责方案设计、技术选型与产出形态，帮主理人把想法落地为结构化产出。",
		Checklist:      []string{"定义产品/成果形态", "给出方案要点与关键取舍", "约定验收标准与不做清单"},
		Roleplay:       "你是方案架构师：把模糊想法收敛为可执行的方案骨架。",
	},
	"execution": {
		Name:           "交付执行向导",
		Responsibility: "负责拆任务、排版本与组织交付，帮主理人把产出推进为可审阅的成果。",
		Checklist:      []string{"拆解交付物与任务", "约定版本与里程碑", "整理交付说明与复盘要点"},
		Roleplay:       "你是执行负责人：把方案拆成一个个可完成、可验收的任务。",
	},
}

// renderVirtualGuideDraft 基于岗位模板 + 用户指令渲染结构化草稿（本地规则，不调模型）。
func renderVirtualGuideDraft(roleKey, instruction, projectType string) string {
	tpl, ok := virtualGuideTemplates[roleKey]
	if !ok {
		tpl = virtualGuideTemplates["project_manager"]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "【虚拟向导 · %s】\n\n", tpl.Name)
	fmt.Fprintf(&b, "▸ 职责：%s\n\n", tpl.Responsibility)
	b.WriteString("▸ 引导清单：\n")
	for i, item := range tpl.Checklist {
		fmt.Fprintf(&b, "   %d. %s\n", i+1, item)
	}
	b.WriteString("\n▸ 角色扮演提示：\n")
	fmt.Fprintf(&b, "   %s\n\n", tpl.Roleplay)
	if projectType != "" {
		fmt.Fprintf(&b, "▸ 项目类型：%s\n\n", projectType)
	}
	if strings.TrimSpace(instruction) != "" {
		fmt.Fprintf(&b, "▸ 你的任务指令：\n   %s\n\n", instruction)
	}
	b.WriteString("▸ 结构化草稿（待你审阅/修改/接受/退回）：\n")
	b.WriteString("   1. 目标：\n   2. 关键交付物：\n   3. 验收标准：\n   4. 本阶段不做清单：\n   5. 下一步行动：\n")
	return b.String()
}

// CreateAITask 基于虚拟向导模板生成结构化草稿（不再调用真实 LLM）。
func (h *VOPCHandler) CreateAITask(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	var in struct {
		RoleKey     string `json:"role_key"`
		Instruction string `json:"instruction"`
		MaxTokens   int    `json:"max_tokens"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.RoleKey = strings.TrimSpace(in.RoleKey)
	in.Instruction = strings.TrimSpace(in.Instruction)
	if in.RoleKey == "" || in.Instruction == "" {
		c.JSON(422, gin.H{"code": 422, "message": "虚拟向导岗位与任务指令必填"})
		return
	}
	if utf8RuneCount(in.Instruction) > 4000 {
		c.JSON(422, gin.H{"code": 422, "message": "任务指令不超过 4000 字"})
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
	var roleName string
	err := tx.QueryRow(`SELECT role_name FROM vopc_ai_roles WHERE project_id=? AND role_key=? AND enabled=1`, id, in.RoleKey).Scan(&roleName)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(422, gin.H{"code": 422, "message": "虚拟向导岗位不存在或未启用"})
		return
	} else if err != nil {
		serverError(c, "虚拟向导岗位读取失败")
		return
	}
	var projectType string
	_ = tx.QueryRow(`SELECT project_type FROM vopc_projects WHERE id=?`, id).Scan(&projectType)

	draft := renderVirtualGuideDraft(in.RoleKey, in.Instruction, projectType)

	res, err := tx.Exec(`INSERT INTO vopc_ai_tasks(project_id,role_key,instruction,provider,model,status,output_content,max_tokens,created_by) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, in.RoleKey, in.Instruction, virtualGuideProvider, virtualGuideModel, "succeeded", draft, 0, u.UserID)
	if err != nil {
		serverError(c, "虚拟向导草稿创建失败")
		return
	}
	taskID, _ := res.LastInsertId()
	if writeEvent(tx, id, u.UserID, "virtual_guide.generated", "", "succeeded", fmt.Sprintf("虚拟向导任务 #%d（%s）", taskID, roleName)) != nil {
		serverError(c, "虚拟向导审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "虚拟向导草稿创建失败")
		return
	}
	c.JSON(201, gin.H{"code": 0, "data": h.getAITask(id, taskID)})
}

// ListAITasks 项目虚拟向导任务列表（读权限）。
func (h *VOPCHandler) ListAITasks(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tx, _, ok2 := h.readableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,role_key,model,status,output_content,prompt_tokens,output_tokens,duration_ms,retry_count,error_msg,final_decision,decision_by,decision_note,created_by,created_at,revision FROM vopc_ai_tasks WHERE project_id=? ORDER BY id DESC`, id)
	if err != nil {
		serverError(c, "虚拟向导任务列表读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var tid int64
		var rk, model, status, out, errMsg, dnote, revision string
		var pt, ot, retry, createdBy int64
		var dur int64
		var dec sql.NullString
		var decBy sql.NullInt64
		var createdAt any
		if rows.Scan(&tid, &rk, &model, &status, &out, &pt, &ot, &dur, &retry, &errMsg, &dec, &decBy, &dnote, &createdBy, &createdAt, &revision) != nil {
			continue
		}
		items = append(items, gin.H{
			"id": tid, "role_key": rk, "model": model, "status": status,
			"output_content": out, "prompt_tokens": pt, "output_tokens": ot, "duration_ms": dur,
			"retry_count": retry, "error_msg": errMsg,
			"final_decision": nullableStringPtr(dec), "decision_by": nullableIntPtr(decBy),
			"decision_note": dnote, "revision": revision, "created_by": createdBy, "created_at": anyString(createdAt),
			"modification_rate": modificationRate(revision),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// GetAITask 单个虚拟向导任务详情（读权限）。
func (h *VOPCHandler) GetAITask(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tid, e := parsePositiveID(c.Param("taskId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "任务 ID 无效"})
		return
	}
	tx, _, ok2 := h.readableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	var pid int64
	if err := tx.QueryRow(`SELECT project_id FROM vopc_ai_tasks WHERE id=?`, tid).Scan(&pid); err != nil || pid != id {
		c.JSON(404, gin.H{"code": 404, "message": "任务不存在或无权访问"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.getAITask(id, tid)})
}

// ReviewAITask 主理人/联合主理人（manage）对虚拟向导草稿审阅：accept/revise/reject/overrule。
func (h *VOPCHandler) ReviewAITask(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tid, e := parsePositiveID(c.Param("taskId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "任务 ID 无效"})
		return
	}
	u := middleware.GetUserContext(c)
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
		Revision string `json:"revision"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Decision = strings.TrimSpace(in.Decision)
	in.Note = strings.TrimSpace(in.Note)
	in.Revision = strings.TrimSpace(in.Revision)
	if !aiDecisions[in.Decision] {
		c.JSON(422, gin.H{"code": 422, "message": "审阅动作必须为 accept/revise/reject/overrule"})
		return
	}
	if in.Decision == "revise" && in.Revision == "" {
		c.JSON(422, gin.H{"code": 422, "message": "修改需提供修订指示"})
		return
	}
	tx, _, ok2 := h.manageableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRow(`SELECT status FROM vopc_ai_tasks WHERE id=? AND project_id=?`, tid, id).Scan(&status); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "任务不存在或无权操作"})
		return
	}
	if status != "succeeded" {
		c.JSON(409, gin.H{"code": 409, "message": "仅可审阅已生成的虚拟向导草稿"})
		return
	}
	var already sql.NullString
	if err := tx.QueryRow(`SELECT final_decision FROM vopc_ai_tasks WHERE id=? AND project_id=?`, tid, id).Scan(&already); err != nil {
		serverError(c, "审阅读取失败")
		return
	}
	if already.Valid {
		c.JSON(409, gin.H{"code": 409, "message": "该任务已审阅"})
		return
	}
	res, err := tx.Exec(`UPDATE vopc_ai_tasks SET final_decision=?,decision_by=?,decision_note=?,revision=?,decided_at=CURRENT_TIMESTAMP WHERE id=? AND status='succeeded' AND final_decision IS NULL`,
		in.Decision, u.UserID, in.Note, in.Revision, tid)
	if err != nil {
		serverError(c, "审阅保存失败")
		return
	}
	if n, rerr := res.RowsAffected(); rerr != nil || n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "该任务已审阅"})
		return
	}
	if writeEvent(tx, id, u.UserID, "virtual_guide.reviewed", "succeeded", "reviewed", fmt.Sprintf("虚拟向导任务 #%d %s", tid, in.Decision)) != nil {
		serverError(c, "审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "审阅保存失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"id": tid, "final_decision": in.Decision}})
}

// RetryAITask 对虚拟向导草稿重新生成（模板重渲染，不调模型）。
func (h *VOPCHandler) RetryAITask(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	tid, e := parsePositiveID(c.Param("taskId"))
	if e != nil {
		c.JSON(400, gin.H{"code": 400, "message": "任务 ID 无效"})
		return
	}
	tx, _, ok2 := h.manageableProject(c, id)
	if !ok2 {
		return
	}
	defer tx.Rollback()
	var status string
	var retry int
	if err := tx.QueryRow(`SELECT status,retry_count FROM vopc_ai_tasks WHERE id=? AND project_id=?`, tid, id).Scan(&status, &retry); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "任务不存在或无权操作"})
		return
	}
	if status == "succeeded" {
		c.JSON(409, gin.H{"code": 409, "message": "已生成的草稿无需重试"})
		return
	}
	if _, err := tx.Exec(`UPDATE vopc_ai_tasks SET status='succeeded',retry_count=retry_count+1,error_msg='',updated_at=CURRENT_TIMESTAMP WHERE id=?`, tid); err != nil {
		serverError(c, "重试失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "重试失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.getAITask(id, tid)})
}

// getAITask 读取某任务当前记录返回 gin.H（含修改率）。
func (h *VOPCHandler) getAITask(projectID, taskID int64) gin.H {
	var t struct {
		ID             int64
		RoleKey, Model string
		Status         string
		Output         string
		PromptT, OutT  int
		Dur            int64
		Retry          int
		Err            string
		Decision       sql.NullString
		DecisionBy     sql.NullInt64
		DecisionNote   string
		Revision       string
		CreatedBy      int64
		CreatedAt      any
	}
	err := h.db.QueryRow(`SELECT id,role_key,model,status,output_content,prompt_tokens,output_tokens,duration_ms,retry_count,error_msg,final_decision,decision_by,decision_note,revision,created_by,created_at FROM vopc_ai_tasks WHERE id=? AND project_id=?`, taskID, projectID).
		Scan(&t.ID, &t.RoleKey, &t.Model, &t.Status, &t.Output, &t.PromptT, &t.OutT, &t.Dur, &t.Retry, &t.Err, &t.Decision, &t.DecisionBy, &t.DecisionNote, &t.Revision, &t.CreatedBy, &t.CreatedAt)
	if err != nil {
		return gin.H{"id": taskID}
	}
	return gin.H{
		"id": t.ID, "role_key": t.RoleKey, "model": t.Model, "status": t.Status,
		"output_content": t.Output,
		"prompt_tokens":  t.PromptT, "output_tokens": t.OutT, "duration_ms": t.Dur,
		"retry_count": t.Retry, "error_msg": t.Err,
		"final_decision": nullableStringPtr(t.Decision), "decision_by": nullableIntPtr(t.DecisionBy),
		"decision_note": t.DecisionNote, "revision": t.Revision,
		"modification_rate": modificationRate(t.Revision),
		"created_by":        t.CreatedBy, "created_at": anyString(t.CreatedAt),
	}
}

// modificationRate 根据 revision 的改动指示估算修改率（教学统计指标）：0=未修改，修订指示越长改动越大。
func modificationRate(revision string) float64 {
	rev := strings.TrimSpace(revision)
	if rev == "" {
		return 0
	}
	lines := strings.Split(rev, "\n")
	score := 0.0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			score += 1.0
		}
	}
	if score > 10 {
		score = 10
	}
	return score / 10.0
}

// utf8RuneCount 计算字符串 rune 数。
func utf8RuneCount(s string) int {
	return len([]rune(s))
}

// nullableStringPtr / nullableIntPtr 兼容 SQLite NULL 扫描。
func nullableStringPtr(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}
func nullableIntPtr(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
