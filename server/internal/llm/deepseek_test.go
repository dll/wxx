package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
)

func TestDeepSeekClient_Name(t *testing.T) {
	cfg := &config.Config{DeepSeekModel: "deepseek-v4-pro"}
	client := NewDeepSeekClient(cfg)

	if client.Name() != "deepseek" {
		t.Errorf("期望 Name=deepseek，得到 %s", client.Name())
	}
}

func TestDeepSeekClient_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != http.MethodPost {
			t.Errorf("期望 POST，得到 %s", r.Method)
		}
		// 验证 Content-Type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("期望 Content-Type=application/json，得到 %s", ct)
		}
		// 验证 Authorization
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			t.Errorf("期望 Authorization='Bearer test-api-key'，得到 %s", auth)
		}

		// 返回模拟响应
		resp := openAIResponse{
			Choices: []openAIChoice{
				{
					Message:      ChatMessage{Role: "assistant", Content: "DeepSeek 回答"},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{
				PromptTokens:     50,
				CompletionTokens: 100,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		DeepSeekAPIKey:  "test-api-key",
		DeepSeekBaseURL: server.URL,
		DeepSeekModel:   "deepseek-v4-pro",
	}
	client := NewDeepSeekClient(cfg)

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "你好"}},
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if resp.Content != "DeepSeek 回答" {
		t.Errorf("期望 DeepSeek 回答，得到 %s", resp.Content)
	}
	if resp.PromptTokens != 50 {
		t.Errorf("期望 PromptTokens=50，得到 %d", resp.PromptTokens)
	}
	if resp.OutputTokens != 100 {
		t.Errorf("期望 OutputTokens=100，得到 %d", resp.OutputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("期望 FinishReason=stop，得到 %s", resp.FinishReason)
	}
}

func TestDeepSeekClient_Chat_DefaultParams(t *testing.T) {
	var receivedBody openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := openAIResponse{
			Choices: []openAIChoice{
				{
					Message:      ChatMessage{Role: "assistant", Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		DeepSeekBaseURL: server.URL,
	}
	client := NewDeepSeekClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	// 验证默认参数
	if receivedBody.Temperature != 0.7 {
		t.Errorf("期望默认 Temperature=0.7，得到 %f", receivedBody.Temperature)
	}
	if receivedBody.MaxTokens != 2048 {
		t.Errorf("期望默认 MaxTokens=2048，得到 %d", receivedBody.MaxTokens)
	}
}

func TestDeepSeekClient_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		DeepSeekBaseURL: server.URL,
	}
	client := NewDeepSeekClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("HTTP 500 应返回错误")
	}
}

func TestDeepSeekClient_Chat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []openAIChoice{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		DeepSeekBaseURL: server.URL,
	}
	client := NewDeepSeekClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("空 choices 应返回错误")
	}
}

func TestDeepSeekClient_Chat_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	cfg := &config.Config{
		DeepSeekBaseURL: server.URL,
	}
	client := NewDeepSeekClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

func TestDeepSeekClient_Chat_CustomTemperature(t *testing.T) {
	var receivedBody openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := openAIResponse{
			Choices: []openAIChoice{
				{
					Message:      ChatMessage{Role: "assistant", Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		DeepSeekBaseURL: server.URL,
	}
	client := NewDeepSeekClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
		Temperature: 0.3,
		MaxTokens:   512,
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	if receivedBody.Temperature != 0.3 {
		t.Errorf("期望 Temperature=0.3，得到 %f", receivedBody.Temperature)
	}
	if receivedBody.MaxTokens != 512 {
		t.Errorf("期望 MaxTokens=512，得到 %d", receivedBody.MaxTokens)
	}
}

func TestDeepSeekClient_Chat_ConnectionRefused(t *testing.T) {
	cfg := &config.Config{
		DeepSeekBaseURL: "http://127.0.0.1:19998",
	}
	client := NewDeepSeekClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("连接失败应返回错误")
	}
}
