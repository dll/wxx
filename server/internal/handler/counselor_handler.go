package handler

import (
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// CounselorHandler 辅导员角色 AI 功能接口
type CounselorHandler struct {
	svc       *service.CounselorService
	phase2Svc *service.Phase2Service // 阶段二真实数据服务（谈心记录），可为 nil
}

// NewCounselorHandler 创建辅导员 handler。svc 可为 nil（兼容旧调用），此时所有 AI 功能走 mock
func NewCounselorHandler(svc *service.CounselorService) *CounselorHandler {
	return &CounselorHandler{svc: svc}
}

// SetPhase2Service 注入阶段二真实数据服务（谈心记录，可选依赖）
func (h *CounselorHandler) SetPhase2Service(svc *service.Phase2Service) {
	h.phase2Svc = svc
}

// DailyFocus AI 今日关注 — 真实数据 + LLM 简报
func (h *CounselorHandler) DailyFocus(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			focus, err := h.svc.GenerateDailyFocus(c.Request.Context(), userCtx)
			if err == nil && focus != nil {
				c.JSON(http.StatusOK, focus)
				return
			}
		}
	}
	h.mockDailyFocus(c)
}

// mockDailyFocus 兜底 mock（svc 未注入或异常时使用）
func (h *CounselorHandler) mockDailyFocus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"date":               time.Now().Format("2006-01-02"),
		"class_health_score": 82.5,
		"top_students": []gin.H{
			{"name": "张明", "reason": "连续3天未签到，情绪波动较大", "risk_level": "high", "suggestion": "建议约谈了解情况"},
			{"name": "李华", "reason": "成绩下滑明显，近期作业未提交", "risk_level": "medium", "suggestion": "关注学业状态"},
			{"name": "王芳", "reason": "社交活动减少，独处时间增加", "risk_level": "low", "suggestion": "适当关心"},
		},
		"overview": gin.H{
			"total":     45,
			"normal":    38,
			"attention": 5,
			"warning":   2,
		},
		"data_source": "fallback",
	})
}

// ClassReport 班级学情日报
func (h *CounselorHandler) ClassReport(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			report := h.svc.GenerateClassReport(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if report != nil {
				c.JSON(http.StatusOK, report)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"date": time.Now().Format("2006-01-02"), "class_name": "计科2301班",
		"active_rate": 0.87, "absent_count": 3, "homework_rate": 0.92, "emotion_alert_count": 2, "checkin_rate": 0.93,
		"anomalies":    []string{"张明连续缺勤", "李华作业未交"},
		"ai_narrative": "班级整体状态良好。需关注张明同学连续缺勤情况。",
	})
}

// TwinBoard 学生数字孪生看板
func (h *CounselorHandler) TwinBoard(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			board := h.svc.GenerateTwinBoard(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if board != nil {
				c.JSON(http.StatusOK, board)
				return
			}
		}
	}
	c.JSON(http.StatusOK, []gin.H{
		{"student_id": "s001", "name": "张明", "academic": 65.0, "social": 45.0, "mental": 55.0, "practice": 70.0, "innovate": 48.0, "risk": "high"},
		{"student_id": "s002", "name": "李华", "academic": 72.0, "social": 80.0, "mental": 78.0, "practice": 60.0, "innovate": 65.0, "risk": "medium"},
		{"student_id": "s003", "name": "王芳", "academic": 88.0, "social": 55.0, "mental": 70.0, "practice": 75.0, "innovate": 82.0, "risk": "low"},
	})
}

// Prediction 预测性预警
func (h *CounselorHandler) Prediction(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			predictions := h.svc.GeneratePredictions(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if predictions != nil {
				c.JSON(http.StatusOK, predictions)
				return
			}
		}
	}
	c.JSON(http.StatusOK, []gin.H{
		{"student_id": "s001", "name": "张明", "risk_type": "dropout", "probability": 0.35, "factors": []string{"出勤率低", "成绩下滑", "社交减少"}, "suggestion": "建议尽快约谈"},
	})
}

