package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// mockAgentService 实现 agentService 接口用于测试
type mockAgentService struct {
	agents []*model.Agent
	err    error
}

func (m *mockAgentService) List() ([]*model.Agent, error) {
	return m.agents, m.err
}

func (m *mockAgentService) Create(req *model.AgentCreateRequest) (*model.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	a := &model.Agent{
		ID:            1,
		AgentID:       req.AgentID,
		Name:          req.Name,
		Description:   req.Description,
		AgentType:     req.AgentType,
		SystemPrompt:  req.SystemPrompt,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
		Temperature:   req.Temperature,
		MaxTokens:     req.MaxTokens,
		Status:        "active",
	}
	return a, nil
}

func (m *mockAgentService) Get(agentID string) (*model.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, a := range m.agents {
		if a.AgentID == agentID {
			return a, nil
		}
	}
	return nil, errors.New("智能体 " + agentID + " 不存在")
}

func (m *mockAgentService) Update(agentID string, req *model.AgentUpdateRequest) (*model.Agent, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, err := m.Get(agentID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Status != nil {
		a.Status = *req.Status
	}
	return a, nil
}

func (m *mockAgentService) Delete(agentID string) error {
	if m.err != nil {
		return m.err
	}
	_, err := m.Get(agentID)
	return err
}

// ═══ NewAgentHandler 构造函数测试 ═══

func TestNewAgentHandler(t *testing.T) {
	h := NewAgentHandler(nil)
	if h == nil {
		t.Fatal("NewAgentHandler 不应返回 nil")
	}
}

func setupAgentTestRouter(mockSvc agentService) (*gin.Engine, *config.Config) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "test-secret-agent",
		JWTExpireHours: 2,
	}

	agentH := &AgentHandler{agentSvc: mockSvc}

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.GET("/agents", agentH.List)
	protected.POST("/agents", agentH.Create)
	protected.GET("/agents/:id", agentH.Get)
	protected.PUT("/agents/:id", agentH.Update)
	protected.DELETE("/agents/:id", agentH.Delete)

	return r, cfg
}

// ═══ List 测试 ═══

func TestAgentHandler_List_Success(t *testing.T) {
	mock := &mockAgentService{
		agents: []*model.Agent{
			{ID: 1, AgentID: "qa-default", Name: "默认问答", AgentType: "qa", Status: "active"},
			{ID: 2, AgentID: "emotion-v1", Name: "情感分析", AgentType: "emotion", Status: "active"},
		},
	}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}

	var body model.AgentListResponse
	json.Unmarshal(resp.Body.Bytes(), &body)
	if body.Code != 0 {
		t.Errorf("期望 code=0，得到 %d", body.Code)
	}
	if body.Total != 2 {
		t.Errorf("期望 total=2，得到 %d", body.Total)
	}
}

func TestAgentHandler_List_Empty(t *testing.T) {
	mock := &mockAgentService{}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", resp.Code)
	}

	var body model.AgentListResponse
	json.Unmarshal(resp.Body.Bytes(), &body)
	if body.Total != 0 {
		t.Errorf("期望 total=0，得到 %d", body.Total)
	}
}

func TestAgentHandler_List_Unauthenticated(t *testing.T) {
	mock := &mockAgentService{}
	r, _ := setupAgentTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

func TestAgentHandler_List_ServiceError(t *testing.T) {
	mock := &mockAgentService{err: errors.New("数据库连接失败")}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

// ═══ Create 测试 ═══

func TestAgentHandler_Create_Success(t *testing.T) {
	mock := &mockAgentService{}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"agent_id":"qa-v2","name":"问答 V2","agent_type":"qa","description":"新版本问答"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}

	var respBody model.AgentDetailResponse
	json.Unmarshal(resp.Body.Bytes(), &respBody)
	if respBody.Data.AgentID != "qa-v2" {
		t.Errorf("期望 agent_id=qa-v2，得到 %s", respBody.Data.AgentID)
	}
}

func TestAgentHandler_Create_InvalidJSON(t *testing.T) {
	mock := &mockAgentService{}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAgentHandler_Create_Unauthenticated(t *testing.T) {
	mock := &mockAgentService{}
	r, _ := setupAgentTestRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

// ═══ Get 测试 ═══

func TestAgentHandler_Get_Success(t *testing.T) {
	mock := &mockAgentService{
		agents: []*model.Agent{
			{ID: 1, AgentID: "qa-default", Name: "默认问答", AgentType: "qa", Status: "active"},
		},
	}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/qa-default", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAgentHandler_Get_NotFound(t *testing.T) {
	mock := &mockAgentService{}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("期望 404，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAgentHandler_Get_Unauthenticated(t *testing.T) {
	mock := &mockAgentService{}
	r, _ := setupAgentTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/qa-default", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

// ═══ Update 测试 ═══

func TestAgentHandler_Update_Success(t *testing.T) {
	mock := &mockAgentService{
		agents: []*model.Agent{
			{ID: 1, AgentID: "qa-default", Name: "默认问答", AgentType: "qa", Status: "active"},
		},
	}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"name":"问答升级版","status":"inactive"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/qa-default", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAgentHandler_Update_NotFound(t *testing.T) {
	mock := &mockAgentService{}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/agents/nonexistent", bytes.NewBufferString(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

// ═══ Delete 测试 ═══

func TestAgentHandler_Delete_Success(t *testing.T) {
	mock := &mockAgentService{
		agents: []*model.Agent{
			{ID: 1, AgentID: "qa-default", Name: "默认问答", AgentType: "qa", Status: "active"},
		},
	}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/qa-default", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAgentHandler_Delete_NotFound(t *testing.T) {
	mock := &mockAgentService{}
	r, cfg := setupAgentTestRouter(mock)

	user := &model.User{ID: 1, Username: "admin1", Role: "sys_admin"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}
