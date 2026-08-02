package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

// setupUploadTestRouter 构造带 JWT 鉴权的 /kb/upload 路由
func setupUploadTestRouter(docSvc *service.DocumentService, kbSvc *service.KBService) (*gin.Engine, *config.Config) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWTSecret:      "test-secret-upload",
		JWTExpireHours: 2,
	}
	h := NewUploadHandler(docSvc, kbSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.POST("/kb/upload", h.Upload)
	return r, cfg
}

func buildUploadMultipart(t *testing.T, filename, content string, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	w.Close()
	return &buf, w.FormDataContentType()
}

func uploadRequest(t *testing.T, r *gin.Engine, cfg *config.Config, filename, content string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return uploadRequestAs(t, r, cfg, "college_admin", "", "", filename, content, fields)
}

// uploadRequestAs 支持指定角色与 owner 范围（college_admin 入库需满足 owner_scope CHECK）
func uploadRequestAs(t *testing.T, r *gin.Engine, cfg *config.Config, role, ownerScope, ownerID, filename, content string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := buildUploadMultipart(t, filename, content, fields)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/upload", body)
	req.Header.Set("Content-Type", ct)
	tok, _ := middleware.GenerateToken(cfg, &model.User{ID: 1, Username: "tester", Role: role, OwnerScope: ownerScope, OwnerID: ownerID})
	req.Header.Set("Authorization", "Bearer "+tok)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

// TestUploadHandler_RejectLowQuality 正文过短/无中文时拒绝自动入库
func TestUploadHandler_RejectLowQuality(t *testing.T) {
	db := testutil.NewTestDB(t)
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	kbSvc := service.NewKBService(repository.NewKBRepo(db), db)
	r, cfg := setupUploadTestRouter(docSvc, kbSvc)

	resp := uploadRequest(t, r, cfg, "note.txt", "短", nil)
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("低质量文档应拒绝(422)，得到 %d: %s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains([]byte(resp.Body.String()), []byte("质量")) {
		t.Errorf("拒绝响应应含质量原因: %s", resp.Body.String())
	}
}

// TestUploadHandler_RejectNoChinese 纯英文/非中文文档应拒绝（无中文 + 明确拒绝语义）
func TestUploadHandler_RejectNoChinese(t *testing.T) {
	db := testutil.NewTestDB(t)
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	kbSvc := service.NewKBService(repository.NewKBRepo(db), db)
	r, cfg := setupUploadTestRouter(docSvc, kbSvc)

	resp := uploadRequest(t, r, cfg, "readme.txt", strings.Repeat("English only text content. ", 8), nil)
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("无中文文档应拒绝(422)，得到 %d: %s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains([]byte(resp.Body.String()), []byte("不含中文字符")) {
		t.Errorf("拒绝响应应含无中文原因: %s", resp.Body.String())
	}
}

// TestUploadHandler_ForceOverride force=1 显式覆盖质量门槛
func TestUploadHandler_ForceOverride(t *testing.T) {
	db := testutil.NewTestDB(t)
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	kbSvc := service.NewKBService(repository.NewKBRepo(db), db)
	r, cfg := setupUploadTestRouter(docSvc, kbSvc)

	resp := uploadRequest(t, r, cfg, "note.txt", "短", map[string]string{"force": "1"})
	if resp.Code != http.StatusOK {
		t.Fatalf("force=1 应放行(200)，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

// TestUploadHandler_OKContent 正常中文文档直接入库
func TestUploadHandler_OKContent(t *testing.T) {
	db := testutil.NewTestDB(t)
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	kbSvc := service.NewKBService(repository.NewKBRepo(db), db)
	r, cfg := setupUploadTestRouter(docSvc, kbSvc)

	content := "为进一步加强校园文化建设，规范本科生第二课堂活动学分认证管理，切实保障学生综合素质评价的公平公正，结合学校实际制定本认证标准。本标准适用于全体在校本科生。"
	resp := uploadRequestAs(t, r, cfg, "college_admin", "college", "college-1", "activity.txt", content, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("正常文档应入库(200)，得到 %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !bytes.Contains([]byte(body), []byte("\"in_knowledge_base\":true")) {
		t.Errorf("正常文档应写入知识库: %s", body)
	}
	if bytes.Contains([]byte(body), []byte("质量")) {
		t.Errorf("正常文档不应带质量原因: %s", body)
	}
}

// TestUploadHandler_Unauthenticated 未认证应 401
func TestUploadHandler_Unauthenticated(t *testing.T) {
	db := testutil.NewTestDB(t)
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	kbSvc := service.NewKBService(repository.NewKBRepo(db), db)
	r, _ := setupUploadTestRouter(docSvc, kbSvc)

	body, ct := buildUploadMultipart(t, "note.txt", "短", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/upload", body)
	req.Header.Set("Content-Type", ct)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}
