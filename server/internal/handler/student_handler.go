package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// StudentHandler 学生角色 AI 功能接口
type StudentHandler struct {
	svc *service.StudentService
}

// NewStudentHandler 创建学生 handler。svc 可为 nil（兼容旧调用），此时所有 AI 功能走兜底
func NewStudentHandler(svc *service.StudentService) *StudentHandler {
	return &StudentHandler{svc: svc}
}

// DailyBriefing 今日速览 — 真实数据 + LLM 个性化生成
func (h *StudentHandler) DailyBriefing(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			briefing, err := h.svc.GenerateDailyBriefing(c.Request.Context(), userCtx.UserID)
			if err == nil && briefing != nil {
				c.JSON(http.StatusOK, briefing)
				return
			}
		}
	}
	// 兜底：未注入 svc 或异常时使用旧 mock
	h.mockDailyBriefing(c)
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
	today := time.Now().Format("2006-01-02")
	c.JSON(http.StatusOK, gin.H{
		"date":            today,
		"courses_studied": []string{"数据结构", "操作系统"},
		"key_points":      []string{"二叉树前序/中序/后序遍历", "进程状态转换图", "死锁四个必要条件"},
		"study_minutes":   185,
		"quiz": []gin.H{
			{"question": "二叉树前序遍历的顺序是？", "options": []string{"根左右", "左根右", "左右根", "右根左"}, "correct_index": 0, "explanation": "前序遍历先访问根节点，再递归左子树，最后右子树"},
		},
		"tomorrow_plan":  "复习操作系统第5章，完成数据结构实验",
		"encouragement":  "今天学习了3小时05分钟，比昨天多了20分钟，继续保持！",
	})
}

// Checkin 打卡
func (h *StudentHandler) Checkin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "打卡成功"})
}

// CheckinHistory 打卡历史
func (h *StudentHandler) CheckinHistory(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	c.JSON(http.StatusOK, gin.H{
		"date":           today,
		"streak":         7,
		"total_days":     42,
		"longest_streak": 15,
		"today_checked":  true,
		"recent_dates": []string{
			today,
			time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
		},
	})
}

// DigitalTwin 数字孪生
func (h *StudentHandler) DigitalTwin(c *gin.Context) {
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
	})
}

// Personality 性格洞察
func (h *StudentHandler) Personality(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"type":               "INTJ",
		"label":              "建筑师型",
		"description":        "富有想象力和战略性的思考者，一切皆在计划之中。",
		"strengths":          []string{"逻辑思维强", "独立自主", "目标明确", "善于规划"},
		"weaknesses":         []string{"有时过于理性", "社交场合可能显得冷淡"},
		"career_suggestions": []string{"软件工程师", "数据分析师", "系统架构师", "研究员"},
		"learning_style":     "偏好系统化学习，喜欢先建立整体框架再深入细节",
	})
}

// Achievements 积分成就
func (h *StudentHandler) Achievements(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total_points":      1250,
		"level":             3,
		"level_name":        "白银",
		"next_level_points": 2000,
		"weekly_rank":       15,
		"badges": []gin.H{
			{"id": "b1", "name": "连续签到7天", "icon": "local_fire_department", "description": "连续打卡7天", "unlocked": true, "unlocked_at": "2026-05-10"},
			{"id": "b2", "name": "学霸之路", "icon": "school", "description": "累计学习100小时", "unlocked": true, "unlocked_at": "2026-05-08"},
			{"id": "b3", "name": "社交达人", "icon": "groups", "description": "参加10次活动", "unlocked": false, "unlocked_at": ""},
		},
	})
}

// CourseMap 课程地图
func (h *StudentHandler) CourseMap(c *gin.Context) {
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
	c.JSON(http.StatusOK, []gin.H{
		{"course_name": "数据结构", "progress": 0.65, "rank_percentile": 25, "knowledge_points": []gin.H{{"name": "链表", "mastery": 0.9}, {"name": "二叉树", "mastery": 0.6}, {"name": "图", "mastery": 0.3}}, "weak_points": []string{"图的遍历", "最短路径算法"}},
		{"course_name": "操作系统", "progress": 0.55, "rank_percentile": 40, "knowledge_points": []gin.H{{"name": "进程管理", "mastery": 0.8}, {"name": "内存管理", "mastery": 0.5}}, "weak_points": []string{"页面置换算法"}},
	})
}

// WeeklyReport 学习周报
func (h *StudentHandler) WeeklyReport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"week":            fmt.Sprintf("第%d周", int(time.Now().YearDay()/7)+1),
		"total_hours":     22.5,
		"courses_count":   5,
		"assignments":     3,
		"rank_change":     2,
		"highlights":      []string{"数据结构实验满分", "英语演讲获得A"},
		"improvements":    []string{"操作系统作业需加强", "体育锻炼不足"},
		"next_week_goals": []string{"完成算法作业", "准备期中考试"},
	})
}

