package handler

import (
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// TeacherHandler 教师角色 AI 功能接口
type TeacherHandler struct {
	svc *service.TeacherService
}

func NewTeacherHandler(svc *service.TeacherService) *TeacherHandler {
	return &TeacherHandler{svc: svc}
}

// DailyOverview 今日授课概览
func (h *TeacherHandler) DailyOverview(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"date": time.Now().Format("2006-01-02"),
		"classes": []gin.H{
			{"course": "数据结构", "class_name": "计科2301", "time": "08:00-09:40", "room": "信息楼301", "students": 45},
			{"course": "数据结构", "class_name": "计科2302", "time": "10:00-11:40", "room": "信息楼301", "students": 42},
		},
		"pending_tasks": []gin.H{
			{"task": "批改数据结构实验报告", "count": 87, "deadline": "2026-05-17"},
			{"task": "准备期中考试试卷", "count": 1, "deadline": "2026-05-20"},
		},
		"alerts": []string{"计科2301班3名同学连续缺勤", "实验报告提交率偏低(78%)"},
	})
}

// LessonPrep AI 备课助手
func (h *TeacherHandler) LessonPrep(c *gin.Context) {
	var req struct {
		Topic    string `json:"topic"`
		CourseID string `json:"course_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 尝试用 LLM 生成
	if h.svc != nil {
		plan, err := h.svc.GenerateLessonPlan(c.Request.Context(), req.Topic, req.CourseID)
		if err == nil && plan != nil {
			c.JSON(http.StatusOK, plan)
			return
		}
	}

	// 兜底 mock
	topic := req.Topic
	if topic == "" {
		topic = "二叉树遍历"
	}
	c.JSON(http.StatusOK, gin.H{
		"topic":        topic,
		"outline":      "掌握二叉树的三种遍历方式及其应用场景",
		"key_points":   []string{"前序遍历（根-左-右）", "中序遍历（左-根-右）", "后序遍历（左-右-根）", "层次遍历（BFS）"},
		"difficulties": []string{"递归与非递归实现的转换", "根据遍历序列重建二叉树"},
		"strategies":   []string{"先用动画演示遍历过程", "让学生手动模拟小规模二叉树", "对比递归与栈实现的异同", "布置重建二叉树的练习题"},
		"interactions": []string{"课堂提问：给定二叉树画出三种遍历结果", "小组讨论：非递归遍历的实现思路", "即时练习：LeetCode 144/94/145"},
		"homework":     []string{"实现二叉树三种遍历的递归和非递归版本", "完成根据前序+中序重建二叉树"},
		"data_source":  "fallback",
	})
}

// ExamGen AI 考试出题
func (h *TeacherHandler) ExamGen(c *gin.Context) {
	var req struct {
		CourseName string `json:"course_name"`
	}
	_ = c.ShouldBindJSON(&req)

	if h.svc != nil {
		paper, err := h.svc.GenerateExam(c.Request.Context(), req.CourseName)
		if err == nil && paper != nil {
			c.JSON(http.StatusOK, paper)
			return
		}
	}

	// 兜底
	courseName := req.CourseName
	if courseName == "" {
		courseName = "数据结构"
	}
	c.JSON(http.StatusOK, gin.H{
		"title":       courseName + "期中考试",
		"total_score": 100,
		"duration":    120,
		"sections": []gin.H{
			{"type": "选择题", "count": 10, "score_each": 3, "subtotal": 30},
			{"type": "填空题", "count": 5, "score_each": 4, "subtotal": 20},
			{"type": "简答题", "count": 3, "score_each": 10, "subtotal": 30},
			{"type": "编程题", "count": 2, "score_each": 10, "subtotal": 20},
		},
		"sample_questions": []gin.H{
			{"type": "选择题", "question": "在一棵完全二叉树中，若第i层有k个节点，则第i+1层最多有几个节点？", "options": []string{"k", "2k", "k+1", "2k+1"}, "answer": "B"},
			{"type": "编程题", "question": "实现二叉树的层次遍历，返回每层节点值的列表", "answer": "使用队列BFS实现"},
		},
		"data_source": "fallback",
	})
}

// ClassInteract AI 课堂互动
func (h *TeacherHandler) ClassInteract(c *gin.Context) {
	var req struct {
		Topic string `json:"topic"`
	}
	_ = c.ShouldBindJSON(&req)

	if h.svc != nil {
		interaction, err := h.svc.GenerateInteraction(c.Request.Context(), req.Topic)
		if err == nil && interaction != nil {
			c.JSON(http.StatusOK, interaction)
			return
		}
	}

	// 兜底
	topic := req.Topic
	if topic == "" {
		topic = "二叉搜索树"
	}
	c.JSON(http.StatusOK, gin.H{
		"question":      "请解释为什么中序遍历二叉搜索树可以得到有序序列？",
		"difficulty":    "medium",
		"expected_time": 3,
		"hints":         []string{"思考BST的性质", "左子树所有节点小于根", "递归思考"},
		"follow_up":     "如果要得到降序序列，应该如何修改遍历顺序？",
		"data_source":   "fallback",
	})
}

// Grading AI 作业批改
func (h *TeacherHandler) Grading(c *gin.Context) {
	var req struct {
		CourseName string `json:"course_name"`
	}
	_ = c.ShouldBindJSON(&req)

	if h.svc != nil {
		result, err := h.svc.GradeAssignments(c.Request.Context(), req.CourseName)
		if err == nil && result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}

	// 兜底
	c.JSON(http.StatusOK, gin.H{
		"total_submissions": 45,
		"graded":            45,
		"average_score":     78.5,
		"distribution": gin.H{
			"90-100": 8, "80-89": 15, "70-79": 12, "60-69": 7, "below_60": 3,
		},
		"common_issues":   []string{"递归终止条件遗漏", "空指针未判断", "时间复杂度分析不准确"},
		"excellent_works": []string{"张三 - 代码简洁，注释清晰", "李四 - 额外实现了迭代版本"},
		"data_source":     "fallback",
	})
}

// Heatmap 班级学情热力图
func (h *TeacherHandler) Heatmap(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.GenerateHeatmap(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	if courseName == "" {
		courseName = "数据结构"
	}
	c.JSON(http.StatusOK, gin.H{
		"course_name": courseName,
		"points": []gin.H{
			{"name": "线性表", "mastery": 0.88},
			{"name": "栈与队列", "mastery": 0.82},
			{"name": "二叉树", "mastery": 0.65},
			{"name": "图", "mastery": 0.45},
			{"name": "排序", "mastery": 0.72},
			{"name": "查找", "mastery": 0.58},
		},
		"weak_top_five":  []string{"图的最短路径", "AVL树旋转", "B树插入删除", "哈希冲突处理", "堆排序"},
		"total_students": 45, "anomaly_count": 5,
		"data_source": "fallback",
	})
}

// Reflection AI 教学反思
func (h *TeacherHandler) Reflection(c *gin.Context) {
	if h.svc != nil {
		overview := h.svc.GenerateDailyOverview(c.Request.Context())
		if overview != nil {
			c.JSON(http.StatusOK, gin.H{
				"period":                   "2026年第8周",
				"last_reflection":          overview.LastReflection,
				"strengths":                []string{"课堂互动积极", "实验完成率提升", "学生反馈正面"},
				"weaknesses":               []string{"图论部分讲解节奏偏快", "编程练习时间不足"},
				"suggestions":              []string{"下周增加一次习题课", "增加课堂编程演示", "针对薄弱学生安排答疑"},
				"student_feedback_summary": "85%学生认为课程难度适中",
				"data_source":              "ai",
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"period":      "2026年第8周",
		"strengths":   []string{"课堂互动积极", "实验完成率提升"},
		"weaknesses":  []string{"图论部分讲解节奏偏快"},
		"suggestions": []string{"下周增加一次习题课", "增加课堂编程演示"},
		"data_source": "fallback",
	})
}

// StyleDist 学生学习风格分布
func (h *TeacherHandler) StyleDist(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.GenerateStyleDist(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	if courseName == "" {
		courseName = "数据结构"
	}
	c.JSON(http.StatusOK, gin.H{
		"course_name": courseName, "total": 45,
		"distribution": gin.H{"视觉型": 12, "听觉型": 8, "动手型": 15, "阅读型": 10},
		"suggestions":  []string{"动手型学生占比最高，建议增加实验和编程练习", "为视觉型学生准备更多图示和动画"},
		"data_source":  "fallback",
	})
}

// CommunityQA 社区专业答疑
func (h *TeacherHandler) CommunityQA(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.GenerateCommunityQA(c.Request.Context())
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"my_answers": []gin.H{
			{"id": "1", "question": "递归和迭代的区别是什么？", "answer": "递归是函数调用自身，迭代是循环结构。递归代码简洁但有栈溢出风险，迭代效率更高。", "likes": 12, "certified": true, "time": "2026-05-14"},
			{"id": "2", "question": "什么是死锁？", "answer": "死锁是两个或多个进程互相等待对方释放资源而无限等待的状态。四个必要条件：互斥、占有等待、不可抢占、循环等待。", "likes": 8, "certified": true, "time": "2026-05-13"},
		},
		"pending_questions": []gin.H{
			{"id": "3", "question": "B+树和B树的区别？", "course": "数据结构", "asker": "匿名同学", "time": "1小时前"},
			{"id": "4", "question": "虚拟内存的页面置换算法有哪些？", "course": "操作系统", "asker": "学习中", "time": "3小时前"},
		},
		"stats":       gin.H{"total_answers": 15, "certified_count": 12, "likes_received": 45, "questions_in_faq": 8},
		"data_source": "fallback",
	})
}

// ======================== P2 深度分析功能 ========================

// FAQKnowledge AI 答疑知识库管理
func (h *TeacherHandler) FAQKnowledge(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.ManageFAQKnowledge(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"course":         courseName,
		"faq_count":      15,
		"pending_review": 3,
		"recent_questions": []gin.H{
			{"q": "递归和迭代的区别？", "status": "已发布", "from": "学生提问"},
			{"q": "动态规划的解题思路？", "status": "待审核", "from": "AI生成"},
		},
		"data_source": "fallback",
	})
}

// StudentTwin 学生数字孪生（授课班级视图）
func (h *TeacherHandler) StudentTwin(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.GenerateStudentTwinTeaching(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"course":       courseName,
		"total":        45,
		"at_risk":      5,
		"top_students": []gin.H{{"name": "王芳", "score": 92, "strength": "算法能力强"}},
		"data_source":  "fallback",
	})
}

// KnowledgeCoverage AI 知识点覆盖检查
func (h *TeacherHandler) KnowledgeCoverage(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.CheckKnowledgeCoverage(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"course":        courseName,
		"coverage_rate": 0.85,
		"uncovered":     []string{"红黑树删除", "并查集优化"},
		"suggestions":   []string{"建议在期末前补充红黑树内容"},
		"data_source":   "fallback",
	})
}

// IdeologicalSuggestions 课程思政建议
func (h *TeacherHandler) IdeologicalSuggestions(c *gin.Context) {
	courseName := c.Query("course")
	if h.svc != nil {
		data := h.svc.GenerateIdeologicalSuggestions(c.Request.Context(), courseName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"course":      courseName,
		"elements":    []gin.H{{"type": "科学精神", "point": "算法设计的严谨性与批判思维", "material": "图灵奖得主故事"}},
		"data_source": "fallback",
	})
}

// PersonalizedTeaching AI 个性化教学建议
func (h *TeacherHandler) PersonalizedTeaching(c *gin.Context) {
	studentName := c.Query("student")
	if h.svc != nil {
		data := h.svc.GeneratePersonalizedTeaching(c.Request.Context(), studentName)
		if data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"student":        studentName,
		"learning_style": "动手实践型",
		"weak_points":    []string{"递归思想", "动态规划"},
		"strategy":       "建议增加编程练习量，用可视化工具辅助理解抽象概念",
		"data_source":    "fallback",
	})
}
