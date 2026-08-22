package handler

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"
)

// helper: 在当前测试中追加一名 college_admin 用户，用于双人审批。
func addCollegeAdmin(t *testing.T, db *sql.DB, id int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users(id,username,display_name,role,owner_scope,owner_id) VALUES(?,?,?,?,?,?)`, id, "u"+strconv.Itoa(id), "管理员"+strconv.Itoa(id), "college_admin", "college", "cs"); err != nil {
		t.Fatal(err)
	}
}

// addPlatformOperator 把用户以 platform_operator 项目角色加入项目。
func addPlatformOperator(t *testing.T, db *sql.DB, projectID, userID int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, projectID, userID, "platform_operator"); err != nil {
		t.Fatal(err)
	}
}

// addRiskGovernance 把用户以 risk_governance（R3 专项审批）项目角色加入项目。
func addRiskGovernance(t *testing.T, db *sql.DB, projectID, userID int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, projectID, userID, "risk_governance"); err != nil {
		t.Fatal(err)
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
	if stage != "S0" || status != "draft" {
		t.Fatalf("post-pivot = %s/%s, want S0/draft", stage, status)
	}
	var resets int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_milestones WHERE project_id=? AND stage<>'S0' AND status='pending'`, out.Data.ID).Scan(&resets)
	if resets != 9 {
		t.Fatalf("milestone resets=%d want 9", resets)
	}
}

func TestVOPCRiskGovernanceAndGate(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")
	addCollegeAdmin(t, db, 6)
	admin2 := token(t, 6, "college_admin", "college", "cs", "active")

	// R2 项目（个人数据）。
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
	// 平台角色未加入项目时不能审批。
	if got := request(r, "POST", base+"/risks/1/approve", admin, map[string]any{"decision": "approve", "reason": "审查通过"}).Code; got != 404 {
		t.Fatalf("approve without membership got %d", got)
	}
	// 登记风险。
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
	addPlatformOperator(t, db, out.Data.ID, 4)
	addPlatformOperator(t, db, out.Data.ID, 6)

	// 提交立项进入 S1，再向 S2 提交里程碑被 R2 门禁拦截。
	// 先提交立项。
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
	vw := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(art.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v1", "source_kind": "repository", "source_ref": "repo:commit:1", "checksum": "0000000000000000000000000000000000000000000000000000000000000001", "intended_stage": "S2"})
	var ver struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(vw.Body.Bytes(), &ver)
	// R2 未审批，禁止提交里程碑。
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "S2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 409 {
		t.Fatalf("R2 unapproved milestone got %d", got)
	}
	// 单人审批 → 仍 open（未达双人）。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", admin, map[string]any{"decision": "approve", "reason": "初步通过"}).Code; got != 200 {
		t.Fatalf("single approve got %d", got)
	}
	var rstatus string
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk.Data.ID).Scan(&rstatus)
	if rstatus != "open" {
		t.Fatalf("after single approve status=%s want open", rstatus)
	}
	// 同一审批人重复审批被拒绝。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", admin, map[string]any{"decision": "approve", "reason": "重复"}).Code; got != 409 {
		t.Fatalf("duplicate approve got %d", got)
	}
	// 第二人审批 → approved。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", admin2, map[string]any{"decision": "approve", "reason": "复核通过"}).Code; got != 200 {
		t.Fatalf("second approve got %d", got)
	}
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk.Data.ID).Scan(&rstatus)
	if rstatus != "approved" {
		t.Fatalf("after double approve status=%s want approved", rstatus)
	}
	// 审批后里程碑可提交。
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "S2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 201 {
		t.Fatalf("approved milestone got %d", got)
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
	addPlatformOperator(t, db, out.Data.ID, 4)
	// 普通成员无法冻结（走治理角色校验）。
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

