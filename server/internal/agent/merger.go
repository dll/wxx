package agent

import (
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
)

// MergedResult 多 Agent 结果汇聚后的最终输出
type MergedResult struct {
	Content    string         // 汇聚后的最终回答
	Sources    []model.Source // 去重合并后的来源列表
	Confidence float64        // 综合置信度
	AgentCount int            // 参与协同的 Agent 数量
}

// ResultMerger 多 Agent 结果汇聚器
// 负责去重、排序、冲突检测与内容合并
type ResultMerger struct{}

// NewMerger 创建结果汇聚器
func NewMerger() *ResultMerger {
	return &ResultMerger{}
}

// Merge 合并多个 Agent 结果
func (m *ResultMerger) Merge(results []*AgentResult) *MergedResult {
	if len(results) == 0 {
		return &MergedResult{
			Content:    "抱歉，当前无法处理您的问题。",
			Confidence: 0,
			Sources:    []model.Source{},
		}
	}

	// 单 Agent 结果直接返回
	if len(results) == 1 {
		r := results[0]
		sources := r.Sources
		if sources == nil {
			sources = []model.Source{}
		}
		return &MergedResult{
			Content:    r.Content,
			Sources:    sources,
			Confidence: r.Confidence,
			AgentCount: 1,
		}
	}

	// 多 Agent 结果汇聚
	merged := &MergedResult{
		AgentCount: len(results),
		Sources:    []model.Source{},
	}

	// 合并内容（按 Agent 名称分段）
	var contentParts []string
	for _, r := range results {
		if r.Content != "" {
			contentParts = append(contentParts,
				fmt.Sprintf("【%s】\n%s", r.AgentName, r.Content))
		}
	}
	merged.Content = strings.Join(contentParts, "\n\n")

	// 去重合并 sources
	sourceSet := make(map[string]model.Source)
	for _, r := range results {
		for _, s := range r.Sources {
			key := s.ResourceID + s.Version
			if _, exists := sourceSet[key]; !exists {
				sourceSet[key] = s
			}
		}
	}
	for _, s := range sourceSet {
		merged.Sources = append(merged.Sources, s)
	}

	// 综合置信度（加权平均）
	var totalConf float64
	for _, r := range results {
		totalConf += r.Confidence
	}
	merged.Confidence = totalConf / float64(len(results))

	return merged
}
