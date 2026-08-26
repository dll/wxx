package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

const testSecret = "vopc-test-secret-at-least-32-characters"

func vopcTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, display_name TEXT NOT NULL DEFAULT '', role TEXT NOT NULL, owner_scope TEXT NOT NULL, owner_id TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active')`); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../migrations/097_vopc_p0.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(raw)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"098_vopc_decisions.sql", "099_vopc_collaboration_delivery.sql", "100_vopc_artifact_version_gates.sql", "101_vopc_close_state_machine.sql", "102_vopc_risk_governance.sql", "103_vopc_private_files.sql", "104_vopc_risk_special_approval.sql", "107_vopc_v2_layers.sql"} {
		migration, readErr := os.ReadFile("../../migrations/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = db.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	for _, user := range []struct {
		id          int
		role, owner string
	}{{1, "student", "cs"}, {2, "student", "cs"}, {3, "student", "cs"}, {4, "college_admin", "cs"}, {5, "student", "business"}} {
		if _, err = db.Exec(`INSERT INTO users(id,username,display_name,role,owner_scope,owner_id) VALUES(?,?,?,?,'college',?)`, user.id, fmt.Sprintf("u%d", user.id), fmt.Sprintf("用户%d", user.id), user.role, user.owner); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func vopcRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	g.GET("/access", h.AccessStatus)
	g.GET("/projects", h.ListProjects)
	g.POST("/projects", h.CreateProject)
	g.GET("/projects/:id", h.GetProject)
	g.PUT("/projects/:id", auth.RequireCapability(auth.VOPCProjectManage), h.UpdateProject)
	g.GET("/projects/:id/tasks", h.ListTasks)
	g.GET("/projects/:id/decisions", h.ListDecisions)
	g.POST("/projects/:id/decisions", auth.RequireCapability(auth.VOPCProjectManage), h.CreateDecision)
	g.PUT("/projects/:id/decisions/:decisionId", auth.RequireCapability(auth.VOPCProjectManage), h.ActDecision)
	g.GET("/projects/:id/members", h.ListMembers)
	g.POST("/projects/:id/members", auth.RequireCapability(auth.VOPCProjectManage), h.InviteMember)
	g.GET("/invitations", auth.RequireCapability(auth.VOPCProjectJoin), h.ListMyInvitations)
	g.POST("/invitations/:invitationId/respond", auth.RequireCapability(auth.VOPCProjectJoin), h.RespondInvitation)
	g.GET("/projects/:id/artifacts", h.ListArtifacts)
	g.POST("/projects/:id/artifacts", auth.RequireCapability(auth.VOPCProjectManage), h.CreateArtifact)
	g.GET("/projects/:id/artifacts/:artifactId/versions", h.ListArtifactVersions)
	g.POST("/projects/:id/artifacts/:artifactId/versions", auth.RequireCapability(auth.VOPCProjectManage), h.CreateArtifactVersion)
	g.POST("/projects/:id/files", auth.RequireCapability(auth.VOPCProjectManage), h.UploadFile)
	g.GET("/projects/:id/files/:key", h.DownloadFile)
	g.GET("/projects/:id/milestone-submissions", h.ListMilestoneSubmissions)
	g.POST("/projects/:id/milestone-submissions", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitMilestone)
	g.POST("/projects/:id/milestone-submissions/:submissionId/review", auth.RequireCapability(auth.VOPCMilestoneReview), h.ReviewMilestone)
	g.POST("/projects/:id/tasks", auth.RequireCapability(auth.VOPCProjectManage), h.CreateTask)
	g.PUT("/projects/:id/tasks/:taskId", h.UpdateTask)
	g.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitProject)
	g.POST("/projects/:id/close", auth.RequireCapability(auth.VOPCProjectManage), h.CloseProject)
	g.GET("/projects/:id/close-records", h.ListCloseRecords)
	g.GET("/projects/:id/risks", h.ListRisks)
	g.POST("/projects/:id/risks", auth.RequireCapability(auth.VOPCProjectManage), h.CreateRisk)
	g.POST("/projects/:id/risks/:riskId/approve", auth.RequireAnyCapability(auth.VOPCRiskManage, auth.VOPCMentorReview), h.ApproveRisk)
	g.GET("/projects/:id/special-approvals", h.ListSpecialApprovals)
	g.POST("/projects/:id/special-approvals", auth.RequireCapability(auth.VOPCRiskManage), h.CreateSpecialApproval)
	g.POST("/projects/:id/freeze", auth.RequireCapability(auth.VOPCRiskManage), h.FreezeProject)
	g.POST("/projects/:id/risk-appeals", auth.RequireCapability(auth.VOPCProjectManage), h.CreateRiskAppeal)
	g.POST("/projects/:id/risk-appeals/:appealId/resolve", auth.RequireCapability(auth.VOPCRiskManage), h.ResolveRiskAppeal)
	g.POST("/projects/:id/governance-roles", auth.RequireAnyCapability(auth.VOPCRiskManage, auth.VOPCAudit), h.GrantGovernanceRole)
	return r
}

