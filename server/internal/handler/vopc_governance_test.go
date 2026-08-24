package handler

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// helper: 在当前测试中追加一名 college_admin 用户。
func addCollegeAdmin(t *testing.T, db *sql.DB, id int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users(id,username,display_name,role,owner_scope,owner_id) VALUES(?,?,?,?,?,?)`, id, "u"+strconv.Itoa(id), "管理员"+strconv.Itoa(id), "college_admin", "college", "cs"); err != nil {
		t.Fatal(err)
	}
}

// helper: 在当前测试中追加一名 teacher 用户，用于 R2「导师」（mentor）单人审核路径。
func addTeacher(t *testing.T, db *sql.DB, id int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users(id,username,display_name,role,owner_scope,owner_id) VALUES(?,?,?,?,?,?)`, id, "u"+strconv.Itoa(id), "教师"+strconv.Itoa(id), "teacher", "college", "cs"); err != nil {
		t.Fatal(err)
	}
}

// grantPlatformOperator 通过真实治理角色授予端点把用户设为项目 platform_operator 成员。
// 不再用 db.Exec 直插 vopc_project_members 自证（对应 audit M-B1：治理角色必须经真实端点可达）。
// asToken 必须是治理系统角色（college_admin/school_admin/sys_admin）令牌，作为授予的调用方。
func grantPlatformOperator(t *testing.T, r *gin.Engine, projectID int64, asToken string, userID int64) {
	t.Helper()
	path := fmtPath("/api/v1/vopc/projects/%d/governance-roles", projectID)
	w := request(r, "POST", path, asToken, map[string]any{"action": "grant", "user_id": userID, "project_role": "platform_operator"})
	if w.Code != 200 {
		t.Fatalf("grant platform_operator to user %d: got %d %s", userID, w.Code, w.Body.String())
	}
}

func TestVOPCCloseStateMachine(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	other := token(t, 2, "student", "college", "cs", "active")

	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)

	// 非管理角色不能 close。
	if got := request(r, "POST", base+"/close", other, map[string]any{"action": "pause", "reason": "越权"}).Code; got != 404 {
		t.Fatalf("non-manager close got %d", got)
	}
	// 非法动作。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "fly", "reason": "x"}).Code; got != 422 {
		t.Fatalf("invalid action got %d", got)
	}
	// draft 状态不能 close。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "close", "reason": "未完成", "outcome_package": "x", "human_decision": "y"}).Code; got != 409 {
		t.Fatalf("draft close got %d", got)
	}
	// 提交立项进入 S1/pending_review，然后 pause → resume。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "pause", "reason": "资源暂缺"}).Code; got != 200 {
		t.Fatalf("pause got %d", got)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status); err != nil || status != "paused" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	// pause 状态禁止普通写操作（提交立项）。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 409 {
		t.Fatalf("paused submit got %d", got)
	}
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "resume", "reason": "资源到位"}).Code; got != 200 {
		t.Fatalf("resume got %d", got)
	}
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status); err != nil || status != "pending_review" {
		t.Fatalf("post-resume status=%s err=%v", status, err)
	}
	// terminate 需要失败证据。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "terminate", "reason": "停止"}).Code; got != 422 {
		t.Fatalf("terminate without evidence got %d", got)
	}
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "terminate", "reason": "方向失效", "failure_evidence": "用户调研无需求"}).Code; got != 200 {
		t.Fatalf("terminate got %d", got)
	}
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status); err != nil || status != "terminated" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	// 终止后不能 close 或 pivot。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "close", "reason": "x", "outcome_package": "y", "human_decision": "z"}).Code; got != 409 {
		t.Fatalf("terminated close got %d", got)
	}
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "pivot", "reason": "x", "human_decision": "y"}).Code; got != 409 {
		t.Fatalf("terminated pivot got %d", got)
	}
	// archive 只允许 completed 或 terminated。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "archive", "reason": "归档"}).Code; got != 200 {
		t.Fatalf("archive got %d", got)
	}
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status); err != nil || status != "archived" {
		t.Fatalf("status=%s err=%v", status, err)
	}
}