// Intervention 干预方案
func (h *CounselorHandler) Intervention(c *gin.Context) {
	var req struct {
		StudentID   string `json:"student_id"`
		StudentName string `json:"student_name"`
		RiskLevel   string `json:"risk_level"`
		Reason      string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if h.svc != nil && (req.StudentName != "" || req.StudentID != "") {
		name := req.StudentName
		if name == "" {
			name = req.StudentID
		}
		plan, err := h.svc.GenerateIntervention(c.Request.Context(), name, req.RiskLevel, req.Reason)
		if err == nil && plan != nil {
			c.JSON(http.StatusOK, plan)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"target_student": req.StudentName,
		"risk_level":     req.RiskLevel,
		"urgent_actions": []string{"一对一谈心了解具体困难", "联系心理健康中心评估"},
		"long_term_plan": []string{"建立定期沟通机制", "安排学业帮扶", "持续跟踪每月反馈"},
		"similar_cases":  "同类案例：早期介入是关键，多部门联动效果更好。",
	})
}

// TalkRecord 谈心谈话记录（真实落库）
func (h *CounselorHandler) TalkRecord(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if c.Request.Method == "POST" {
		var req service.TalkRecordInput
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}
		// LLM 结构化提取（内容为空时也允许保存，若提供了 content）
		if h.svc != nil && req.Content != "" {
			if record, err := h.svc.GenerateTalkRecord(c.Request.Context(), &service.TalkRecordRequest{
				StudentName: req.StudentName, Content: req.Content,
			}); err == nil && record != nil {
				if req.Summary == "" {
					req.Summary = record.Summary
				}
				if req.Topic == "" {
					req.Topic = record.Topic
				}
				if req.Emotion == "" {
					req.Emotion = record.Emotion
				}
				if len(req.FollowUps) == 0 && record.FollowUp != "" {
					req.FollowUps = []string{record.FollowUp}
				}
			}
		}
		if h.phase2Svc != nil {
			if id, err := h.phase2Svc.SaveTalkRecord(userCtx.UserID, &req); err == nil {
				c.JSON(http.StatusOK, gin.H{"code": 0, "id": id, "message": "记录保存成功"})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "谈心记录保存失败"})
		return
	}

	// GET：真实记录
	if h.phase2Svc != nil {
		if records, err := h.phase2Svc.ListTalkRecords(userCtx.UserID, 100); err == nil {
			c.JSON(http.StatusOK, records)
			return
		}
	}
	c.JSON(http.StatusOK, []gin.H{})
}

// TalkTips 谈话话术推荐
func (h *CounselorHandler) TalkTips(c *gin.Context) {
	// 前端传 scene/type，后端兼容 profile/scene/type 三种参数
	profile := c.Query("profile")
	if profile == "" {
		profile = c.Query("type")
	}
	if profile == "" {
		profile = c.Query("scene")
	}

	if h.svc != nil && profile != "" {
		tip, err := h.svc.GenerateTalkTips(c.Request.Context(), profile)
		if err == nil && tip != nil {
			// 统一返回 {tips:[...]} 供前端渲染（开场白 + 提问建议）
			tips := []string{tip.OpeningLine}
			tips = append(tips, tip.Questions...)
			c.JSON(http.StatusOK, gin.H{"tips": tips})
			return
		}
	}

	// 兜底
	scene := c.Query("scene")
	tips := []string{
		"你最近感觉怎么样？有什么想和我聊聊的吗？",
		"我注意到你最近的状态有些变化，能和我说说发生了什么吗？",
		"遇到困难是很正常的，我们一起想想办法好吗？",
		"你觉得目前最大的压力来源是什么？",
		"有没有什么我可以帮到你的地方？",
	}
	if scene == "academic" {
		tips = []string{
			"最近课程学习上有没有遇到什么困难？",
			"你觉得哪门课最有挑战性？我们可以一起想想解决办法。",
			"学习方法上有没有需要调整的地方？",
			"要不要我帮你联系学长学姐做一些辅导？",
		}
	}
	c.JSON(http.StatusOK, gin.H{"tips": tips})
}

// Ideological 学生思想档案
func (h *CounselorHandler) Ideological(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			data := h.svc.GenerateIdeologicalSummary(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if data != nil {
				c.JSON(http.StatusOK, data)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"summary":    "班级整体思想状态积极向上，政治学习参与率95%",
		"highlights": []string{"3名同学递交入党申请书", "班级志愿服务时长达标"},
		"concerns":   []string{"个别同学对时事关注度不够"},
		"students": []gin.H{
			{"name": "赵强", "status": "预备党员", "evaluation": "思想觉悟高，积极参与组织活动"},
			{"name": "刘洋", "status": "入党积极分子", "evaluation": "表现良好，建议加强理论学习"},
		},
		"data_source": "fallback",
	})
}

