package handler

// vOPC L1 概念层 + L2 虚拟向导模板（v2.0 新增）。
//
// L1 概念层（/vopc/learning）：纯内容 + 交互，零环境依赖，无需建项目。返回 OPC 核心知识卡、
// OPC 核心流程图（idea → validate → build → deliver → feedback 五步）与自测小问卷。
// L2 虚拟向导模板（/vopc/guides）：按项目类型返回虚拟向导模板 + 角色扮演提示。
//
// 内容本质是静态知识，但也提供数据表（vopc_learning_cards / vopc_quizzes）兜底扩展位——
// 若表未播种，则回退到内置默认内容，保证接口在任何环境（含无迁移种子）都能返回有效数据。

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// opcFlowSteps 是 OPC 核心五步闭环（idea → validate → build → deliver → feedback）。
var opcFlowSteps = []gin.H{
	{"key": "idea", "title": "想法", "desc": "找到要解决的问题：为什么做、为谁做、价值假设是什么。"},
	{"key": "validate", "title": "验证", "desc": "低成本验证假设：访谈、问卷、模拟反馈，判断想法是否成立。"},
	{"key": "build", "title": "构建", "desc": "做出最小可审阅的虚拟产出：文档、设计、数据、脚本或演示。"},
	{"key": "deliver", "title": "交付", "desc": "整理成果、约定验收标准，把产出交付给目标用户/评审者。"},
	{"key": "feedback", "title": "反馈", "desc": "收集反馈、复盘要点，决定继续、转向、暂停还是终止。"},
}

// opcKnowledgeCards 是 OPC 核心知识卡（一人公司最小闭环的四要素）。
var opcKnowledgeCards = []gin.H{
	{"key": "what", "title": "OPC 是什么", "body": "OPC（One-Person Company，一人公司）：一个人像一家公司一样，独立承担产品、市场、交付等多数角色，把一个小想法推进为可交付的成果。"},
	{"key": "why", "title": "一人公司为什么成立", "body": "单点专注、成本低、决策快、反馈短。关键不是「一个人干所有事」，而是「承担所有关键责任」。"},
	{"key": "loop", "title": "最小闭环", "body": "一个 OPC ≈ 需求方 + 产品/服务 + 交付 + 反馈。四者缺一不可，循环闭环即生意。"},
	{"key": "mindset", "title": "核心心态", "body": "先验证再投入，先交付再完善；每一步都要能回溯、能复盘、能讲清楚。"},
}

// opcQuizzes 是 L1 自测小问卷（5 分钟掌握核心思想的自查）。
var opcQuizzes = []gin.H{
	{"q": "OPC 的「最小闭环」由哪些要素构成？", "options": []string{"需求方 + 产品/服务 + 交付 + 反馈", "招聘 + 融资 + 办公 + 法务", "研发 + 测试 + 运维 + 销售"}, "answer": 0},
	{"q": "OPC 强调一人公司的核心是什么？", "options": []string{"一个人什么都不做", "一个人承担产品、市场、交付等多数关键角色", "一个人雇佣很多人"}, "answer": 1},
	{"q": "OPC 核心流程的正确顺序是？", "options": []string{"build → idea → feedback → validate → deliver", "idea → validate → build → deliver → feedback", "deliver → feedback → idea → build → validate"}, "answer": 1},
	{"q": "验证想法的正确姿势是？", "options": []string{"直接大规模投入", "低成本先验证假设，再决定是否投入", "等所有人都说好再做"}, "answer": 1},
}

// opcGuideTemplates 是虚拟向导模板（按项目类型列出的引导角色与提示）。
var opcGuideTemplates = []gin.H{
	{"project_type": "软件与 AI 产品", "guides": []gin.H{
		{"role_key": "project_manager", "name": "产品经理向导", "prompt": "用一句话说清要解决的软件问题与目标用户。"},
		{"role_key": "product_solution", "name": "产品与方案向导", "prompt": "给出 MVP 形态与技术取舍，约定验收标准。"},
		{"role_key": "execution", "name": "交付执行向导", "prompt": "把 MVP 拆成可验收的任务与版本。"},
	}},
	{"project_type": "内容与知识产品", "guides": []gin.H{
		{"role_key": "market_user", "name": "市场与用户向导", "prompt": "识别内容的目标受众与需求痛点。"},
		{"role_key": "product_solution", "name": "产品与方案向导", "prompt": "设计内容结构与分发方式，明确交付形态。"},
	}},
	{"project_type": "校园服务创新", "guides": []gin.H{
		{"role_key": "market_user", "name": "市场与用户向导", "prompt": "访谈目标校园用户，归纳真实痛点。"},
		{"role_key": "execution", "name": "交付执行向导", "prompt": "设计最小可运行的服务流程与验证方式。"},
	}},
}

// Learning 返回 L1 概念层内容：知识卡、核心流程图、自测问卷（零环境依赖，纯内容）。
// GET /api/v1/vopc/learning
func (h *VOPCHandler) Learning(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"knowledge_cards": opcKnowledgeCards,
			"flow_steps":      opcFlowSteps,
			"quizzes":         opcQuizzes,
		},
	})
}

// Guides 返回 L2 虚拟向导模板 + 角色扮演提示（按项目类型）。
// GET /api/v1/vopc/guides
func (h *VOPCHandler) Guides(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"templates":   opcGuideTemplates,
			"role_options": []gin.H{
				{"role_key": "project_manager", "name": "产品经理向导"},
				{"role_key": "market_user", "name": "市场与用户向导"},
				{"role_key": "product_solution", "name": "产品与方案向导"},
				{"role_key": "execution", "name": "交付执行向导"},
			},
		},
	})
}
