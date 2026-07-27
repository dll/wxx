package agent

import (
	"context"
	"fmt"
	"log"
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

func (a *ProcessAgent) Key() string  { return "process-guide" }
func (a *ProcessAgent) Name() string { return "流程指引" }

func (a *ProcessAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	results, err := kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, a.searchTopK)
	if err != nil {
		log.Printf("ProcessAgent 检索失败: %v", err)
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
	var bestScore float64
	for i, r := range processResults {
		parts = append(parts, fmt.Sprintf(
			"流程%d：%s\n内容：%s\n链接：%s",
			i+1,
			r.Resource.Title,
			truncate(r.Resource.Content, 800),
			r.Resource.SourceLink,
		))
		if r.Score < bestScore {
			bestScore = r.Score
		}
	}

	roleHint := rolePerspective(userCtx)
	content := fmt.Sprintf(
		"请为以下问题生成步骤清单式回答「%s」。要求：\n1. 分步骤列出（每步含：做什么、材料、入口、时限）\n2. 标注办理地点和联系方式\n3. 提醒注意事项\n\n%s\n\n参考资料：\n%s",
		question,
		roleHint,
		strings.Join(parts, "\n\n"),
	)

	sources := kbResultsToSources(processResults)

	countRatio := float64(len(processResults)) / float64(a.searchTopK)
	raw := -bestScore
	if raw < 0 {
		raw = 0
	}
	scoreNorm := raw / 20.0
	if scoreNorm > 1.0 {
		scoreNorm = 1.0
	}
	confidence := 0.4 + scoreNorm*0.3 + countRatio*0.3
	if confidence > 0.95 {
		confidence = 0.95
	}

	return &AgentResult{
		AgentName:  a.Name(),
		Content:    content,
		Sources:    sources,
		Confidence: confidence,
	}, nil
}
