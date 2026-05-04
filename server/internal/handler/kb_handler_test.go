package handler

import (
	"encoding/json"
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

func setupKBTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		JWTSecret:      "test-secret-kb",
		JWTExpireHours: 2,
	}

	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo)
	kbHandler := NewKBHandler(kbSvc)

	r := gin.New()
	r.Use(middleware.TraceID())

	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/kb/resources", kbHandler.ListResources)
	protected.GET("/kb/resources/:id", kbHandler.GetResource)

	// KB 写操作需要 counselor 及以上权限
	kbWrite := r.Group("/api/v1")
	kbWrite.Use(middleware.JWTAuth(cfg))
	kbWrite.Use(middleware.RequireRole("counselor"))
	kbWrite.POST("/kb/resources", kbHandler.CreateResource)
	kbWrite.PUT("/kb/resources/:id", kbHandler.UpdateResource)

	return r, cfg
}

func TestKBHandler_ListResources_Empty(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "admin", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/kb/resources?page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}

	var resp model.KBListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("期望 total=0，得到 %d", resp.Total)
	}
}

func TestKBHandler_ListResources_Unauthenticated(t *testing.T) {
	r, _ := setupKBTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/kb/resources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestKBHandler_CreateResource_Success(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "counselor1", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{
		"resource_type": "Policy",
		"owner_scope": "school",
		"role_scope": "[\"student\",\"counselor\"]",
		"title": "测试政策",
		"content": "这是测试内容",
		"summary": "测试摘要"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/resources", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望 201，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.KBDetailResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatal("应返回创建的资源")
	}
	if resp.Data.Title != "测试政策" {
		t.Errorf("期望 title=测试政策，得到 %s", resp.Data.Title)
	}
}

func TestKBHandler_CreateResource_StudentForbidden(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"resource_type":"Policy","owner_scope":"school","role_scope":"[\"student\"]","title":"测试","content":"内容"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/resources", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("student 应被拒绝，期望 403 得到 %d", w.Code)
	}
}

func TestKBHandler_CreateResource_ValidationError(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "c1", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	// 缺少必填字段 title
	body := `{"resource_type":"Policy","owner_scope":"school","role_scope":"[\"student\"]"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/resources", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少必填字段应返回 400，得到 %d", w.Code)
	}
}

func TestKBHandler_GetResource_NotFound(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "u1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/kb/resources/nonexistent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，得到 %d", w.Code)
	}
}

func TestKBHandler_UpdateResource_Success(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "c1", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	// 先创建
	createBody := `{
		"resource_type": "Policy",
		"owner_scope": "school",
		"role_scope": "[\"student\"]",
		"title": "原始标题",
		"content": "原始内容"
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/kb/resources", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var createResp model.KBDetailResponse
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	resourceID := createResp.Data.ResourceID

	// 然后更新
	updateBody := `{"title": "新标题", "status": "published"}`
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/kb/resources/"+resourceID,
		strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", updateW.Code, updateW.Body.String())
	}

	var updateResp model.KBDetailResponse
	json.Unmarshal(updateW.Body.Bytes(), &updateResp)
	if updateResp.Data.Title != "新标题" {
		t.Errorf("期望 title=新标题，得到 %s", updateResp.Data.Title)
	}
	if updateResp.Data.Status != "published" {
		t.Errorf("期望 status=published，得到 %s", updateResp.Data.Status)
	}
}
