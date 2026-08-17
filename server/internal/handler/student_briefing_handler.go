package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *StudentHandler) DailyBriefing(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			briefing, err := h.svc.GenerateDailyBriefing(c.Request.Context(), userCtx.UserID)
			if err == nil && briefing != nil {
				h.enrichBriefingWithRealData(briefing, userCtx.UserID)
				c.JSON(http.StatusOK, briefing)
				return
			}
		}
	}
	// 兜底：未注入 svc 或异常时使用旧 mock
	h.mockDailyBriefing(c)
}

// enrichBriefingWithRealData 用 course_schedules / study_plan_tasks / academic_calendar_events
// 的真实数据覆盖速览中的课程、待办与活动（仅当存在真实数据时覆盖，否则保留服务层兜底）
func (h *StudentHandler) enrichBriefingWithRealData(b *service.DailyBriefing, userID int64) {
	if b == nil || h.db == nil {
		return
	}

	today := time.Now().Format("2006-01-02")
	weekday := int(time.Now().Weekday())
	if weekday == 0 {
		weekday = 7
	}
	calendar, _ := h.resolveCurrentCalendar()

	// 今日课程 → courses
	if calendar != nil {
		if realCourses := h.getTodayCourses(userID, weekday, calendar); len(realCourses) > 0 {
			courses := make([]map[string]interface{}, 0, len(realCourses))
			for _, c := range realCourses {
				subtitle := c.Location
				if c.Teacher != "" && subtitle != "" {
					subtitle = c.Location + " · " + c.Teacher
				} else if c.Teacher != "" {
					subtitle = c.Teacher
				}
				courses = append(courses, map[string]interface{}{
					"title":    c.CourseName,
					"subtitle": subtitle,
					"time":     c.Time,
					"icon":     "book",
				})
			}
			b.Courses = courses
		}
	}

	// 今日计划任务 → deadlines
	if realTasks := h.getTodayTasks(userID, today); len(realTasks) > 0 {
		deadlines := make([]map[string]interface{}, 0, len(realTasks))
		for _, t := range realTasks {
			if t.Status == "done" || t.Status == "skipped" {
				continue
			}
			deadlines = append(deadlines, map[string]interface{}{
				"title":    t.Title,
				"subtitle": "今日计划任务",
				"time":     "今天",
				"icon":     "assignment",
			})
		}
		if len(deadlines) > 0 {
			b.Deadlines = deadlines
		}
	}

	// 近期考试/活动 → activities
	if calendar != nil {
		if realEvents := h.getUpcomingEvents(today, calendar); len(realEvents) > 0 {
			activities := make([]map[string]interface{}, 0, len(realEvents))
			for _, e := range realEvents {
				timeLabel := "今天"
				if e.DaysLeft > 0 {
					timeLabel = fmt.Sprintf("%d 天后", e.DaysLeft)
				}
				activities = append(activities, map[string]interface{}{
					"title":    e.EventName,
					"subtitle": eventTypeLabel(e.EventType),
					"time":     timeLabel,
					"icon":     "event",
				})
			}
			b.Activities = activities
		}
	}
}

// eventTypeLabel 校历事件类型的中文标签
func eventTypeLabel(t string) string {
	switch t {
	case "exam":
		return "考试"
	case "deadline":
		return "截止"
	case "holiday":
		return "节假日"
	case "activity":
		return "校园活动"
	case "register":
		return "报到注册"
	case "break":
		return "放假"
	default:
		return "校历"
	}
}

// mockDailyBriefing 兜底 mock 数据（保留给开发环境/svc 未注入场景）
func (h *StudentHandler) mockDailyBriefing(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	c.JSON(http.StatusOK, gin.H{
		"date":     today,
		"greeting": "早上好！今天有3节课，1个作业截止。",
		"courses": []gin.H{
			{"title": "数据结构", "subtitle": "第8周 · 二叉树遍历", "time": "08:00-09:40", "icon": "book"},
			{"title": "操作系统", "subtitle": "第8周 · 进程调度", "time": "10:00-11:40", "icon": "computer"},
			{"title": "英语听说", "subtitle": "Unit 6 Presentation", "time": "14:00-15:40", "icon": "language"},
		},
		"deadlines": []gin.H{
			{"title": "数据结构实验报告", "subtitle": "二叉树实现", "time": "今天 23:59", "icon": "assignment"},
		},
		"activities": []gin.H{
			{"title": "ACM 训练赛", "subtitle": "信息楼 301", "time": "19:00", "icon": "emoji_events"},
		},
		"weather":     "晴 26°C",
		"motto":       "学如逆水行舟，不进则退。",
		"data_source": "fallback",
	})
}

// LearningDiary 学习日记
func (h *StudentHandler) LearningDiary(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			diary, err := h.svc.GenerateLearningDiary(c.Request.Context(), userCtx.UserID)
			if err == nil && diary != nil {
				c.JSON(http.StatusOK, diary)
				return
			}
		}
	}
	// 兜底
	h.mockLearningDiary(c)
}

func (h *StudentHandler) mockLearningDiary(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	c.JSON(http.StatusOK, gin.H{
		"date":            today,
		"courses_studied": []string{"数据结构", "操作系统"},
		"key_points":      []string{"二叉树前序/中序/后序遍历", "进程状态转换图", "死锁四个必要条件"},
		"study_minutes":   185,
		"quiz": []gin.H{
			{"question": "二叉树前序遍历的顺序是？", "options": []string{"根左右", "左根右", "左右根", "右根左"}, "correct_index": 0, "explanation": "前序遍历先访问根节点，再递归左子树，最后右子树"},
		},
		"tomorrow_plan": "复习操作系统第5章，完成数据结构实验",
		"encouragement": "今天学习了3小时05分钟，比昨天多了20分钟，继续保持！",
	})
}
