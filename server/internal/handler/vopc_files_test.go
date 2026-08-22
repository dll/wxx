package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// vopcFilesRouter 复刻 vopcRouter 的路由，注入临时受控上传目录，避免污染仓库 .uploads。
func vopcFilesRouter(t *testing.T, db *sql.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: testSecret}
	h := NewVOPCHandler(db, "cs")
	h.SetUploadDir(t.TempDir())
	g := r.Group("/api/v1/vopc")
	g.Use(middleware.JWTAuth(cfg))
	g.Use(CollegeAccess("cs"))
	g.POST("/projects", h.CreateProject)
	g.POST("/projects/:id/submit", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitProject)
	g.POST("/projects/:id/members", auth.RequireCapability(auth.VOPCProjectManage), h.InviteMember)
	g.POST("/invitations/:invitationId/respond", auth.RequireCapability(auth.VOPCProjectJoin), h.RespondInvitation)
	g.POST("/projects/:id/artifacts", auth.RequireCapability(auth.VOPCProjectManage), h.CreateArtifact)
	g.POST("/projects/:id/artifacts/:artifactId/versions", auth.RequireCapability(auth.VOPCProjectManage), h.CreateArtifactVersion)
	g.POST("/projects/:id/milestone-submissions", auth.RequireCapability(auth.VOPCProjectManage), h.SubmitMilestone)
	g.POST("/projects/:id/files", auth.RequireCapability(auth.VOPCProjectManage), h.UploadFile)
	g.GET("/projects/:id/files/:key", h.DownloadFile)
	return r
}

