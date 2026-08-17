package service

// D4-3 功能补齐专项测试（纯新增，不改既有生产代码与既有测试断言语义）。
// 覆盖「AnswerCard 显式标注参与 Agent」的重点验证点：
//   1. 多 Agent 编排路径：card.Agents 携带实际参与智能体名称列表（来自 merger.Merge 的 AgentName）
//   2. 无 Agent 信息路径（单 Agent/降级/兜底，multiAgentResult 为 nil）：card.Agents 为空，不硬编
//   3. 新增 agents 字段不破坏既有 AnswerCard 字段（Conclusion/Sources/Confidence/Fallback/Model）

import (
	"testing"

	"github.com/dll/wxx/server/internal/agent"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// buildAgentsTestCardSvc 构造一个带 mock LLM 的 ChatService（buildAnswerCard 内会读取 s.llmClient.Model()）。
func buildAgentsTestCardSvc(t *testing.T) *ChatService {
	t.Helper()
	db := testutil.NewTestDBFull(t)
	t.Cleanup(func() { db.Close() })
	return NewChatService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		repository.NewAgentRepo(db),
		llm.NewMockClient("test-agents"),
	)
}

func TestBuildAnswerCard_Agents_MultiAgentPathPopulated(t *testing.T) {
	svc := buildAgentsTestCardSvc(t)
	multi := &agent.MergedResult{
		Content:    "汇聚回答",
		Confidence: 0.85,
		AgentCount: 2,
		Agents:     []string{"政策解读", "流程指引"},
		Sources: []model.Source{
			{ResourceID: "r1", Title: "政策文件", Version: "v1", RelevanceScore: 0.9},
			{ResourceID: "r2", Title: "流程文档", Version: "v1", RelevanceScore: 0.8},
		},
	}
	// 提供相关性高的知识命中，使卡片走非兜底分支（isRelevant 命中）
	hits := []*repository.SearchResult{
		{
			Resource: model.KBResource{
				ResourceID: "kb1", Title: "奖学金申请政策解读", ResourceType: "Policy",
				Version: "v1", Summary: "奖学金申请条件与流程", Content: "奖学金申请材料规范",
			},
			Score:         -15,
			LowConfidence: false,
		},
	}

	card := svc.buildAnswerCard("最终回答", hits, "trace-agents-1", multi)

	// 1. agents 字段：多 Agent 路径应携带参与列表
	if len(card.Agents) != 2 {
		t.Fatalf("多 Agent 路径 card.Agents 应为 2 项，实际: %v", card.Agents)
	}
	if card.Agents[0] != "政策解读" || card.Agents[1] != "流程指引" {
		t.Errorf("card.Agents 内容不符，实际: %v", card.Agents)
	}

	// 2. 既有 AnswerCard 字段不被破坏
	if card.Conclusion != "最终回答" {
		t.Errorf("Conclusion 被破坏，实际: %q", card.Conclusion)
	}
	if card.Fallback {
		t.Error("有知识命中时 Fallback 应为 false")
	}
	if card.Confidence != 0.8 {
		t.Errorf("有知识命中时默认 Confidence 应=0.8，实际: %f", card.Confidence)
	}
	// 知识库来源 + 多智能体来源都会并入 Sources
	if len(card.Sources) < 2 {
		t.Errorf("Sources 应≥2 条，实际: %d", len(card.Sources))
	}
	if card.Model != "test-agents" {
		t.Errorf("Model 应来自 mock LLM，实际: %q", card.Model)
	}
}

func TestBuildAnswerCard_Agents_NoAgentPathEmpty(t *testing.T) {
	svc := buildAgentsTestCardSvc(t)

	// 无 Agent 信息（multiAgentResult 为 nil）：agents 必须为空，不硬编
	card := svc.buildAnswerCard("降级回答", nil, "trace-agents-2", nil)

	if len(card.Agents) != 0 {
		t.Errorf("无 Agent 信息时 card.Agents 应为空，实际: %v", card.Agents)
	}
	// 既有兜底语义不变（无知识命中 → 低置信 + Fallback）
	if !card.Fallback {
		t.Error("无知识命中应为 Fallback")
	}
	if card.Confidence != 0.3 {
		t.Errorf("无知识命中 Confidence 应=0.3，实际: %f", card.Confidence)
	}
	if card.Conclusion == "" {
		t.Error("Conclusion 不应为空")
	}
}

func TestBuildAnswerCard_Agents_EmptyAgentsList(t *testing.T) {
	svc := buildAgentsTestCardSvc(t)
	// 编排返回了 AgentCount>0 但无智能体名（异常兜底）：agents 为空，不硬编
	multi := &agent.MergedResult{
		Content:    "内容",
		Confidence: 0.9,
		AgentCount: 1,
		Agents:     []string{},
		Sources: []model.Source{
			{ResourceID: "r1", Title: "资料", Version: "v1", RelevanceScore: 0.9},
		},
	}
	hits := []*repository.SearchResult{
		{
			Resource: model.KBResource{
				ResourceID: "kb2", Title: "办事流程指引", ResourceType: "Process",
				Version: "v1", Summary: "流程说明", Content: "流程步骤",
			},
			Score:         -12,
			LowConfidence: false,
		},
	}
	card := svc.buildAnswerCard("回答", hits, "trace-agents-3", multi)
	if len(card.Agents) != 0 {
		t.Errorf("Agents 列表为空时应保持为空，实际: %v", card.Agents)
	}
	// 既有字段不受影响
	if len(card.Sources) < 1 {
		t.Errorf("Sources 应≥1 条（含多智能体来源），实际: %d", len(card.Sources))
	}
	if card.Fallback {
		t.Error("有来源命中时不应为 Fallback")
	}
}
