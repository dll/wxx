package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// StudentHandler 学生角色 AI 功能接口
type StudentHandler struct {
	svc            *service.StudentService
	twinSvc        *service.TwinService        // 数字孪生五维聚合服务，可为 nil（走兜底 mock）
	checkinSvc     *service.CheckinService     // 打卡服务，可为 nil
	personalitySvc *service.PersonalityService // 性格洞察服务，可为 nil
	phase2Svc      *service.Phase2Service      // 阶段二真实数据服务（积分/问答），可为 nil
	db             *sql.DB
}

// NewStudentHandler 创建学生 handler。svc 可为 nil（兼容旧调用），此时所有 AI 功能走兜底
func NewStudentHandler(svc *service.StudentService, db *sql.DB) *StudentHandler {
	return &StudentHandler{svc: svc, db: db}
}

// SetTwinService 注入数字孪生服务（可选依赖，装配期调用）
func (h *StudentHandler) SetTwinService(twinSvc *service.TwinService) {
	h.twinSvc = twinSvc
}

// SetCheckinService 注入打卡服务（可选依赖，装配期调用）
func (h *StudentHandler) SetCheckinService(svc *service.CheckinService) {
	h.checkinSvc = svc
}

// SetPersonalityService 注入性格洞察服务（可选依赖，装配期调用）
func (h *StudentHandler) SetPersonalityService(svc *service.PersonalityService) {
	h.personalitySvc = svc
}

// SetPhase2Service 注入阶段二真实数据服务（积分/问答，可选依赖）
func (h *StudentHandler) SetPhase2Service(svc *service.Phase2Service) {
	h.phase2Svc = svc
}

// DailyBriefing 今日速览 — 真实数据 + LLM 个性化生成
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

// Checkin 打卡
func (h *StudentHandler) Checkin(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	var req struct {
		Mood string `json:"mood"` // happy/normal/tired/sad
		Note string `json:"note"` // 一句话感想
	}
	_ = c.ShouldBindJSON(&req)

	if h.checkinSvc != nil {
		alreadyChecked := h.checkinSvc.GetHistory(userCtx.UserID).TodayChecked
		result := h.checkinSvc.DoCheckin(userCtx.UserID, req.Mood, req.Note)
		// 首次打卡加分（当日幂等，避免重复刷积分）
		if !alreadyChecked && h.phase2Svc != nil && result.Success {
			_ = h.phase2Svc.AddPoints(userCtx.UserID, 5, "今日学习打卡", "checkin")
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": result.Message, "data": result})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "打卡成功"})
}

// CheckinHistory 打卡历史
func (h *StudentHandler) CheckinHistory(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if h.checkinSvc != nil {
		result := h.checkinSvc.GetHistory(userCtx.UserID)
		// 附加最近 30 天打卡日期（供日历渲染）
		dates, _ := h.checkinSvc.GetRecentDates(userCtx.UserID, 30)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "recent_dates": dates})
		return
	}

	// 兜底 mock
	today := time.Now().Format("2006-01-02")
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"date":           today,
			"streak":         0,
			"total_days":     0,
			"longest_streak": 0,
			"today_checked":  false,
		},
		"recent_dates": []string{},
	})
}

// DigitalTwin 数字孪生
// GrowthPath 成长路径（S1 功能7）：优先真实五维快照分阶段路线图，失败回落通用 AI 文案
func (h *StudentHandler) GrowthPath(c *gin.Context) {
	if h.svc != nil {
		if userCtx := middleware.GetUserContext(c); userCtx != nil {
			if result, err := h.svc.GenerateGrowthPath(c.Request.Context(), userCtx.UserID); err == nil && result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	// 兜底：无孪生快照时走通用 LLM 文案（与原 GenericAI 一致，保证前端可用）
	h.GenericAI("growth-path")(c)
}

func (h *StudentHandler) DigitalTwin(c *gin.Context) {
	// 优先走真实五维聚合服务（S1.1 数字孪生数据底座）
	if h.twinSvc != nil {
		if userCtx := middleware.GetUserContext(c); userCtx != nil {
			result, err := h.twinSvc.GetDigitalTwin(c.Request.Context(), userCtx.UserID)
			if err == nil && result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}

	// 兜底：未注入 twinSvc 或聚合异常时返回 mock（保证前端可用，不阻断）
	c.JSON(http.StatusOK, gin.H{
		"dimensions": []gin.H{
			{"name": "学业", "score": 78.5, "label": "良好"},
			{"name": "社交", "score": 65.0, "label": "中等"},
			{"name": "身心", "score": 82.0, "label": "良好"},
			{"name": "实践", "score": 70.0, "label": "中等"},
			{"name": "创新", "score": 55.0, "label": "待提升"},
		},
		"ideal_dimensions": []gin.H{
			{"name": "学业", "score": 90.0, "label": "优秀"},
			{"name": "社交", "score": 80.0, "label": "良好"},
			{"name": "身心", "score": 85.0, "label": "良好"},
			{"name": "实践", "score": 85.0, "label": "良好"},
			{"name": "创新", "score": 75.0, "label": "良好"},
		},
		"ai_summary":  "你的学业和身心维度表现良好，社交和实践维度有提升空间。建议多参加社团活动和实习项目。",
		"suggestions": []string{"参加下周的企业宣讲会", "加入一个技术社团", "每周运动3次以上"},
		"fallback":    true,
	})
}

// Personality 性格洞察
func (h *StudentHandler) Personality(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if h.personalitySvc != nil {
		result, err := h.personalitySvc.GetPersonality(c.Request.Context(), userCtx.UserID)
		if err == nil && result != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
			return
		}
	}

	// 兜底 mock
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"type":               "待评估",
			"label":              "数据不足",
			"description":        "暂无足够行为数据推断性格画像，请多使用系统积累数据后再查看。",
			"strengths":          []string{"持续使用中"},
			"weaknesses":         []string{"数据样本不足"},
			"career_suggestions": []string{"继续探索中"},
			"learning_style":     "需要更多学习记录来判断",
			"data_source":        "fallback",
		},
	})
}

