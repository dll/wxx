package agent

import (
	"strings"
)

// Intent 问题意图分类
type Intent int

const (
	IntentUnknown   Intent = iota // 未知（走默认 QA Agent）
	IntentPolicy                  // 政策条款解释
	IntentProcess                 // 办事流程指引
	IntentActivity                 // 活动通知
	IntentFAQ                     // 常见问答
	IntentEmotion                 // 心理/情感
)

// domainKeywords 业务域关键词映射（规则路由）
var domainKeywords = map[Intent][]string{
	IntentPolicy: {
		"政策", "规定", "条例", "办法", "细则", "通知", "文件",
		"申请条件", "评定标准", "资格", "金额", "比例", "名额",
		"奖学金", "助学金", "助学贷款", "勤工助学", "困难补助",
		"评奖", "评优", "三好学生", "优秀毕业生",
		"处分", "违纪", "申诉", "撤销",
	},
	IntentProcess: {
		"流程", "步骤", "怎么办", "怎么申请", "怎么办理", "如何申请", "如何办理",
		"材料", "提交", "审核", "审批", "盖章", "签字",
		"请假", "休学", "复学", "退学", "转专业", "转学",
		"入学", "报到", "毕业", "离校", "退宿",
		"宿舍", "住宿", "走读", "调换",
		"校园卡", "补办", "挂失", "充值",
		"户籍", "档案", "派遣", "报到证",
	},
	IntentActivity: {
		"活动", "比赛", "竞赛", "讲座", "宣讲会", "招聘会",
		"志愿者", "社团", "学生会", "团委", "党支",
		"报名", "参加", "参与", "组织", "举办",
		"大创", "创新创业", "学科竞赛", "挑战杯", "互联网+",
		"社会实践", "实习", "见习",
	},
	IntentEmotion: {
		"焦虑", "抑郁", "失眠", "压力", "困惑", "迷茫",
		"心理", "咨询", "预约心理咨询", "心理中心",
		"情绪", "心情", "不开心", "难过", "烦躁",
	},
	IntentFAQ: {
		"是什么", "什么是", "什么意思", "怎么理解",
		"在哪里", "去哪里", "电话", "联系方式", "地址",
		"几点", "什么时候", "时间", "多久", "截止",
		"学费", "学分", "绩点", "成绩", "考试", "补考", "重修",
		"选课", "课表", "教室",
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

// classify 根据关键词匹配返回意图列表
func (r *Router) classify(lowerQuestion string) []Intent {
	var intents []Intent
	scores := make(map[Intent]int)

	for intent, keywords := range domainKeywords {
		for _, kw := range keywords {
			if strings.Contains(lowerQuestion, kw) {
				scores[intent]++
			}
		}
	}

	// 按匹配分数降序排列
	for intent, score := range scores {
		if score > 0 {
			intents = append(intents, intent)
		}
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
