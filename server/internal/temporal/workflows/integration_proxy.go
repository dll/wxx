package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ── 校外系统代理工作流输入/输出类型 ──

// IntegrationProxyInput 代理请求工作流输入
type IntegrationProxyInput struct {
	System string            `json:"system"` // xuegong / ybt
	Path   string            `json:"path"`
	Query  map[string]string `json:"query"`
}

// IntegrationProxyOutput 代理请求工作流输出
type IntegrationProxyOutput struct {
	BodyJSON string `json:"body_json"` // 响应 JSON
}

// IntegrationProxyWorkflow 校外系统代理工作流
// 对学工/一表通 HTTP 代理请求进行重试保护
func IntegrationProxyWorkflow(ctx workflow.Context, input IntegrationProxyInput) (*IntegrationProxyOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("校外系统代理工作流开始", "system", input.System, "path", input.Path)

	// 外部 API 调用：3 次退避重试
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    10 * time.Second,
	}

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         retryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var result IntegrationProxyOutput
	err := workflow.ExecuteActivity(ctx, "IntegrationProxyActivity", input).Get(ctx, &result)
	if err != nil {
		logger.Error("代理请求失败（已重试）", "error", err, "system", input.System)
		return nil, err
	}

	logger.Info("校外系统代理工作流完成", "system", input.System)
	return &result, nil
}
