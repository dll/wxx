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

		// 兜底 mock
		responses := map[string]gin.H{
			"freshman-plan":       {"content": "大一规划建议", "response": "建议重点关注数学和编程基础课程，积极参加社团活动拓展视野。", "data_source": "fallback"},
			"growth-path":         {"content": "成长路径分析", "response": "你目前处于大二下学期，建议本学期提升算法能力，暑假寻找实习机会。", "data_source": "fallback"},
			"political-study":     {"content": "政治学习", "response": "本周学习主题：习近平新时代中国特色社会主义思想。已整理学习要点。", "data_source": "fallback"},
			"ideological-record":  {"content": "思想档案", "response": "思想政治表现良好，建议继续保持对时事的关注，多参与志愿服务。", "data_source": "fallback"},
			"party-progress":      {"content": "入党进度", "response": "当前阶段：入党积极分子。下一步参加组织考察，建议积极参与志愿服务。", "data_source": "fallback"},
			"campus-life":         {"content": "校园生活", "response": "本周推荐：周三技术沙龙、周五篮球赛、周末志愿者活动。", "data_source": "fallback"},
			"schedule":            {"content": "日程管理", "response": "今日：上午2节课，下午1节课，晚上建议复习数据结构。", "data_source": "fallback"},
			"competition-match":   {"content": "竞赛推荐", "response": "推荐：ACM程序设计竞赛(95%)、数学建模(80%)、创新创业大赛(70%)。", "data_source": "fallback"},
			"study-buddy":         {"content": "学伴匹配", "response": "推荐3位学伴：张三(数据结构)、李四(算法练习)、王五(英语口语)。", "data_source": "fallback"},
			"mental-health":       {"content": "心理健康", "response": "整体心理状态良好。建议保持规律作息，适当运动放松。", "data_source": "fallback"},
			"digital-mentor":      {"content": "AI导师", "response": "本周建议重点关注数据结构中图的相关算法，这是目前的薄弱环节。", "data_source": "fallback"},
			"classroom-extension": {"content": "课堂延伸", "response": "课后要点：1.理解核心概念 2.掌握典型例题 3.思考实际应用。建议用思维导图整理知识结构。", "data_source": "fallback"},
			"values-guidance":     {"content": "价值观引导", "response": "诚信、责任、奉献、感恩是大学生应具备的核心价值观。建议从日常生活小事做起。", "data_source": "fallback"},
		}
		if resp, ok := responses[feature]; ok {
			c.JSON(http.StatusOK, resp)
		} else {
			c.JSON(http.StatusOK, gin.H{"content": feature, "response": "功能开发中", "data_source": "fallback"})
		}
	}
}
