package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// MajorAgent 学科专业子智能体
// 专注学科/专业介绍、培养方案、课程体系、学科竞赛、就业方向与前沿技术，
// 结合结构化知识与 FTS 检索回答，覆盖「专业认知 + 课程竞赛就业 + 前沿技术」。
type MajorAgent struct {
	searchTopK int
}

// NewMajorAgent 创建学科专业 Agent
func NewMajorAgent() *MajorAgent {
	return &MajorAgent{searchTopK: 6}
}

func (a *MajorAgent) Key() string  { return "major-guide" }
func (a *MajorAgent) Name() string { return "学科专业" }

func (a *MajorAgent) Execute(ctx context.Context, question string, userCtx *model.UserContext, kbRepo *repository.KBRepo) (*AgentResult, error) {
	results, err := kbRepo.Search(question, userCtx.OwnerScope, userCtx.OwnerID, userCtx.Role, a.searchTopK)
	if err != nil {
		log.Printf("MajorAgent 检索失败: %v", err)
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0,
		}, nil
	}

	// 优先筛选学科专业相关资源类型
	var majorResults []*repository.SearchResult
	for _, r := range results {
		switch r.Resource.ResourceType {
		case "Major", "Course", "Process", "FAQ", "Activity":
			majorResults = append(majorResults, r)
		}
	}
	if len(majorResults) == 0 {
		majorResults = results // 退化为通用检索结果
	}

	if len(majorResults) == 0 {
		return &AgentResult{
			AgentName:  a.Name(),
			Content:    "",
			Confidence: 0.1,
			Sources:    []model.Source{},
		}, nil
	}

	var parts []string
	for i, r := range majorResults {
		parts = append(parts, fmt.Sprintf("资料%d：%s（类型：%s）\n%s",
			i+1, r.Resource.Title, r.Resource.ResourceType, truncate(r.Resource.Content, 900)))
	}

	roleHint := rolePerspective(userCtx)
	content := fmt.Sprintf(
		"你是滁州学院计算机科学与工程学院（网络空间安全学院）的学科专业助手。请基于以下知识库资料回答用户关于学科专业的问题「%s」。\n\n"+
			"回答要求：\n"+
			"1. 涉及专业培养方案时，介绍培养目标、核心课程与能力要求\n"+
			"2. 涉及学科竞赛时，说明可参加的竞赛与备赛建议\n"+
			"3. 涉及就业时，说明就业方向、岗位与前景\n"+
			"4. 涉及前沿技术时，结合人工智能、大数据、网络安全等方向介绍\n"+
			"5. 如资料不足，结合常识补充，并标注「以下为通用信息」\n\n%s\n\n%s",
		question,
		roleHint,
		strings.Join(parts, "\n\n"),
	)

	sources := kbResultsToSources(majorResults)
	confidence := 0.5 + float64(len(majorResults))/float64(a.searchTopK)*0.35
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
