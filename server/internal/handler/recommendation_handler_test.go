package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

func setupRecTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		JWTSecret:      "test-secret-rec",
		JWTExpireHours: 2,
	}

	kbRepo := repository.NewKBRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	recSvc := service.NewRecommendationService(kbRepo, messageRepo)
	recHandler := NewRecommendationHandler(recSvc)

	// 插入种子数据
	for i := 0; i < 10; i++ {
		kbRepo.Create(&model.KBResource{
			ResourceID:   "rec-test-" + string(rune('0'+i%10)),
			ResourceType: "Policy",
			OwnerScope:   "school",
			RoleScope:    `["student"]`,
			Version:      "1.0",
			Status:       "published",
			Title:        "测试政策文档",
			Summary:      "包含奖学金和入学信息",
			Content:      "详细政策内容",
			UpdatedBy:    "test",
		})
	}

	r := gin.New()
	r.Use(middleware.TraceID())

	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/recommendations", recHandler.GetRecommendations)

	return r, cfg
}

func TestRecHandler_GetRecommendations_Success(t *testing.T) {
	r, cfg := setupRecTestRouter(t)

	user := &model.User{ID: 1, Username: "student01", Role: "student", OwnerScope: "school", OwnerID: "school-1"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    []service.RecommendItem `json:"data"`
		Total   int                     `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
	if resp.Total == 0 {
		t.Error("冷启动应返回内容")
	}
}

func TestRecHandler_GetRecommendations_Unauthenticated(t *testing.T) {
	r, _ := setupRecTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

func TestRecHandler_GetRecommendations_DefaultLimit(t *testing.T) {
	r, cfg := setupRecTestRouter(t)

	user := &model.User{ID: 1, Username: "student01", Role: "student", OwnerScope: "school", OwnerID: "school-1"}
	token, _ := middleware.GenerateToken(cfg, user)

	// 不传 limit 参数
	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}

	var resp struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    []service.RecommendItem `json:"data"`
		Total   int                     `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if resp.Total > 10 {
		t.Errorf("默认 limit=10，不应超过 10 条，得到 %d", resp.Total)
	}
}

func TestRecHandler_GetRecommendations_InvalidLimit(t *testing.T) {
	r, cfg := setupRecTestRouter(t)

	user := &model.User{ID: 1, Username: "student01", Role: "student", OwnerScope: "school", OwnerID: "school-1"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?limit=999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}

	var resp struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Data    []service.RecommendItem `json:"data"`
		Total   int                     `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if resp.Total > 10 {
		t.Errorf("limit>50 应重置为 10，得到 %d 条", resp.Total)
	}
}