func token(t *testing.T, id int64, role, scope, owner, status string) string {
	t.Helper()
	v, err := middleware.GenerateToken(&config.Config{JWTSecret: testSecret, JWTExpireHours: 2}, &model.User{ID: id, Username: "u" + strconv.FormatInt(id, 10), Role: role, OwnerScope: scope, OwnerID: owner, Status: status, TokenVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func request(r http.Handler, method, path, tok string, body any) *httptest.ResponseRecorder {
	var b bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&b).Encode(body)
	}
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func validProject() map[string]any {
	return map[string]any{"name": "真实项目", "summary": "摘要", "problem_statement": "真实问题", "target_users": "学院用户", "expected_outcome": "可运行产品", "validation_plan": "用户验证", "project_type": "软件与 AI 产品", "project_source": "self_proposed", "product_form": "Web 应用", "project_cycle": "8 周", "acceptance_criteria": "通过验收", "mentor_needs": "技术导师", "resource_needs": "测试环境", "risk_level": "R0", "data_type": "公开数据"}
}

func TestVOPCAccessHTTPMatrix(t *testing.T) {
	r := vopcRouter(vopcTestDB(t))
	cases := []struct {
		name, tok string
		want      int
	}{{"未登录", "", 401}, {"inactive token", token(t, 1, "student", "college", "cs", "disabled"), 403}, {"guest", token(t, 1, "guest", "college", "cs", "active"), 403}, {"外院", token(t, 1, "student", "college", "business", "active"), 403}, {"school scope", token(t, 1, "student", "school", "cs", "active"), 403}, {"合法 cs", token(t, 1, "student", "college", "CS", "active"), 200}, {"sys admin system scope", token(t, 9, "sys_admin", "system", "", "active"), 200}, {"inactive sys admin", token(t, 9, "sys_admin", "system", "", "disabled"), 403}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := request(r, "GET", "/api/v1/vopc/access", tc.tok, nil).Code; got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestVOPCPrivateProjectIsolationAndSubmit(t *testing.T) {
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
	ownerDetail := request(r, "GET", fmtPath("/api/v1/vopc/projects/%d", out.Data.ID), owner, nil)
	if ownerDetail.Code != 200 || !bytes.Contains(ownerDetail.Body.Bytes(), []byte(`"can_manage":true`)) {
		t.Fatalf("owner manage flag missing: %d %s", ownerDetail.Code, ownerDetail.Body.String())
	}
	if got := request(r, "GET", fmtPath("/api/v1/vopc/projects/%d", out.Data.ID), other, nil).Code; got != 404 {
		t.Fatalf("private guess got %d", got)
	}
	w = request(r, "GET", "/api/v1/vopc/projects", other, nil)
	if w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte("真实项目")) {
		t.Fatalf("private list leaked: %s", w.Body.String())
	}
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), other, nil).Code; got != 404 {
		t.Fatalf("wrong owner got %d", got)
	}
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), owner, nil).Code; got != 200 {
		t.Fatalf("owner submit got %d", got)
	}
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), owner, nil).Code; got != 409 {
		t.Fatalf("repeat submit got %d", got)
	}
}

