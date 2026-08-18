package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// KnowledgeGovernanceService 知识治理智能体
//
// 定位：在不触碰「确定性格式校验」的前提下，补齐「准确/质量」增强层。
// 规则（确定性）保规范与格式，LLM 审计做准确性/时效性/重复性的增强复盘，
// 但只产出建议报告，绝不自动改写知识内容（防止 AI 改错）。
//
// 数据源诚实原则：无资源或未注入 LLM 时返回 degraded/data_source=real 的诚实空，
// 不编造审计结论。
type KnowledgeGovernanceService struct {
	kbRepo    *repository.KBRepo
	llmClient llm.ChatClient
}

// NewKnowledgeGovernanceService 创建知识治理服务
func NewKnowledgeGovernanceService(kbRepo *repository.KBRepo) *KnowledgeGovernanceService {
	return &KnowledgeGovernanceService{kbRepo: kbRepo}
}

// SetLLMClient 注入 LLM 客户端（启用准确性增强审计）
func (s *KnowledgeGovernanceService) SetLLMClient(c llm.ChatClient) {
	s.llmClient = c
}

// GovernanceAudit 执行一次知识库治理审计。
//
// scope（可选）：owner_scope/owner_id 限定范围；空=全量。
// withLLM：为 true 且已注入 LLM 时，对抽样资源做准确性风险审计。
// limit：参与检查的资源上限（默认 200）。
func (s *KnowledgeGovernanceService) GovernanceAudit(ctx context.Context, ownerScope, ownerID string, withLLM bool, limit int) *model.KnowledgeGovernanceResult {
	res := &model.KnowledgeGovernanceResult{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Summary:     model.KGSummary{},
		Issues:      []*model.KGIssue{},
		DataSource:  "real",
	}

	if limit <= 0 {
		limit = 200
	}

	if s.kbRepo == nil {
		res.Issues = append(res.Issues, &model.KGIssue{
			Category: "audit",
			Level:    "info",
			Message:  "知识库仓储未初始化，无法执行治理审计。",
		})
		return res
	}

	// 1) 拉取资源（仅 published，避免对 draft 误报）
	resources, _, err := s.kbRepo.ListAdvanced(&model.KBQuery{
		Status:     "published",
		OwnerScope: ownerScope,
		OwnerID:    ownerID,
		Page:       1,
		PageSize:   limit,
	})
	if err != nil {
		log.Printf("[KG] 拉取知识资源失败: %v", err)
		return res
	}
	res.Summary.Scanned = len(resources)
	if len(resources) == 0 {
		res.Issues = append(res.Issues, &model.KGIssue{
			Category: "audit",
			Level:    "info",
			Message:  "当前范围暂无已发布知识资源，无需治理。",
		})
		return res
	}

	// 2) 确定性检查（规则层，可靠）
	res.Issues = append(res.Issues, s.deterministicChecks(resources)...)
	res.Summary.Determined = len(res.Issues)

	// 3) LLM 准确性增强审计（可选，只读建议）
	if withLLM && s.llmClient != nil {
		findings := s.llmAudit(ctx, resources)
		for _, f := range findings {
			res.Issues = append(res.Issues, f...)
			res.Summary.LLMFindings += len(f)
		}
		res.Summary.LLMChecked = len(findings)
	}

	return res
}

