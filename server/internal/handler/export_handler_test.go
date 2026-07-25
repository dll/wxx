package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// mockExportService 实现 exportService 接口用于测试
type mockExportService struct {
	resources []*model.KBResource
	err       error
}

func (m *mockExportService) ExportResources(ctx context.Context, resourceType, sinceCursor, callerScope, callerOwnerID string) ([]*model.KBResource, error) {
	if m.err != nil {
		return nil, m.err
	}

	// 简单过滤逻辑用于测试
	var filtered []*model.KBResource
	for _, r := range m.resources {
		if resourceType != "" && r.ResourceType != resourceType {
			continue
		}
		if sinceCursor != "" && r.UpdatedAt < sinceCursor {
			continue
		}
		filtered = append(filtered, r)
	}
	if filtered == nil {
		return []*model.KBResource{}, nil
	}
	return filtered, nil
}

// ═══ NewExportHandler 构造函数测试 ═══

func TestNewExportHandler(t *testing.T) {
	h := NewExportHandler(nil, nil)
	if h == nil {
		t.Fatal("NewExportHandler 不应返回 nil")
	}
}

func setupExportTestRouter(mockSvc exportService) (*gin.Engine, *config.Config) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "test-secret-export",
		JWTExpireHours: 2,
	}

	exportH := &ExportHandler{kbSvc: mockSvc}

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/export", exportH.Export)

	return r, cfg
}

// ═══ Export 测试 ═══

func TestExportHandler_Export_Success(t *testing.T) {
	mock := &mockExportService{
		resources: []*model.KBResource{
			{ID: 1, ResourceID: "res-001", ResourceType: "Policy", Title: "奖学金评定办法", Status: "published", UpdatedAt: time.Now().UTC().Format(time.RFC3339)},
			{ID: 2, ResourceID: "res-002", ResourceType: "Process", Title: "入学流程", Status: "published", UpdatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	r, cfg := setupExportTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}

	var body model.ExportResponse
	json.Unmarshal(resp.Body.Bytes(), &body)
	if body.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", body.Code)
	}
	if body.Manifest.Count != 2 {
		t.Errorf("期望 count=2，得到 %d", body.Manifest.Count)
	}
	if body.Manifest.Format != "json" {
		t.Errorf("期望 format=json，得到 %s", body.Manifest.Format)
	}
	if len(body.Data) != 2 {
		t.Errorf("期望 data 长度=2，得到 %d", len(body.Data))
	}
}

func TestExportHandler_Export_WithFilters(t *testing.T) {
	mock := &mockExportService{
		resources: []*model.KBResource{
			{ID: 1, ResourceID: "res-001", ResourceType: "Policy", Title: "奖学金评定办法", Status: "published", UpdatedAt: "2026-05-01T00:00:00Z"},
			{ID: 2, ResourceID: "res-002", ResourceType: "Process", Title: "入学流程", Status: "published", UpdatedAt: "2026-05-03T00:00:00Z"},
			{ID: 3, ResourceID: "res-003", ResourceType: "Policy", Title: "离校规定", Status: "published", UpdatedAt: "2025-12-01T00:00:00Z"},
		},
	}
	r, cfg := setupExportTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export?resource_type=Policy&since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}

	var body model.ExportResponse
	json.Unmarshal(resp.Body.Bytes(), &body)
	if body.Manifest.Count != 1 {
		t.Errorf("期望过滤后 count=1，得到 %d", body.Manifest.Count)
	}
}

func TestExportHandler_Export_EmptyResult(t *testing.T) {
	mock := &mockExportService{}
	r, cfg := setupExportTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}

	var body model.ExportResponse
	json.Unmarshal(resp.Body.Bytes(), &body)
	if body.Manifest.Count != 0 {
		t.Errorf("期望 count=0，得到 %d", body.Manifest.Count)
	}
}

func TestExportHandler_Export_Unauthenticated(t *testing.T) {
	mock := &mockExportService{}
	r, _ := setupExportTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

func TestExportHandler_Export_ServiceError(t *testing.T) {
	mock := &mockExportService{err: errors.New("数据库查询超时")}
	r, cfg := setupExportTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，得到 %d: %s", resp.Code, resp.Body.String())
	}
}