// Achievements 积分成就（真实积分计算）
func (h *StudentHandler) Achievements(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if h.phase2Svc != nil {
		if data, err := h.phase2Svc.GetAchievements(userCtx.UserID); err == nil && data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}

	// 兜底：服务未注入时返回空成就（不展示假数据）
	c.JSON(http.StatusOK, gin.H{
		"total_points": 0, "level": 1, "level_name": "青铜", "next_level_points": 300,
		"weekly_rank": 0, "badges": []gin.H{}, "data_source": "fallback",
	})
}

// CourseMap 课程地图
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

// QAPlaza 问答广场（真实帖子 + 知识库 FAQ 合并）
func (h *StudentHandler) QAPlaza(c *gin.Context) {
	realPosts := []map[string]interface{}{}
	if h.phase2Svc != nil {
		if posts, err := h.phase2Svc.ListRealPosts(8); err == nil {
			for _, p := range posts {
				realPosts = append(realPosts, map[string]interface{}{
					"id":        p.ID,
					"title":     p.Title,
					"author":    p.Author,
					"answers":   p.Answers,
					"views":     p.Views,
					"ai_answer": "",
					"category":  p.Category,
					"real":      true,
				})
			}
		}
	}

	// 知识库 FAQ 作为补充
	var faqPosts []map[string]interface{}
	if h.svc != nil {
		plaza := h.svc.GenerateQAPlaza(c.Request.Context())
		if plaza != nil {
			faqPosts = plaza.HotQuestions
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"hot_questions": append(realPosts, faqPosts...),
		"categories":    []string{"学业", "生活", "政策", "心理", "就业", "竞赛", "综合"},
		"my_posts":      len(realPosts),
		"data_source":   "real",
	})
}

// CreateQAPost 发布问题
func (h *StudentHandler) CreateQAPost(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	var req struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "问题标题不能为空"})
		return
	}
	if h.phase2Svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "问答服务未就绪"})
		return
	}
	id, err := h.phase2Svc.CreateQAPost(userCtx.UserID, req.Title, req.Content, req.Category)
	if err != nil {
		log.Printf("创建问答帖失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "id": id, "message": "发布成功"})
}

// ListQAPosts 真实帖子列表
func (h *StudentHandler) ListQAPosts(c *gin.Context) {
	if h.phase2Svc == nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	posts, err := h.phase2Svc.ListRealPosts(50)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	list := make([]gin.H, 0, len(posts))
	for _, p := range posts {
		list = append(list, gin.H{
			"id": p.ID, "title": p.Title, "content": p.Content, "author": p.Author,
			"category": p.Category, "answers": p.Answers, "views": p.Views, "created_at": p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, list)
}

// GetQAPostDetail 帖子详情（含回答）
func (h *StudentHandler) GetQAPostDetail(c *gin.Context) {
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	if h.phase2Svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "问答服务未就绪"})
		return
	}
	post, err := h.phase2Svc.GetPost(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "问题不存在"})
		return
	}
	answers, _ := h.phase2Svc.ListAnswers(id)
	c.JSON(http.StatusOK, gin.H{
		"id": post.ID, "title": post.Title, "content": post.Content, "author": post.Author,
		"category": post.Category, "views": post.Views, "created_at": post.CreatedAt,
		"answers": answers,
	})
}

