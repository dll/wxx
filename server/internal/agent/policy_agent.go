package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// PolicyAgent 政策条款子智能体
// 专注政策类检索，强调原文引用与来源可追溯
type PolicyAgent struct {
	searchTopK int
}

// NewPolicyAgent 创建政策 Agent
func NewPolicyAgent() *PolicyAgent {
	return &PolicyAgent{searchTopK: 5}
}

func (a *PolicyAgent) Key() string  { return "policy-expert" }
func (a *PolicyAgent) Name() string { return "政策解读" }

func (a *PolicyAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	// 优先检索 Policy 类型资源
	results, err := kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, a.searchTopK)
	if err != nil {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0,
		}, nil
	}

	// 筛选 Policy 类型
	var policyResults []*repository.SearchResult
	for _, r := range results {
		if r.Resource.ResourceType == "Policy" || r.Resource.ResourceType == "FAQ" {
			policyResults = append(policyResults, r)
		}
	}

	if len(policyResults) == 0 {
		// CE-01 修复：政策类零命中时，不得把非政策检索结果伪装成政策来源返回，
		// 否则兜底回答会携带误导性 sources，违反「政策类零命中禁止编造来源」的硬约束。
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0,
			Sources:    nil,
		}, nil
	}

	var parts []string
	for _, r := range policyResults {
		parts = append(parts, fmt.Sprintf(
			"【%s】%s（版本：%s）\n%s",
			r.Resource.Title,
			r.Resource.SourceVersion,
			r.Resource.Version,
			truncate(r.Resource.Content, 1000),
		))
	}

	roleHint := rolePerspective(userCtx)
	content := fmt.Sprintf(
		"请严格基于以下政策原文回答用户问题「%s」。要求：\n1. 必须引用原文条款\n2. 标注版本号和生效日期\n3. 不确定的内容明确说明\n\n%s\n\n%s",
		question,
		roleHint,
		strings.Join(parts, "\n\n"),
	)

	sources := kbResultsToSources(policyResults)

	return &AgentResult{
		AgentName:  a.Name(),
		Content:    content,
		Sources:    sources,
		Confidence: 0.85,
	}, nil
}
