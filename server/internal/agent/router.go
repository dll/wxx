package agent

import (
	"sort"
	"strings"
)

// Intent 问题意图分类
type Intent int

const (
	IntentUnknown  Intent = iota // 未知（走默认 QA Agent）
	IntentPolicy                 // 政策条款解释
	IntentProcess                // 办事流程指引
	IntentActivity               // 活动通知
	IntentFAQ                    // 常见问答
	IntentEmotion                // 心理/情感
)

// keyword 带权重的关键词
type keyword struct {
	word   string // 关键词文本
	weight int    // 权重：3=核心关键词，1=一般关键词
}

// intentPriority 意图优先级（数值越小优先级越高）
// 用于多个意图得分相同时的排序：policy > process > activity > emotion > faq
var intentPriority = map[Intent]int{
	IntentPolicy:   1,
	IntentProcess:  2,
	IntentActivity: 3,
	IntentEmotion:  4,
	IntentFAQ:      5,
}

// domainKeywords 业务域关键词映射（规则路由）
// 每个意图 30+ 个关键词，核心关键词权重 3，一般关键词权重 1
var domainKeywords = map[Intent][]keyword{
	IntentPolicy: {
		// 核心关键词（权重 3）
		{word: "奖学金", weight: 3},
		{word: "国家奖学金", weight: 3},
		{word: "国家励志", weight: 3},
		{word: "助学金", weight: 3},
		{word: "助学贷款", weight: 3},
		{word: "学费减免", weight: 3},
		{word: "困难补助", weight: 3},
		{word: "评优评先", weight: 3},
		{word: "三好学生", weight: 3},
		{word: "优秀毕业生", weight: 3},
		{word: "处分", weight: 3},
		{word: "违纪", weight: 3},
		{word: "学籍", weight: 3},
		{word: "休学", weight: 3},
		{word: "复学", weight: 3},
		{word: "转学", weight: 3},
		{word: "转专业", weight: 3},
		{word: "退学", weight: 3},
		{word: "政策", weight: 3},
		{word: "规定", weight: 3},
		{word: "条例", weight: 3},
		{word: "办法", weight: 3},
		{word: "细则", weight: 3},
		// 一般关键词（权重 1）
		{word: "申请条件", weight: 1},
		{word: "评定标准", weight: 1},
		{word: "资格", weight: 1},
		{word: "金额", weight: 1},
		{word: "比例", weight: 1},
		{word: "名额", weight: 1},
		{word: "通知", weight: 1},
		{word: "文件", weight: 1},
		{word: "评奖", weight: 1},
		{word: "评优", weight: 1},
		{word: "申诉", weight: 1},
		{word: "撤销", weight: 1},
		{word: "补助", weight: 1},
		{word: "减免", weight: 1},
		{word: "奖励", weight: 1},
	},
	IntentProcess: {
		// 核心关键词（权重 3）
		{word: "请假流程", weight: 3},
		{word: "报销流程", weight: 3},
		{word: "入党流程", weight: 3},
		{word: "入团流程", weight: 3},
		{word: "请假", weight: 3},
		{word: "休学", weight: 3},
		{word: "复学", weight: 3},
		{word: "退学", weight: 3},
		{word: "转专业", weight: 3},
		{word: "转学", weight: 3},
		{word: "报到", weight: 3},
		{word: "注册", weight: 3},
		{word: "毕业", weight: 3},
		{word: "离校", weight: 3},
		{word: "补办", weight: 3},
		{word: "流程", weight: 3},
		{word: "步骤", weight: 3},
		// 一般关键词（权重 1）
		{word: "怎么办", weight: 1},
		{word: "怎么申请", weight: 1},
		{word: "怎么办理", weight: 1},
		{word: "如何申请", weight: 1},
		{word: "如何办理", weight: 1},
		{word: "材料", weight: 1},
		{word: "提交", weight: 1},
		{word: "审核", weight: 1},
		{word: "审批", weight: 1},
		{word: "盖章", weight: 1},
		{word: "签字", weight: 1},
		{word: "入学", weight: 1},
		{word: "退宿", weight: 1},
		{word: "宿舍", weight: 1},
		{word: "住宿", weight: 1},
		{word: "走读", weight: 1},
		{word: "调换", weight: 1},
		{word: "校园卡", weight: 1},
		{word: "挂失", weight: 1},
		{word: "充值", weight: 1},
		{word: "户籍", weight: 1},
		{word: "档案", weight: 1},
		{word: "派遣", weight: 1},
		{word: "报到证", weight: 1},
	},
	IntentActivity: {
		// 核心关键词（权重 3）
		{word: "讲座", weight: 3},
		{word: "报告", weight: 3},
		{word: "比赛", weight: 3},
		{word: "竞赛", weight: 3},
		{word: "晚会", weight: 3},
		{word: "演出", weight: 3},
		{word: "展览", weight: 3},
		{word: "社团", weight: 3},
		{word: "招新", weight: 3},
		{word: "志愿", weight: 3},
		{word: "实践", weight: 3},
		{word: "活动", weight: 3},
		{word: "宣讲会", weight: 3},
		{word: "招聘会", weight: 3},
		// 一般关键词（权重 1）
		{word: "志愿者", weight: 1},
		{word: "学生会", weight: 1},
		{word: "团委", weight: 1},
		{word: "党支", weight: 1},
		{word: "报名", weight: 1},
		{word: "参加", weight: 1},
		{word: "参与", weight: 1},
		{word: "组织", weight: 1},
		{word: "举办", weight: 1},
		{word: "大创", weight: 1},
		{word: "创新创业", weight: 1},
		{word: "学科竞赛", weight: 1},
		{word: "挑战杯", weight: 1},
		{word: "互联网+", weight: 1},
		{word: "社会实践", weight: 1},
		{word: "实习", weight: 1},
		{word: "见习", weight: 1},
	},
	IntentEmotion: {
		// 核心关键词（权重 3）
		{word: "心情不好", weight: 3},
		{word: "焦虑", weight: 3},
		{word: "抑郁", weight: 3},
		{word: "失眠", weight: 3},
		{word: "压力", weight: 3},
		{word: "孤独", weight: 3},
		{word: "迷茫", weight: 3},
		{word: "心理咨询", weight: 3},
		{word: "情绪", weight: 3},
		{word: "情感", weight: 3},
		{word: "失恋", weight: 3},
		{word: "心理", weight: 3},
		// 一般关键词（权重 1）
		{word: "咨询", weight: 1},
		{word: "预约心理咨询", weight: 1},
		{word: "心理中心", weight: 1},
		{word: "心情", weight: 1},
		{word: "不开心", weight: 1},
		{word: "难过", weight: 1},
		{word: "烦躁", weight: 1},
		{word: "困惑", weight: 1},
		{word: "郁闷", weight: 1},
		{word: "低落", weight: 1},
		{word: "想哭", weight: 1},
		{word: "睡不着", weight: 1},
		{word: "焦虑症", weight: 1},
		{word: "抑郁症", weight: 1},
		{word: "心理问题", weight: 1},
		{word: "心理疏导", weight: 1},
	},
	IntentFAQ: {
		// 核心关键词（权重 3）
		{word: "招聘", weight: 3},
		{word: "求职", weight: 3},
		{word: "简历", weight: 3},
		{word: "面试", weight: 3},
		{word: "实习", weight: 3},
		{word: "校招", weight: 3},
		{word: "双选会", weight: 3},
		{word: "就业", weight: 3},
		{word: "创业", weight: 3},
		{word: "选课", weight: 3},
		{word: "成绩", weight: 3},
		{word: "绩点", weight: 3},
		{word: "补考", weight: 3},
		{word: "重修", weight: 3},
		{word: "考试", weight: 3},
		{word: "学分", weight: 3},
		{word: "培养方案", weight: 3},
		{word: "课程", weight: 3},
		// 一般关键词（权重 1）
		{word: "是什么", weight: 1},
		{word: "什么是", weight: 1},
		{word: "什么意思", weight: 1},
		{word: "怎么理解", weight: 1},
		{word: "在哪里", weight: 1},
		{word: "去哪里", weight: 1},
		{word: "电话", weight: 1},
		{word: "联系方式", weight: 1},
		{word: "地址", weight: 1},
		{word: "几点", weight: 1},
		{word: "什么时候", weight: 1},
		{word: "时间", weight: 1},
		{word: "多久", weight: 1},
		{word: "截止", weight: 1},
		{word: "学费", weight: 1},
		{word: "考查", weight: 1},
		{word: "课表", weight: 1},
		{word: "教室", weight: 1},
		{word: "宣讲会", weight: 1},
	},
}

