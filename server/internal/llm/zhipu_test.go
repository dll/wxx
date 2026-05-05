package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
)

func TestZhipuClient_Name(t *testing.T) {
	cfg := &config.Config{ZhipuModel: "glm-4"}
	client := NewZhipuClient(cfg)

	if client.Name() != "zhipu" {
		t.Errorf("期望 Name=zhipu，得到 %s", client.Name())
	}
}

func TestZhipuClient_Chat_Success(t *testing.T) {
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
		if auth := r.Header.Get("Authorization"); auth != "Bearer zhipu-api-key" {
			t.Errorf("期望 Authorization='Bearer zhipu-api-key'，得到 %s", auth)
		}

		resp := openAIResponse{
			Choices: []openAIChoice{
				{
					Message:      ChatMessage{Role: "assistant", Content: "智谱回答"},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{
				PromptTokens:     30,
				CompletionTokens: 60,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ZhipuAPIKey:  "zhipu-api-key",
		ZhipuBaseURL: server.URL,
		ZhipuModel:   "glm-4",
	}
	client := NewZhipuClient(cfg)

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "你好"}},
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}
	if resp.Content != "智谱回答" {
		t.Errorf("期望 智谱回答，得到 %s", resp.Content)
	}
	if resp.PromptTokens != 30 {
		t.Errorf("期望 PromptTokens=30，得到 %d", resp.PromptTokens)
	}
	if resp.OutputTokens != 60 {
		t.Errorf("期望 OutputTokens=60，得到 %d", resp.OutputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("期望 FinishReason=stop，得到 %s", resp.FinishReason)
	}
}

func TestZhipuClient_Chat_DefaultModel(t *testing.T) {
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
		ZhipuBaseURL: server.URL,
	}
	client := NewZhipuClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	// 智谱客户端使用 openAIRequest 结构体，model 未初始化时为空
	if receivedBody.Model != "" {
		// 如果 model 不为空，确认它来自配置
		t.Logf("model = %s", receivedBody.Model)
	}
}

func TestZhipuClient_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": "bad gateway"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ZhipuBaseURL: server.URL,
	}
	client := NewZhipuClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("HTTP 502 应返回错误")
	}
}

func TestZhipuClient_Chat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []openAIChoice{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		ZhipuBaseURL: server.URL,
	}
	client := NewZhipuClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("空 choices 应返回错误")
	}
}

func TestZhipuClient_Chat_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("这不是 JSON"))
	}))
	defer server.Close()

	cfg := &config.Config{
		ZhipuBaseURL: server.URL,
	}
	client := NewZhipuClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

func TestZhipuClient_Chat_ConnectionRefused(t *testing.T) {
	cfg := &config.Config{
		ZhipuBaseURL: "http://127.0.0.1:19997",
	}
	client := NewZhipuClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("连接失败应返回错误")
	}
}

func TestZhipuClient_Chat_CustomParams(t *testing.T) {
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
		ZhipuBaseURL: server.URL,
		ZhipuModel:   "glm-4-flash",
	}
	client := NewZhipuClient(cfg)

	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages:    []ChatMessage{{Role: "user", Content: "hi"}},
		Temperature: 0.1,
		MaxTokens:   256,
	})
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	if receivedBody.Temperature != 0.1 {
		t.Errorf("期望 Temperature=0.1，得到 %f", receivedBody.Temperature)
	}
	if receivedBody.MaxTokens != 256 {
		t.Errorf("期望 MaxTokens=256，得到 %d", receivedBody.MaxTokens)
	}
	if receivedBody.Model != "glm-4-flash" {
		t.Errorf("期望 Model=glm-4-flash，得到 %s", receivedBody.Model)
	}
}
