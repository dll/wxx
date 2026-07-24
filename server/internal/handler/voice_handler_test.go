package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// mockVoiceClient 实现 voiceService 接口用于测试
type mockVoiceClient struct {
	asrText  string
	asrErr   error
	ttsAudio []byte
	ttsErr   error
}

func (m *mockVoiceClient) ASR(_ context.Context, _ []byte) (string, error) {
	return m.asrText, m.asrErr
}

func (m *mockVoiceClient) TTS(_ context.Context, _ string, _ string) ([]byte, error) {
	return m.ttsAudio, m.ttsErr
}

// ═══ NewVoiceHandler 构造函数测试 ═══

func TestNewVoiceHandler(t *testing.T) {
	h := NewVoiceHandler(nil)
	if h == nil {
		t.Fatal("NewVoiceHandler 不应返回 nil")
	}
}

func setupVoiceTestRouter(mockSvc voiceService) (*gin.Engine, *config.Config) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret:      "test-secret-voice",
		JWTExpireHours: 2,
	}

	voiceH := &VoiceHandler{voiceSvc: mockSvc}

	r := gin.New()
	r.Use(middleware.TraceID())
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	protected.POST("/voice/asr", voiceH.ASR)
	protected.POST("/voice/tts", voiceH.TTS)

	return r, cfg
}

// ═══ ASR 测试 ═══

func TestVoiceHandler_ASR_Success(t *testing.T) {
	mock := &mockVoiceClient{asrText: "你好蔚小芯"}
	r, cfg := setupVoiceTestRouter(mock)

	// 构造 multipart 请求
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("audio", "test.pcm")
	part.Write([]byte{0x00, 0x01, 0x02, 0x03}) // 伪造 PCM 数据
	w.Close()

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/asr", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &body)
	if body["code"].(float64) != 0 {
		t.Errorf("期望 code=0，得到 %v", body["code"])
	}
	data := body["data"].(map[string]interface{})
	if data["text"].(string) != "你好蔚小芯" {
		t.Errorf("期望 text=你好蔚小芯，得到 %s", data["text"])
	}
}

func TestVoiceHandler_ASR_NoFile(t *testing.T) {
	mock := &mockVoiceClient{}
	r, cfg := setupVoiceTestRouter(mock)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/asr", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestVoiceHandler_ASR_EmptyFile(t *testing.T) {
	mock := &mockVoiceClient{}
	r, cfg := setupVoiceTestRouter(mock)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("audio", "empty.pcm")
	// 不写入任何数据，得到空文件
	part.Write([]byte{})
	w.Close()

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/asr", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestVoiceHandler_ASR_Unauthenticated(t *testing.T) {
	mock := &mockVoiceClient{}
	r, _ := setupVoiceTestRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/asr", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

func TestVoiceHandler_ASR_ClientError(t *testing.T) {
	mock := &mockVoiceClient{asrErr: errors.New("讯飞 ASR 超时")}
	r, cfg := setupVoiceTestRouter(mock)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("audio", "test.pcm")
	part.Write([]byte{0x00, 0x01})
	w.Close()

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/asr", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

// ═══ TTS 测试 ═══

func TestVoiceHandler_TTS_Success(t *testing.T) {
	mockAudio := []byte{0xFF, 0xFB, 0x90} // MP3 帧头
	mock := &mockVoiceClient{ttsAudio: mockAudio}
	r, cfg := setupVoiceTestRouter(mock)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"text":"你好世界"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/tts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "audio/mpeg" {
		t.Errorf("期望 Content-Type=audio/mpeg，得到 %s", resp.Header().Get("Content-Type"))
	}
	if !bytes.Equal(resp.Body.Bytes(), mockAudio) {
		t.Errorf("返回的音频数据与 mock 不匹配")
	}
}

func TestVoiceHandler_TTS_MissingText(t *testing.T) {
	mock := &mockVoiceClient{}
	r, cfg := setupVoiceTestRouter(mock)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"voice":"x_xiaoyan"}` // 缺少必填 text 字段
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/tts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestVoiceHandler_TTS_InvalidJSON(t *testing.T) {
	mock := &mockVoiceClient{}
	r, cfg := setupVoiceTestRouter(mock)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/tts", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("期望 400，得到 %d: %s", resp.Code, resp.Body.String())
	}
}

func TestVoiceHandler_TTS_Unauthenticated(t *testing.T) {
	mock := &mockVoiceClient{}
	r, _ := setupVoiceTestRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/tts", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，得到 %d", resp.Code)
	}
}

func TestVoiceHandler_TTS_ClientError(t *testing.T) {
	mock := &mockVoiceClient{ttsErr: errors.New("讯飞 TTS 超时")}
	r, cfg := setupVoiceTestRouter(mock)

	user := &model.User{ID: 1, Username: "student1", Role: "student"}
	token, _ := middleware.GenerateToken(cfg, user)

	body := `{"text":"你好世界"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/tts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，得到 %d: %s", resp.Code, resp.Body.String())
	}
}