// AnswerQAPost 回答问题
func (h *StudentHandler) AnswerQAPost(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "回答内容不能为空"})
		return
	}
	if h.phase2Svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "问答服务未就绪"})
		return
	}
	aid, err := h.phase2Svc.AnswerPost(id, userCtx.UserID, req.Content)
	if err != nil {
		log.Printf("回答问答帖失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "回答失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "answer_id": aid, "message": "回答成功"})
}

// HotTopics 热点关注
func (h *StudentHandler) HotTopics(c *gin.Context) {
	if h.svc != nil {
		topics := h.svc.GenerateHotTopics(c.Request.Context())
		if topics != nil {
			c.JSON(http.StatusOK, topics)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"topics": []gin.H{
			{"id": "1", "title": "期中考试安排", "heat": 95, "trend": "rising", "posts": 23, "summary": "本学期期中考试集中在第10-11周，数据结构和高数为重点关注科目"},
			{"id": "2", "title": "暑期实习招聘", "heat": 82, "trend": "rising", "posts": 15, "summary": "多家互联网公司开放暑期实习岗位，建议提前准备简历和算法"},
			{"id": "3", "title": "校园网升级", "heat": 68, "trend": "stable", "posts": 12, "summary": "校园网将于下周升级至千兆，届时可能短暂断网"},
			{"id": "4", "title": "社团招新", "heat": 55, "trend": "falling", "posts": 8, "summary": "本学期第二轮社团招新已结束，共12个社团完成纳新"},
		},
		"updated_at":  time.Now().Format("2006-01-02 15:04"),
		"data_source": "fallback",
	})
}

// QALeaderboard 问答排行榜
func (h *StudentHandler) QALeaderboard(c *gin.Context) {
	if h.svc != nil {
		lb := h.svc.GenerateQALeaderboard(c.Request.Context())
		if lb != nil {
			c.JSON(http.StatusOK, lb)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"hot_questions": []gin.H{
			{"rank": 1, "title": "ACM竞赛如何入门？", "views": 256, "answers": 8, "score": 92.5},
			{"rank": 2, "title": "转专业需要什么条件？", "views": 128, "answers": 5, "score": 85.0},
			{"rank": 3, "title": "考研还是就业？", "views": 198, "answers": 12, "score": 80.3},
		},
		"top_answerers": []gin.H{
			{"rank": 1, "name": "知识达人", "answers": 23, "adopted": 15, "score": 95.0},
			{"rank": 2, "name": "热心学长", "answers": 18, "adopted": 10, "score": 82.5},
			{"rank": 3, "name": "编程高手", "answers": 12, "adopted": 8, "score": 78.0},
		},
		"contributors": []gin.H{
			{"rank": 1, "name": "知识达人", "contributions": 15, "quality_score": 4.8},
			{"rank": 2, "name": "热心学长", "contributions": 10, "quality_score": 4.5},
			{"rank": 3, "name": "学霸笔记", "contributions": 8, "quality_score": 4.3},
		},
		"period":      "本周",
		"data_source": "fallback",
	})
}

// PrivateChat 站内私聊
func (h *StudentHandler) PrivateChat(c *gin.Context) {
	if h.svc != nil {
		chat := h.svc.GeneratePrivateChat(c.Request.Context())
		if chat != nil {
			c.JSON(http.StatusOK, chat)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"conversations": []gin.H{
			{"id": "1", "name": "李辅导员", "role": "counselor", "last_message": "明天下午来办公室聊聊", "time": "10:30", "unread": 1},
			{"id": "2", "name": "张学长", "role": "student", "last_message": "ACM训练资料已发你邮箱", "time": "昨天", "unread": 0},
			{"id": "3", "name": "AI学友-王同学", "role": "student", "last_message": "明天一起去图书馆复习吧", "time": "昨天", "unread": 0},
		},
		"recommended_contacts": []gin.H{
			{"name": "赵学姐", "reason": "同专业大三，擅长算法", "match_score": 88},
			{"name": "刘同学", "reason": "学习风格互补，可组队复习", "match_score": 82},
		},
		"data_source": "fallback",
	})
}

// ProcessEnhanced AI 办事流程增强 — 按 type 参数从 KB + process_steps 拼装真实数据
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

// ======================== P2 深度分析功能 ========================

// MockInterview AI 模拟面试
func (h *StudentHandler) MockInterview(c *gin.Context) {
	position := c.Query("position")
	if h.svc != nil {
		result := h.svc.GenerateMockInterview(c.Request.Context(), position)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"position":    position,
		"questions":   []gin.H{{"q": "请做一下自我介绍", "tip": "突出技术栈和项目经验"}, {"q": "你如何看待失败？", "tip": "展示成长心态"}},
		"score":       78,
		"suggestions": []string{"技术表达可以更结构化", "建议多准备行为面试问题"},
		"data_source": "fallback",
	})
}

