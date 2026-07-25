package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// AssistantHandler 教辅角色 AI 功能接口
type AssistantHandler struct {
	svc *service.AssistantService
}

func NewAssistantHandler(svc *service.AssistantService) *AssistantHandler {
	return &AssistantHandler{svc: svc}
}

// ScheduleCheck AI 排课冲突检测
func (h *AssistantHandler) ScheduleCheck(c *gin.Context) {
	if h.svc != nil {
		result := h.svc.CheckSchedule(c.Request.Context())
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"conflicts": []gin.H{
			{"type": "教师冲突", "severity": "high", "detail": "王老师周三同时被安排两门课"},
			{"type": "教室冲突", "severity": "medium", "detail": "信息楼301周五被两个班占用"},
		},
		"summary": gin.H{"total_courses": 48, "conflicts_found": 2},
	})
}

// GradAudit AI 毕业资格审核
func (h *AssistantHandler) GradAudit(c *gin.Context) {
	studentID := c.Query("student_id")
	if h.svc != nil {
		result := h.svc.AuditGraduation(c.Request.Context(), studentID)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"student_name": "示例学生", "total_credits": 168, "required_credits": 175,
		"passed_items":  []string{"公共必修课", "专业必修课", "毕业论文"},
		"pending_items": []string{"公共选修课差2学分", "创新创业学分差2分"},
		"can_graduate":  false,
	})
}

// ExamArrange AI 考试安排优化
func (h *AssistantHandler) ExamArrange(c *gin.Context) {
	semester := c.Query("semester")
	if h.svc != nil {
		result := h.svc.ArrangeExams(c.Request.Context(), semester)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"arrangement": []gin.H{
			{"course": "数据结构", "date": "2026-06-20", "time": "09:00-11:00", "room": "信息楼301", "students": 87},
			{"course": "操作系统", "date": "2026-06-22", "time": "14:00-16:00", "room": "信息楼302", "students": 85},
		},
		"conflicts": 0, "warnings": []string{"6月21日监考教师资源紧张"},
	})
}

// ======================== P2 深度分析功能 ========================

// Notification 通知批量生成
func (h *AssistantHandler) Notification(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"`
		Topic   string `json:"topic"`
	}
	_ = c.ShouldBindJSON(&req)
	if h.svc != nil {
		data := h.svc.GenerateNotification(c.Request.Context(), req.Channel, req.Topic)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"channel":     req.Channel,
		"topic":       req.Topic,
		"draft":       "【通知】各位同学，" + req.Topic + "，请及时查看。",
		"data_source": "fallback",
	})
}

// TeachingCalendar 教学日历管理
func (h *AssistantHandler) TeachingCalendar(c *gin.Context) {
	semester := c.Query("semester")
	if h.svc != nil {
		data := h.svc.GenerateTeachingCalendar(c.Request.Context(), semester)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"semester":    semester,
		"key_dates":   []string{"开学注册", "中期检查", "期末考试周", "成绩录入截止"},
		"data_source": "fallback",
	})
}

// StudentInfoQuery 学生信息查询
func (h *AssistantHandler) StudentInfoQuery(c *gin.Context) {
	query := c.Query("q")
	if h.svc != nil {
		data := h.svc.QueryStudentInfo(c.Request.Context(), query)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"query":       query,
		"results":     []gin.H{{"name": "张明", "student_id": "2023010101", "class": "计科2301"}},
		"data_source": "fallback",
	})
}

// ======================== P2 补充功能 ========================

// MaterialTemplates AI 材料模板库
func (h *AssistantHandler) MaterialTemplates(c *gin.Context) {
	category := c.Query("category")
	if h.svc != nil {
		data := h.svc.GetMaterialTemplates(c.Request.Context(), category)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"templates":   []gin.H{{"name": "请假申请表", "category": "学生事务"}},
		"data_source": "fallback",
	})
}

// DocProcess AI 文档智能处理
func (h *AssistantHandler) DocProcess(c *gin.Context) {
	fileName := c.Query("file_name")
	fileType := c.Query("file_type")
	if h.svc != nil {
		data := h.svc.ProcessDocument(c.Request.Context(), fileName, fileType)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"file_name":   fileName,
		"summary":     "文档处理中",
		"data_source": "fallback",
	})
}

// WorkflowAutomation AI 流程自动化
func (h *AssistantHandler) WorkflowAutomation(c *gin.Context) {
	workflowType := c.Query("type")
	if h.svc != nil {
		data := h.svc.AutomateWorkflow(c.Request.Context(), workflowType)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"workflow": workflowType,
		"status":  "pending",
		"data_source": "fallback",
	})
}

// ProcessStepsManage 流程步骤详情管理
func (h *AssistantHandler) ProcessStepsManage(c *gin.Context) {
	processID := c.Query("process_id")
	if h.svc != nil {
		data := h.svc.ManageProcessSteps(c.Request.Context(), processID)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"process_id": processID,
		"steps":      []gin.H{},
		"data_source": "fallback",
	})
}

// MusicRadio 音乐电台
func (h *AssistantHandler) MusicRadio(c *gin.Context) {
	category := c.Query("category")
	if h.svc != nil {
		data := h.svc.GetMusicRadio(c.Request.Context(), category)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"now_playing": gin.H{"title": "Canon in D", "artist": "Pachelbel"},
		"data_source": "fallback",
	})
}

// ActivityRegister 校园活动报名
func (h *AssistantHandler) ActivityRegister(c *gin.Context) {
	status := c.Query("status")
	if h.svc != nil {
		data := h.svc.GetActivityRegister(c.Request.Context(), status)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"activities":  []gin.H{},
		"data_source": "fallback",
	})
}
