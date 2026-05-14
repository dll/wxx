package agent

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
)

func TestMerger_EmptyResults(t *testing.T) {
	m := NewMerger()
	result := m.Merge(nil)
	if result.Content == "" {
		t.Error("空结果应返回兜底回复")
	}
	if result.Confidence != 0 {
		t.Errorf("空结果置信度应为 0，实际: %f", result.Confidence)
	}
}

func TestMerger_SingleResult(t *testing.T) {
	m := NewMerger()
	input := []*AgentResult{
		{
			AgentName:  "通用问答",
			Content:    "这是回答内容",
			Confidence: 0.8,
			Sources: []model.Source{
				{ResourceID: "r1", Title: "资料1", Version: "v1"},
			},
		},
	}
	result := m.Merge(input)
	if result.Content != "这是回答内容" {
		t.Errorf("单结果应直接返回内容，实际: %s", result.Content)
	}
	if result.AgentCount != 1 {
		t.Errorf("AgentCount 应为 1，实际: %d", result.AgentCount)
	}
	if result.Confidence != 0.8 {
		t.Errorf("置信度应为 0.8，实际: %f", result.Confidence)
	}
	if len(result.Sources) != 1 {
		t.Errorf("Sources 数量应为 1，实际: %d", len(result.Sources))
	}
}

func TestMerger_MultipleResults(t *testing.T) {
	m := NewMerger()
	input := []*AgentResult{
		{
			AgentName:  "政策解读",
			Content:    "政策内容",
			Confidence: 0.9,
			Sources: []model.Source{
				{ResourceID: "r1", Title: "政策文件", Version: "v1"},
			},
		},
		{
			AgentName:  "流程指引",
			Content:    "流程内容",
			Confidence: 0.7,
			Sources: []model.Source{
				{ResourceID: "r2", Title: "流程文档", Version: "v1"},
			},
		},
	}
	result := m.Merge(input)
	if result.AgentCount != 2 {
		t.Errorf("AgentCount 应为 2，实际: %d", result.AgentCount)
	}
	expectedConf := (0.9 + 0.7) / 2
	if result.Confidence != expectedConf {
		t.Errorf("置信度应为 %.2f，实际: %f", expectedConf, result.Confidence)
	}
	if len(result.Sources) != 2 {
		t.Errorf("Sources 数量应为 2，实际: %d", len(result.Sources))
	}
}

func TestMerger_SourceDedup(t *testing.T) {
	m := NewMerger()
	input := []*AgentResult{
		{
			AgentName:  "Agent1",
			Content:    "内容1",
			Confidence: 0.8,
			Sources: []model.Source{
				{ResourceID: "r1", Title: "共享资料", Version: "v1"},
			},
		},
		{
			AgentName:  "Agent2",
			Content:    "内容2",
			Confidence: 0.7,
			Sources: []model.Source{
				{ResourceID: "r1", Title: "共享资料", Version: "v1"},
				{ResourceID: "r2", Title: "独有资料", Version: "v1"},
			},
		},
	}
	result := m.Merge(input)
	if len(result.Sources) != 2 {
		t.Errorf("去重后 Sources 应为 2，实际: %d", len(result.Sources))
	}
}

func TestMerger_EmptyContentFiltered(t *testing.T) {
	m := NewMerger()
	input := []*AgentResult{
		{AgentName: "Agent1", Content: "有效内容", Confidence: 0.8},
		{AgentName: "Agent2", Content: "", Confidence: 0.1},
	}
	result := m.Merge(input)
	if result.AgentCount != 2 {
		t.Errorf("AgentCount 应为 2，实际: %d", result.AgentCount)
	}
	if len(result.Content) == 0 {
		t.Error("合并结果不应为空")
	}
}