// uploadFile 构造 multipart/form-data 上传请求，支持显式声明 MIME 类型。
func uploadFile(r *gin.Engine, path, tok, filename, declaredMime string, content []byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	partHead := make(map[string][]string)
	partHead["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	if declaredMime != "" {
		partHead["Content-Type"] = []string{declaredMime}
	}
	pw, err := w.CreatePart(partHead)
	if err != nil {
		panic(err)
	}
	_, _ = pw.Write(content)
	_ = w.Close()
	req := httptest.NewRequest("POST", path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestVOPCFileUploadPermissionMatrix(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcFilesRouter(t, db)
	owner := token(t, 1, "student", "college", "cs", "active")
	other := token(t, 2, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	path := fmtPath("/api/v1/vopc/projects/%d/files", out.Data.ID)

	if got := uploadFile(r, path, other, "a.pdf", "application/pdf", []byte("data")).Code; got != 404 {
		t.Fatalf("非成员上传 got %d, want 404", got)
	}
	ext := token(t, 5, "student", "college", "business", "active")
	if got := uploadFile(r, path, ext, "a.pdf", "application/pdf", []byte("data")).Code; got != 403 {
		t.Fatalf("外院上传 got %d, want 403", got)
	}
	if got := uploadFile(r, path, "", "a.pdf", "application/pdf", []byte("data")).Code; got != 401 {
		t.Fatalf("未登录上传 got %d, want 401", got)
	}
	got := uploadFile(r, path, owner, "a.pdf", "application/pdf", []byte("hello vopc"))
	if got.Code != 201 {
		t.Fatalf("owner 上传 got %d %s", got.Code, got.Body.String())
	}
}

func TestVOPCFileUploadRejectsSizeTypeAndInjection(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcFilesRouter(t, db)
	owner := token(t, 1, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	path := fmtPath("/api/v1/vopc/projects/%d/files", out.Data.ID)

	big := make([]byte, vopcMaxFileBytes+1)
	if got := uploadFile(r, path, owner, "big.pdf", "application/pdf", big).Code; got != 413 {
		t.Fatalf("超限 got %d, want 413", got)
	}
	if got := uploadFile(r, path, owner, "evil.exe", "application/x-msdownload", []byte("MZ")).Code; got != 415 {
		t.Fatalf("危险类型 got %d, want 415", got)
	}
	res := uploadFile(r, path, owner, "../../../etc/passwd", "text/plain", []byte("x"))
	if res.Code != 201 {
		t.Fatalf("注入文件名 got %d %s", res.Code, res.Body.String())
	}
	var data struct {
		Data struct {
			Key      string `json:"object_key"`
			FileName string `json:"file_name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &data)
	if data.Data.Key == "" || strings.Contains(data.Data.FileName, "/") || strings.Contains(data.Data.FileName, "..") {
		t.Fatalf("文件名/object_key 未净化: key=%q fileName=%q", data.Data.Key, data.Data.FileName)
	}
	if !validObjectKey(data.Data.Key) {
		t.Fatalf("object_key 非法: %q", data.Data.Key)
	}
}

func TestVOPCFileDownloadAuthorization(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcFilesRouter(t, db)
	owner := token(t, 1, "student", "college", "cs", "active")
	member := token(t, 2, "student", "college", "cs", "active")
	other := token(t, 3, "student", "college", "cs", "active")
	w := request(r, "POST", "/api/v1/vopc/projects", owner, validProject())
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	base := fmtPath("/api/v1/vopc/projects/%d", out.Data.ID)
	inv := request(r, "POST", base+"/members", owner, map[string]any{"user_id": 2, "project_role": "member"})
	var iid struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(inv.Body.Bytes(), &iid)
	if got := request(r, "POST", "/api/v1/vopc/invitations/"+strconv.FormatInt(iid.Data.ID, 10)+"/respond", member, map[string]any{"action": "accept"}).Code; got != 200 {
		t.Fatalf("accept got %d", got)
	}

	up := uploadFile(r, base+"/files", owner, "report.pdf", "application/pdf", []byte("secret bytes"))
	if up.Code != 201 {
		t.Fatalf("upload got %d %s", up.Code, up.Body.String())
	}
	var f struct {
		Data struct {
			Key string `json:"object_key"`
		} `json:"data"`
	}
	_ = json.Unmarshal(up.Body.Bytes(), &f)
	dl := base + "/files/" + f.Data.Key

	if got := request(r, "GET", dl, owner, nil).Code; got != 200 {
		t.Fatalf("owner 下载 got %d", got)
	}
	resp := request(r, "GET", dl, member, nil)
	if resp.Code != 200 || !bytes.Contains(resp.Body.Bytes(), []byte("secret bytes")) {
		t.Fatalf("member 下载 got %d body=%q", resp.Code, resp.Body.String())
	}
	if got := request(r, "GET", dl, other, nil).Code; got != 404 {
		t.Fatalf("非成员下载 got %d, want 404", got)
	}
	if got := request(r, "GET", base+"/files/..%2F..", owner, nil).Code; got != 404 {
		t.Fatalf("路径注入下载 got %d, want 404", got)
	}
}

func TestVOPCFileMilestoneGateIntegration(t *testing.T) {
	db := vopcTestDB(t)
	r := vopcFilesRouter(t, db)
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
	up := uploadFile(r, base+"/files", owner, "news.md", "text/markdown", []byte("# 标题\n正文"))
	if up.Code != 201 {
		t.Fatalf("upload got %d %s", up.Code, up.Body.String())
	}
	var f struct {
		Data struct {
			Key      string `json:"object_key"`
			Checksum string `json:"checksum"`
		} `json:"data"`
	}
	_ = json.Unmarshal(up.Body.Bytes(), &f)
	art := request(r, "POST", base+"/artifacts", owner, map[string]any{"name": "受控文档", "artifact_type": "file", "visibility": "private"})
	var aid struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(art.Body.Bytes(), &aid)
	ver := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(aid.Data.ID, 10)+"/versions", owner, map[string]any{
		"version": "v1", "source_kind": "storage_ref", "source_ref": f.Data.Key, "checksum": f.Data.Checksum, "intended_stage": "S2",
	})
	if ver.Code != 201 {
		t.Fatalf("storage_ref 版本创建 got %d %s", ver.Code, ver.Body.String())
	}
	var vid struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(ver.Body.Bytes(), &vid)

	if got := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(aid.Data.ID, 10)+"/versions", owner, map[string]any{
		"version": "v2", "source_kind": "storage_ref", "source_ref": "not-a-key", "checksum": f.Data.Checksum, "intended_stage": "S2",
	}).Code; got != 422 {
		t.Fatalf("非法 storage_ref 版本 got %d, want 422", got)
	}
	if got := request(r, "POST", base+"/artifacts/"+strconv.FormatInt(aid.Data.ID, 10)+"/versions", owner, map[string]any{
		"version": "v3", "source_kind": "storage_ref", "source_ref": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "checksum": f.Data.Checksum, "intended_stage": "S2",
	}).Code; got != 422 {
		t.Fatalf("不存在 key 版本 got %d, want 422", got)
	}

	if _, err := db.Exec(`INSERT INTO vopc_project_members(project_id,user_id,project_role) VALUES(?,?,?)`, out.Data.ID, 4, "reviewer"); err != nil {
		t.Fatal(err)
	}
	sub := request(r, "POST", base+"/milestone-submissions", owner, map[string]any{
		"stage": "S2", "evidence": "受控文件作为阶段性证据", "artifact_version_ids": []int64{vid.Data.ID}, "reviewer_user_id": 4,
	})
	if sub.Code != 201 {
		t.Fatalf("里程碑提交 got %d %s", sub.Code, sub.Body.String())
	}
}
