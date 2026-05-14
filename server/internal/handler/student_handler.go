package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StudentHandler 学生角色 AI 功能接口
type StudentHandler struct{}

func NewStudentHandler() *StudentHandler {
	return &StudentHandler{}
}

// DailyBriefing 今日速览
func (h *StudentHandler) DailyBriefing(c *gin.Context) {
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
		"weather": "晴 26°C",
		"motto":   "学如逆水行舟，不进则退。",
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