func TestVOPCPivotResetsProject(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "pivot", "reason": "改变方向", "human_decision": "转向新市场"}).Code; got != 200 {
		t.Fatalf("pivot got %d", got)
	}
	var stage, status string
	if err := db.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&stage, &status); err != nil {
		t.Fatal(err)
	}
	if stage != "G0" || status != "draft" {
		t.Fatalf("post-pivot = %s/%s, want G0/draft", stage, status)
	}
	var resets int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_milestones WHERE project_id=? AND stage<>'G0' AND status='pending'`, out.Data.ID).Scan(&resets)
	if resets != 4 {
		t.Fatalf("milestone resets=%d want 4", resets)
	}
}

func TestVOPCRiskGovernanceAndGate(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")
	// 导师路径：teacher 持有 vopc.mentor.review，作为本项目 mentor 成员可单人审核 R2。
	addTeacher(t, db, 9)
	mentor := token(t, 9, "teacher", "college", "cs", "active")
	// 越权者：teacher 持有 vopc.mentor.review 但非本项目 mentor 成员，应被 in-handler 拒绝。
	addTeacher(t, db, 10)
	stranger := token(t, 10, "teacher", "college", "cs", "active")
	// R2 项目（个人数据触发升档）。
	proj := validProject()
	proj["data_type"] = "个人数据"
	w := request(r, "POST", "/api/v1/vopc/projects", owner, proj)
	if w.Code != 201 {
		t.Fatalf("create R2 %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)

	// 将 teacher #9 设为本项目 mentor 成员（普通项目角色，可由主理人邀请，测试直插作 setup）。
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, 9, "mentor"); err != nil {
		t.Fatal(err)
	}

	// 登记 R2 风险。
	w = request(r, "POST", base+"/risks", owner, map[string]any{"risk_level": "R2", "title": "个人数据风险", "description": "涉及个人数据"})
	if w.Code != 201 {
		t.Fatalf("create risk %d %s", w.Code, w.Body.String())
	}
	var risk struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &risk)

	// 提交立项进入 S1。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	// 创建成果版本用于里程碑。
	aw := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "章程", "artifact_type": "document", "visibility": "private"})
	var art struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(aw.Body.Bytes(), &art)
	vw := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(art.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v1", "source_kind": "repository", "source_ref": "repo:commit:1", "checksum": "0000000000000000000000000000000000000000000000000000000000000001", "intended_stage": "G2"})
	var ver struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(vw.Body.Bytes(), &ver)
	// R2 未审批，禁止提交里程碑。
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 9}).Code; got != 409 {
		t.Fatalf("R2 unapproved milestone got %d", got)
	}
	// 越权：teacher 但非本项目 mentor 成员 → 403。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", stranger, map[string]any{"decision": "approve", "reason": "越权"}).Code; got != 403 {
		t.Fatalf("non-mentor approve got %d, want 403", got)
	}
	// 导师单人审核通过 → approved（不再需要两名不同审批人）。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", mentor, map[string]any{"decision": "approve", "reason": "审核通过"}).Code; got != 200 {
		t.Fatalf("mentor approve got %d", got)
	}
	var rstatus string
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk.Data.ID).Scan(&rstatus)
	if rstatus != "approved" {
		t.Fatalf("after single mentor approve status=%s want approved", rstatus)
	}
	// 同一审批人重复审批被拒。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", mentor, map[string]any{"decision": "approve", "reason": "重复"}).Code; got != 409 {
		t.Fatalf("duplicate approve got %d", got)
	}
	// 审批后里程碑可提交。
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 9}).Code; got != 201 {
		t.Fatalf("approved milestone got %d", got)
	}
	// 「管理员」单人审核路径：登记第二条 R2 风险，由授权管理员单人 approve 即 approved。
	w = request(r, "POST", base+"/risks", owner, map[string]any{"risk_level": "R2", "title": "真实部署风险", "description": "涉及真实部署"})
	if w.Code != 201 {
		t.Fatalf("create risk2 %d %s", w.Code, w.Body.String())
	}
	var risk2 struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &risk2)
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk2.Data.ID, 10)+"/approve", admin, map[string]any{"decision": "approve", "reason": "管理员审核通过"}).Code; got != 200 {
		t.Fatalf("admin approve got %d", got)
	}
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk2.Data.ID).Scan(&rstatus)
	if rstatus != "approved" {
		t.Fatalf("after single admin approve status=%s want approved", rstatus)
	}
}

func TestVOPCRiskFreezeAndAppeal(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)
	// 普通成员无法冻结（无 vopc.risk.manage 能力，路由层即 403）。
	if got := request(r, "POST", base+"/freeze", owner, map[string]any{"action": "freeze", "reason": "违规"}).Code; got != 403 {
		t.Fatalf("owner freeze got %d", got)
	}
	if got := request(r, "POST", base+"/freeze", admin, map[string]any{"action": "freeze", "reason": "风险处置"}).Code; got != 200 {
		t.Fatalf("admin freeze got %d", got)
	}
	var status string
	_ = db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status)
	if status != "risk_frozen" {
		t.Fatalf("status=%s want risk_frozen", status)
	}
	// 冻结后禁止提交立项。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 409 {
		t.Fatalf("frozen submit got %d", got)
	}
	// 申诉。
	if got := request(r, "POST", base+"/risk-appeals", owner, map[string]any{"reason": "处置过当"}).Code; got != 201 {
		t.Fatalf("appeal got %d", got)
	}
	// 直接查询最后一条申诉 id。
	var aid int64
	_ = db.QueryRow(`SELECT id FROM vopc_risk_appeals WHERE project_id=? ORDER BY id DESC LIMIT 1`, out.Data.ID).Scan(&aid)
	if got := request(r, "POST", base+"/risk-appeals/"+strconv.FormatInt(aid, 10)+"/resolve", owner, map[string]any{"decision": "dismissed", "resolution": "维持"}).Code; got != 403 {
		t.Fatalf("owner resolve got %d", got)
	}
	if got := request(r, "POST", base+"/risk-appeals/"+strconv.FormatInt(aid, 10)+"/resolve", admin, map[string]any{"decision": "dismissed", "resolution": "风险确实存在"}).Code; got != 200 {
		t.Fatalf("admin resolve got %d", got)
	}
	// 解冻。
	if got := request(r, "POST", base+"/freeze", admin, map[string]any{"action": "unfreeze", "reason": "处置完成"}).Code; got != 200 {
		t.Fatalf("unfreeze got %d", got)
	}
	_ = db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status)
	if status != "pending_review" {
		t.Fatalf("post-unfreeze status=%s", status)
	}
}

// TestVOPCR3SpecialGovernanceChannel 验证 PRD §13.1 R3 口径：默认禁止、按学校制度专项审批。
// 覆盖：普通 manager 创建 R3 被拒、授权管理员可登记 R3、R3 缺专项审批时里程碑被拦、
// R3 不经平台普通审批（approve 被拒）、登记专项审批记录后里程碑放行、非授权管理员
// 不可登记专项审批。
func TestVOPCR3SpecialGovernanceChannel(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")

	// R0 项目（公开数据，不触发 R2/R3 自动升档），用于隔离 R3 门禁。
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	if w.Code != 201 {
		t.Fatalf("create R0 %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)

	// 指定 reviewer（用户 4 作为 reviewer 成员），供里程碑提交的 reviewer_user_id 校验。
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, 4, "reviewer"); err != nil {
		t.Fatal(err)
	}

	// 1) 普通 manager（owner）登记 R3 风险 → 403（R3 仅授权管理员可登记）。
	if got := request(r, "POST", base+"/risks", owner, map[string]any{"risk_level": "R3", "title": "真实支付风险", "description": "涉及真实支付"}).Code; got != 403 {
		t.Fatalf("owner create R3 got %d, want 403", got)
	}

	// 2) 授权管理员登记 R3 风险 → 201。
	w = request(r, "POST", base+"/risks", admin, map[string]any{"risk_level": "R3", "title": "真实支付风险", "description": "涉及真实支付"})
	if w.Code != 201 {
		t.Fatalf("admin create R3 %d %s", w.Code, w.Body.String())
	}
	var risk struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &risk)

	// R3 缺专项审批时里程碑被拦（项目为 R0，但挂有 R3 风险 → R3-tier）。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	aw := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "章程", "artifact_type": "document", "visibility": "private"})
	var art struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(aw.Body.Bytes(), &art)
	vw := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(art.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v1", "source_kind": "repository", "source_ref": "repo:commit:1", "checksum": "0000000000000000000000000000000000000000000000000000000000000001", "intended_stage": "G2"})
	var ver struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(vw.Body.Bytes(), &ver)
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 409 {
		t.Fatalf("R3 unapproved milestone got %d, want 409", got)
	}

	// 3) R3 不进入平台普通审批：即使授权管理员 approve 也被拒（按学校制度专项审批）。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", admin, map[string]any{"decision": "approve", "reason": "尝试普通审批"}).Code; got != 409 {
		t.Fatalf("R3 ordinary approve got %d, want 409", got)
	}
	// R3 风险仍为 open，里程碑仍被拦。
	var rstatus string
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk.Data.ID).Scan(&rstatus)
	if rstatus != "open" {
		t.Fatalf("after failed approve R3 status=%s want open", rstatus)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 409 {
		t.Fatalf("R3 still-blocked milestone got %d, want 409", got)
	}

	// 4) 非授权管理员（owner/student）不可登记专项审批 → 403（路由层 vopc.risk.manage 拦截）。
	if got := request(r, "POST", base+"/special-approvals", owner, map[string]any{"reason": "学校批准", "approver": "校科研处", "ref": "批文-2026-001"}).Code; got != 403 {
		t.Fatalf("owner special-approval got %d, want 403", got)
	}

	// 5) 授权管理员登记专项审批记录 → 201，里程碑放行。
	if got := request(r, "POST", base+"/special-approvals", admin, map[string]any{"reason": "学校制度批准", "approver": "校科研处", "ref": "批文-2026-001"}).Code; got != 201 {
		t.Fatalf("admin special-approval got %d", got)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 201 {
		t.Fatalf("R3 special-approved milestone got %d, want 201", got)
	}
}

// TestVOPCMilestoneGateTOCTOU 验证 H-B1：提交里程碑后、评审前登记 R2/R3 风险，
// reviewer 直接 pass 必须被风险门禁复核拦截（修复原 TOCTOU 绕过）。
func TestVOPCMilestoneGateTOCTOU(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	reviewer := token(t, 4, "college_admin", "college", "cs", "active")

	// R0 项目（公开数据，无自动升档），隔离风险门禁。
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	if w.Code != 201 {
		t.Fatalf("create R0 %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)

	// 指定 reviewer（用户 4 作为 reviewer 成员）。
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, 4, "reviewer"); err != nil {
		t.Fatal(err)
	}

	// 提交立项进入 S1。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	// 创造成果版本用于里程碑。
	aw := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "章程", "artifact_type": "document", "visibility": "private"})
	var art struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(aw.Body.Bytes(), &art)
	vw := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(art.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v1", "source_kind": "repository", "source_ref": "repo:commit:1", "checksum": "0000000000000000000000000000000000000000000000000000000000000001", "intended_stage": "G2"})
	var ver struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(vw.Body.Bytes(), &ver)

	// 提交 S2 里程碑（此时无 R3 风险，门禁通过）。
	sw := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4})
	if sw.Code != 201 {
		t.Fatalf("submit milestone got %d %s", sw.Code, sw.Body.String())
	}
	var sub struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(sw.Body.Bytes(), &sub)

	// 提交后、评审前：授权管理员（用户 7 = college_admin）登记一条 R3 风险。
	addCollegeAdmin(t, db, 7)
	gov := token(t, 7, "college_admin", "college", "cs", "active")
	rw := request(r, "POST", base+"/risks", gov, map[string]any{"risk_level": "R3", "title": "真实支付风险", "description": "涉及真实支付"})
	if rw.Code != 201 {
		t.Fatalf("gov create R3 %d %s", rw.Code, rw.Body.String())
	}

	// reviewer 直接 pass：必须被风险门禁复核拦住（409），而不是推进到 S2。
	if got := request(r, "POST", base+"/milestone-submissions/"+strconv.FormatInt(sub.Data.ID, 10)+"/review", reviewer, map[string]any{"result": "pass", "note": "直接通过"}).Code; got != 409 {
		t.Fatalf("TOCTOU pass got %d, want 409", got)
	}
	// 项目阶段不应推进到 S2。
	var stage, status string
	_ = db.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&stage, &status)
	if stage != "G1" {
		t.Fatalf("stage=%s want G1 (未推进)", stage)
	}
	if status == "developing" {
		t.Fatalf("status=%s 不应被推进", status)
	}
	// H-B1 全量回滚断言：submission 应仍为 pending，评审记录未落库，无伪造审计误写。
	var subStatus string
	_ = db.QueryRow(`SELECT status FROM vopc_milestone_submissions WHERE id=?`, sub.Data.ID).Scan(&subStatus)
	if subStatus != "pending" {
		t.Fatalf("submission status=%s want pending (应全量回滚)", subStatus)
	}
	var reviewCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_milestone_reviews WHERE submission_id=?`, sub.Data.ID).Scan(&reviewCount)
	if reviewCount != 0 {
		t.Fatalf("milestone review leaked %d, want 0 (应回滚)", reviewCount)
	}
	var eventCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action='milestone.reviewed'`, out.Data.ID).Scan(&eventCount)
	if eventCount != 0 {
		t.Fatalf("milestone.reviewed event leaked %d, want 0 (无审计误写)", eventCount)
	}
}

// TestVOPCFreezeBlocksBusinessWrites 验证 H-B2：risk_frozen 项目应拒绝成果/版本/风险登记与结构性流转。
func TestVOPCFreezeBlocksBusinessWrites(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")

	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)

	// 冻结前：先造一个成果用于后续版本冻结测试（成果已在冻结前创建）。
	aw := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "预建成果", "artifact_type": "document", "visibility": "private"})
	var art struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(aw.Body.Bytes(), &art)

	// 授权管理员（vopc.risk.manage）冻结项目。
	if got := request(r, "POST", base+"/freeze", admin, map[string]any{"action": "freeze", "reason": "风险处置"}).Code; got != 200 {
		t.Fatalf("freeze got %d", got)
	}

	// 冻结后：创建成果应被拒（409）且不落库。
	if got := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "冻结后成果", "artifact_type": "document", "visibility": "private"}).Code; got != 409 {
		t.Fatalf("frozen CreateArtifact got %d, want 409", got)
	}
	var frozenArtifacts int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_artifacts WHERE project_id=? AND name='冻结后成果'`, out.Data.ID).Scan(&frozenArtifacts)
	if frozenArtifacts != 0 {
		t.Fatalf("frozen artifact leaked %d", frozenArtifacts)
	}

	// 冻结后：为首个成果新增版本应被拒（409）。
	if got := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(art.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v1", "source_kind": "repository", "source_ref": "repo:commit:1", "checksum": "0000000000000000000000000000000000000000000000000000000000000001", "intended_stage": "G2"}).Code; got != 409 {
		t.Fatalf("frozen CreateArtifactVersion got %d, want 409", got)
	}

	// 冻结后：登记风险（R0 等普通风险）应被拒（409）。
	if got := request(r, "POST", base+"/risks", owner, map[string]any{"risk_level": "R1", "title": "冻结后风险", "description": "x"}).Code; got != 409 {
		t.Fatalf("frozen CreateRisk got %d, want 409", got)
	}
	var frozenRisks int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_risks WHERE project_id=? AND title='冻结后风险'`, out.Data.ID).Scan(&frozenRisks)
	if frozenRisks != 0 {
		t.Fatalf("frozen risk leaked %d", frozenRisks)
	}

	// 冻结后：项目主理人不能 pivot/terminate 绕过治理冻结。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "pivot", "reason": "绕过", "human_decision": "x"}).Code; got != 409 {
		t.Fatalf("frozen pivot got %d, want 409", got)
	}
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "terminate", "reason": "绕过", "failure_evidence": "证据"}).Code; got != 409 {
		t.Fatalf("frozen terminate got %d, want 409", got)
	}

	// 申诉（remedy 路径）仍应可用：冻结后主理人可申诉。
	if got := request(r, "POST", base+"/risk-appeals", owner, map[string]any{"reason": "处置过当"}).Code; got != 201 {
		t.Fatalf("frozen appeal got %d, want 201", got)
	}
}

// TestVOPCGovernanceRoleProvisioning 验证 platform_operator 治理角色的受控授予/撤销端点。
// 覆盖：普通 manager（owner）不可授予（403）、非治理系统角色不可授予（403）、非 platform_operator 角色拒绝（422）、
// 治理系统角色可授予（200）并撤销（200）、重复授予/对非 platform_operator 撤销均 409。
func TestVOPCGovernanceRoleProvisioning(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	op := token(t, 4, "college_admin", "college", "cs", "active")
	student := token(t, 2, "student", "college", "cs", "active")

	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	path := fmtPath("/api/v1/vopc/projects/%d/governance-roles", out.Data.ID)

	// 1) owner（普通 manager）授予 → 403（路由层 vopc.audit/risk.manage 能力拦截）。
	if got := request(r, "POST", path, owner, map[string]any{"action": "grant", "user_id": 4, "project_role": "platform_operator"}).Code; got != 403 {
		t.Fatalf("owner grant got %d, want 403", got)
	}

	// 2) 非治理系统角色（student）授予 → 403。
	if got := request(r, "POST", path, student, map[string]any{"action": "grant", "user_id": 4, "project_role": "platform_operator"}).Code; got != 403 {
		t.Fatalf("student grant got %d, want 403", got)
	}

	// 3) 非 platform_operator 角色 → 422（fail-closed：防写入 owner/co_owner 提权）。
	if got := request(r, "POST", path, op, map[string]any{"action": "grant", "user_id": 4, "project_role": "owner"}).Code; got != 422 {
		t.Fatalf("non-platform_operator grant got %d, want 422", got)
	}

	// 4) 治理系统角色授予 platform_operator → 200。
	if got := request(r, "POST", path, op, map[string]any{"action": "grant", "user_id": 4, "project_role": "platform_operator"}).Code; got != 200 {
		t.Fatalf("gov grant got %d, want 200", got)
	}
	var role string
	_ = db.QueryRow(`SELECT project_role FROM vopc_project_members WHERE project_id=? AND user_id=? AND status='active'`, out.Data.ID, 4).Scan(&role)
	if role != "platform_operator" {
		t.Fatalf("member role=%s want platform_operator", role)
	}
	var grantEvent int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action='governance_role.granted'`, out.Data.ID).Scan(&grantEvent)
	if grantEvent != 1 {
		t.Fatalf("grant event=%d want 1", grantEvent)
	}

	// 5) 重复授予已存在的 platform_operator 成员 → 409。
	if got := request(r, "POST", path, op, map[string]any{"action": "grant", "user_id": 4, "project_role": "platform_operator"}).Code; got != 409 {
		t.Fatalf("duplicate grant got %d, want 409", got)
	}

	// 6) 撤销非 platform_operator 成员（owner 本人 or 普通用户）→ 409。
	if got := request(r, "POST", path, op, map[string]any{"action": "revoke", "user_id": 1, "project_role": "platform_operator"}).Code; got != 409 {
		t.Fatalf("revoke non-member got %d, want 409", got)
	}

	// 7) 治理系统角色撤销 platform_operator → 200。
	if got := request(r, "POST", path, op, map[string]any{"action": "revoke", "user_id": 4, "project_role": "platform_operator"}).Code; got != 200 {
		t.Fatalf("gov revoke got %d, want 200", got)
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_project_members WHERE project_id=? AND user_id=? AND project_role='platform_operator' AND status='active'`, out.Data.ID, 4).Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("after revoke platform_operator count=%d want 0", cnt)
	}
	var revokeEvent int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action='governance_role.revoked'`, out.Data.ID).Scan(&revokeEvent)
	if revokeEvent != 1 {
		t.Fatalf("revoke event=%d want 1", revokeEvent)
	}
}