// ClassProfile 班级性格画像
func (h *CounselorHandler) ClassProfile(c *gin.Context) {
	if h.svc != nil {
		profile := h.svc.GenerateClassProfile(c.Request.Context(), c.Query("class"))
		if profile != nil {
			c.JSON(http.StatusOK, profile)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"class_name": "计科2301班", "total": 45,
		"distribution":    gin.H{"外向型": 18, "内向型": 12, "分析型": 8, "感性型": 7},
		"characteristics": []string{"整体偏理性思维", "团队协作意愿强", "创新意识较好"},
		"suggestions":     []string{"多组织团队活动促进内向同学融入", "利用分析型同学带动学术氛围"},
		"data_source":     "fallback",
	})
}

// CommunityManage 社区问答管理
func (h *CounselorHandler) CommunityManage(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GenerateCommunityManage(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"pending_review": []gin.H{
			{"id": "1", "title": "感觉压力很大怎么办", "author": "匿名", "type": "心理求助", "risk": "medium", "time": "2小时前"},
			{"id": "2", "title": "奖学金评定标准有误？", "author": "张同学", "type": "政策误读", "risk": "low", "time": "5小时前"},
		},
		"flagged_posts": []gin.H{{"id": "3", "title": "对某课程评价", "reason": "内容争议", "reports": 3}},
		"stats":         gin.H{"total_posts_today": 12, "reviewed": 8, "official_responses": 2, "hidden": 1},
		"data_source":   "fallback",
	})
}

// HotTopicSense 热点话题感知
func (h *CounselorHandler) HotTopicSense(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GenerateHotTopicSense(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"hot_topics": []gin.H{
			{"title": "期中考试焦虑", "heat": 92, "sentiment": "negative", "affected_students": 15, "suggestion": "建议组织考前辅导和心理疏导"},
			{"title": "实习招聘信息", "heat": 78, "sentiment": "neutral", "affected_students": 22, "suggestion": "可组织就业指导讲座"},
			{"title": "宿舍空调报修", "heat": 65, "sentiment": "negative", "affected_students": 8, "suggestion": "已反馈后勤处，预计3天内解决"},
		},
		"keywords":     []string{"考试", "实习", "焦虑", "空调", "选课"},
		"alert_topics": []gin.H{{"title": "期中考试焦虑", "reason": "多名学生表达负面情绪，需关注心理状态"}},
		"data_source":  "fallback",
	})
}

// ProcessEdit 流程步骤编辑
func (h *CounselorHandler) ProcessEdit(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GetEditableProcesses(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"editable_processes": []gin.H{
			{"id": "1", "title": "请假审批流程", "steps_count": 4, "last_updated": "2026-05-10", "status": "active"},
			{"id": "2", "title": "缓考申请流程", "steps_count": 3, "last_updated": "2026-05-08", "status": "active"},
			{"id": "3", "title": "学生证补办流程", "steps_count": 5, "last_updated": "2026-04-20", "status": "active"},
		},
		"recent_edits": []gin.H{
			{"process": "请假审批流程", "step": "辅导员审批", "field": "office_hours", "old_value": "9:00-17:00", "new_value": "8:30-17:30", "time": "2026-05-12"},
		},
		"permissions": gin.H{"can_edit_contact": true, "can_edit_location": true, "can_edit_faq": true, "can_edit_media": false},
		"data_source": "fallback",
	})
}

