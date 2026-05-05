package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
		for _, stmt := range splitSQLStatements(string(sql)) {
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

// helpers

func splitSQLStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inTrigger := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "CREATE TRIGGER") {
			inTrigger = true
		}
		current.WriteString(line)
		current.WriteString("\n")
		if inTrigger && strings.HasSuffix(trimmed, "END;") {
			statements = append(statements, current.String())
			current.Reset()
			inTrigger = false
			continue
		}
		if !inTrigger && strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		statements = append(statements, remaining)
	}
	return statements
}
