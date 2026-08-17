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
	Agents     []string       // 实际参与本回答的 Agent 名称（人类可读，如 "政策解读"/"流程指引"），用于前端透明化展示
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
			Agents:     []string{},
		}
	}

	// 单 Agent 结果直接返回
	if len(results) == 1 {
		r := results[0]
		sources := r.Sources
		if sources == nil {
			sources = []model.Source{}
		}
		// 空 AgentName 不计入参与列表（不硬编）
		agents := []string{}
		if r.AgentName != "" {
			agents = []string{r.AgentName}
		}
		return &MergedResult{
			Content:    r.Content,
			Sources:    sources,
			Confidence: r.Confidence,
			AgentCount: 1,
			Agents:     agents,
		}
	}

	// 多 Agent 结果汇聚
	agentSet := make(map[string]bool)
	var agentNames []string
	for _, r := range results {
		if r.AgentName == "" {
			continue
		}
		if agentSet[r.AgentName] {
			continue
		}
		agentSet[r.AgentName] = true
		agentNames = append(agentNames, r.AgentName)
	}
	if agentNames == nil {
		agentNames = []string{}
	}

	merged := &MergedResult{
		AgentCount: len(results),
		Sources:    []model.Source{},
		Agents:     agentNames,
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
