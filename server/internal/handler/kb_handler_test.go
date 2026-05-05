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
	kbWrite.POST("/kb/import", kbHandler.Import)

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

func TestKBHandler_Import_JSONWrapper(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "counselor1", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"resources":[
		{"resource_id":"imp-001","resource_type":"FAQ","owner_scope":"school","role_scope":"[\"student\"]","title":"e5¯¼e585a5FAQ1","content":"e5¯¼e585a5e58685e5aeb91","version":"1.0","status":"published"},
		{"resource_id":"imp-002","resource_type":"Policy","owner_scope":"school","role_scope":"[\"student\"]","title":"e5¯¼e585a5Policy2","content":"e5¯¼e585a5e58685e5aeb92","version":"1.0","status":"draft"}
	]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("e6e69b 200efbc8ce5bee997 %d: %s", w.Code, w.Body.String())
	}

	var resp model.KBImportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("e8a7a3e69e90e5938de5ba94e5a4b1e8b4a5: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("e6e69b total=2efbc8ce5bee997 %d", resp.Total)
	}
	if resp.Created != 2 {
		t.Errorf("e6e69b created=2efbc8ce5bee997 %d", resp.Created)
	}
}

func TestKBHandler_Import_Idempotent(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "counselor1", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"resources":[
		{"resource_id":"idem-001","resource_type":"FAQ","owner_scope":"school","role_scope":"[\"student\"]","title":"e5b982e7ad89e6b58be8af95","content":"e58685e5aeb9","version":"1.0","status":"published"}
	]}`

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/kb/import", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	var resp1 model.KBImportResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.Created != 1 {
		t.Errorf("e9a696e6aca1e5afbce585a5e5ba94 created=1efbc8ce5bee997 %d", resp1.Created)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/kb/import", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 model.KBImportResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Skipped != 1 {
		t.Errorf("e5908ce78988e69cace9878de5a48de5afbce585a5e5ba94 skipped=1efbc8ce5bee997 %d", resp2.Skipped)
	}
}

func TestKBHandler_Import_HigherVersion(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "counselor1", Role: "counselor"}
	token, _ := middleware.GenerateToken(cfg, user)

	body1 := `{"resources":[
		{"resource_id":"ver-001","resource_type":"FAQ","owner_scope":"school","role_scope":"[\"student\"]","title":"e78988e69cace6b58be8af95","content":"v1e58685e5aeb9","version":"1.0","status":"published"}
	]}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/kb/import", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(httptest.NewRecorder(), req1)

	body2 := `{"resources":[
		{"resource_id":"ver-001","resource_type":"FAQ","owner_scope":"school","role_scope":"[\"student\"]","title":"e78988e69cace6b58be8af95v2","content":"v2e58685e5aeb9","version":"2.0","status":"published"}
	]}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/kb/import", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp model.KBImportResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp.Updated != 1 {
		t.Errorf("e9ab98e78988e69cace5afbce585a5e5ba94 updated=1efbc8ce5bee997 %d", resp.Updated)
	}
}

func setupKBBrowseTestRouter(t *testing.T) (*gin.Engine, *config.Config, *repository.KBRepo) {
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
	protected.GET("/knowledge", kbHandler.BrowseKnowledge)

	return r, cfg, kbRepo
}

func TestKBHandler_BrowseKnowledge_Empty(t *testing.T) {
	r, cfg, _ := setupKBBrowseTestRouter(t)

	user := &model.User{ID: 1, Username: "student1", Role: "student", OwnerScope: "school"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.KnowledgeBrowseResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
}

func TestKBHandler_BrowseKnowledge_WithData(t *testing.T) {
	r, cfg, kbRepo := setupKBBrowseTestRouter(t)

	// 插入已发布的知识资源
	kbRepo.Create(&model.KBResource{
		ResourceID: "browse-001", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student,counselor", Title: "奖学金办法", Content: "内容",
		Version: "1.0", Status: "published",
	})
	kbRepo.Create(&model.KBResource{
		ResourceID: "browse-002", ResourceType: "Process", OwnerScope: "school",
		RoleScope: "student", Title: "入学流程", Content: "步骤",
		Version: "1.0", Status: "published",
	})

	user := &model.User{ID: 1, Username: "student1", Role: "student", OwnerScope: "school"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.KnowledgeBrowseResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) < 1 {
		t.Errorf("期望至少 1 个分组，得到 %d", len(resp.Data))
	}
}

func TestKBHandler_BrowseKnowledge_WithTypeFilter(t *testing.T) {
	r, cfg, kbRepo := setupKBBrowseTestRouter(t)

	kbRepo.Create(&model.KBResource{
		ResourceID: "bf-001", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: "student", Title: "政策A", Content: "内容",
		Version: "1.0", Status: "published",
	})
	kbRepo.Create(&model.KBResource{
		ResourceID: "bf-002", ResourceType: "Process", OwnerScope: "school",
		RoleScope: "student", Title: "流程B", Content: "内容",
		Version: "1.0", Status: "published",
	})

	user := &model.User{ID: 1, Username: "student1", Role: "student", OwnerScope: "school"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge?type=Policy", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.KnowledgeBrowseResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	// 应该只返回 Policy 类型
	if cards, ok := resp.Data["Policy"]; !ok || len(cards) == 0 {
		t.Errorf("期望返回 Policy 分组，得到 %v", resp.Data)
	}
	if _, ok := resp.Data["Process"]; ok {
		t.Error("不应返回 Process 分组")
	}
}

func TestKBHandler_BrowseKnowledge_Unauthenticated(t *testing.T) {
	r, _, _ := setupKBBrowseTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestKBHandler_Import_StudentForbidden(t *testing.T) {
	r, cfg := setupKBTestRouter(t)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"resources":[{"resource_id":"x","resource_type":"FAQ","title":"x","content":"x"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("student e5ba94e8a2abe68b92e7bb9defbc8ce69c9fe69b 403 e5bee997 %d", w.Code)
	}
}
