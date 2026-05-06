package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

func setupEmotionTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDBFull(t)
	t.Cleanup(func() { db.Close() })

	// 执行情感预警相关迁移（emotion_logs 需要 004 + 006 的完整 schema）
	for _, m := range []string{
		"../../migrations/004_emotion_enhance.sql",
		"../../migrations/006_fix_emotion_risk_level.sql",
	} {
		sql, err := os.ReadFile(m)
		if err != nil {
			// 尝试从 server/ 根目录查找
			sql, err = os.ReadFile(strings.TrimPrefix(m, "../../"))
			if err != nil {
				t.Fatalf("读取迁移文件 %s 失败: %v", m, err)
			}
		}
		for _, stmt := range testutil.SplitSQL(string(sql)) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("执行迁移语句失败 [%s]: %v\nSQL: %s", m, err, stmt[:min(len(stmt), 100)])
			}
		}
	}

	// 插入测试用户（emotion 查询 JOIN users 表）
	db.Exec(`INSERT INTO users (id, username, display_name, role, owner_scope, owner_id)
		VALUES (1, 'student1', '测试学生', 'student', 'college', 'cs')`)
	db.Exec(`INSERT INTO users (id, username, display_name, role, owner_scope, owner_id)
		VALUES (2, 'counselor1', '测试辅导员', 'counselor', 'college', 'cs')`)

	// 插入测试情感数据（确保有趋势数据）
	db.Exec(`INSERT INTO emotion_logs (alert_id, user_id, username, session_id, message_text, score, risk_level, analysis_json, notified, status)
		VALUES ('alert-test-001', 1, 'student1', 'sess-001', '心情不好', -0.5, 'medium', '{}', 0, 'pending')`)
	db.Exec(`INSERT INTO emotion_logs (alert_id, user_id, username, session_id, message_text, score, risk_level, analysis_json, notified, status)
		VALUES ('alert-test-002', 1, 'student1', 'sess-001', '非常焦虑', -0.8, 'high', '{}', 1, 'pending')`)

	cfg := &config.Config{
		JWTSecret:      "test-secret-emotion",
		JWTExpireHours: 2,
	}

	emotionRepo := repository.NewEmotionRepo(db)
	emotionSvc := service.NewEmotionService(emotionRepo, nil) // trends 不依赖 LLM
	emotionHandler := NewEmotionHandler(emotionSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/emotion/stats", emotionHandler.GetStats)
	protected.GET("/emotion/trends", emotionHandler.Trends)

	return r, cfg
}

func TestEmotionHandler_Trends(t *testing.T) {
	r, cfg := setupEmotionTestRouter(t)

	user := &model.User{
		ID: 1, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotion/trends?days=7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionTrendResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("data 不应为 nil")
	}
	if resp.Data.Days != 7 {
		t.Errorf("期望 days=7，得到 %d", resp.Data.Days)
	}
	if len(resp.Data.Points) == 0 {
		t.Error("趋势数据点不应为空")
	}
}

func TestEmotionHandler_GetStats(t *testing.T) {
	r, cfg := setupEmotionTestRouter(t)

	user := &model.User{
		ID: 1, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotion/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("data 不应为 nil")
	}
}

// ═══ 全路由测试（Analyze / ListAlerts / UpdateAlert） ═══

func setupEmotionFullTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDBFull(t)
	t.Cleanup(func() { db.Close() })

	// 执行情感预警相关迁移
	for _, m := range []string{
		"../../migrations/004_emotion_enhance.sql",
		"../../migrations/006_fix_emotion_risk_level.sql",
	} {
		sql, err := os.ReadFile(m)
		if err != nil {
			sql, err = os.ReadFile(strings.TrimPrefix(m, "../../"))
			if err != nil {
				t.Fatalf("读取迁移文件 %s 失败: %v", m, err)
			}
		}
		for _, stmt := range testutil.SplitSQL(string(sql)) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("执行迁移语句失败 [%s]: %v", m, err)
			}
		}
	}

	// 插入测试用户
	db.Exec(`INSERT INTO users (id, username, display_name, role, owner_scope, owner_id)
		VALUES (1, "student1", "测试学生", "student", "college", "cs")`)
	db.Exec(`INSERT INTO users (id, username, display_name, role, owner_scope, owner_id)
		VALUES (2, "counselor1", "测试辅导员", "counselor", "college", "cs")`)

	// 插入测试情感数据
	db.Exec(`INSERT INTO emotion_logs (alert_id, user_id, username, session_id, message_text, score, risk_level, analysis_json, notified, status)
		VALUES ("alert-test-001", 1, "student1", "sess-001", "心情不好", -0.5, "medium", "{}", 0, "pending")`)
	db.Exec(`INSERT INTO emotion_logs (alert_id, user_id, username, session_id, message_text, score, risk_level, analysis_json, notified, status)
		VALUES ("alert-test-002", 1, "student1", "sess-001", "非常焦虑", -0.8, "high", "{}", 1, "pending")`)

	cfg := &config.Config{
		JWTSecret:      "test-secret-emo-full",
		JWTExpireHours: 2,
	}

	// 使用 Mock LLM 客户端
	mockLLM := llm.NewMockClient("test-emo")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Content:      `{"score": -0.7, "risk_level": "high", "emotions": ["焦虑", "压力"], "keywords": ["焦虑"], "reasoning": "用户表达了明显的焦虑情绪"}`,
			FinishReason: "stop",
			PromptTokens: 50,
			OutputTokens: 80,
		}, nil
	}

	emotionRepo := repository.NewEmotionRepo(db)
	emotionSvc := service.NewEmotionService(emotionRepo, mockLLM)
	emotionHandler := NewEmotionHandler(emotionSvc)

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.POST("/emotion/analyze", emotionHandler.Analyze)
	protected.GET("/emotion/alerts", emotionHandler.ListAlerts)
	protected.PUT("/emotion/alerts/:id", emotionHandler.UpdateAlert)
	protected.GET("/emotion/stats", emotionHandler.GetStats)
	protected.GET("/emotion/trends", emotionHandler.Trends)

	return r, cfg
}

// ═══ Analyze 测试 ═══

func TestEmotionHandler_Analyze_Success(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 1, Username: "student1", Role: "student",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"message_text":"我最近压力很大，睡不着觉","session_id":"sess-001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/emotion/analyze", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionAnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("分析结果不应为 nil")
	}
	if resp.Data.RiskLevel != "high" {
		t.Errorf("期望 risk_level=high，得到 %s", resp.Data.RiskLevel)
	}
}

func TestEmotionHandler_Analyze_MissingFields(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/emotion/analyze", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少必填字段应返回 400，得到 %d", w.Code)
	}
}

func TestEmotionHandler_Analyze_Unauthenticated(t *testing.T) {
	r, _ := setupEmotionFullTestRouter(t)

	body := `{"message_text":"test","session_id":"sess-001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/emotion/analyze", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

// ═══ ListAlerts 测试 ═══

func TestEmotionHandler_ListAlerts_Success(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 2, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotion/alerts?page=1&page_size=20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
	if resp.Total < 1 {
		t.Errorf("期望至少 1 条告警，得到 total=%d", resp.Total)
	}
}

func TestEmotionHandler_ListAlerts_WithFilters(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 2, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotion/alerts?risk_level=high&status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("过滤 high 后应只有 1 条，得到 total=%d", resp.Total)
	}
}

func TestEmotionHandler_ListAlerts_EmptyResult(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 2, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotion/alerts?risk_level=urgent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionListResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("无 urgent 告警时应返回 total=0，得到 %d", resp.Total)
	}
}

func TestEmotionHandler_ListAlerts_Unauthenticated(t *testing.T) {
	r, _ := setupEmotionFullTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotion/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}

// ═══ UpdateAlert 测试 ═══

func TestEmotionHandler_UpdateAlert_Success(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 2, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"status":"acknowledged"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/emotion/alerts/alert-test-001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", w.Code, w.Body.String())
	}

	var resp model.EmotionAnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", resp.Code)
	}
}

func TestEmotionHandler_UpdateAlert_InvalidStatus(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 2, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"status":"invalid"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/emotion/alerts/alert-test-001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效 status 应返回 400，得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestEmotionHandler_UpdateAlert_NotFound(t *testing.T) {
	r, cfg := setupEmotionFullTestRouter(t)

	user := &model.User{
		ID: 2, Username: "counselor1", Role: "counselor",
		OwnerScope: "college", OwnerID: "cs",
	}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"status":"acknowledged"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/emotion/alerts/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("不存在的告警应返回 500，得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestEmotionHandler_UpdateAlert_Unauthenticated(t *testing.T) {
	r, _ := setupEmotionFullTestRouter(t)

	body := `{"status":"acknowledged"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/emotion/alerts/alert-test-001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", w.Code)
	}
}