// Router 意图路由器
// 基于关键词规则匹配问题意图，决定激活哪些子 Agent
type Router struct{}

// NewRouter 创建路由器
func NewRouter() *Router {
	return &Router{}
}

// Route 根据问题路由到目标 Agent 列表
// 返回应激活的 Agent 名称列表（按优先级排序）
func (r *Router) Route(question string) []string {
	lower := strings.ToLower(question)
	intents := r.classify(lower)

	// 去重并按优先级生成 Agent 列表
	agents := make([]string, 0, len(intents))
	seen := make(map[string]bool)

	for _, intent := range intents {
		name := intentToAgent(intent)
		if !seen[name] {
			seen[name] = true
			agents = append(agents, name)
		}
	}

	// 至少激活默认 QA Agent
	if len(agents) == 0 {
		agents = append(agents, "qa-default")
	}
	return agents
}

// classify 根据关键词匹配返回意图列表（按得分降序 + 优先级排序）
// 得分最高的意图排在前面；最高得分 < 2 时返回空切片（触发兜底 qa-default）
func (r *Router) classify(lowerQuestion string) []Intent {
	// 计算每个意图的得分
	scores := make(map[Intent]int)
	for intent, keywords := range domainKeywords {
		score := 0
		for _, kw := range keywords {
			if strings.Contains(lowerQuestion, kw.word) {
				score += kw.weight
			}
		}
		if score > 0 {
			scores[intent] = score
		}
	}

	// 没有匹配到任何关键词，返回空
	if len(scores) == 0 {
		return nil
	}

	// 找出最高得分
	maxScore := 0
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
	}

	// 最高得分 < 2，返回空（置信度不足，走兜底）
	if maxScore < 2 {
		return nil
	}

	// 收集所有有得分的意图
	type intentScore struct {
		intent Intent
		score  int
	}
	var items []intentScore
	for intent, score := range scores {
		items = append(items, intentScore{intent: intent, score: score})
	}

	// 排序规则：
	// 1. 得分高的排前面
	// 2. 得分相同时，按 intentPriority 优先级排前面（数值越小优先级越高）
	sort.Slice(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return intentPriority[items[i].intent] < intentPriority[items[j].intent]
	})

	// 提取意图列表
	intents := make([]Intent, 0, len(items))
	for _, item := range items {
		intents = append(intents, item.intent)
	}

	return intents
}

// intentToAgent 将意图映射到 Agent 名称
func intentToAgent(intent Intent) string {
	switch intent {
	case IntentPolicy:
		return "policy-expert"
	case IntentProcess:
		return "process-guide"
	case IntentActivity:
		return "qa-default" // 活动类暂由 QA 处理
	case IntentEmotion:
		return "emotion-counselor"
	case IntentFAQ:
		return "qa-default"
	default:
		return "qa-default"
	}
}
