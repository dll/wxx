package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CounselorHandler 辅导员角色 AI 功能接口
type CounselorHandler struct{}

func NewCounselorHandler() *CounselorHandler {
	return &CounselorHandler{}
}

// DailyFocus AI 今日关注
func (h *CounselorHandler) DailyFocus(c *gin.Context) {
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
	})
}

// ClassReport 班级学情日报
func (h *CounselorHandler) ClassReport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"date":                 time.Now().Format("2006-01-02"),
		"class_name":           "计科2301班",
		"active_rate":          0.87,
		"absent_count":         3,
		"homework_rate":        0.92,
		"emotion_alert_count":  2,
		"checkin_rate":         0.93,
		"anomalies":           []string{"张明连续缺勤", "李华作业未交"},
		"ai_narrative":        "班级整体状态良好，出勤率93%，作业提交率92%。需关注张明同学连续缺勤情况，建议及时沟通了解原因。",
	})
}

// TwinBoard 学生数字孪生看板
func (h *CounselorHandler) TwinBoard(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{
		{"student_id": "s001", "name": "张明", "academic": 65.0, "social": 45.0, "mental": 55.0, "practice": 70.0, "risk": "high"},
		{"student_id": "s002", "name": "李华", "academic": 72.0, "social": 80.0, "mental": 78.0, "practice": 60.0, "risk": "medium"},
		{"student_id": "s003", "name": "王芳", "academic": 88.0, "social": 55.0, "mental": 70.0, "practice": 75.0, "risk": "low"},
		{"student_id": "s004", "name": "赵强", "academic": 90.0, "social": 85.0, "mental": 88.0, "practice": 82.0, "risk": "none"},
		{"student_id": "s005", "name": "刘洋", "academic": 78.0, "social": 72.0, "mental": 80.0, "practice": 68.0, "risk": "none"},
	})
}

// Prediction 预测性预警
func (h *CounselorHandler) Prediction(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{
		{"student_id": "s001", "name": "张明", "risk_type": "dropout", "probability": 0.35, "factors": []string{"出勤率低", "成绩下滑", "社交减少"}, "suggestion": "建议尽快约谈"},
		{"student_id": "s006", "name": "陈静", "risk_type": "academic", "probability": 0.28, "factors": []string{"数学成绩持续下降", "作业质量下滑"}, "suggestion": "安排学业辅导"},
	})
}

// Intervention 干预方案
func (h *CounselorHandler) Intervention(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"student_id":   c.PostForm("student_id"),
		"plan":         "分阶段干预方案",
		"steps": []gin.H{
			{"phase": "第一阶段", "action": "一对一谈心，了解具体困难", "timeline": "本周内"},
			{"phase": "第二阶段", "action": "协调学业帮扶，安排学伴", "timeline": "1-2周"},
			{"phase": "第三阶段", "action": "持续跟踪，每周反馈", "timeline": "持续1个月"},
		},
		"resources":    []string{"心理咨询中心", "学业辅导站", "班级互助小组"},
		"expected_outcome": "预计2-4周内出勤率恢复正常",
	})
}

// TalkRecord 谈心谈话记录
func (h *CounselorHandler) TalkRecord(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.JSON(http.StatusOK, []gin.H{
			{"id": "t001", "student_name": "张明", "date": "2026-05-13", "topic": "学业困难", "emotion": "焦虑", "summary": "反映课程压力大，数学跟不上进度", "follow_ups": []string{"安排数学辅导", "下周复查"}, "status": "following"},
			{"id": "t002", "student_name": "李华", "date": "2026-05-12", "topic": "人际关系", "emotion": "低落", "summary": "与室友产生矛盾，影响休息", "follow_ups": []string{"协调宿舍关系", "建议心理咨询"}, "status": "resolved"},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录保存成功"})
	}
}

// TalkTips 谈话话术推荐
func (h *CounselorHandler) TalkTips(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{
		"summary":    "班级整体思想状态积极向上，政治学习参与率95%",
		"highlights": []string{"3名同学递交入党申请书", "班级志愿服务时长达标"},
		"concerns":   []string{"个别同学对时事关注度不够"},
		"students": []gin.H{
			{"name": "赵强", "status": "预备党员", "evaluation": "思想觉悟高，积极参与组织活动"},
			{"name": "刘洋", "status": "入党积极分子", "evaluation": "表现良好，建议加强理论学习"},
		},
	})
}

// ClassProfile 班级性格画像
func (h *CounselorHandler) ClassProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"class_name": "计科2301班",
		"total":      45,
		"distribution": gin.H{
			"外向型": 18,
			"内向型": 12,
			"分析型": 8,
			"感性型": 7,
		},
		"characteristics": []string{"整体偏理性思维", "团队协作意愿强", "创新意识较好"},
		"suggestions":     []string{"多组织团队活动促进内向同学融入", "利用分析型同学带动学术氛围"},
	})
}