// Resume AI 智能简历生成
func (h *StudentHandler) Resume(c *gin.Context) {
	position := c.Query("position")
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateResume(c.Request.Context(), userCtx.UserID, position)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"template":    "modern",
		"sections":    []string{"个人信息", "教育背景", "项目经历", "技能特长", "荣誉奖项"},
		"data_source": "fallback",
	})
}

// CareerSimulation 职业模拟器
func (h *StudentHandler) CareerSimulation(c *gin.Context) {
	careerPath := c.Query("career")
	if h.svc != nil {
		result := h.svc.GenerateCareerSimulation(c.Request.Context(), careerPath)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"career":      careerPath,
		"stages":      []string{"应届生", "3年经验", "5年+", "资深/管理"},
		"data_source": "fallback",
	})
}

// StudyBuddyMatch AI 学友匹配
func (h *StudentHandler) StudyBuddyMatch(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateStudyBuddyMatches(c.Request.Context(), userCtx.UserID)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"matches":     []gin.H{{"name": "张同学", "reason": "数据结构互补", "match_score": 88}},
		"data_source": "fallback",
	})
}

// MentalHealthReport AI 心理健康评估报告
func (h *StudentHandler) MentalHealthReport(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateMentalHealthReport(c.Request.Context(), userCtx.UserID)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"dimensions":  gin.H{"压力": "中等", "情绪": "良好", "社交": "活跃", "韧性": "较强"},
		"suggestions": []string{"保持规律作息", "适当运动放松"},
		"data_source": "fallback",
	})
}

// NoteAssistant AI 笔记助手
func (h *StudentHandler) NoteAssistant(c *gin.Context) {
	content := c.Query("content")
	if h.svc != nil && content != "" {
		result := h.svc.GenerateNoteAssistant(c.Request.Context(), content)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"key_points":  []string{"知识点1", "知识点2"},
		"mind_map":    "中心主题 → 分支1 → 细节",
		"quiz":        []string{"自测题1", "自测题2"},
		"data_source": "fallback",
	})
}

// AlumniMatch AI 前辈连线
func (h *StudentHandler) AlumniMatch(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateAlumniMatch(c.Request.Context(), userCtx.UserID)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"matches":     []gin.H{{"name": "陈学长", "company": "字节跳动", "match_score": 92}},
		"data_source": "fallback",
	})
}

// ======================== P3 生态扩展 ========================

// DynamicMentor 数字人导师（动态形象版）
func (h *StudentHandler) DynamicMentor(c *gin.Context) {
	style := c.Query("style")
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateDynamicMentor(c.Request.Context(), userCtx.UserID, style)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"name":         "蔚小芯",
		"avatar_style": style,
		"greeting":     fmt.Sprintf("你好！我是你的%s风格AI导师。今天我们一起努力吧！", style),
		"data_source":  "fallback",
	})
}

// EnhancedCareerSim 职业模拟器增强版
func (h *StudentHandler) EnhancedCareerSim(c *gin.Context) {
	careerPath := c.Query("career")
	if h.svc != nil {
		result := h.svc.GenerateEnhancedCareerSimulation(c.Request.Context(), careerPath)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"career_path": careerPath,
		"stages":      []string{"在校期", "应届生", "3年经验", "5年+"},
		"data_source": "fallback",
	})
}

// ═══════════════════════════════════════════════
// 学生首页数据接口
// ═══════════════════════════════════════════════

// HomeStudentCourse 今日课程
type HomeStudentCourse struct {
	CourseName string `json:"course_name"`
	Time       string `json:"time"`
	Location   string `json:"location"`
	Teacher    string `json:"teacher"`
	Color      string `json:"color"`
}

// HomeStudentTask 今日任务
type HomeStudentTask struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	PlanID   int64  `json:"plan_id"`
	Status   string `json:"status"`
	Duration int    `json:"duration"`
}

// HomeStudentEvent 近期事件
type HomeStudentEvent struct {
	ID        int64  `json:"id"`
	EventName string `json:"event_name"`
	EventType string `json:"event_type"`
	StartDate string `json:"start_date"`
	DaysLeft  int    `json:"days_left"`
}

// HomeStudentStats 统计数据
type HomeStudentStats struct {
	UnreadNotifications int `json:"unread_notifications"`
	PendingFeedback     int `json:"pending_feedback"`
	PlansInProgress     int `json:"plans_in_progress"`
}

// HomeStudentQuickEntry 功能入口
type HomeStudentQuickEntry struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Route string `json:"route"`
}