// StudentList 学生列表
func (h *CounselorHandler) StudentList(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			data := h.svc.GetStudentList(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if data != nil {
				c.JSON(http.StatusOK, data)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"students": []gin.H{
			{"name": "张明", "student_id": "2023010101", "class_name": "计科2301", "status": "warning", "gpa": 3.2, "checkin_days": 35},
			{"name": "李华", "student_id": "2023010102", "class_name": "计科2301", "status": "alert", "gpa": 2.8, "checkin_days": 28},
			{"name": "王芳", "student_id": "2023010103", "class_name": "计科2301", "status": "normal", "gpa": 3.8, "checkin_days": 42},
			{"name": "赵强", "student_id": "2023010104", "class_name": "计科2301", "status": "normal", "gpa": 3.5, "checkin_days": 40},
			{"name": "刘洋", "student_id": "2023010105", "class_name": "计科2301", "status": "normal", "gpa": 3.6, "checkin_days": 41},
			{"name": "陈静", "student_id": "2023010106", "class_name": "计科2301", "status": "normal", "gpa": 3.9, "checkin_days": 42},
			{"name": "周磊", "student_id": "2023010107", "class_name": "计科2301", "status": "warning", "gpa": 2.9, "checkin_days": 30},
			{"name": "吴敏", "student_id": "2023010108", "class_name": "计科2301", "status": "normal", "gpa": 3.4, "checkin_days": 38},
			{"name": "孙浩", "student_id": "2023010109", "class_name": "计科2301", "status": "normal", "gpa": 3.7, "checkin_days": 39},
			{"name": "郑雪", "student_id": "2023010110", "class_name": "计科2301", "status": "normal", "gpa": 4.0, "checkin_days": 42},
		},
		"total":       45,
		"data_source": "fallback",
	})
}

// ======================== P2 深度分析功能 ========================

// FollowUpReminders 谈话跟进提醒
func (h *CounselorHandler) FollowUpReminders(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			data := h.svc.GenerateFollowUpReminders(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if data != nil {
				c.JSON(http.StatusOK, data)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"reminders": []gin.H{
			{"student": "张明", "task": "期中成绩面谈", "deadline": "2026-05-20", "priority": "high", "status": "pending"},
			{"student": "李华", "task": "心理状态回访", "deadline": "2026-05-22", "priority": "high", "status": "overdue"},
			{"student": "周磊", "task": "学业改进计划", "deadline": "2026-05-25", "priority": "medium", "status": "pending"},
		},
		"data_source": "fallback",
	})
}

// CheckinStats 班级打卡统计
func (h *CounselorHandler) CheckinStats(c *gin.Context) {
	className := c.Query("class")
	if h.svc != nil {
		data := h.svc.GenerateCheckinStats(c.Request.Context(), className)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"class_name":          className,
		"total_students":      45,
		"today_rate":          0.93,
		"streak_distribution": gin.H{"连续7天+": 18, "连续3-6天": 15, "连续1-2天": 7, "今日未打卡": 3},
		"data_source":         "fallback",
	})
}

// SmartNotify 智能群发助手
func (h *CounselorHandler) SmartNotify(c *gin.Context) {
	var req struct {
		Content       string   `json:"content" binding:"required"`
		AudienceTypes []string `json:"audience_types"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}
	if h.svc != nil {
		data := h.svc.GenerateSmartNotification(c.Request.Context(), req.Content, req.AudienceTypes)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"drafts": []gin.H{
			{"channel": "群聊", "content": "【通知】" + req.Content},
			{"channel": "邮件", "content": "各位同学：\n" + req.Content},
		},
		"data_source": "fallback",
	})
}

// MonthlyBrief AI 月度工作简报
func (h *CounselorHandler) MonthlyBrief(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			data := h.svc.GenerateMonthlyBrief(c.Request.Context(), userCtx.OwnerScope, userCtx.OwnerID)
			if data != nil {
				c.JSON(http.StatusOK, data)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"month":          "2026年5月",
		"talks_count":    12,
		"alerts_handled": 5,
		"key_changes":    []string{"学业预警减少2人", "心理健康关注增加1人"},
		"data_source":    "fallback",
	})
}

// SessionInsight AI 会话洞察
func (h *CounselorHandler) SessionInsight(c *gin.Context) {
	studentName := c.Query("student")
	if h.svc != nil {
		var messages []string
		// 尝试从请求体解析消息列表
		var req struct {
			Messages []string `json:"messages"`
		}
		if c.ShouldBindJSON(&req) == nil && len(req.Messages) > 0 {
			messages = req.Messages
		}
		data := h.svc.GenerateSessionInsight(c.Request.Context(), studentName, messages)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"student":     studentName,
		"emotion":     "中性偏积极",
		"key_topics":  []string{"学业规划", "人际关系"},
		"concerns":    []string{"对期末考试成绩有些担忧"},
		"data_source": "fallback",
	})
}
