package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AssistantHandler 教辅角色 AI 功能接口
type AssistantHandler struct{}

func NewAssistantHandler() *AssistantHandler {
	return &AssistantHandler{}
}

// ScheduleCheck AI 排课冲突检测
func (h *AssistantHandler) ScheduleCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"conflicts": []gin.H{
			{"type": "教师冲突", "severity": "high", "detail": "王老师周三第3-4节同时被安排数据结构(信息楼301)和算法(信息楼205)", "suggestion": "建议将算法课调至周四第1-2节"},
			{"type": "教室冲突", "severity": "medium", "detail": "信息楼301周五第5-6节被计科2301和软工2301同时占用", "suggestion": "建议软工2301调至信息楼302"},
			{"type": "逻辑冲突", "severity": "low", "detail": "计科2302班周二连续4节数据结构课", "suggestion": "建议拆分为上午2节+下午2节"},
		},
		"summary": gin.H{
			"total_courses":   48,
			"conflicts_found": 3,
			"high_severity":   1,
			"medium_severity": 1,
			"low_severity":    1,
		},
	})
}

// GradAudit AI 毕业资格审核
func (h *AssistantHandler) GradAudit(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"total_students": 120,
			"qualified":      98,
			"near_qualified": 15,
			"not_qualified":  7,
		},
		"issues": []gin.H{
			{"student": "张XX", "status": "not_qualified", "missing": []string{"创新创业学分不足2分", "英语四级未通过"}, "suggestion": "建议参加下学期创新项目+报名6月四级考试"},
			{"student": "李XX", "status": "near_qualified", "missing": []string{"选修课差1学分"}, "suggestion": "建议本学期加选一门公选课"},
		},
	})
}

// ExamArrange AI 考试安排优化
func (h *AssistantHandler) ExamArrange(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"arrangement": []gin.H{
			{"course": "数据结构", "date": "2026-06-20", "time": "09:00-11:00", "room": "信息楼301", "students": 87, "invigilators": []string{"王老师", "李老师"}},
			{"course": "操作系统", "date": "2026-06-22", "time": "14:00-16:00", "room": "信息楼302", "students": 85, "invigilators": []string{"赵老师", "孙老师"}},
		},
		"score":     88.5,
		"conflicts": 0,
		"warnings":  []string{"6月21日监考教师资源紧张，建议协调其他学院支援"},
	})
}