// HomeStudentUserInfo 用户信息
type HomeStudentUserInfo struct {
	Name      string `json:"name"`
	StudentID string `json:"student_id"`
	College   string `json:"college"`
	Major     string `json:"major"`
	Grade     string `json:"grade"`
}

// HomeStudentToday 今日信息
type HomeStudentToday struct {
	Date         string `json:"date"`
	Weekday      string `json:"weekday"`
	WeekNo       int    `json:"week_no"`
	SemesterName string `json:"semester_name"`
}

// Home 学生首页数据
// GET /api/v1/student/home
func (h *StudentHandler) Home(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    503,
			Message: "数据库未初始化",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Code:    401,
			Message: "未获取到用户信息",
			TraceID: middleware.GetTraceID(c),
		})
		return
	}

	today := time.Now()
	todayStr := today.Format("2006-01-02")
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekdayNames := []string{"", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}

	// 1. 获取用户信息
	userInfo := h.getUserInfo(userCtx.UserID)

	// 2. 获取当前学期和教学周
	calendar, weekNo := h.resolveCurrentCalendar()
	semesterName := ""
	if calendar != nil {
		semesterName = calendar.SemesterName
		if calendar.Status == "completed" {
			semesterName += "（已结束）"
		} else if calendar.Status == "upcoming" {
			semesterName += "（未开始）"
		}
	}

	// 3. 获取今日课程
	todayCourses := h.getTodayCourses(userCtx.UserID, weekday, calendar)

	// 4. 获取今日任务
	todayTasks := h.getTodayTasks(userCtx.UserID, todayStr)

	// 5. 获取近期事件（未来7天 + 正在进行中的）
	upcomingEvents := h.getUpcomingEvents(todayStr, calendar)

	// 6. 获取统计数据
	stats := h.getHomeStats(userCtx.UserID)

	// 7. 功能入口（固定配置）
	quickEntries := []HomeStudentQuickEntry{
		{Icon: "chat", Title: "AI问答", Route: "/chat"},
		{Icon: "study_plan", Title: "学习计划", Route: "/student/study-plan"},
		{Icon: "timetable", Title: "我的课表", Route: "/student/timetable"},
		{Icon: "career", Title: "就业服务", Route: "/student/career"},
		{Icon: "study", Title: "学业服务", Route: "/student/study"},
		{Icon: "mental", Title: "心理健康", Route: "/student/mental"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"user_info": userInfo,
			"today": HomeStudentToday{
				Date:         todayStr,
				Weekday:      weekdayNames[weekday],
				WeekNo:       weekNo,
				SemesterName: semesterName,
			},
			"today_courses":   todayCourses,
			"today_tasks":     todayTasks,
			"upcoming_events": upcomingEvents,
			"stats":           stats,
			"quick_entries":   quickEntries,
		},
	})
}

// getUserInfo 获取用户信息
func (h *StudentHandler) getUserInfo(userID int64) HomeStudentUserInfo {
	info := HomeStudentUserInfo{
		Name:      "同学",
		StudentID: "",
		College:   "",
		Major:     "",
		Grade:     "",
	}
	if h.db == nil {
		return info
	}

	var displayName, username, college, major, enrollmentYear sql.NullString
	err := h.db.QueryRow(
		"SELECT display_name, username, college, major, enrollment_year FROM users WHERE id = ?",
		userID,
	).Scan(&displayName, &username, &college, &major, &enrollmentYear)
	if err != nil {
		log.Printf("获取用户信息失败 user_id=%d: %v", userID, err)
		return info
	}
	if displayName.Valid {
		info.Name = displayName.String
	}
	if username.Valid {
		info.StudentID = username.String
	}
	if college.Valid {
		info.College = college.String
	}
	if major.Valid {
		info.Major = major.String
	}
	if enrollmentYear.Valid && enrollmentYear.String != "" {
		info.Grade = enrollmentYear.String + "级"
	}
	return info
}

