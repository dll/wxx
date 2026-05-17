package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// CollegeHandler 学院管理员角色 AI 功能接口
type CollegeHandler struct {
	svc *service.CollegeService
}

func NewCollegeHandler(svc *service.CollegeService) *CollegeHandler {
	return &CollegeHandler{svc: svc}
}

// TwinScreen 学院数字孪生大屏
func (h *CollegeHandler) TwinScreen(c *gin.Context) {
	college := c.Query("college")

	if h.svc != nil {
		data := h.svc.GenerateTwinScreen(c.Request.Context(), college)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"college": "信息学院",
		"overview": gin.H{
			"total_students": 580, "health_score": 85.2, "risk_students": 12, "active_rate": 0.78,
		},
		"departments": []gin.H{
			{"name": "计算机科学", "students": 240, "health": 87.5, "risk": 4},
			{"name": "软件工程", "students": 180, "health": 83.0, "risk": 5},
		},
		"data_source": "fallback",
	})
}

// DataAnalysis AI 全量数据分析
func (h *CollegeHandler) DataAnalysis(c *gin.Context) {
	query := c.Query("q")

	if h.svc != nil {
		result := h.svc.AnalyzeData(c.Request.Context(), query)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"content":     "信息学院数据分析报告：全院平均绩点3.12，挂科率4.2%，出勤率92.5%。",
		"data_source": "fallback",
	})
}

// ======================== P2 深度分析功能 ========================

// DecisionAdvice AI 决策建议
func (h *CollegeHandler) DecisionAdvice(c *gin.Context) {
	topic := c.Query("topic")
	if h.svc != nil {
		data := h.svc.GenerateDecisionAdvice(c.Request.Context(), topic)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"topic":       topic,
		"suggestions": []gin.H{{"action": "增加心理健康活动频次", "reason": "心理预警人数上升"}},
		"data_source": "fallback",
	})
}

// TeacherEfficiency 教师效能分析
func (h *CollegeHandler) TeacherEfficiency(c *gin.Context) {
	teacherName := c.Query("teacher")
	if h.svc != nil {
		data := h.svc.AnalyzeTeacherEfficiency(c.Request.Context(), teacherName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"teacher_name": teacherName,
		"scores":       gin.H{"教学": 88.0, "学情": 82.5},
		"data_source":  "fallback",
	})
}

// CourseQuality 课程质量评估
func (h *CollegeHandler) CourseQuality(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.EvaluateCourseQuality(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"course_name": courseName,
		"grade":       "B",
		"metrics":     gin.H{"pass_rate": 0.88, "avg_score": 76.5},
		"data_source": "fallback",
	})
}

// CollegeReport 周报/月报
func (h *CollegeHandler) CollegeReport(c *gin.Context) {
	period := c.Query("period")
	if h.svc != nil {
		data := h.svc.GenerateCollegeReport(c.Request.Context(), period)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"period":      period,
		"key_metrics": gin.H{"avg_health": 82.5, "avg_academic": 76.0},
		"data_source": "fallback",
	})
}
