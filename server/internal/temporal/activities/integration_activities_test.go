package activities

import (
	"context"
	"encoding/json"
	"testing"
)

func TestIntegrationProxyActivity_Xuegong(t *testing.T) {
	expectedBody := json.RawMessage(`{"students":[{"name":"张三"}]}`)

	acts := &IntegrationActivities{
		ProxyXuegong: func(path string, query map[string]string) (json.RawMessage, error) {
			if path != "/api/students" {
				t.Errorf("期望 path=/api/students，得到 %s", path)
			}
			return expectedBody, nil
		},
	}

	result, err := acts.IntegrationProxyActivity(context.Background(), IntegrationProxyInput{
		System: "xuegong",
		Path:   "/api/students",
		Query:  map[string]string{"page": "1"},
	})
	if err != nil {
		t.Fatalf("代理失败: %v", err)
	}
	if result.BodyJSON != string(expectedBody) {
		t.Errorf("BodyJSON 不匹配")
	}
}

func TestIntegrationProxyActivity_YBT(t *testing.T) {
	expectedBody := json.RawMessage(`{"data":[]}`)

	acts := &IntegrationActivities{
		ProxyYBT: func(path string, query map[string]string) (json.RawMessage, error) {
			return expectedBody, nil
		},
	}

	result, err := acts.IntegrationProxyActivity(context.Background(), IntegrationProxyInput{
		System: "ybt",
		Path:   "/api/data",
	})
	if err != nil {
		t.Fatalf("代理失败: %v", err)
	}
	if result.BodyJSON != string(expectedBody) {
		t.Errorf("BodyJSON 不匹配")
	}
}

func TestIntegrationProxyActivity_UnknownSystem(t *testing.T) {
	acts := &IntegrationActivities{}

	_, err := acts.IntegrationProxyActivity(context.Background(), IntegrationProxyInput{
		System: "unknown",
		Path:   "/api/test",
	})
	if err == nil {
		t.Error("未知系统应返回错误")
	}
}

func TestIntegrationProxyActivity_NilFunction(t *testing.T) {
	acts := &IntegrationActivities{
		ProxyXuegong: nil, // 未配置
	}

	_, err := acts.IntegrationProxyActivity(context.Background(), IntegrationProxyInput{
		System: "xuegong",
		Path:   "/api/test",
	})
	if err == nil {
		t.Error("未配置代理函数时应返回错误")
	}
}