// resolveCurrentCalendar 获取当前学期校历与教学周
func (h *StudentHandler) resolveCurrentCalendar() (*AcademicCalendar, int) {
	if h.db == nil {
		return nil, 0
	}

	today := time.Now().Format("2006-01-02")

	calendar := &AcademicCalendar{}
	err := h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE start_date <= ? AND end_date >= ? ORDER BY id DESC LIMIT 1",
		today, today,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == nil {
		return calendar, calcHomeCurrentWeek(calendar.StartDate, today)
	}
	if err != sql.ErrNoRows {
		log.Printf("查询当前学期校历失败: %v", err)
		return nil, 0
	}

	calendar = &AcademicCalendar{}
	err = h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE start_date > ? ORDER BY start_date ASC LIMIT 1",
		today,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == nil {
		return calendar, 0
	}
	if err != sql.ErrNoRows {
		log.Printf("查询未来学期校历失败: %v", err)
		return nil, 0
	}

	calendar = &AcademicCalendar{}
	err = h.db.QueryRow(
		"SELECT id, academic_year, semester, semester_code, semester_name, start_date, end_date, "+
			"register_date, total_weeks, week_start_day, status, created_at, updated_at "+
			"FROM academic_calendars WHERE end_date < ? ORDER BY end_date DESC LIMIT 1",
		today,
	).Scan(&calendar.ID, &calendar.AcademicYear, &calendar.Semester, &calendar.SemesterCode,
		&calendar.SemesterName, &calendar.StartDate, &calendar.EndDate,
		&calendar.RegisterDate, &calendar.TotalWeeks, &calendar.WeekStartDay,
		&calendar.Status, &calendar.CreatedAt, &calendar.UpdatedAt)
	if err == sql.ErrNoRows {
		log.Printf("无任何校历记录")
		return nil, 0
	}
	if err != nil {
		log.Printf("查询过往学期校历失败: %v", err)
		return nil, 0
	}
	return calendar, 0
}

// calcHomeCurrentWeek 计算当前教学周
func calcHomeCurrentWeek(startDate, today string) int {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0
	}
	now, err := time.Parse("2006-01-02", today)
	if err != nil {
		return 0
	}
	if now.Before(start) {
		return 0
	}
	days := int(now.Sub(start).Hours() / 24)
	return days/7 + 1
}

// getTodayCourses 获取今日课程
func (h *StudentHandler) getTodayCourses(userID int64, weekday int, calendar *AcademicCalendar) []HomeStudentCourse {
	courses := make([]HomeStudentCourse, 0)
	if h.db == nil || calendar == nil {
		return courses
	}

	rows, err := h.db.Query(
		"SELECT course_name, start_period, end_period, location, teacher, color "+
			"FROM course_schedules WHERE user_id = ? AND semester_code = ? AND weekday = ? "+
			"ORDER BY start_period ASC, id ASC",
		userID, calendar.SemesterCode, weekday,
	)
	if err != nil {
		return courses
	}
	defer rows.Close()

	periodTimes := []string{
		"", "08:00-08:45", "08:55-09:40",
		"10:00-10:45", "10:55-11:40",
		"14:00-14:45", "14:55-15:40",
		"16:00-16:45", "16:55-17:40",
		"19:00-19:45", "19:55-20:40",
	}

	for rows.Next() {
		var courseName, location, teacher, color sql.NullString
		var startPeriod, endPeriod int
		if err := rows.Scan(&courseName, &startPeriod, &endPeriod, &location, &teacher, &color); err != nil {
			continue
		}
		timeStr := ""
		if startPeriod >= 1 && startPeriod <= 10 && endPeriod >= startPeriod && endPeriod <= 10 {
			startTime := periodTimes[startPeriod][:5]
			endTime := periodTimes[endPeriod][6:]
			timeStr = startTime + "-" + endTime
		}
		courses = append(courses, HomeStudentCourse{
			CourseName: courseName.String,
			Time:       timeStr,
			Location:   location.String,
			Teacher:    teacher.String,
			Color:      color.String,
		})
	}
	return courses
}

// getTodayTasks 获取今日任务
func (h *StudentHandler) getTodayTasks(userID int64, todayStr string) []HomeStudentTask {
	tasks := make([]HomeStudentTask, 0)
	if h.db == nil {
		return tasks
	}

	rows, err := h.db.Query(
		"SELECT t.id, t.title, t.plan_id, t.status, t.scheduled_duration "+
			"FROM study_plan_tasks t JOIN study_plans p ON t.plan_id = p.id "+
			"WHERE p.user_id = ? AND t.scheduled_date = ? "+
			"ORDER BY t.sort_order ASC, t.id ASC",
		userID, todayStr,
	)
	if err != nil {
		log.Printf("查询今日任务失败 user_id=%d: %v", userID, err)
		return tasks
	}
	defer rows.Close()

	for rows.Next() {
		var title sql.NullString
		var duration sql.NullInt64
		task := HomeStudentTask{}
		if err := rows.Scan(&task.ID, &title, &task.PlanID, &task.Status, &duration); err != nil {
			continue
		}
		task.Title = title.String
		task.Duration = int(duration.Int64)
		tasks = append(tasks, task)
	}
	return tasks
}