func TestVOPCDraftUpdateCompleteAndPermission(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	other := token(t, 2, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, map[string]any{"name": "待补齐草稿"})
	if w.Code != 201 {
		t.Fatalf("create=%d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	path := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)
	if got := request(r, "PUT", path, other, validProject()).Code; got != 404 {
		t.Fatalf("non member update=%d", got)
	}
	if got := request(r, "PUT", path, owner, validProject()).Code; got != 200 {
		t.Fatalf("owner update=%d", got)
	}
	if got := request(r, "POST", path+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit=%d", got)
	}
	if got := request(r, "PUT", path, owner, validProject()).Code; got != 409 {
		t.Fatalf("submitted update=%d", got)
	}
	var events int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action='project.draft_updated'`, out.Data.ID).Scan(&events)
	if events != 1 {
		t.Fatalf("draft update events=%d", events)
	}
}

func TestVOPCCreateAtomicRollbackAndRiskValidation(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	tok := token(t, 1, "student", "college", "cs", "active")
	bad := validProject()
	bad["project_type"] = "未知类型"
	if got := request(r, "POST", "/api/v1/vopc/projects", tok, bad).Code; got != 422 {
		t.Fatalf("unknown enum got %d", got)
	}
	r3 := validProject()
	r3["funds_involved"] = true
	if got := request(r, "POST", "/api/v1/vopc/projects", tok, r3).Code; got != 422 {
		t.Fatalf("funds got %d", got)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_event BEFORE INSERT ON vopc_events BEGIN SELECT RAISE(ABORT, 'event failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if got := request(r, "POST", "/api/v1/vopc/projects", tok, validProject()).Code; got != 500 {
		t.Fatalf("failure injection got %d", got)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM vopc_projects`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("transaction left %d projects err=%v", n, err)
	}
}

func TestVOPCProjectRoleBoundariesAndBlockedStates(t *testing.T) {
	roles := []struct {
		role string
		want int
	}{{"co_owner", 200}, {"platform_operator", 200}, {"member", 404}, {"mentor", 404}, {"reviewer", 404}}
	for i, tc := range roles {
		t.Run(tc.role, func(t *testing.T) {
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
			_ = db.QueryRow(`SELECT id FROM vopc_projects LIMIT 1`).Scan(&out.Data.ID)
			_, _ = db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, int64(i+2), tc.role)
			member := token(t, int64(i+2), "student", "college", "cs", "active")
			got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), member, nil).Code
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
	for _, status := range []string{"paused", "risk_frozen", "terminated", "archived"} {
		t.Run(status, func(t *testing.T) {
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
			_, _ = db.Exec(`UPDATE vopc_projects SET status=? WHERE id=?`, status, out.Data.ID)
			if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), owner, nil).Code; got != 409 {
				t.Fatalf("got %d", got)
			}
		})
	}
}

