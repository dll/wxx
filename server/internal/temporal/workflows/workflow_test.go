package workflows

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

var mockErr = errors.New("mock error")

func registerMockActivity(env *testsuite.TestWorkflowEnvironment, name string) {
	env.RegisterActivityWithOptions(
		func(context.Context, interface{}) (interface{}, error) { return nil, nil },
		activity.RegisterOptions{Name: name},
	)
}

func TestChatAskWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "ValidateSessionActivity")
	registerMockActivity(env, "SearchKnowledgeActivity")
	registerMockActivity(env, "CallLLMActivity")
	registerMockActivity(env, "BuildAnswerActivity")

	env.OnActivity("ValidateSessionActivity", mock.Anything, mock.Anything).
		Return(&ValidateSessionResult{SessionID: "sess-new-001"}, nil)

	env.OnActivity("SearchKnowledgeActivity", mock.Anything, mock.Anything).
		Return(&SearchKnowledgeResult{ResultsJSON: `[]`}, nil)

	env.OnActivity("CallLLMActivity", mock.Anything, mock.Anything).
		Return(&CallLLMResult{
			Content: "申请奖学金需要提交申请表和成绩单。",
			Tokens:  `{"prompt":100,"output":50}`,
		}, nil)

	expectedCardJSON := `{"conclusion":"申请奖学金需要提交申请表和成绩单。","trace_id":"trace-001","confidence":0.8}`
	env.OnActivity("BuildAnswerActivity", mock.Anything, mock.Anything).
		Return(&BuildAnswerResult{AnswerCardJSON: expectedCardJSON}, nil)

	env.ExecuteWorkflow(ChatAskWorkflow, ChatAskInput{
		UserID:     1,
		Username:   "student1",
		Role:       "student",
		OwnerScope: "school",
		OwnerID:    "",
		SessionID:  "",
		Question:   "如何申请奖学金？",
		AgentID:    "",
		TraceID:    "trace-001",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("工作流返回错误: %v", err)
	}

	var output ChatAskOutput
	if err := env.GetWorkflowResult(&output); err != nil {
		t.Fatalf("获取工作流结果失败: %v", err)
	}
	if output.SessionID != "sess-new-001" {
		t.Errorf("期望 SessionID=sess-new-001，得到 %s", output.SessionID)
	}
	if output.AnswerCardJSON != expectedCardJSON {
		t.Errorf("AnswerCardJSON 不匹配")
	}
}

func TestChatAskWorkflow_SearchFails_StillCompletes(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "ValidateSessionActivity")
	registerMockActivity(env, "SearchKnowledgeActivity")
	registerMockActivity(env, "CallLLMActivity")
	registerMockActivity(env, "BuildAnswerActivity")

	env.OnActivity("ValidateSessionActivity", mock.Anything, mock.Anything).
		Return(&ValidateSessionResult{SessionID: "sess-002"}, nil)

	env.OnActivity("SearchKnowledgeActivity", mock.Anything, mock.Anything).
		Return(nil, mockErr)

	env.OnActivity("CallLLMActivity", mock.Anything, mock.Anything).
		Return(&CallLLMResult{Content: "兜底回答", Tokens: `{"prompt":50,"output":20}`}, nil)

	env.OnActivity("BuildAnswerActivity", mock.Anything, mock.Anything).
		Return(&BuildAnswerResult{AnswerCardJSON: `{"conclusion":"兜底回答"}`}, nil)

	env.ExecuteWorkflow(ChatAskWorkflow, ChatAskInput{
		UserID:     2,
		Username:   "student2",
		Role:       "student",
		OwnerScope: "school",
		Question:   "测试问题",
		TraceID:    "trace-002",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成（知识检索失败不应中断工作流）")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("工作流返回错误: %v", err)
	}
}

