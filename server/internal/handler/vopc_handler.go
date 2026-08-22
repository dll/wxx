package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

var (
	projectTypes    = setOf("软件与 AI 产品", "内容与知识产品", "校园服务创新", "创新创业项目", "科研与技术实验", "公益与社会实践", "教学改革项目", "自由探索项目")
	riskLevels      = setOf("R0", "R1", "R2", "R3")
	dataTypes       = setOf("公开数据", "校内非敏感数据", "个人数据", "敏感个人数据", "学籍数据", "心理健康数据", "医疗健康数据")
	blockedStatuses = setOf("paused", "risk_frozen", "terminated", "archived")
	// stageStatuses[9] 不再是 completed：S9 里程碑通过后项目进入 closeable（可结项）
	// 状态，必须由项目管理角色发起 close 才落为 completed。
	stageStatuses = []string{"draft", "pending_review", "company_formed", "requirement_baselined", "solution_approved", "developing", "testing", "production", "operating", "closeable"}
	// statusCloseable 表示 S9 已通过、等待人工结项决策。
	statusCloseable = "closeable"
	// completedLike 表示 S9 已通过或已结项，禁止再创建任务/决策、推进里程碑。
	completedLike = setOf("completed", "closeable")
	// closeActions 是结项/异常状态机的合法动作。
	closeActions = setOf("close", "pause", "resume", "pivot", "terminate", "archive")

	taskPriorities = setOf("low", "normal", "high", "urgent")
	taskStatuses   = setOf("todo", "in_progress", "review", "done", "cancelled")
)

func setOf(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

// VOPCHandler 提供 vOPC P0 项目、成员和服务端阶段门禁。
type VOPCHandler struct {
	db        *sql.DB
	collegeID string
}

func NewVOPCHandler(db *sql.DB, collegeID ...string) *VOPCHandler {
	id := "cs"
	if len(collegeID) > 0 && strings.TrimSpace(collegeID[0]) != "" {
		id = strings.TrimSpace(collegeID[0])
	}
	return &VOPCHandler{db: db, collegeID: id}
}

func CollegeAccess(collegeID ...string) gin.HandlerFunc {
	id := "cs"
	if len(collegeID) > 0 && strings.TrimSpace(collegeID[0]) != "" {
		id = strings.TrimSpace(collegeID[0])
	}
	return func(c *gin.Context) {
		u := middleware.GetUserContext(c)
		if u == nil || u.Status != "active" || u.Role == "guest" || u.OwnerScope != "college" || !strings.EqualFold(u.OwnerID, id) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅计算机学院已授权用户可使用 vOPC"})
			return
		}
		c.Next()
	}
}

// UpdateProject saves an editable S0 draft. Once submitted, baseline changes
// must go through the milestone/change process rather than silently mutating it.
func (h *VOPCHandler) UpdateProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in projectInput
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	if msg, code := in.normalizeAndValidate(false); code != 0 {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "草稿保存失败")
		return
	}
	done := false
	defer func() {
		if !done {
			_ = tx.Rollback()
		}
	}()
	var owner int64
	var stage, status string
	if tx.QueryRow(`SELECT owner_user_id,stage,status FROM vopc_projects WHERE id=?`, id).Scan(&owner, &stage, &status) != nil {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	allowed, e := projectPolicy(tx, id, u.UserID, owner, "manage")
	if e != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	if stage != "S0" || status != "draft" {
		c.JSON(409, gin.H{"code": 409, "message": "仅 S0 草稿可直接编辑，已提交项目请走变更评审"})
		return
	}
	res, err := tx.Exec(`UPDATE vopc_projects SET name=?,summary=?,problem_statement=?,target_users=?,expected_outcome=?,validation_plan=?,project_type=?,project_source=?,product_form=?,project_cycle=?,acceptance_criteria=?,mentor_needs=?,resource_needs=?,risk_level=?,data_type=?,real_user_trial=?,external_publish=?,funds_involved=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND stage='S0' AND status='draft'`, in.Name, in.Summary, in.Problem, in.Target, in.Outcome, in.Validation, in.Type, in.Source, in.ProductForm, in.Cycle, in.Acceptance, in.MentorNeeds, in.ResourceNeeds, in.Risk, in.DataType, in.RealTrial, in.ExternalPublish, in.FundsInvolved, id)
	if err != nil {
		serverError(c, "草稿保存失败")
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "草稿已变化，请刷新重试"})
		return
	}
	if writeEvent(tx, id, u.UserID, "project.draft_updated", "S0/draft", "S0/draft", "更新并补齐项目草稿") != nil {
		serverError(c, "草稿审计写入失败")
		return
	}
	if tx.Commit() != nil {
		serverError(c, "草稿保存失败")
		return
	}
	done = true
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"id": id, "risk_level": in.Risk}})
}