func TestVOPCG0ToG4FormalMilestoneFlowAndNoTextAdvanceRoute(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	reviewer := token(t, 4, "college_admin", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	if w.Code != http.StatusCreated {
		t.Fatalf("create got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != http.StatusOK {
		t.Fatalf("G0 -> G1 got %d", got)
	}
	if got := request(r, "POST", base+"/milestones/G2/advance", owner, map[string]any{"evidence": "文本直推"}).Code; got != http.StatusNotFound {
		t.Fatalf("formal router must not expose text advance route, got %d", got)
	}
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, 4, "reviewer"); err != nil {
		t.Fatal(err)
	}
	// G1→G2→G3→G4 三个里程碑；每阶段一个符合类型要求的成果版本。
	stageTypes := map[string]string{"G2": "document", "G3": "document", "G4": "document"}
	for _, g := range []string{"G2", "G3", "G4"} {
		stage := g
		artifact := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "阶段交付物" + stage, "artifact_type": stageTypes[stage], "visibility": "private"})
		var artifactOut struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(artifact.Body.Bytes(), &artifactOut); err != nil || artifactOut.Data.ID == 0 {
			t.Fatalf("artifact %s: %d %s", stage, artifact.Code, artifact.Body.String())
		}
		version := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(artifactOut.Data.ID, 10)+"/versions", owner, map[string]any{"version": "v" + stage, "source_kind": "repository", "source_ref": "repo:commit:stage-" + stage, "checksum": fmt.Sprintf("%064x", stageIndexOf(stage)), "intended_stage": stage})
		var versionOut struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(version.Body.Bytes(), &versionOut); err != nil || versionOut.Data.ID == 0 {
			t.Fatalf("version %s: %d %s", stage, version.Code, version.Body.String())
		}
		w = request(r, "POST", base+"/milestone-submissions", owner, map[string]any{
			"stage": stage, "evidence": "阶段证据与人工确认记录", "artifact_version_ids": []int64{versionOut.Data.ID}, "reviewer_user_id": 4,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("submit milestone %s got %d: %s", stage, w.Code, w.Body.String())
		}
		var submission struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &submission); err != nil {
			t.Fatal(err)
		}
		path := base + "/milestone-submissions/" + strconv.FormatInt(submission.Data.ID, 10) + "/review"
		if got := request(r, "POST", path, reviewer, map[string]any{"result": "pass", "note": "指定评审人工通过"}).Code; got != http.StatusOK {
			t.Fatalf("review %s got %d", stage, got)
		}
	}
	var stage, status string
	if err := db.QueryRow(`SELECT stage,status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&stage, &status); err != nil {
		t.Fatal(err)
	}
	if stage != "G4" || status != "closeable" {
		t.Fatalf("final state = %s/%s, want G4/closeable", stage, status)
	}
	// closeable 需由项目管理角色发起 close 才结项。
	if got := request(r, "POST", base+"/close", owner, map[string]any{"action": "close", "reason": "完成交付并复盘", "outcome_package": "成果包与用户反馈", "human_decision": "结项并归档"}).Code; got != http.StatusOK {
		t.Fatalf("close got %d", got)
	}
	if err := db.QueryRow(`SELECT status FROM vopc_projects WHERE id=?`, out.Data.ID).Scan(&status); err != nil || status != "completed" {
		t.Fatalf("post-close status=%s err=%v", status, err)
	}
}

func TestVOPCTaskLifecycleAuthorizationAndAudit(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	member := token(t, 2, "student", "college", "cs", "active")
	other := token(t, 3, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Data.ID == 0 {
		t.Fatalf("create: %s", w.Body.String())
	}
	if got := request(r, "POST", fmtPath("/api/v1/vopc/projects/%d/submit", out.Data.ID), owner, nil).Code; got != 200 {
		t.Fatalf("submit got %d", got)
	}
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, 2, "member"); err != nil {
		t.Fatal(err)
	}
	base := fmtPath("/api/v1/vopc/projects/%d/tasks", out.Data.ID)
	invalid := map[string]any{"title": "缺验收", "priority": "normal"}
	if got := request(r, "POST", base, owner, invalid).Code; got != 422 {
		t.Fatalf("missing acceptance got %d", got)
	}
	task := map[string]any{"title": "完成原型", "description": "交付可运行原型", "assignee_user_id": 2, "acceptance_criteria": "可启动并通过主流程", "priority": "high"}
	w = request(r, "POST", base, owner, task)
	if w.Code != 201 {
		t.Fatalf("create task %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if got := request(r, "GET", base, other, nil).Code; got != 404 {
		t.Fatalf("other list got %d", got)
	}
	taskPath := base + "/" + strconv.FormatInt(created.Data.ID, 10)
	if got := request(r, "PUT", taskPath, other, map[string]any{"status": "in_progress"}).Code; got != 404 {
		t.Fatalf("other update got %d", got)
	}
	if got := request(r, "PUT", taskPath, member, map[string]any{"status": "in_progress"}).Code; got != 200 {
		t.Fatalf("assignee update got %d", got)
	}
	if got := request(r, "PUT", taskPath, member, map[string]any{"status": "done"}).Code; got != 409 {
		t.Fatalf("skip review got %d", got)
	}
	if got := request(r, "PUT", taskPath, owner, map[string]any{"status": "review"}).Code; got != 200 {
		t.Fatalf("owner review got %d", got)
	}
	if got := request(r, "PUT", taskPath, owner, map[string]any{"status": "done"}).Code; got != 200 {
		t.Fatalf("owner done got %d", got)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM vopc_tasks WHERE id=?`, created.Data.ID).Scan(&status); err != nil || status != "done" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	var events int
	if err := db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action LIKE 'task.%'`, out.Data.ID).Scan(&events); err != nil || events != 4 {
		t.Fatalf("task events=%d err=%v", events, err)
	}
}

func fmtPath(format string, id int64) string {
	return stringsReplace(format, "%d", strconv.FormatInt(id, 10))
}
func stringsReplace(s, old, new string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func TestVOPCInvitationArtifactAndMilestoneReview(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcRouter(db)
	owner := token(t, 1, "student", "college", "cs", "active")
	member := token(t, 2, "student", "college", "cs", "active")
	reviewer := token(t, 4, "college_admin", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var project struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &project)
	base := fmtPath("/api/v1/vopc/projects/%d", project.Data.ID)
	w = request(r, "POST", base+"/members", owner, map[string]any{"user_id": 2, "project_role": "member", "message": "欢迎"})
	if w.Code != 201 {
		t.Fatalf("invite: %d %s", w.Code, w.Body.String())
	}
	var invitation struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &invitation)
	if got := request(r, "POST", "/api/v1/vopc/invitations/"+strconv.FormatInt(invitation.Data.ID, 10)+"/respond", member, map[string]any{"action": "accept"}).Code; got != 200 {
		t.Fatalf("accept=%d", got)
	}
	if got := request(r, "POST", base+"/members", owner, map[string]any{"user_id": 5, "project_role": "member"}).Code; got != 422 {
		t.Fatalf("external college invite=%d", got)
	}
	if got := request(r, "POST", base+"/members", owner, map[string]any{"user_id": 3, "project_role": "platform_operator"}).Code; got != 422 {
		t.Fatalf("owner must not grant platform_operator, got=%d", got)
	}
	w = request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "项目章程", "artifact_type": "document", "visibility": "private"})
	if w.Code != 201 {
		t.Fatalf("artifact %d %s", w.Code, w.Body.String())
	}
	var artifact struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &artifact)
	versionPath := base + "/artifacts/" + strconv.FormatInt(artifact.Data.ID, 10) + "/versions"
	w = request(r, "POST", versionPath, owner, map[string]any{"version": "v1.0.0", "source_kind": "repository", "source_ref": "repo:commit:abc", "checksum": fmt.Sprintf("%064x", 1), "intended_stage": "G2", "release_notes": "首版"})
	if w.Code != 201 {
		t.Fatalf("version %d %s", w.Code, w.Body.String())
	}
	var version struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &version)
	if got := request(r, "POST", base+"/submit", owner, nil).Code; got != 200 {
		t.Fatalf("submit project=%d", got)
	}
	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, project.Data.ID, 4, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "只有文本不能通过", "artifact_version_ids": []int64{}, "reviewer_user_id": 4}).Code; got != 422 {
		t.Fatalf("empty artifact versions=%d", got)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "重复版本", "artifact_version_ids": []int64{version.Data.ID, version.Data.ID}, "reviewer_user_id": 4}).Code; got != 422 {
		t.Fatalf("duplicate versions=%d", got)
	}
	if _, err := db.Exec(`UPDATE vopc_artifact_versions SET intended_stage='G3' WHERE id=?`, version.Data.ID); err != nil {
		t.Fatal(err)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "跨阶段版本", "artifact_version_ids": []int64{version.Data.ID}, "reviewer_user_id": 4}).Code; got != 422 {
		t.Fatalf("wrong stage version=%d", got)
	}
	if _, err := db.Exec(`UPDATE vopc_artifact_versions SET intended_stage='G2',status='invalid' WHERE id=?`, version.Data.ID); err != nil {
		t.Fatal(err)
	}
	if got := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "失效版本", "artifact_version_ids": []int64{version.Data.ID}, "reviewer_user_id": 4}).Code; got != 422 {
		t.Fatalf("invalid version=%d", got)
	}
	if _, err := db.Exec(`UPDATE vopc_artifact_versions SET status='active' WHERE id=?`, version.Data.ID); err != nil {
		t.Fatal(err)
	}
	w = request(r, "POST", base+"/milestone-submissions", owner, map[string]any{"stage": "G2", "evidence": "完整的项目章程与组织证据", "artifact_version_ids": []int64{version.Data.ID}, "reviewer_user_id": 4})
	if w.Code != 201 {
		t.Fatalf("milestone submit %d %s", w.Code, w.Body.String())
	}
	var submission struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &submission)
	reviewPath := base + "/milestone-submissions/" + strconv.FormatInt(submission.Data.ID, 10) + "/review"
	if got := request(r, "POST", reviewPath, member, map[string]any{"result": "pass", "note": "无权"}).Code; got != 403 {
		t.Fatalf("member review=%d", got)
	}
	if got := request(r, "POST", reviewPath, reviewer, map[string]any{"result": "pass", "note": "材料完整，同意通过"}).Code; got != 200 {
		t.Fatalf("review=%d", got)
	}
	var stage string
	if err := db.QueryRow(`SELECT stage FROM vopc_projects WHERE id=?`, project.Data.ID).Scan(&stage); err != nil || stage != "G2" {
		t.Fatalf("stage=%s err=%v", stage, err)
	}
	var events int
	_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action IN ('member.invited','member.invitation_responded','artifact.created','artifact.version_created','milestone.submitted','milestone.reviewed')`, project.Data.ID).Scan(&events)
	if events != 6 {
		t.Fatalf("events=%d", events)
	}
}

