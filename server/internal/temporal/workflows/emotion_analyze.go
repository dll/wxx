package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ── 情感分析工作流输入/输出类型 ──

// EmotionAnalyzeInput 情感分析工作流输入
type EmotionAnalyzeInput struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	SessionID   string `json:"session_id"`
	MessageText string `json:"message_text"`
}

// EmotionAnalyzeOutput 情感分析工作流输出
type EmotionAnalyzeOutput struct {
	EmotionLogJSON string `json:"emotion_log_json"` // 序列化的 EmotionLog
}

// EmotionAnalyzeWorkflow 情感分析工作流
// 调用 LLM 分析文本情感，高风险时触发预警活动
func EmotionAnalyzeWorkflow(ctx workflow.Context, input EmotionAnalyzeInput) (*EmotionAnalyzeOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("情感分析工作流开始", "user_id", input.UserID)

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

	var result EmotionAnalyzeOutput
	err := workflow.ExecuteActivity(ctx, "EmotionAnalyzeActivity", input).Get(ctx, &result)
	if err != nil {
		logger.Warn("情感分析失败（已重试），返回兜底结果", "error", err)
		// 返回低风险兜底
		return &EmotionAnalyzeOutput{EmotionLogJSON: `{"risk_level":"low","score":0}`}, nil
	}

	logger.Info("情感分析工作流完成", "user_id", input.UserID)
	return &result, nil
}
