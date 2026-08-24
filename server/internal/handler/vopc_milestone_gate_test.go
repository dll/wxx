package handler

// vOPC A4 里程碑完整业务门禁测试：评分量表 / conditional pass 闭环 / 豁免 / 甲方结构化证据。
// 纯增量：复用既有 vopcTestDB/token/request/fmtPath/validProject/addTeacher/grantPlatformOperator helper；
// 本文件 self-contained 注册 A4 路由到独立 router，不与既有 vopcRouter 冲突。
// 迁移：在 vopcTestDB（097-104）基础上追加执行 105，构建 A4 表结构。

import (
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// vopcA4DB 复用 vopcTestDB 的 schema（097-104）并追加 105 迁移，构建 A4 表。
func vopcA4DB(t *testing.T) *sql.DB {
	t.Helper()
	db := vopcTestDB(t)
	migration, err := os.ReadFile("../../migrations/105_vopc_milestone_rubric_waiver_evidence.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	// 播种默认量表：S2/S5 维度（仅供测试断言 ruby 读取）
	for _, row := range []struct {
		stage, key, title, desc string
		max, min                int64
	}{
		{"G2", "completeness", "交付物完整性", "交付物齐备度评分", 5, 3},
		{"G2", "acceptability", "可验收性", "反馈可验收标准满足度", 5, 3},
		{"G3", "completeness", "交付物完整性", "交付物齐备度评分", 5, 3},
		{"G3", "runnable", "可运行性", "成果可自证可验证", 5, 3},
	} {
		if _, err = db.Exec(`INSERT INTO vopc_rubrics(stage,dimension_key,title,max_score,min_pass,description) VALUES(?,?,?,?,?,?)`, row.stage, row.key, row.title, row.max, row.min, row.desc); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// vopcA4Router 注册 vOPC 组 + A4 新增子资源路由（含既有路由以支持全流程）。
func vopcA4Router(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	// 核心创建/提交/评审链路
	g.POST("/projects", h.CreateProject)
	g.GET("/projects/:id", h.GetProject)
	g.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitProject)
	g.POST("/projects/:id/milestone-submissions", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitMilestone)
	g.POST("/projects/:id/milestone-submissions/:submissionId/review", auth.RequireCapability(auth.VOPCMilestoneReview), h.ReviewMilestone)
	g.POST("/projects/:id/artifacts", auth.RequireCapability(auth.VOPCProjectManage), h.CreateArtifact)
	g.POST("/projects/:id/artifacts/:artifactId/versions", auth.RequireCapability(auth.VOPCProjectManage), h.CreateArtifactVersion)
	g.POST("/projects/:id/risks", auth.RequireCapability(auth.VOPCProjectManage), h.CreateRisk)
	g.POST("/projects/:id/special-approvals", auth.RequireCapability(auth.VOPCRiskManage), h.CreateSpecialApproval)
	g.POST("/projects/:id/governance-roles", auth.RequireAnyCapability(auth.VOPCRiskManage, auth.VOPCAudit), h.GrantGovernanceRole)
	// A4 新增
	g.GET("/projects/:id/rubrics", h.ListRubrics)
	g.GET("/projects/:id/milestone-submissions/:submissionId/review", h.GetSubmissionReview)
	g.PUT("/projects/:id/milestone-submissions/:submissionId/conditions/:conditionId", auth.RequireCapability(auth.VOPCProjectManage), h.MarkConditionSatisfied)
	g.POST("/projects/:id/milestone-submissions/:submissionId/finalize", auth.RequireCapability(auth.VOPCMilestoneReview), h.FinalizeMilestone)
	g.GET("/projects/:id/milestone-waivers", h.ListMilestoneWaivers)
	g.POST("/projects/:id/milestone-waivers", auth.RequireCapability(auth.VOPCProjectManage), h.CreateMilestoneWaiver)
	g.POST("/projects/:id/milestone-waivers/:waiverId/review", auth.RequireAnyCapability(auth.VOPCMentorReview, auth.VOPCMilestoneReview, auth.VOPCRiskManage), h.ReviewMilestoneWaiver)
	g.GET("/projects/:id/client-evidence", h.ListClientEvidence)
	g.POST("/projects/:id/client-evidence", auth.RequireCapability(auth.VOPCProjectManage), h.CreateClientEvidence)
	g.PUT("/projects/:id/client-evidence/:evidenceId", auth.RequireCapability(auth.VOPCProjectManage), h.UpdateClientEvidence)
	return r
}

// fmtPathA4 replaces every %d placeholder with the given ids in order.
func fmtPathA4(format string, ids ...int64) string {
	out := format
	for _, i := range ids {
		out = strings.Replace(out, "%d", strconv.FormatInt(i, 10), 1)
	}
	return out
}

// mustA4VersionStage 创建成果+版本并指定 intended_stage，返回版本 id。
func mustA4VersionStage(t *testing.T, r *gin.Engine, tok string, projectID int64, stage string) int64 {
	t.Helper()
	// 成果名加 stage 后缀避免唯一冲突
	base := fmtPath("/api/v1/vopc/projects/%d", projectID)
	aw := request(r, "POST", base+"/artifacts", tok, map[string]any{"name": stage + "交付物", "artifact_type": "document", "description": "", "visibility": "private"})
	if aw.Code != 201 {
		t.Fatalf("create artifact got %d %s", aw.Code, aw.Body.String())
	}
	var aout struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(aw.Body.Bytes(), &aout)
	vw := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/artifacts/%d/versions", projectID, aout.Data.ID), tok, map[string]any{
		"version": "v1", "source_kind": "link", "source_ref": "https://example.com/" + stage,
		"checksum": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "intended_stage": stage,
	})
	if vw.Code != 201 {
		t.Fatalf("create version got %d %s", vw.Code, vw.Body.String())
	}
	var vout struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(vw.Body.Bytes(), &vout)
	return vout.Data.ID
}

// 创建项目并返回 id（复用 validProject + R0）。
func mustA4Project(t *testing.T, r *gin.Engine, tok string) int64 {
	t.Helper()
	w := request(r, "POST", "/api/v1/vopc/projects", tok, validProject())
	if w.Code != 201 {
		t.Fatalf("create got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Data.ID
}

// ---- ① 评分量表 ----

func TestVOPCA4ListRubrics(t *testing.T) {
	db := vopcA4DB(t)
	r := vopcA4Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)

	w := request(r, "GET", fmtPath("/api/v1/vopc/projects/%d/rubrics", id), owner, nil)
	if w.Code != 200 {
		t.Fatalf("rubrics got %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Data) != 4 {
		t.Fatalf("rubrics count=%d want 4", len(out.Data))
	}
}

func TestVOPCA4ReviewConditionalAndFinalize(t *testing.T) {
	db := vopcA4DB(t)
	r := vopcA4Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")
	id := mustA4Project(t, r, owner)

	// 授予用户4 为 platform_operator（作为指定评审，通过真实治理端点）
	grantPlatformOperator(t, r, id, admin, 4)

	// 提交立项进入 G1/pending_review
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", id), owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	// 创建 G2 成果版本并提交 G2 里程碑（G1 不在 milestoneArtifactTypes，直接走 G2）
	vid2 := mustA4VersionStage(t, r, owner, id, "G2")
	sw := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/milestone-submissions", id), owner, map[string]any{"stage": "G2", "evidence": "G2 交付物", "reviewer_user_id": 4, "artifact_version_ids": []int64{vid2}})
	if sw.Code != 201 {
		t.Fatalf("G2 milestone submit got %d %s", sw.Code, sw.Body.String())
	}
	var sout struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(sw.Body.Bytes(), &sout)
	subBase := fmtPathA4("/api/v1/vopc/projects/%d/milestone-submissions/%d", id, sout.Data.ID)

	// conditional_pass 无 conditions → 422
	bad := request(r, "POST", subBase+"/review", admin, map[string]any{"result": "conditional_pass", "note": "待条件"})
	if bad.Code != 422 {
		t.Fatalf("conditional without conditions got %d", bad.Code)
	}
	// conditional_pass + conditions + scores → 200，submission 状态 condition_pending，阶段不推进
	cw := request(r, "POST", subBase+"/review", admin, map[string]any{
		"result": "conditional_pass", "note": "主体已达标，待补验收",
		"scores":    []map[string]any{{"dimension_key": "completeness", "score": 4, "comment": "基本齐全"}},
		"conditions": []map[string]any{{"description": "补验收台账", "due_at": ""}},
	})
	if cw.Code != 200 {
		t.Fatalf("conditional got %d %s", cw.Code, cw.Body.String())
	}
	var stage string
	if err := db.QueryRow(`SELECT stage FROM vopc_projects WHERE id=?`, id).Scan(&stage); err != nil || stage != "G1" {
		t.Fatalf("stage=%s err=%v want G1 (conditional 不推进)", stage, err)
	}
	var subStatus string
	if err := db.QueryRow(`SELECT status FROM vopc_milestone_submissions WHERE id=?`, sout.Data.ID).Scan(&subStatus); err != nil || subStatus != "condition_pending" {
		t.Fatalf("subStatus=%s err=%v want condition_pending", subStatus, err)
	}
	// 获取 conditionId
	var condID int64
	if err := db.QueryRow(`SELECT id FROM vopc_milestone_conditions WHERE submission_id=?`, sout.Data.ID).Scan(&condID); err != nil {
		t.Fatal(err)
	}
	// 未全部满足时 finalize → 409
	if got := request(r, "POST", subBase+"/finalize", admin, nil).Code; got != 409 {
		t.Fatalf("finalize with open condition got %d", got)
	}
	// 主理人标记条件满足
	condBase := fmtPathA4("/api/v1/vopc/projects/%d/milestone-submissions/%d/conditions/%d", id, sout.Data.ID, condID)
	if got := request(r, "PUT", condBase, owner, nil).Code; got != 200 {
		t.Fatalf("mark satisfied got %d", got)
	}
	// 全部满足后 finalize → 200，阶段推进到 S2
	fw := request(r, "POST", subBase+"/finalize", admin, nil)
	if fw.Code != 200 {
		t.Fatalf("finalize got %d %s", fw.Code, fw.Body.String())
	}
	if err := db.QueryRow(`SELECT stage FROM vopc_projects WHERE id=?`, id).Scan(&stage); err != nil || stage != "G2" {
		t.Fatalf("post-finalize stage=%s err=%v want G2", stage, err)
	}
}

// ---- ③ 豁免 waiver ----

func TestVOPCA4WaiverFlow(t *testing.T) {
	db := vopcA4DB(t)
	r := vopcA4Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	admin := token(t, 4, "college_admin", "college", "cs", "active")
	addTeacher(t, db, 9)
	mentor := token(t, 9, "teacher", "college", "cs", "active")
	id := mustA4Project(t, r, owner)
	base := fmtPath("/api/v1/vopc/projects/%d", id)

	// 主理人申请豁免（R0 项目）
	w := request(r, "POST", base+"/milestone-waivers", owner, map[string]any{"stage": "G2", "required_evidence": "可运行成果", "reason": "纯软件交付，无需部署文档"})
	if w.Code != 201 {
		t.Fatalf("create waiver got %d %s", w.Code, w.Body.String())
	}
	var wout struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &wout)
	// R0 项目：导师（VOPCMentorReview）可单签 approve
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/milestone-waivers/%d/review", id, wout.Data.ID), mentor, map[string]any{"action": "approve", "note": "同意豁免"}).Code; got != 200 {
		t.Fatalf("mentor approve got %d", got)
	}
	var wkStatus string
	if err := db.QueryRow(`SELECT status FROM vopc_milestone_waivers WHERE id=?`, wout.Data.ID).Scan(&wkStatus); err != nil || wkStatus != "approved" {
		t.Fatalf("waiver status=%s err=%v", wkStatus, err)
	}

	// R2 项目豁免：导师单签必须被拒（须治理角色复核）
	r2Body := validProject()
	r2Body["risk_level"] = "R2"
	w2 := request(r, "POST", "/api/v1/vopc/projects", owner, r2Body)
	var r2out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &r2out)
	w2base := fmtPath("/api/v1/vopc/projects/%d", r2out.Data.ID)
	ww2 := request(r, "POST", w2base+"/milestone-waivers", owner, map[string]any{"stage": "G3", "reason": "测试环境受控"})
	if ww2.Code != 201 {
		t.Fatalf("create r2 waiver got %d", ww2.Code)
	}
	var w2out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(ww2.Body.Bytes(), &w2out)
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/milestone-waivers/%d/review", r2out.Data.ID, w2out.Data.ID), mentor, map[string]any{"action": "approve", "note": "导师单签"}).Code; got != 403 {
		t.Fatalf("R2 mentor-only approve got %d, want 403", got)
	}
	if got := request(r, "POST", fmtPathA4("/api/v1/vopc/projects/%d/milestone-waivers/%d/review", r2out.Data.ID, w2out.Data.ID), admin, map[string]any{"action": "approve", "note": "治理复核通过"}).Code; got != 200 {
		t.Fatalf("R2 admin approve got %d", got)
	}
}