func TestChatAskWorkflow_LLMFails_StillCompletes(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "ValidateSessionActivity")
	registerMockActivity(env, "SearchKnowledgeActivity")
	registerMockActivity(env, "CallLLMActivity")
	registerMockActivity(env, "BuildAnswerActivity")

	env.OnActivity("ValidateSessionActivity", mock.Anything, mock.Anything).
		Return(&ValidateSessionResult{SessionID: "sess-003"}, nil)

	env.OnActivity("SearchKnowledgeActivity", mock.Anything, mock.Anything).
		Return(&SearchKnowledgeResult{ResultsJSON: `[]`}, nil)

	env.OnActivity("CallLLMActivity", mock.Anything, mock.Anything).
		Return(nil, mockErr)

	env.OnActivity("BuildAnswerActivity", mock.Anything, mock.Anything).
		Return(&BuildAnswerResult{AnswerCardJSON: `{"conclusion":"兜底回答","fallback":true}`}, nil)

	env.ExecuteWorkflow(ChatAskWorkflow, ChatAskInput{
		UserID:     3,
		Username:   "student3",
		Role:       "student",
		OwnerScope: "school",
		Question:   "测试问题",
		TraceID:    "trace-003",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成（LLM 失败不应中断工作流）")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("工作流返回错误: %v", err)
	}
}

func TestEmotionAnalyzeWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "EmotionAnalyzeActivity")

	env.OnActivity("EmotionAnalyzeActivity", mock.Anything, mock.Anything).
		Return(&EmotionAnalyzeOutput{
			EmotionLogJSON: `{"risk_level":"medium","score":-0.5}`,
		}, nil)

	env.ExecuteWorkflow(EmotionAnalyzeWorkflow, EmotionAnalyzeInput{
		UserID:      1,
		Username:    "student1",
		SessionID:   "sess-001",
		MessageText: "最近学习压力很大",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("工作流返回错误: %v", err)
	}
}

func TestEmotionAnalyzeWorkflow_ActivityFails_ReturnsFallback(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "EmotionAnalyzeActivity")

	env.OnActivity("EmotionAnalyzeActivity", mock.Anything, mock.Anything).
		Return(nil, mockErr)

	env.ExecuteWorkflow(EmotionAnalyzeWorkflow, EmotionAnalyzeInput{
		UserID:      2,
		Username:    "student2",
		MessageText: "测试",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成（失败应返回兜底）")
	}

	var output EmotionAnalyzeOutput
	if err := env.GetWorkflowResult(&output); err != nil {
		t.Fatalf("获取结果失败: %v", err)
	}
	if output.EmotionLogJSON != `{"risk_level":"low","score":0}` {
		t.Errorf("失败时应返回低风险兜底，得到: %s", output.EmotionLogJSON)
	}
}

func TestIntegrationProxyWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "IntegrationProxyActivity")

	env.OnActivity("IntegrationProxyActivity", mock.Anything, mock.Anything).
		Return(&IntegrationProxyOutput{BodyJSON: `{"students":[]}`}, nil)

	env.ExecuteWorkflow(IntegrationProxyWorkflow, IntegrationProxyInput{
		System: "xuegong",
		Path:   "/api/students",
		Query:  map[string]string{"page": "1"},
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("工作流返回错误: %v", err)
	}
}

func TestKBImportWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	registerMockActivity(env, "KBImportActivity")

	expectedResultJSON := `{"code":0,"message":"导入完成","data":[],"total":1,"created":1,"updated":0,"skipped":0}`
	env.OnActivity("KBImportActivity", mock.Anything, mock.Anything).
		Return(&KBImportOutput{ImportResultJSON: expectedResultJSON}, nil)

	env.ExecuteWorkflow(KBImportWorkflow, KBImportInput{
		NDJSONData: `{"resource_id":"test","resource_type":"Policy","title":"测试","content":"正文"}`,
		Username:   "admin",
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("工作流未完成")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("工作流返回错误: %v", err)
	}
}