// getUpcomingEvents 获取近期事件（未来7天 + 正在进行中的）
func (h *StudentHandler) getUpcomingEvents(todayStr string, calendar *AcademicCalendar) []HomeStudentEvent {
	events := make([]HomeStudentEvent, 0)
	if h.db == nil || calendar == nil {
		return events
	}

	fromDate := todayStr
	toDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	rows, err := h.db.Query(
		"SELECT id, event_name, event_type, start_date, end_date "+
			"FROM academic_calendar_events WHERE semester_code = ? "+
			"AND (start_date <= ? AND (end_date >= ? OR end_date IS NULL)) "+
			"ORDER BY start_date ASC, id ASC LIMIT 10",
		calendar.SemesterCode, toDate, fromDate,
	)
	if err != nil {
		return events
	}
	defer rows.Close()

	today, _ := time.Parse("2006-01-02", todayStr)

	for rows.Next() {
		var eventName, eventType, startDate sql.NullString
		var endDate sql.NullString
		var id int64
		if err := rows.Scan(&id, &eventName, &eventType, &startDate, &endDate); err != nil {
			continue
		}
		start, err := time.Parse("2006-01-02", startDate.String)
		if err != nil {
			continue
		}
		daysLeft := int(start.Sub(today).Hours() / 24)
		events = append(events, HomeStudentEvent{
			ID:        id,
			EventName: eventName.String,
			EventType: eventType.String,
			StartDate: startDate.String,
			DaysLeft:  daysLeft,
		})
	}
	return events
}

// getHomeStats 获取首页统计数据
func (h *StudentHandler) getHomeStats(userID int64) HomeStudentStats {
	stats := HomeStudentStats{}
	if h.db == nil {
		return stats
	}

	// 进行中的学习计划数
	countErr := h.db.QueryRow(
		"SELECT COUNT(*) FROM study_plans WHERE user_id = ? AND status = 'active'",
		userID,
	).Scan(&stats.PlansInProgress)
	if countErr != nil {
		log.Printf("获取首页统计失败 user_id=%d: %v", userID, countErr)
	}

	return stats
}