// ---- ④ 甲方结构化证据 client evidence ----

func TestVOPCA4ClientEvidence(t *testing.T) {
	db := vopcA4DB(t)
	r := vopcA4Router(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	id := mustA4Project(t, r, owner)
	base := fmtPath("/api/v1/vopc/projects/%d", id)

	// 非法阶段 → 422
	if got := request(r, "POST", base+"/client-evidence", owner, map[string]any{"stage": "G1", "client_rep": "反馈方", "conclusion": "confirmed"}).Code; got != 422 {
		t.Fatalf("invalid stage got %d", got)
	}
	// 非法结论 → 422
	if got := request(r, "POST", base+"/client-evidence", owner, map[string]any{"stage": "G2", "client_rep": "反馈方", "conclusion": "maybe"}).Code; got != 422 {
		t.Fatalf("invalid conclusion got %d", got)
	}
	// 合法创建 → 201
	w := request(r, "POST", base+"/client-evidence", owner, map[string]any{"stage": "G3", "client_rep": "学院实验室", "client_contact": "", "conclusion": "confirmed", "sign_method": "系统确认", "file_ref": "", "note": "反馈确认"})
	if w.Code != 201 {
		t.Fatalf("create evidence got %d %s", w.Code, w.Body.String())
	}
	var eout struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &eout)
	// 列表可见
	lw := request(r, "GET", base+"/client-evidence", owner, nil)
	if lw.Code != 200 {
		t.Fatalf("list evidence got %d", lw.Code)
	}
	// 更新
	if got := request(r, "PUT", fmtPathA4("/api/v1/vopc/projects/%d/client-evidence/%d", id, eout.Data.ID), owner, map[string]any{"client_rep": "甲方2", "conclusion": "reserved"}).Code; got != 200 {
		t.Fatalf("update evidence got %d", got)
	}
}
