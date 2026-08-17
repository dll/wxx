package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

func (h *StudentHandler) CourseMap(c *gin.Context) {
	if h.svc != nil {
		kg := h.svc.GenerateKnowledgeGraph(c.Request.Context(), c.Query("course"))
		if kg != nil {
			c.JSON(http.StatusOK, kg)
			return
		}
	}
	c.JSON(http.StatusOK, []gin.H{
		{"id": "cs101", "name": "程序设计基础", "credits": 4, "semester": 1, "status": "completed", "prerequisites": []string{}, "category": "专业核心"},
		{"id": "cs201", "name": "数据结构", "credits": 4, "semester": 3, "status": "current", "prerequisites": []string{"cs101"}, "category": "专业核心"},
		{"id": "cs301", "name": "算法设计", "credits": 3, "semester": 5, "status": "pending", "prerequisites": []string{"cs201"}, "category": "专业核心"},
		{"id": "cs202", "name": "操作系统", "credits": 4, "semester": 3, "status": "current", "prerequisites": []string{"cs101"}, "category": "专业核心"},
		{"id": "math101", "name": "高等数学", "credits": 5, "semester": 1, "status": "completed", "prerequisites": []string{}, "category": "公共基础"},
	})
}

// CourseAnalytics 课程学情
func (h *StudentHandler) CourseAnalytics(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			// 优先真实课程学情看板（成绩 + 班级匿名基准 + LLM 建议）
			if result, err := h.svc.GenerateCourseAnalytics(c.Request.Context(), userCtx.UserID); err == nil && result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, []gin.H{
		{"course_name": "数据结构", "progress": 0.65, "rank_percentile": 25, "knowledge_points": []gin.H{{"name": "链表", "mastery": 0.9}, {"name": "二叉树", "mastery": 0.6}, {"name": "图", "mastery": 0.3}}, "weak_points": []string{"图的遍历", "最短路径算法"}},
		{"course_name": "操作系统", "progress": 0.55, "rank_percentile": 40, "knowledge_points": []gin.H{{"name": "进程管理", "mastery": 0.8}, {"name": "内存管理", "mastery": 0.5}}, "weak_points": []string{"页面置换算法"}},
	})
}

// WeeklyReport 学习周报
func (h *StudentHandler) WeeklyReport(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			report := h.svc.GenerateWeeklyReport(c.Request.Context(), userCtx.UserID)
			if report != nil {
				c.JSON(http.StatusOK, report)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"week":            fmt.Sprintf("第%d周", int(time.Now().YearDay()/7)+1),
		"total_hours":     22.5,
		"courses_count":   5,
		"assignments":     3,
		"rank_change":     2,
		"highlights":      []string{"数据结构实验满分", "英语演讲获得A"},
		"improvements":    []string{"操作系统作业需加强", "体育锻炼不足"},
		"next_week_goals": []string{"完成算法作业", "准备期中考试"},
		"data_source":     "fallback",
	})
}