// QAPlaza 问答广场
func (h *StudentHandler) QAPlaza(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"hot_questions": []gin.H{
			{"id": "1", "title": "转专业需要什么条件？", "author": "匿名同学", "answers": 5, "views": 128, "ai_answer": "转专业一般需要：1.大一第一学期结束后申请 2.绩点达到3.0以上 3.通过目标专业考核", "tags": []string{"政策", "学业"}},
			{"id": "2", "title": "图书馆自习室怎么预约？", "author": "学习达人", "answers": 3, "views": 89, "ai_answer": "通过校园APP→图书馆→座位预约，每天22:00开放次日预约", "tags": []string{"生活", "图书馆"}},
			{"id": "3", "title": "ACM竞赛如何入门？", "author": "编程新手", "answers": 8, "views": 256, "ai_answer": "建议从C++基础开始，刷LeetCode简单题，参加校内训练赛", "tags": []string{"竞赛", "学业"}},
		},
		"categories": []string{"学业", "生活", "政策", "心理", "就业", "竞赛"},
		"my_posts":   2,
		"my_answers": 5,
	})
}

// HotTopics 热点关注
func (h *StudentHandler) HotTopics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"topics": []gin.H{
			{"id": "1", "title": "期中考试安排", "heat": 95, "trend": "rising", "posts": 23, "summary": "本学期期中考试集中在第10-11周，数据结构和高数为重点关注科目"},
			{"id": "2", "title": "暑期实习招聘", "heat": 82, "trend": "rising", "posts": 15, "summary": "多家互联网公司开放暑期实习岗位，建议提前准备简历和算法"},
			{"id": "3", "title": "校园网升级", "heat": 68, "trend": "stable", "posts": 12, "summary": "校园网将于下周升级至千兆，届时可能短暂断网"},
			{"id": "4", "title": "社团招新", "heat": 55, "trend": "falling", "posts": 8, "summary": "本学期第二轮社团招新已结束，共12个社团完成纳新"},
		},
		"updated_at": time.Now().Format("2006-01-02 15:04"),
	})
}

// QALeaderboard 问答排行榜
func (h *StudentHandler) QALeaderboard(c *gin.Context) {
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
		"period": "本周",
	})
}

// PrivateChat 站内私聊
func (h *StudentHandler) PrivateChat(c *gin.Context) {
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
	})
}

// ProcessEnhanced AI 办事流程增强
func (h *StudentHandler) ProcessEnhanced(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"processes": []gin.H{
			{
				"id": "1", "title": "缓考申请流程", "status": "in_progress", "current_step": 2,
				"steps": []gin.H{
					{"step": 1, "title": "填写缓考申请表", "status": "completed", "contact": "教务处张老师", "phone": "0550-3510XXX", "location": "行政楼B102", "office_hours": "周一至五 8:30-11:30, 14:00-17:00", "faq": []gin.H{{"q": "申请表在哪下载？", "a": "教务系统→表格下载→缓考申请表"}}},
					{"step": 2, "title": "辅导员签字", "status": "in_progress", "contact": "李辅导员", "phone": "0550-3510XXX", "location": "信息楼205", "office_hours": "周一至五 9:00-17:00", "faq": []gin.H{{"q": "辅导员不在怎么办？", "a": "可先在系统提交电子版，辅导员线上审批"}}},
					{"step": 3, "title": "教务处审批", "status": "pending", "contact": "教务处", "phone": "0550-3510XXX", "location": "行政楼A201", "office_hours": "周一至五 8:30-17:00", "faq": []gin.H{{"q": "审批需要多久？", "a": "一般3个工作日内完成"}}},
				},
			},
		},
		"reminders": []gin.H{
			{"title": "缓考申请截止", "deadline": "2026-05-20", "days_left": 5},
		},
	})
}

// GenericAI 通用 AI 响应（用于多个简单功能）
func (h *StudentHandler) GenericAI(feature string) gin.HandlerFunc {
	responses := map[string]gin.H{
		"freshman-plan":      {"content": "大一规划建议", "response": "建议重点关注数学和编程基础课程，积极参加社团活动拓展视野。"},
		"growth-path":        {"content": "成长路径分析", "response": "你目前处于大二下学期，建议本学期提升算法能力，暑假寻找实习机会。"},
		"political-study":    {"content": "政治学习", "response": "本周学习主题：习近平新时代中国特色社会主义思想。已整理学习要点。"},
		"ideological-record": {"content": "思想档案", "response": "思想政治表现良好，建议继续保持对时事的关注，多参与志愿服务。"},
		"party-progress":     {"content": "入党进度", "response": "当前阶段：入党积极分子。下一步参加组织考察，建议积极参与志愿服务。"},
		"campus-life":        {"content": "校园生活", "response": "本周推荐：周三技术沙龙、周五篮球赛、周末志愿者活动。"},
		"schedule":           {"content": "日程管理", "response": "今日：上午2节课，下午1节课，晚上建议复习数据结构。"},
		"competition-match":  {"content": "竞赛推荐", "response": "推荐：ACM程序设计竞赛(95%)、数学建模(80%)、创新创业大赛(70%)。"},
		"study-buddy":        {"content": "学伴匹配", "response": "推荐3位学伴：张三(数据结构)、李四(算法练习)、王五(英语口语)。"},
		"mental-health":      {"content": "心理健康", "response": "整体心理状态良好。建议保持规律作息，适当运动放松。"},
		"digital-mentor":     {"content": "AI导师", "response": "本周建议重点关注数据结构中图的相关算法，这是目前的薄弱环节。"},
	}
	return func(c *gin.Context) {
		if resp, ok := responses[feature]; ok {
			c.JSON(http.StatusOK, resp)
		} else {
			c.JSON(http.StatusOK, gin.H{"content": feature, "response": "功能开发中"})
		}
	}
}
