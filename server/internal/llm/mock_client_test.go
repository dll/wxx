package llm

import (
	"context"
	"errors"
	"testing"
)

func TestMockClient_Chat_Default(t *testing.T) {
	mc := NewMockClient("test-mock")

	resp, err := mc.Chat(context.Background(), &ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "你好"},
		},
	})
	if err != nil {
		t.Fatalf("默认 Chat 不应失败: %v", err)
	}
	if resp.Content != "这是模拟的回答内容。" {
		t.Errorf("期望默认回答内容，得到: %s", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("期望 FinishReason=stop，得到: %s", resp.FinishReason)
	}
}

func TestMockClient_Chat_CustomFunc(t *testing.T) {
	mc := NewMockClient("custom")

	mc.ChatFunc = func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
		return &ChatResponse{
			Content:      "自定义回答",
			FinishReason: "stop",
			PromptTokens: 10,
			OutputTokens: 20,
		}, nil
	}

	resp, err := mc.Chat(context.Background(), &ChatRequest{})
	if err != nil {
		t.Fatalf("自定义 Chat 不应失败: %v", err)
	}
	if resp.Content != "自定义回答" {
		t.Errorf("期望 自定义回答，得到: %s", resp.Content)
	}
	if resp.PromptTokens != 10 {
		t.Errorf("期望 PromptTokens=10，得到 %d", resp.PromptTokens)
	}
	if resp.OutputTokens != 20 {
		t.Errorf("期望 OutputTokens=20，得到 %d", resp.OutputTokens)
	}
}

func TestMockClient_Chat_Error(t *testing.T) {
	mc := NewMockClient("error-mock")

	mc.ChatFunc = func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
		return nil, errors.New("模拟错误")
	}

	_, err := mc.Chat(context.Background(), &ChatRequest{})
	if err == nil {
		t.Fatal("应返回错误")
	}
	if err.Error() != "模拟错误" {
		t.Errorf("期望 模拟错误，得到: %s", err.Error())
	}
}

func TestMockClient_Name(t *testing.T) {
	mc := NewMockClient("my-name")
	if mc.Name() != "my-name" {
		t.Errorf("期望 Name=my-name，得到: %s", mc.Name())
	}
}

func TestMockClient_Name_EmptyDefaults(t *testing.T) {
	mc := NewMockClient("")
	if mc.Name() != "mock" {
		t.Errorf("期望 Name=mock，得到: %s", mc.Name())
	}
}

func TestMockClient_Reset(t *testing.T) {
	mc := NewMockClient("reset-test")

	mc.ChatFunc = func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
		return &ChatResponse{Content: "自定义"}, nil
	}

	// 重置
	mc.Reset()

	// 重置后应返回默认内容
	resp, err := mc.Chat(context.Background(), &ChatRequest{})
	if err != nil {
		t.Fatalf("重置后 Chat 不应失败: %v", err)
	}
	if resp.Content != "这是模拟的回答内容。" {
		t.Errorf("重置后应返回默认内容，得到: %s", resp.Content)
	}
}
