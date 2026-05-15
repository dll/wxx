package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CollegeHandler 学院管理员角色 AI 功能接口
type CollegeHandler struct{}

func NewCollegeHandler() *CollegeHandler {
	return &CollegeHandler{}
}

// TwinScreen 学院数字孪生大屏
func (h *CollegeHandler) TwinScreen(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"college":    "信息学院",
		"updated_at": time.Now().Format("2006-01-02 15:04"),
		"overview": gin.H{
			"total_students": 580,
			"health_score":   85.2,
			"risk_students":  12,
			"active_rate":    0.78,
		},
		"departments": []gin.H{
			{"name": "计算机科学", "students": 240, "health": 87.5, "risk": 4},
			{"name": "软件工程", "students": 180, "health": 83.0, "risk": 5},
			{"name": "信息安全", "students": 160, "health": 85.8, "risk": 3},
		},
		"trends": gin.H{
			"academic": []float64{82, 83, 85, 84, 86, 85.2},
			"emotion":  []float64{78, 80, 79, 82, 81, 83},
			"activity": []float64{70, 72, 75, 73, 76, 78},
		},
	})
}

// DataAnalysis AI 全量数据分析
func (h *CollegeHandler) DataAnalysis(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"response": "信息学院数据分析报告(2026年5月): 全院平均绩点3.12(较上月+0.05), 挂科率4.2%(较上月-0.8%), 课堂出勤率92.5%, 心理预警学生12人(较上月-3人), 情感正向率78%, 本月活动参与率65%, 预计期末挂科率将降至3.5%",
	})
}
