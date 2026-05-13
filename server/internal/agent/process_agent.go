package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// ProcessAgent 办事流程子智能体
// 专注流程类检索，输出步骤清单式回答
type ProcessAgent struct {
	searchTopK int
}

// NewProcessAgent 创建流程 Agent
func NewProcessAgent() *ProcessAgent {
	return &ProcessAgent{searchTopK: 5}
}

func (a *ProcessAgent) Name() string { return "流程指引" }

func (a *ProcessAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	results, err := kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, a.searchTopK)
	if err != nil {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0,
		}, nil
	}

	// 优先 Process 类型
	var processResults []*repository.SearchResult
	for _, r := range results {
		if r.Resource.ResourceType == "Process" || r.Resource.ResourceType == "FAQ" {
			processResults = append(processResults, r)
		}
	}

	if len(processResults) == 0 {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0.1,
			Sources:    kbResultsToSources(results),
		}, nil
	}

	var parts []string
	for i, r := range processResults {
		parts = append(parts, fmt.Sprintf(
			"流程%d：%s\n内容：%s\n链接：%s",
			i+1,
			r.Resource.Title,
			truncate(r.Resource.Content, 800),
			r.Resource.SourceLink,
		))
	}

	content := fmt.Sprintf(
		"请为以下问题生成步骤清单式回答「%s」。要求：\n1. 分步骤列出（每步含：做什么、材料、入口、时限）\n2. 标注办理地点和联系方式\n3. 提醒注意事项\n\n参考资料：\n%s",
		question,
		strings.Join(parts, "\n\n"),
	)

	sources := kbResultsToSources(processResults)

	return &AgentResult{
		AgentName:  a.Name(),
		Content:    content,
		Sources:    sources,
		Confidence: 0.8,
	}, nil
}