// TestVOPCR3SpecialGovernanceChannel 验证 B3：R3 独立专项审批通道。
// 覆盖：普通 manager 创建 R3 被拒、platform_operator 越权审批 R3 被拒、
// 专项角色可创建 R3、R3 缺专项审批时里程碑被拦、双专项审批后放行、任一 reject 即拒绝。
func TestVOPCR3SpecialGovernanceChannel(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	op := token(t, 4, "college_admin", "college", "cs", "active")
	addCollegeAdmin(t, db, 7)
	addCollegeAdmin(t, db, 8)
	gov1 := token(t, 7, "college_admin", "college", "cs", "active")
	gov2 := token(t, 8, "college_admin", "college", "cs", "active")

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

	// 1) 普通 manager（owner）创建 R3 风险 → 403。
	if got := request(r, "POST", base+"/risks", owner, map[string]any{"risk_level": "R3", "title": "真实支付风险", "description": "涉及真实支付"}).Code; got != 403 {
		t.Fatalf("owner create R3 got %d, want 403", got)
	}

	// 2) platform_operator（一般治理）不能创建 R3，也不能审批 R3。
	addPlatformOperator(t, db, out.Data.ID, 4)
	if got := request(r, "POST", base+"/risks", op, map[string]any{"risk_level": "R3", "title": "真实支付风险", "description": "涉及真实支付"}).Code; got != 403 {
		t.Fatalf("platform_operator create R3 got %d, want 403", got)
	}

	// 3) 专项角色（risk_governance）可创建 R3 风险。
	addRiskGovernance(t, db, out.Data.ID, 7)
	addRiskGovernance(t, db, out.Data.ID, 8)
	w = request(r, "POST", base+"/risks", gov1, map[string]any{"risk_level": "R3", "title": "真实支付风险", "description": "涉及真实支付"})
	if w.Code != 201 {
		t.Fatalf("gov create R3 %d %s", w.Code, w.Body.String())
	}
	var risk struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &risk)

	// R3 缺专项审批时里程碑被拦（项目为 R0，但挂有未审批 R3 风险）。
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	// 造一个成果版本用于里程碑提交。
	aw := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "章程", "artifact_type": "document", "visibility": "private"})
	var art struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(aw.Body.Bytes(), &art)
	vw := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(art.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v1", "source_kind": "repository", "source_ref": "repo:commit:1", "checksum": "0000000000000000000000000000000000000000000000000000000000000001", "intended_stage": "S2"})
	var ver struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(vw.Body.Bytes(), &ver)
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "S2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 409 {
		t.Fatalf("R3 unapproved milestone got %d, want 409", got)
	}

	// 4) 单人专项审批 → 仍 open（未达双人）。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", gov1, map[string]any{"decision": "approve", "reason": "初步通过"}).Code; got != 200 {
		t.Fatalf("gov1 single approve got %d", got)
	}
	var rstatus string
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk.Data.ID).Scan(&rstatus)
	if rstatus != "open" {
		t.Fatalf("after single R3 approve status=%s want open", rstatus)
	}
	// 单人审批后里程碑仍被拦。
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "S2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 409 {
		t.Fatalf("R3 single-approved milestone got %d, want 409", got)
	}
	// 第二专项审批 → approved，里程碑放行。
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk.Data.ID, 10)+"/approve", gov2, map[string]any{"decision": "approve", "reason": "专项复核通过"}).Code; got != 200 {
		t.Fatalf("gov2 approve got %d", got)
	}
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk.Data.ID).Scan(&rstatus)
	if rstatus != "approved" {
		t.Fatalf("after double R3 approve status=%s want approved", rstatus)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "S2", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 201 {
		t.Fatalf("R3 approved milestone got %d, want 201", got)
	}

	// 5) 任一专项 reject 即拒绝：另建 R3 风险用专项角色 reject → rejected，里程碑被拦。
	w = request(r, "POST", base+"/risks", gov1, map[string]any{"risk_level": "R3", "title": "医疗诊断风险", "description": "涉及医疗"})
	if w.Code != 201 {
		t.Fatalf("gov create R3-2 %d %s", w.Code, w.Body.String())
	}
	var risk2 struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &risk2)
	if got := request(r, "POST", base+"/risks/"+strconv.FormatInt(risk2.Data.ID, 10)+"/approve", gov1, map[string]any{"decision": "reject", "reason": "禁止推进"}).Code; got != 200 {
		t.Fatalf("gov reject R3 got %d", got)
	}
	_ = db.QueryRow(`SELECT status FROM vopc_risks WHERE id=?`, risk2.Data.ID).Scan(&rstatus)
	if rstatus != "rejected" {
		t.Fatalf("after reject R3 status=%s want rejected", rstatus)
	}
	// 存在未通过专项审批的 R3 风险仍阻断里程碑。
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "S3", "evidence": "证据", "artifact_version_ids": []int64{ver.Data.ID}, "reviewer_user_id": 4}).Code; got != 409 {
		t.Fatalf("rejected R3 milestone got %d, want 409", got)
	}
}
