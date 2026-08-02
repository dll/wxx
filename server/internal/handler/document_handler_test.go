package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// setupDocumentTestRouter 构造带 JWT 鉴权的文档 handler 路由
func setupDocumentTestRouter(docSvc *service.DocumentService) (*gin.Engine, *config.Config) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "test-secret-document",
		JWTExpireHours: 2,
	}

	h := NewDocumentHandler(docSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.POST("/documents/refine", h.RefineDocument)

	return r, cfg
}

func tokenFor(r *gin.Engine, cfg *config.Config, role string) string {
	t, _ := middleware.GenerateToken(cfg, &model.User{ID: 1, Username: "tester", Role: role})
	return t
}

func TestDocumentHandler_Refine_Success(t *testing.T) {
	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content: `{"title":"国家奖学金评选办法","summary":"国家奖学金用于奖励特别优秀的学生，每人每年8000元。","keywords":["奖学金","评选"]}`,
		}, nil
	}
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	docSvc.SetLLMClient(mock)

	r, cfg := setupDocumentTestRouter(docSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/refine",
		bytes.NewBufferString(`{"content":"国家奖学金评选办法正文……","title":"旧标题","summary":"旧摘要","keywords":["旧"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(r, cfg, "college_admin"))
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !bytes.Contains([]byte(body), []byte("国家奖学金评选办法")) {
		t.Errorf("响应应包含精修后的标题: %s", body)
	}
	if bytes.Contains([]byte(body), []byte("\\\"refined\\\":false")) {
		t.Errorf("精修成功时 refined 应为 true: %s", body)
	}
}

func TestDocumentHandler_Refine_FallbackWhenLLMFails(t *testing.T) {
	mock := llm.NewMockClient("mock")
	mock.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{Content: "非 JSON 输出"}, nil
	}
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	docSvc.SetLLMClient(mock)

	r, cfg := setupDocumentTestRouter(docSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/refine",
		bytes.NewBufferString(`{"content":"正文内容","title":"原标题","summary":"原摘要","keywords":["原"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(r, cfg, "college_admin"))
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200（兜底），得到 %d: %s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains([]byte(resp.Body.String()), []byte("fallback")) {
		t.Errorf("兜底响应应含 fallback 标记: %s", resp.Body.String())
	}
}

func TestDocumentHandler_Refine_Unauthenticated(t *testing.T) {
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	r, _ := setupDocumentTestRouter(docSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/refine",
		bytes.NewBufferString(`{"content":"正文"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

func TestDocumentHandler_Refine_ForbiddenForStudent(t *testing.T) {
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	r, cfg := setupDocumentTestRouter(docSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/refine",
		bytes.NewBufferString(`{"content":"正文"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(r, cfg, "student"))
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Errorf("学生应无精修权限，期望 403，得到 %d", resp.Code)
	}
}

func TestDocumentHandler_Refine_MissingContent(t *testing.T) {
	docSvc := service.NewDocumentService(t.TempDir(), 10)
	r, cfg := setupDocumentTestRouter(docSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/refine",
		bytes.NewBufferString(`{"title":"仅有标题"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(r, cfg, "college_admin"))
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("缺 content 应返回 400，得到 %d", resp.Code)
	}
}
