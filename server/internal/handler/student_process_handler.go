package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// type: enrollment（入学）/ graduation（离校）/ major_change（转专业）/ student_loan（助学贷款）/ leave（请假）/ scholarship（奖学金）
func (h *StudentHandler) ProcessEnhanced(c *gin.Context) {
	flowType := c.DefaultQuery("type", "enrollment")

	if h.svc != nil {
		kb, steps, card, err := h.svc.GetProcessEnhanced(flowType, "", "")
		if err == nil {
			flowTitle := defaultFlowTitle(flowType)
			if kb != nil {
				flowTitle = kb.Title
			}
			resp := gin.H{
				"processes": []gin.H{
					{
						"id":           "1",
						"title":        flowTitle,
						"status":       "in_progress",
						"current_step": 0,
						"steps":        steps,
					},
				},
				"reminders": []gin.H{},
			}
			if card != nil {
				resp["answer_card"] = card
			}
			c.JSON(http.StatusOK, resp)
			return
		}
	}
	c.JSON(http.StatusInternalServerError, model.ErrorResponse{
		Code:    500,
		Message: "服务不可用",
		TraceID: middleware.GetTraceID(c),
	})
}

// FreshmenGuide 返回聚合后的新生指南知识资源与报到步骤。
// GET /api/v1/student/freshmen-guide
func (h *StudentHandler) FreshmenGuide(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "新生指南服务不可用",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	guide, err := h.svc.GetFreshmenGuide()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    500,
			Message: "新生指南加载失败",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}
	c.JSON(http.StatusOK, guide)
}

// defaultFlowTitle 根据 flowType 返回默认流程标题（当 KB 未命中时使用）
func defaultFlowTitle(flowType string) string {
	switch flowType {
	case "graduation":
		return "毕业生离校流程"
	case "major-transfer", "major_transfer", "major_change":
		return "转专业流程"
	case "student-loan", "student_loan":
		return "助学贷款申请流程"
	case "leave":
		return "学生请假办理流程"
	case "scholarship":
		return "奖学金申请流程"
	default:
		return "新生入学报到流程"
	}
}

// GenericAI 通用 AI 响应（用于多个简单功能）
func (h *StudentHandler) GenericAI(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试用 LLM 生成
		if h.svc != nil {
			userCtx := middleware.GetUserContext(c)
			if userCtx != nil {
				result := h.svc.GenerateAIResponse(c.Request.Context(), feature, userCtx.UserID)
				if result != nil {
					c.JSON(http.StatusOK, result)
					return
				}
			}
		}

		// 服务未装配时仍返回可编辑的通用建议，不伪造课程、活动、成绩或个人阶段。
		responses := map[string]gin.H{
			"freshman-plan":       {"content": "大一规划建议", "response": "规划模板：适应校园与课程节奏，探索兴趣方向，建立学习和健康习惯，再根据复盘结果调整目标。具体课程与校历请以教务系统为准。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "填写你的目标和可投入时间。"},
			"growth-path":         {"content": "成长路径分析", "response": "成长模板：从课程、项目、协作和身心状态四方面设定里程碑，用每周可验证成果复盘。年级、成绩和薄弱项请由本人补充。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "补充当前目标后生成计划。"},
			"political-study":     {"content": "政治学习", "response": "学习模板：选择一个正式主题，记录权威来源、核心观点和个人思考；政策与时事内容请以官方发布为准。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "提供主题或材料。"},
			"ideological-record":  {"content": "思想档案", "response": "记录模板：按时间填写理论学习、实践参与和反思证据，不自动推断次数、评价或表现。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "填写真实经历后生成摘要。"},
			"party-progress":      {"content": "入党进度", "response": "流程参考：申请、积极分子、发展对象、预备党员、转正。阶段条件和材料由组织部门核验，请不要把示例当作个人进度。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "核对组织部门通知。"},
			"campus-life":         {"content": "校园生活", "response": "生活规划模板：列出就餐、学习空间、交通和活动需求，再通过校园官方平台核对实时信息。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "补充日期与实际需求。"},
			"schedule":            {"content": "日程管理", "response": "日程模板：导入真实课表后按优先级安排课程、作业、复习和休息，给每项任务设置下一步。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "提供课表或待办事项。"},
			"competition-match":   {"content": "竞赛推荐", "response": "竞赛筛选模板：比较兴趣、技能、团队、投入周期和官方赛程，不生成虚构匹配度或名额。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "填写方向和目标时间。"},
			"study-buddy":         {"content": "学伴匹配", "response": "学伴需求模板：明确课程或技能、可投入时间和协作方式；匹配结果应以真实用户同意和平台数据为准。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "填写希望协作的主题。"},
			"mental-health":       {"content": "心理健康", "response": "关怀建议：先描述当下感受和压力来源，再选择呼吸、散步、规律作息或联系可信任的人。如持续困扰，请联系学校心理中心。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "必要时预约专业支持。"},
			"digital-mentor":      {"content": "AI导师", "response": "导师模板：根据你主动提供的课程、目标和困难拆解练习任务，不替你判断成绩或薄弱项。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "描述一个具体困难。"},
			"classroom-extension": {"content": "课堂延伸", "response": "复习模板：整理核心概念、例题、疑点和应用场景；课程内容和扩展材料请以任课教师及正式教材为准。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "粘贴课堂笔记或问题。"},
			"values-guidance":     {"content": "价值观引导", "response": "反思模板：从诚信、责任、奉献、感恩中选择一个具体场景，记录事实、选择和改进动作，避免空泛评价。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "描述需要复盘的场景。"},
		}
		if resp, ok := responses[feature]; ok {
			c.JSON(http.StatusOK, resp)
		} else {
			c.JSON(http.StatusOK, gin.H{"content": feature, "response": "请描述你的目标、现状和限制条件，我会给出可编辑的行动建议。具体信息请以正式来源为准。", "data_source": "rule", "review_required": true, "sources": []string{}, "next_action": "补充上下文后重新生成。"})
		}
	}
}