func TestVOPCInvitationAcceptRechecksIdentityAtomically(t *testing.T) {
	cases := []struct{ name, column, value string }{
		{"external college", "owner_id", "business"},
		{"guest", "role", "guest"},
		{"inactive", "status", "disabled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := vopcTestDB(t)
			r := vopcRouter(db)
			owner := token(t, 1, "student", "college", "cs", "active")
			invitee := token(t, 2, "student", "college", "cs", "active")
			w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
			var project struct {
				Data struct {
					ID int64 `json:"id"`
				} `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &project)
			base := fmtPath("/api/v1/vopc/projects/%d", project.Data.ID)
			w = request(r, "POST", base+"/members", owner, map[string]any{"user_id": 2, "project_role": "member"})
			var invitation struct {
				Data struct {
					ID int64 `json:"id"`
				} `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &invitation)
			if _, err := db.Exec("UPDATE users SET "+tc.column+"=? WHERE id=2", tc.value); err != nil {
				t.Fatal(err)
			}
			if got := request(r, "POST", "/api/v1/vopc/invitations/"+strconv.FormatInt(invitation.Data.ID, 10)+"/respond", invitee, map[string]any{"action": "accept"}).Code; got != 403 {
				t.Fatalf("accept=%d", got)
			}
			var status string
			if err := db.QueryRow(`SELECT status FROM vopc_invitations WHERE id=?`, invitation.Data.ID).Scan(&status); err != nil || status != "pending" {
				t.Fatalf("status=%s err=%v", status, err)
			}
			var members, events int
			_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_project_members WHERE project_id=? AND user_id=2`, project.Data.ID).Scan(&members)
			_ = db.QueryRow(`SELECT COUNT(*) FROM vopc_events WHERE project_id=? AND action='member.invitation_responded'`, project.Data.ID).Scan(&events)
			if members != 0 || events != 0 {
				t.Fatalf("atomicity members=%d events=%d", members, events)
			}
		})
	}
}