// PersonalProfile 学生个人信息档案聚合
// GET /api/v1/student/profile
// 并发聚合：基础信息 + 五维孪生 + 性格 + 学业 + 竞赛 + 入党 + 社团 + 打卡 + 积分
// 错误容忍：单个子查询失败不影响整体，返回空数组/零值
func (h *StudentHandler) PersonalProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	userID := userCtx.UserID
	result := gin.H{"user_id": userID, "username": userCtx.Username, "display_name": userCtx.DisplayName}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 用匿名函数并发查询，写回 result
	query := func(key string, fn func() (interface{}, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := fn()
			if err != nil {
				log.Printf("聚合[%s]失败 user_id=%d: %v", key, userID, err)
				return
			}
			mu.Lock()
			result[key] = val
			mu.Unlock()
		}()
	}

	// 1. 基础信息（users 表）
	query("basic_info", func() (interface{}, error) {
		var college, major, className, enrollmentDate, enrollmentYear, status string
		err := h.db.QueryRow(
			"SELECT college, major, class_name, enrollment_date, enrollment_year, status FROM users WHERE id = ?",
			userID,
		).Scan(&college, &major, &className, &enrollmentDate, &enrollmentYear, &status)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"college": college, "major": major, "class_name": className,
			"enrollment_date": enrollmentDate, "enrollment_year": enrollmentYear, "status": status,
		}, nil
	})

	// 2. 学业成绩汇总（student_grades）
	query("academic", func() (interface{}, error) {
		var courseCount int
		var credits, totalScore, gpa float64
		var passedCount, totalCount int
		err := h.db.QueryRow(
			"SELECT COUNT(*), COALESCE(SUM(credits_earned),0), COALESCE(AVG(score),0), "+
				"COALESCE(AVG(gpa),0), COALESCE(SUM(CASE WHEN passed=1 THEN 1 ELSE 0 END),0), COUNT(*) "+
				"FROM student_grades WHERE user_id = ?",
			fmt.Sprintf("%d", userID),
		).Scan(&courseCount, &credits, &totalScore, &gpa, &passedCount, &totalCount)
		if err != nil {
			return nil, err
		}
		passRate := 0.0
		if totalCount > 0 {
			passRate = float64(passedCount) / float64(totalCount) * 100
		}
		return gin.H{
			"course_count": courseCount, "total_credits": credits,
			"avg_score": totalScore, "avg_gpa": gpa, "pass_rate": passRate,
		}, nil
	})

	// 3. 竞赛报名（competition_registrations）
	query("competitions", func() (interface{}, error) {
		rows, err := h.db.Query(
			"SELECT competition_id, student_name, team_name, advisor_name, status, award_level FROM competition_registrations WHERE user_id = ? ORDER BY id DESC LIMIT 10",
			userID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []gin.H
		for rows.Next() {
			var cid int64
			var studentName, teamName, advisor, status, award string
			if err := rows.Scan(&cid, &studentName, &teamName, &advisor, &status, &award); err != nil {
				continue
			}
			list = append(list, gin.H{
				"competition_id": cid, "student_name": studentName, "team_name": teamName,
				"advisor_name": advisor, "status": status, "award_level": award,
			})
		}
		return list, nil
	})

	// 4. 入党进度（party_progress）
	query("party", func() (interface{}, error) {
		var currentStage, status string
		var applyDate string
		err := h.db.QueryRow(
			"SELECT current_stage, status, COALESCE(apply_date,'') FROM party_progress WHERE user_id = ? ORDER BY id DESC LIMIT 1",
			userID,
		).Scan(&currentStage, &status, &applyDate)
		if err != nil {
			if err == sql.ErrNoRows {
				return gin.H{"current_stage": "", "status": "", "apply_date": ""}, nil
			}
			return nil, err
		}
		return gin.H{"current_stage": currentStage, "status": status, "apply_date": applyDate}, nil
	})

	// 5. 社团参与（club_members）
	query("clubs", func() (interface{}, error) {
		rows, err := h.db.Query(
			"SELECT club_id, role, join_date FROM club_members WHERE user_id = ? AND (leave_date IS NULL OR leave_date = '') ORDER BY join_date DESC LIMIT 5",
			userID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []gin.H
		for rows.Next() {
			var cid int64
			var role, joinDate string
			if err := rows.Scan(&cid, &role, &joinDate); err != nil {
				continue
			}
			list = append(list, gin.H{"club_id": cid, "role": role, "join_date": joinDate})
		}
		return list, nil
	})

	// 6. 打卡记录（student_checkins）
	query("checkin", func() (interface{}, error) {
		var total int
		var lastDate string
		err := h.db.QueryRow(
			"SELECT COUNT(*), COALESCE(MAX(check_date),'') FROM student_checkins WHERE user_id = ?",
			userID,
		).Scan(&total, &lastDate)
		if err != nil {
			return nil, err
		}
		return gin.H{"total_days": total, "last_date": lastDate}, nil
	})

	// 7. 积分（student_points）
	query("points", func() (interface{}, error) {
		var total int
		err := h.db.QueryRow(
			"SELECT COALESCE(SUM(points),0) FROM student_points WHERE user_id = ?",
			userID,
		).Scan(&total)
		if err != nil {
			return nil, err
		}
		return gin.H{"total_points": total}, nil
	})

	wg.Wait()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// ---- 数字孪生/性格（复用现有 service）----

// TwinProfile 个人数字孪生五维画像
func (h *StudentHandler) TwinProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	if h.twinSvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	result, err := h.twinSvc.GetDigitalTwin(c.Request.Context(), userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// PersonalityProfile 个人性格洞察
func (h *StudentHandler) PersonalityProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	if h.personalitySvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	result, err := h.personalitySvc.GetPersonality(c.Request.Context(), userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// decodeJSON 解析 JSON 字符串字段（辅助）
func decodeJSON(raw string) []interface{} {
	var arr []interface{}
	if raw == "" {
		return arr
	}
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return arr
	}
	return arr
}

// UploadAvatar 上传/更新当前用户头像
// POST /api/v1/user/avatar  （multipart/form-data: file）
// 头像 base64 存 users.avatar_base64（SQLite 跨实例持久）
func (h *StudentHandler) UploadAvatar(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未获取到上传文件"})
		return
	}
	defer file.Close()

	// 限制 3MB
	if header.Size > 3*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "头像文件超过 3MB 限制"})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持 png/jpg/webp/gif 图片"})
		return
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文件失败"})
		return
	}

	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}

	encoded := base64.StdEncoding.EncodeToString(bytes)
	if _, err := h.db.Exec(
		"UPDATE users SET avatar_base64 = ?, avatar_mime = ?, updated_at = datetime('now') WHERE id = ?",
		encoded, mime, userCtx.UserID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "头像已更新",
		"data": gin.H{"user_id": userCtx.UserID, "size": header.Size, "mime": mime},
	})
}

// ServeAvatar GET /api/v1/user/avatar/:user_id — 返回用户头像图片字节（base64 解码）
func (h *StudentHandler) ServeAvatar(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 user_id"})
		return
	}

	var b64, mime string
	err := h.db.QueryRow(
		"SELECT COALESCE(avatar_base64,''), COALESCE(avatar_mime,'image/png') FROM users WHERE id = ?",
		userID,
	).Scan(&b64, &mime)
	if err != nil || b64 == "" {
		c.Status(http.StatusNotFound)
		return
	}

	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", mime)
	c.Data(http.StatusOK, mime, raw)
}