// deterministicChecks 确定性规则检查：缺失字段/正文过短/无标签/重复标题/已失效。
func (s *KnowledgeGovernanceService) deterministicChecks(resources []*model.KBResource) []*model.KGIssue {
	var issues []*model.KGIssue

	seenTitle := map[string]string{} // title -> resource_id
	for _, r := range resources {
		// 缺失核心字段
		if strings.TrimSpace(r.Title) == "" {
			issues = append(issues, &model.KGIssue{
				ResourceID: r.ResourceID, Title: r.Title, Category: "missing_field",
				Level: "critical", Message: "标题为空，检索与展示将严重受影响。",
			})
		}
		if strings.TrimSpace(r.Summary) == "" {
			issues = append(issues, &model.KGIssue{
				ResourceID: r.ResourceID, Title: r.Title, Category: "missing_field",
				Level: "warning", Message: "摘要为空，建议补充摘要以提升检索命中率。",
			})
		}
		if strings.TrimSpace(r.Content) == "" {
			issues = append(issues, &model.KGIssue{
				ResourceID: r.ResourceID, Title: r.Title, Category: "missing_field",
				Level: "critical", Message: "正文为空，用户提问将无内容可用。",
			})
		}
		// 正文过短（<20 有效字符）
		contentRunes := len([]rune(r.Content))
		if contentRunes > 0 && contentRunes < 20 {
			issues = append(issues, &model.KGIssue{
				ResourceID: r.ResourceID, Title: r.Title, Category: "short_content",
				Level: "warning", Message: "正文过短（<20 有效字符），信息量可能不足。",
			})
		}
		// 无标签
		if strings.TrimSpace(parseTagsStr(r.Tags)) == "" {
			issues = append(issues, &model.KGIssue{
				ResourceID: r.ResourceID, Title: r.Title, Category: "no_tags",
				Level: "info", Message: "未设置标签，结构化为检索弱化，建议补充关键词标签。",
			})
		}
		// 重复标题
		titleKey := strings.TrimSpace(r.Title)
		if titleKey != "" {
			if prev, ok := seenTitle[titleKey]; ok {
				issues = append(issues, &model.KGIssue{
					ResourceID: r.ResourceID, Title: r.Title, Category: "duplicate",
					Level: "warning",
					Message: fmt.Sprintf("与资源 %s 标题重复，存在内容冗余风险，建议核对合并。", prev),
				})
			} else {
				seenTitle[titleKey] = r.ResourceID
			}
		}
		// 已失效
		if r.ExpiredAt != nil && *r.ExpiredAt != "" && *r.ExpiredAt <= time.Now().Format("2006-01-02") {
			issues = append(issues, &model.KGIssue{
				ResourceID: r.ResourceID, Title: r.Title, Category: "expired",
				Level: "warning",
				Message: fmt.Sprintf("已到失效日期 %s，但仍为 published 状态，建议退役或续期。", *r.ExpiredAt),
			})
		}
	}
	return issues
}

// llmAudit 对抽样资源做 LLM 准确性风险审计。返回按资源分组的发现列表。
func (s *KnowledgeGovernanceService) llmAudit(ctx context.Context, resources []*model.KBResource) [][]*model.KGIssue {
	// 抽样上限，控制 LLM 成本与接口耗时
	const sampleMax = 12
	sample := resources
	if len(sample) > sampleMax {
		sample = sample[:sampleMax]
	}

	groups := make([][]*model.KGIssue, 0, len(sample))
	for _, r := range sample {
		groups = append(groups, s.auditOne(ctx, r))
	}
	return groups
}

func (s *KnowledgeGovernanceService) auditOne(ctx context.Context, r *model.KBResource) []*model.KGIssue {
	content := r.Content
	if len([]rune(content)) > 2000 {
		content = string([]rune(content)[:2000])
	}
	prompt := fmt.Sprintf(`你是知识库质量审计员。请审阅下面这条知识资源的准确性、时效性与重复性,并用严格 JSON 返回(不要输出其它内容):
{"risk":"high|medium|low|none","reasons":["..."],"suggest":"...","confidence":0.0-1.0}
只依据给定内容判断,无法判断时 risk=none 并注明原因。
标题:%s
摘要:%s
正文:%s`, r.Title, r.Summary, content)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.2,
		MaxTokens:   300,
	})
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		log.Printf("[KG] LLM 审计失败 resource=%s err=%v", r.ResourceID, err)
		return nil
	}

	// 提取 JSON（容忍模型在必要时包裹 ```json ... ```）
	raw := extractJSONObject(resp.Content)
	var f model.KGLLMFindings
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		log.Printf("[KG] LLM 审计 JSON 解析失败 resource=%s err=%v raw=%s", r.ResourceID, err, raw[:min(120, len(raw))])
		return nil
	}

	if f.Risk != "high" && f.Risk != "medium" {
		return nil // 低风险/无法判断：不入报告，避免噪声
	}
	level := "warning"
	if f.Risk == "high" {
		level = "critical"
	}
	reason := strings.Join(f.Reasons, "；")
	if reason == "" {
		reason = f.Suggest
	}
	return []*model.KGIssue{{
		ResourceID: r.ResourceID,
		Title:      r.Title,
		Category:   "accuracy_risk",
		Level:      level,
		Message:    fmt.Sprintf("[LLM审计·置信%.0f%%] %s。建议：%s", f.Confidence*100, reason, f.Suggest),
	}}
}

// parseTagsStr 返回去重后的标签逗号串（空则返回零值）。
func parseTagsStr(tagsJSON string) string {
	tags := parseTags(tagsJSON)
	return strings.Join(tags, ",")
}

// extractJSONObject 从模型输出中提取最外层 JSON 对象（兼容 ```json 包裹）。
func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// 去掉首尾代码围栏
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = strings.TrimSuffix(s, "```")
		}
		s = strings.TrimSpace(s)
	}
	if strings.HasPrefix(s, "{") {
		return s
	}
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			return s[i : j+1]
		}
	}
	return s
}
