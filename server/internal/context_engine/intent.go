package context_engine

import (
	"strings"
)

// ── CE-06: 意图分类（固定优先级，消除 map 遍历不稳定） ──

// Intent 意图分类结果
type Intent struct {
	Category   string  // policy / process / faq / activity / chat / enrollment / graduation
	Confidence float64 // 0~1 置信度
}

// intentRule 意图规则（按优先级排列）
type intentRule struct {
	category string
	keywords []string
	weight   float64 // 基础置信度
}

// intentRules 按优先级排列，先匹配的优先（CE-06 修复遍历不稳定）
var intentRules = []intentRule{
	// 高优先级：明确业务意图
	{category: "enrollment", keywords: []string{"入学", "报到", "新生", "迎新", "报道", "开学"}, weight: 0.9},
	{category: "graduation", keywords: []string{"毕业", "离校", "学位", "答辩", "论文送审", "毕业设计"}, weight: 0.9},
	{category: "scholarship", keywords: []string{"奖学金", "助学金", "国奖", "励志奖", "评优", "综测"}, weight: 0.9},
	{category: "leave", keywords: []string{"请假", "缓考", "休学", "复学", "退学", "保留学籍"}, weight: 0.85},
	{category: "process", keywords: []string{"流程", "步骤", "办理", "申请", "审批", "手续", "怎么办"}, weight: 0.8},
	{category: "policy", keywords: []string{"政策", "规定", "条例", "制度", "管理办法", "文件", "通知"}, weight: 0.8},
	{category: "activity", keywords: []string{"活动", "比赛", "竞赛", "讲座", "志愿", "社团", "报名"}, weight: 0.75},
	{category: "mental", keywords: []string{"心理", "情绪", "压力", "焦虑", "咨询", "困扰"}, weight: 0.75},
	{category: "career", keywords: []string{"就业", "实习", "招聘", "简历", "面试", "考研", "考公"}, weight: 0.75},
	{category: "course", keywords: []string{"选课", "课程", "学分", "绩点", "成绩", "挂科", "补考", "重修"}, weight: 0.8},
	// 低优先级：通用问答
	{category: "faq", keywords: []string{"在哪", "电话", "地址", "时间", "几点", "工作日"}, weight: 0.6},
	{category: "chat", keywords: []string{}, weight: 0.3},
}

// ClassifyIntent 对问题进行意图分类
// 使用固定优先级规则列表（slice），避免 map 遍历顺序不稳定（CE-06）
// 返回最高置信度的匹配意图；无匹配则返回 chat
func ClassifyIntent(question string) Intent {
	q := strings.ToLower(question)
	bestIntent := Intent{Category: "chat", Confidence: 0.3}

	for _, rule := range intentRules {
		if len(rule.keywords) == 0 {
			continue
		}

		matchCount := 0
		for _, kw := range rule.keywords {
			if strings.Contains(q, kw) {
				matchCount++
			}
		}

		if matchCount == 0 {
			continue
		}

		// 置信度 = 基础权重 × (1 + 多关键词命中加成)
		confidence := rule.weight
		if matchCount > 1 {
			confidence += 0.05 * float64(matchCount-1)
		}
		if confidence > 1.0 {
			confidence = 1.0
		}

		// 取最高置信度（优先级已由 slice 顺序保证稳定性）
		if confidence > bestIntent.Confidence {
			bestIntent = Intent{
				Category:   rule.category,
				Confidence: confidence,
			}
		}
	}

	return bestIntent
}

// IntentToResourceTypes 将意图映射到优先检索的资源类型（用于检索加权）
func IntentToResourceTypes(intent Intent) []string {
	switch intent.Category {
	case "policy", "scholarship", "leave":
		return []string{"Policy", "Process"}
	case "process", "enrollment", "graduation":
		return []string{"Process", "Policy"}
	case "activity":
		return []string{"Activity", "FAQ"}
	case "course", "career":
		return []string{"FAQ", "Policy", "Process"}
	case "faq", "mental":
		return []string{"FAQ", "Policy"}
	default:
		return []string{"Policy", "Process", "FAQ", "Activity"}
	}
}
