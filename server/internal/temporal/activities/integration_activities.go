package activities

import (
	"context"
	"encoding/json"
	"fmt"
)

// IntegrationActivities 校外系统代理相关活动集合
// 使用函数字段注入而非直接引用 service 包（避免循环依赖）
type IntegrationActivities struct {
	// ProxyXuegong 代理学工系统请求
	ProxyXuegong func(path string, query map[string]string) (json.RawMessage, error)
	// ProxyYBT 代理一表通请求
	ProxyYBT func(path string, query map[string]string) (json.RawMessage, error)
}

// IntegrationProxyInput 代理活动输入
type IntegrationProxyInput struct {
	System string            `json:"system"` // xuegong / ybt
	Path   string            `json:"path"`
	Query  map[string]string `json:"query"`
}

// IntegrationProxyOutput 代理活动输出
type IntegrationProxyOutput struct {
	BodyJSON string `json:"body_json"`
}

// IntegrationProxyActivity 代理转发外部系统请求
func (a *IntegrationActivities) IntegrationProxyActivity(ctx context.Context, input IntegrationProxyInput) (*IntegrationProxyOutput, error) {
	var respBody json.RawMessage
	var err error

	switch input.System {
	case "xuegong":
		if a.ProxyXuegong == nil {
			return nil, fmt.Errorf("学工系统代理未配置")
		}
		respBody, err = a.ProxyXuegong(input.Path, input.Query)
	case "ybt":
		if a.ProxyYBT == nil {
			return nil, fmt.Errorf("一表通代理未配置")
		}
		respBody, err = a.ProxyYBT(input.Path, input.Query)
	default:
		return nil, fmt.Errorf("未知系统: %s", input.System)
	}

	if err != nil {
		return nil, err
	}

	return &IntegrationProxyOutput{BodyJSON: string(respBody)}, nil
}
