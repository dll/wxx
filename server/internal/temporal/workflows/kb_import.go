package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ── 知识导入工作流输入/输出类型 ──

// KBImportInput 知识导入工作流输入
type KBImportInput struct {
	NDJSONData string `json:"ndjson_data"` // 原始 NDJSON 数据
	Username   string `json:"username"`    // 操作者
}

// KBImportOutput 知识导入工作流输出
type KBImportOutput struct {
	ImportResultJSON string `json:"import_result_json"` // KBImportResponse JSON
}

// KBImportWorkflow 知识批量导入工作流
// 对批量 NDJSON 导入进行重试和心跳保护（适用于大文件长时间操作）
func KBImportWorkflow(ctx workflow.Context, input KBImportInput) (*KBImportOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("知识导入工作流开始", "username", input.Username)

	// 知识导入重试 2 次（导入失败通常为数据格式问题，过多重试无意义）
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 1.5,
		MaximumAttempts:    2,
	}

	// 大文件导入可能耗时较长
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy:         retryPolicy,
		HeartbeatTimeout:    30 * time.Second, // 长任务心跳
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var result KBImportOutput
	err := workflow.ExecuteActivity(ctx, "KBImportActivity", input).Get(ctx, &result)
	if err != nil {
		logger.Error("知识导入失败", "error", err)
		return nil, err
	}

	logger.Info("知识导入工作流完成", "username", input.Username)
	return &result, nil
}
