package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ── 问答工作流输入/输出类型 ──

// ChatAskInput 问答工作流输入
type ChatAskInput struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	OwnerScope string `json:"owner_scope"`
	OwnerID    string `json:"owner_id"`
	SessionID  string `json:"session_id"`
	Question   string `json:"question"`
	AgentID    string `json:"agent_id"`
	TraceID    string `json:"trace_id"`
}

// ChatAskOutput 问答工作流输出
type ChatAskOutput struct {
	AnswerCardJSON string `json:"answer_card_json"` // AnswerCard 序列化为 JSON
	SessionID      string `json:"session_id"`
}

// ── 活动输入/输出类型 ──

// ValidateSessionInput 会话验证活动输入
type ValidateSessionInput struct {
	UserID    int64  `json:"user_id"`
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
	TraceID   string `json:"trace_id"`
}

// ValidateSessionResult 会话验证活动输出
type ValidateSessionResult struct {
	SessionID string `json:"session_id"`
}

// SearchKnowledgeInput 知识检索活动输入
type SearchKnowledgeInput struct {
	Question   string `json:"question"`
	OwnerScope string `json:"owner_scope"`
	OwnerID    string `json:"owner_id"`
	Role       string `json:"role"`
}

// SearchKnowledgeResult 知识检索活动输出
type SearchKnowledgeResult struct {
	ResultsJSON string `json:"results_json"` // 序列化的 []*repository.SearchResult
}

// CallLLMInput LLM 调用活动输入
type CallLLMInput struct {
	SessionID     string `json:"session_id"`
	Question      string `json:"question"`
	AgentID       string `json:"agent_id"`
	SearchResults string `json:"search_results"` // JSON，空字符串表示无结果
}

// CallLLMResult LLM 调用活动输出
type CallLLMResult struct {
	Content string `json:"content"`
	Tokens  string `json:"tokens"` // JSON: {"prompt":0,"output":0}
}

// BuildAnswerInput 构造回答活动输入
type BuildAnswerInput struct {
	SessionID     string `json:"session_id"`
	Question      string `json:"question"`
	LLMContent    string `json:"llm_content"`
	LLMTokens     string `json:"llm_tokens"`
	SearchResults string `json:"search_results"` // JSON
	TraceID       string `json:"trace_id"`
}

// BuildAnswerResult 构造回答活动输出
type BuildAnswerResult struct {
	AnswerCardJSON string `json:"answer_card_json"`
}

// ── 工作流定义 ──

// ChatAskWorkflow 问答工作流：编排 4 个活动完成问答链路
// 活动顺序：验证会话 → 知识检索 → LLM 调用 → 构造回答
// 知识检索失败不中断链路（降级兜底），LLM 调用享有独立重试策略
func ChatAskWorkflow(ctx workflow.Context, input ChatAskInput) (*ChatAskOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("问答工作流开始", "trace_id", input.TraceID, "user_id", input.UserID)

	// 标准重试策略（2 次尝试）
	standardRetry := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 1.5,
		MaximumAttempts:    2,
	}

	// LLM 重试策略（3 次尝试，指数退避）
	llmRetry := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    10 * time.Second,
	}

	// ── 步骤 1：验证/创建会话，保存用户消息 ──
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         standardRetry,
	}
	ctx1 := workflow.WithActivityOptions(ctx, activityOpts)

	var sessionResult ValidateSessionResult
	err := workflow.ExecuteActivity(ctx1, "ValidateSessionActivity", ValidateSessionInput{
		UserID:    input.UserID,
		SessionID: input.SessionID,
		Question:  input.Question,
		TraceID:   input.TraceID,
	}).Get(ctx1, &sessionResult)
	if err != nil {
		logger.Error("会话验证失败", "error", err)
		return nil, err
	}

	// ── 步骤 2：知识检索（只读，检索失败不中断链路）──
	var searchResult SearchKnowledgeResult
	err = workflow.ExecuteActivity(ctx1, "SearchKnowledgeActivity", SearchKnowledgeInput{
		Question:   input.Question,
		OwnerScope: input.OwnerScope,
		OwnerID:    input.OwnerID,
		Role:       input.Role,
	}).Get(ctx1, &searchResult)
	if err != nil {
		logger.Warn("知识检索失败，继续走兜底", "error", err)
		searchResult.ResultsJSON = "" // 空结果，后续活动走兜底
	}

	// ── 步骤 3：LLM 调用（高风险操作，独立重试策略）──
	llmOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 90 * time.Second,
		RetryPolicy:         llmRetry,
		HeartbeatTimeout:    15 * time.Second,
	}
	ctx3 := workflow.WithActivityOptions(ctx, llmOpts)

	var llmResult CallLLMResult
	err = workflow.ExecuteActivity(ctx3, "CallLLMActivity", CallLLMInput{
		SessionID:     sessionResult.SessionID,
		Question:      input.Question,
		AgentID:       input.AgentID,
		SearchResults: searchResult.ResultsJSON,
	}).Get(ctx3, &llmResult)
	if err != nil {
		logger.Warn("LLM 调用失败（已重试），走兜底", "error", err)
		llmResult.Content = "" // 空内容 = 兜底回答
		llmResult.Tokens = `{"prompt":0,"output":0}`
	}

	// ── 步骤 4：构造 AnswerCard 并保存助手消息 ──
	var answerResult BuildAnswerResult
	err = workflow.ExecuteActivity(ctx1, "BuildAnswerActivity", BuildAnswerInput{
		SessionID:     sessionResult.SessionID,
		Question:      input.Question,
		LLMContent:    llmResult.Content,
		LLMTokens:     llmResult.Tokens,
		SearchResults: searchResult.ResultsJSON,
		TraceID:       input.TraceID,
	}).Get(ctx1, &answerResult)
	if err != nil {
		logger.Error("构造回答失败", "error", err)
		return nil, err
	}

	logger.Info("问答工作流完成", "trace_id", input.TraceID)
	return &ChatAskOutput{
		AnswerCardJSON: answerResult.AnswerCardJSON,
		SessionID:      sessionResult.SessionID,
	}, nil
}