func (h *VOPCHandler) AccessStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"allowed": true, "college_id": h.collegeID}})
}

func (h *VOPCHandler) ListProjects(c *gin.Context) {
	u := middleware.GetUserContext(c)
	rows, err := h.db.Query(`SELECT p.id,p.name,p.summary,p.project_type,p.stage,p.status,p.visibility,p.risk_level,p.owner_user_id,p.updated_at FROM vopc_projects p WHERE p.owner_user_id=? OR EXISTS (SELECT 1 FROM vopc_project_members m WHERE m.project_id=p.id AND m.user_id=? AND m.status='active') ORDER BY p.updated_at DESC LIMIT 100`, u.UserID, u.UserID)
	if err != nil {
		serverError(c, "项目列表读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var id, owner int64
		var name, summary, typ, stage, status, visibility, risk, updated string
		if err := rows.Scan(&id, &name, &summary, &typ, &stage, &status, &visibility, &risk, &owner, &updated); err != nil {
			serverError(c, "项目列表读取失败")
			return
		}
		items = append(items, gin.H{"id": id, "name": name, "summary": summary, "project_type": typ, "stage": stage, "status": status, "visibility": visibility, "risk_level": risk, "owner_user_id": owner, "updated_at": updated})
	}
	if err := rows.Err(); err != nil {
		serverError(c, "项目列表读取失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

type projectInput struct {
	Name            string `json:"name"`
	Summary         string `json:"summary"`
	Problem         string `json:"problem_statement"`
	Target          string `json:"target_users"`
	Outcome         string `json:"expected_outcome"`
	Validation      string `json:"validation_plan"`
	Type            string `json:"project_type"`
	Source          string `json:"project_source"`
	ProductForm     string `json:"product_form"`
	Cycle           string `json:"project_cycle"`
	Acceptance      string `json:"acceptance_criteria"`
	MentorNeeds     string `json:"mentor_needs"`
	ResourceNeeds   string `json:"resource_needs"`
	Risk            string `json:"risk_level"`
	DataType        string `json:"data_type"`
	RealTrial       bool   `json:"real_user_trial"`
	ExternalPublish bool   `json:"external_publish"`
	FundsInvolved   bool   `json:"funds_involved"`
}

func (in *projectInput) normalizeAndValidate(submit bool) (string, int) {
	trim := func(s *string) { *s = strings.TrimSpace(*s) }
	for _, p := range []*string{&in.Name, &in.Summary, &in.Problem, &in.Target, &in.Outcome, &in.Validation, &in.Type, &in.Source, &in.ProductForm, &in.Cycle, &in.Acceptance, &in.MentorNeeds, &in.ResourceNeeds, &in.Risk, &in.DataType} {
		trim(p)
	}
	if in.Type == "" {
		in.Type = "自由探索项目"
	}
	if in.Risk == "" {
		in.Risk = "R0"
	}
	if in.DataType == "" {
		in.DataType = "公开数据"
	}
	if in.Source == "" {
		in.Source = "self_proposed"
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 100 {
		return "项目名称必填且不超过 100 字", 422
	}
	for label, value := range map[string]string{"项目摘要": in.Summary, "问题陈述": in.Problem, "目标用户": in.Target, "预期成果": in.Outcome, "验证计划": in.Validation, "验收标准": in.Acceptance, "产品形态": in.ProductForm, "项目周期": in.Cycle} {
		if utf8.RuneCountInString(value) > 4000 {
			return label + "超过 4000 字", 422
		}
	}
	if !projectTypes[in.Type] || !riskLevels[in.Risk] || !dataTypes[in.DataType] {
		return "项目类型、风险等级或数据类型不在允许范围", 422
	}
	if in.Source != "self_proposed" && in.Source != "client_requirement" {
		return "未知项目来源", 422
	}
	requiredRisk := "R0"
	if in.RealTrial || in.ExternalPublish || in.FundsInvolved || in.DataType != "公开数据" {
		requiredRisk = "R2"
	}
	if in.DataType == "敏感个人数据" || in.DataType == "心理健康数据" || in.DataType == "医疗健康数据" || in.FundsInvolved {
		requiredRisk = "R3"
	}
	if in.Risk < requiredRisk {
		in.Risk = requiredRisk
	}
	if in.Risk == "R3" {
		return "R3 项目默认拒绝，需线下专项审批后由平台运营录入", 422
	}
	if submit {
		for label, value := range map[string]string{"项目摘要": in.Summary, "问题陈述": in.Problem, "目标用户": in.Target, "预期成果": in.Outcome, "验证计划": in.Validation, "产品形态": in.ProductForm, "项目周期": in.Cycle, "验收标准": in.Acceptance} {
			if value == "" {
				return "提交前需补齐" + label, 422
			}
		}
	}
	return "", 0
}

func (h *VOPCHandler) CreateProject(c *gin.Context) {
	u := middleware.GetUserContext(c)
	var in projectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	if msg, code := in.normalizeAndValidate(false); code != 0 {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "项目保存失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.Exec(`INSERT INTO vopc_projects(name,summary,problem_statement,target_users,expected_outcome,validation_plan,project_type,project_source,product_form,project_cycle,acceptance_criteria,mentor_needs,resource_needs,risk_level,data_type,real_user_trial,external_publish,funds_involved,owner_user_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, in.Name, in.Summary, in.Problem, in.Target, in.Outcome, in.Validation, in.Type, in.Source, in.ProductForm, in.Cycle, in.Acceptance, in.MentorNeeds, in.ResourceNeeds, in.Risk, in.DataType, in.RealTrial, in.ExternalPublish, in.FundsInvolved, u.UserID)
	if err != nil {
		serverError(c, "项目保存失败")
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		serverError(c, "项目保存失败")
		return
	}
	if err = execOne(tx, `INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, id, u.UserID, "owner"); err != nil {
		serverError(c, "项目成员初始化失败")
		return
	}
	roles := []struct{ k, n string }{{"project_manager", "项目经理 Bot"}, {"market_user", "市场与用户 Bot"}, {"product_solution", "产品与方案 Bot"}, {"execution", "执行专家 Bot"}}
	for _, r := range roles {
		if err = execOne(tx, `INSERT INTO vopc_ai_roles(project_id,role_key,role_name) VALUES(?,?,?)`, id, r.k, r.n); err != nil {
			serverError(c, "AI 岗位初始化失败")
			return
		}
	}
	for i := 0; i < 10; i++ {
		if err = execOne(tx, `INSERT INTO vopc_milestones(project_id,stage,required_evidence) VALUES(?,?,?)`, id, fmt.Sprintf("S%d", i), milestoneEvidence(i)); err != nil {
			serverError(c, "里程碑初始化失败")
			return
		}
	}
	if err = writeEvent(tx, id, u.UserID, "project.created", "", "S0/draft", "创建项目草稿"); err != nil {
		serverError(c, "项目审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "项目保存失败")
		return
	}
	committed = true
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": id, "stage": "S0", "status": "draft", "risk_level": in.Risk}})
}

func execOne(tx *sql.Tx, q string, args ...any) error {
	res, err := tx.Exec(q, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("affected rows %d", n)
	}
	return nil
}
func writeEvent(tx *sql.Tx, id, actor int64, action, before, after, detail string) error {
	payload, _ := json.Marshal(gin.H{"before": before, "after": after, "detail": detail})
	return execOne(tx, `INSERT INTO vopc_events(project_id,actor_user_id,action,detail) VALUES(?,?,?,?)`, id, actor, action, string(payload))
}

func (h *VOPCHandler) GetProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	var p projectInput
	var stage, status, visibility, created, updated string
	var pid, owner int64
	err := h.db.QueryRow(`SELECT id,name,summary,problem_statement,target_users,expected_outcome,validation_plan,project_type,project_source,product_form,project_cycle,acceptance_criteria,mentor_needs,resource_needs,stage,status,visibility,risk_level,data_type,real_user_trial,external_publish,funds_involved,owner_user_id,created_at,updated_at FROM vopc_projects WHERE id=? AND (owner_user_id=? OR EXISTS (SELECT 1 FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'))`, id, u.UserID, id, u.UserID).Scan(&pid, &p.Name, &p.Summary, &p.Problem, &p.Target, &p.Outcome, &p.Validation, &p.Type, &p.Source, &p.ProductForm, &p.Cycle, &p.Acceptance, &p.MentorNeeds, &p.ResourceNeeds, &stage, &status, &visibility, &p.Risk, &p.DataType, &p.RealTrial, &p.ExternalPublish, &p.FundsInvolved, &owner, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		return
	}
	if err != nil {
		serverError(c, "项目读取失败")
		return
	}
	canManage := u.UserID == owner
	if !canManage {
		var role string
		_ = h.db.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, id, u.UserID).Scan(&role)
		canManage = role == "co_owner" || role == "platform_operator"
	}
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"id": pid, "name": p.Name, "summary": p.Summary, "problem_statement": p.Problem, "target_users": p.Target, "expected_outcome": p.Outcome, "validation_plan": p.Validation, "project_type": p.Type, "project_source": p.Source, "product_form": p.ProductForm, "project_cycle": p.Cycle, "acceptance_criteria": p.Acceptance, "mentor_needs": p.MentorNeeds, "resource_needs": p.ResourceNeeds, "stage": stage, "status": status, "visibility": visibility, "risk_level": p.Risk, "data_type": p.DataType, "real_user_trial": p.RealTrial, "external_publish": p.ExternalPublish, "funds_involved": p.FundsInvolved, "owner_user_id": owner, "can_manage": canManage, "created_at": created, "updated_at": updated}})
}

type taskInput struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	AssigneeUserID     *int64 `json:"assignee_user_id"`
	AssigneeAIRole     string `json:"assignee_ai_role"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Priority           string `json:"priority"`
	DueAt              string `json:"due_at"`
}

func (in *taskInput) normalizeAndValidate() (string, int) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.AssigneeAIRole = strings.TrimSpace(in.AssigneeAIRole)
	in.AcceptanceCriteria = strings.TrimSpace(in.AcceptanceCriteria)
	in.Priority = strings.TrimSpace(in.Priority)
	in.DueAt = strings.TrimSpace(in.DueAt)
	if in.Priority == "" {
		in.Priority = "normal"
	}
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 200 {
		return "任务标题必填且不超过 200 字", 422
	}
	if in.AcceptanceCriteria == "" {
		return "任务必须填写验收标准", 422
	}
	if utf8.RuneCountInString(in.Description) > 4000 || utf8.RuneCountInString(in.AcceptanceCriteria) > 4000 {
		return "任务描述或验收标准超过 4000 字", 422
	}
	if !taskPriorities[in.Priority] {
		return "任务优先级无效", 422
	}
	if in.AssigneeUserID != nil && *in.AssigneeUserID <= 0 {
		return "真人负责人无效", 422
	}
	if in.AssigneeUserID != nil && in.AssigneeAIRole != "" {
		return "真人负责人和 AI 岗位只能选择一个", 422
	}
	return "", 0
}

func (h *VOPCHandler) ListTasks(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "任务列表读取失败")
		return
	}
	defer tx.Rollback()
	var owner int64
	if err = tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		} else {
			serverError(c, "任务列表读取失败")
		}
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "read")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权访问"})
		return
	}
	rows, err := tx.Query(`SELECT id,title,description,assignee_user_id,assignee_ai_role,acceptance_criteria,priority,status,due_at,created_by,created_at,updated_at FROM vopc_tasks WHERE project_id=? ORDER BY CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 ELSE 3 END, id DESC`, id)
	if err != nil {
		serverError(c, "任务列表读取失败")
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			serverError(c, "任务列表读取失败")
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		serverError(c, "任务列表读取失败")
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": items})
}

func (h *VOPCHandler) CreateTask(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var in taskInput
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
		serverError(c, "任务创建失败")
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
	if err = tx.QueryRow(`SELECT owner_user_id,stage,status FROM vopc_projects WHERE id=?`, id).Scan(&owner, &stage, &status); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "manage")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	stageNo, _ := strconv.Atoi(strings.TrimPrefix(stage, "S"))
	if blockedStatuses[status] || stageNo < 1 || stageNo >= 9 {
		c.JSON(409, gin.H{"code": 409, "message": "当前项目阶段或治理状态禁止创建任务"})
		return
	}
	if err = validateTaskAssignee(tx, id, in.AssigneeUserID, in.AssigneeAIRole); err != nil {
		c.JSON(422, gin.H{"code": 422, "message": err.Error()})
		return
	}
	res, err := tx.Exec(`INSERT INTO vopc_tasks(project_id,title,description,assignee_user_id,assignee_ai_role,acceptance_criteria,priority,due_at,created_by) VALUES(?,?,?,?,?,?,?,?,?)`, id, in.Title, in.Description, in.AssigneeUserID, nullableString(in.AssigneeAIRole), in.AcceptanceCriteria, in.Priority, nullableString(in.DueAt), u.UserID)
	if err != nil {
		serverError(c, "任务创建失败")
		return
	}
	taskID, err := res.LastInsertId()
	if err != nil {
		serverError(c, "任务创建失败")
		return
	}
	if err = writeEvent(tx, id, u.UserID, "task.created", "", "todo", fmt.Sprintf("任务 #%d：%s", taskID, in.Title)); err != nil {
		serverError(c, "任务审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "任务创建失败")
		return
	}
	committed = true
	c.JSON(201, gin.H{"code": 0, "data": gin.H{"id": taskID, "status": "todo"}})
}

func (h *VOPCHandler) UpdateTask(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(400, gin.H{"code": 400, "message": "任务 ID 无效"})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	body.Status = strings.TrimSpace(body.Status)
	if !taskStatuses[body.Status] {
		c.JSON(422, gin.H{"code": 422, "message": "任务状态无效"})
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "任务状态更新失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var owner int64
	var projectStatus, current string
	var assignee sql.NullInt64
	if err = tx.QueryRow(`SELECT p.owner_user_id,p.status,t.status,t.assignee_user_id FROM vopc_tasks t JOIN vopc_projects p ON p.id=t.project_id WHERE t.id=? AND t.project_id=?`, taskID, id).Scan(&owner, &projectStatus, &current, &assignee); err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "任务不存在或无权操作"})
		return
	}
	manage, manageErr := projectPolicy(tx, id, u.UserID, owner, "manage")
	read, readErr := projectPolicy(tx, id, u.UserID, owner, "read")
	isAssignee := readErr == nil && read && assignee.Valid && assignee.Int64 == u.UserID
	if (manageErr != nil || !manage) && !isAssignee {
		c.JSON(404, gin.H{"code": 404, "message": "任务不存在或无权操作"})
		return
	}
	if blockedStatuses[projectStatus] || completedLike[projectStatus] {
		c.JSON(409, gin.H{"code": 409, "message": "当前项目状态禁止更新任务"})
		return
	}
	if !validTaskTransition(current, body.Status) {
		c.JSON(409, gin.H{"code": 409, "message": "任务状态流转不合法"})
		return
	}
	res, err := tx.Exec(`UPDATE vopc_tasks SET status=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND project_id=? AND status=?`, body.Status, taskID, id, current)
	if err != nil {
		serverError(c, "任务状态更新失败")
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "任务状态已变化，请刷新后重试"})
		return
	}
	if err = writeEvent(tx, id, u.UserID, "task.status_changed", current, body.Status, fmt.Sprintf("任务 #%d", taskID)); err != nil {
		serverError(c, "任务审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "任务状态更新失败")
		return
	}
	committed = true
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"id": taskID, "status": body.Status}})
}

type taskScanner interface{ Scan(...any) error }

func scanTask(row taskScanner) (gin.H, error) {
	var id, createdBy int64
	var title, description, acceptance, priority, status, created, updated string
	var userID sql.NullInt64
	var aiRole, dueAt sql.NullString
	if err := row.Scan(&id, &title, &description, &userID, &aiRole, &acceptance, &priority, &status, &dueAt, &createdBy, &created, &updated); err != nil {
		return nil, err
	}
	item := gin.H{"id": id, "title": title, "description": description, "acceptance_criteria": acceptance, "priority": priority, "status": status, "created_by": createdBy, "created_at": created, "updated_at": updated}
	if userID.Valid {
		item["assignee_user_id"] = userID.Int64
	}
	if aiRole.Valid {
		item["assignee_ai_role"] = aiRole.String
	}
	if dueAt.Valid {
		item["due_at"] = dueAt.String
	}
	return item, nil
}

func validateTaskAssignee(tx *sql.Tx, projectID int64, userID *int64, aiRole string) error {
	if userID != nil {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, projectID, *userID).Scan(&n); err != nil || n != 1 {
			return errors.New("真人负责人必须是项目内有效成员")
		}
	}
	if aiRole != "" {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM vopc_ai_roles WHERE project_id=? AND role_key=? AND enabled=1`, projectID, aiRole).Scan(&n); err != nil || n != 1 {
			return errors.New("AI 负责人必须是项目内已启用岗位")
		}
	}
	return nil
}

func validTaskTransition(from, to string) bool {
	allowed := map[string]map[string]bool{
		"todo":        {"in_progress": true, "cancelled": true},
		"in_progress": {"todo": true, "review": true, "cancelled": true},
		"review":      {"in_progress": true, "done": true},
	}
	return allowed[from][to]
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (h *VOPCHandler) SubmitProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "状态更新失败")
		return
	}
	defer tx.Rollback()

	var owner int64
	var stage, status string
	var in projectInput
	err = tx.QueryRow(`SELECT owner_user_id,stage,status,name,summary,problem_statement,target_users,expected_outcome,validation_plan,project_type,project_source,product_form,project_cycle,acceptance_criteria,mentor_needs,resource_needs,risk_level,data_type,real_user_trial,external_publish,funds_involved FROM vopc_projects WHERE id=?`, id).Scan(&owner, &stage, &status, &in.Name, &in.Summary, &in.Problem, &in.Target, &in.Outcome, &in.Validation, &in.Type, &in.Source, &in.ProductForm, &in.Cycle, &in.Acceptance, &in.MentorNeeds, &in.ResourceNeeds, &in.Risk, &in.DataType, &in.RealTrial, &in.ExternalPublish, &in.FundsInvolved)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在"})
		return
	}
	if err != nil {
		serverError(c, "状态读取失败")
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "manage")
	if err != nil || !allowed {
		c.JSON(404, gin.H{"code": 404, "message": "项目不存在或无权推进"})
		return
	}
	if blockedStatuses[status] {
		c.JSON(409, gin.H{"code": 409, "message": "当前治理状态禁止推进"})
		return
	}
	if stage != "S0" || status != "draft" {
		c.JSON(409, gin.H{"code": 409, "message": "立项提交仅适用于 S0 草稿，后续阶段必须通过正式里程碑评审推进"})
		return
	}
	if msg, code := in.normalizeAndValidate(true); code != 0 {
		c.JSON(code, gin.H{"code": code, "message": msg})
		return
	}

	before := stage + "/" + status
	res, err := tx.Exec(`UPDATE vopc_projects SET stage='S1',status=?,submitted_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=? AND stage='S0' AND status='draft'`, stageStatuses[1], id)
	if err != nil {
		serverError(c, "状态更新失败")
		return
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		c.JSON(409, gin.H{"code": 409, "message": "项目状态已变化，请刷新后重试"})
		return
	}
	if err = execOne(tx, `UPDATE vopc_milestones SET status='passed',review_note=? WHERE project_id=? AND stage='S0'`, "立项资料校验通过", id); err != nil {
		serverError(c, "里程碑更新失败")
		return
	}
	if err = writeEvent(tx, id, u.UserID, "project.stage_changed", before, "S1/"+stageStatuses[1], "提交立项"); err != nil {
		serverError(c, "状态审计写入失败")
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(c, "状态更新失败")
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": gin.H{"stage": "S1", "status": stageStatuses[1]}})
}

// projectPolicy is the project-level authorization boundary. Global
// capabilities only permit attempting an action; active membership and the
// project role decide whether the action is allowed for this project.
func projectPolicy(tx *sql.Tx, id, user, owner int64, action string) (bool, error) {
	if user == owner {
		return action == "read" || action == "manage", nil
	}
	var role string
	err := tx.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, id, user).Scan(&role)
	if err != nil {
		return false, err
	}
	switch action {
	case "read":
		return true, nil
	case "manage":
		return role == "co_owner" || role == "platform_operator", nil
	default:
		return false, nil
	}
}
func milestoneEvidence(i int) string {
	return []string{"需求/创意卡与风险确认", "项目章程、组织与范围", "需求基线与验收标准", "产品技术方案与风险评估", "可运行成果与部署方案", "测试报告与验收记录", "上线审批与运维手册", "真实用户反馈与迭代记录", "价值/成本收益复盘", "成果包、复盘与归档"}[i]
}
func serverError(c *gin.Context, msg string) { c.JSON(500, gin.H{"code": 500, "message": msg}) }
func projectID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"code": 400, "message": "项目 ID 无效"})
		return 0, false
	}
	return id, true
}
