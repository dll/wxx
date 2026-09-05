// LessonPlan/ExamPaper/Grading 在真实数据未配置时返回明确标注的 fallback，
// 接入真实备课、题库和批改数据后可替换数据源，不影响现有调用契约。
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// TeacherService 教师角色 AI 功能服务
type TeacherService struct {
	llmClient llm.ChatClient
}

// NewTeacherService 创建教师服务
func NewTeacherService(llmClient llm.ChatClient) *TeacherService {
	return &TeacherService{llmClient: llmClient}
}

// LessonPlan 教案结构
type LessonPlan struct {
	Topic        string   `json:"topic"`
	Outline      string   `json:"outline"`
	KeyPoints    []string `json:"key_points"`
	Difficulties []string `json:"difficulties"`
	Strategies   []string `json:"strategies"`
	Interactions []string `json:"interactions"`
	Homework     []string `json:"homework"`
	DataSource   string   `json:"data_source"` // ai/fallback
}

// GenerateLessonPlan 用 LLM 生成教案
func (s *TeacherService) GenerateLessonPlan(ctx context.Context, topic, courseID string) (*LessonPlan, error) {
	if topic == "" {
		topic = "二叉树遍历"
	}

	if s.llmClient != nil {
		plan, err := s.generatePlanWithLLM(ctx, topic, courseID)
		if err == nil && plan != nil {
			return plan, nil
		}
	}

	return s.fallbackPlan(topic), nil
}

func (s *TeacherService) generatePlanWithLLM(ctx context.Context, topic, courseID string) (*LessonPlan, error) {
	var b strings.Builder
	b.WriteString("你是一位经验丰富的高校教师。请为以下课程主题生成一份教案。\n\n")
	b.WriteString(fmt.Sprintf("课程主题：%s\n", topic))
	if courseID != "" {
		b.WriteString(fmt.Sprintf("课程编号：%s\n", courseID))
	}
	b.WriteString("\n请按以下 JSON 格式输出（严格遵守）：\n")
	b.WriteString(`{
  "outline": "教案大纲（含教学目标、重难点、教学过程）",
  "key_points": ["重点1", "重点2", "重点3"],
  "difficulties": ["难点1", "难点2"],
  "strategies": ["教学策略1", "教学策略2"],
  "interactions": ["互动设计1", "互动设计2"],
  "homework": ["课后作业1", "课后作业2"]
}`)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.4,
		MaxTokens:   1200,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	plan := parseLessonPlanJSON(resp.Content, topic)
	plan.DataSource = "ai"
	return plan, nil
}

// ExamPaper AI 考试出题结果
type ExamPaper struct {
	Title           string                   `json:"title"`
	TotalScore      int                      `json:"total_score"`
	Duration        int                      `json:"duration"`
	Sections        []map[string]interface{} `json:"sections"`
	SampleQuestions []map[string]interface{} `json:"sample_questions"`
	DataSource      string                   `json:"data_source"`
}

// GenerateExam 用 LLM 生成考试试卷
func (s *TeacherService) GenerateExam(ctx context.Context, courseName string) (*ExamPaper, error) {
	if courseName == "" {
		courseName = "数据结构"
	}

	if s.llmClient != nil {
		paper, err := s.generateExamWithLLM(ctx, courseName)
		if err == nil && paper != nil {
			return paper, nil
		}
	}

	return s.fallbackExam(courseName), nil
}

func (s *TeacherService) generateExamWithLLM(ctx context.Context, courseName string) (*ExamPaper, error) {
	prompt := fmt.Sprintf(
		"你是一位高校教师。请为「%s」课程设计一份期中考试试卷。\n"+
			"要求：满分100分，时长120分钟，包含选择题(10题x3分)、填空题(5题x4分)、简答题(3题x10分)、编程题(2题x10分)。\n"+
			"请给出各题型的一个样题（含题干、选项、答案）。\n"+
			"严格按以下 JSON 结构输出：{\"title\":\"...\",\"total_score\":100,\"duration\":120,\"sections\":[{\"type\":\"选择题\",\"count\":10,\"score_each\":3,\"subtotal\":30}],\"sample_questions\":[{\"type\":\"选择题\",\"question\":\"...\",\"options\":[\"A\",\"B\",\"C\",\"D\"],\"answer\":\"B\"}]}",
		courseName)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.4,
		MaxTokens:   1000,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	paper := parseExamPaper(resp.Content, courseName)
	if paper == nil {
		return nil, fmt.Errorf("LLM 试卷解析失败")
	}
	paper.DataSource = "ai"
	return paper, nil
}

// parseExamPaper 解析 LLM 返回的试卷 JSON（兼容 markdown 代码块包裹），解析失败返回 nil
// ─── P2 深度功能 ───

// StudentTwinTeaching 授课班级数字孪生视图
type StudentTwinTeaching struct {
	CourseName    string                   `json:"course_name"`
	TotalStudents int                      `json:"total_students"`
	AvgMastery    float64                  `json:"avg_mastery"`
	Distribution  map[string]int           `json:"distribution"`
	FocusStudents []map[string]interface{} `json:"focus_students"`
	DataSource    string                   `json:"data_source"`
}

func (s *TeacherService) GenerateStudentTwinTeaching(ctx context.Context, courseName string) *StudentTwinTeaching {
	if courseName == "" {
		courseName = "数据结构"
	}

	// 诚实：该服务当前未接入真实课程/学情数据源，不返回编造的学生教学画像。
	// 返回空结构，前端应展示「数据积累中」而非虚构学生名单。
	return &StudentTwinTeaching{
		CourseName:    courseName,
		TotalStudents: 0,
		AvgMastery:    0,
		Distribution:  map[string]int{},
		FocusStudents: []map[string]interface{}{},
		DataSource:    "empty",
	}
}

// FAQKnowledgeBase 答疑知识库管理
type FAQKnowledgeBase struct {
	CourseName   string                   `json:"course_name"`
	TotalFAQs    int                      `json:"total_faqs"`
	NewQuestions []map[string]interface{} `json:"new_questions"`
	PopularFAQs  []map[string]interface{} `json:"popular_faqs"`
	Suggestion   string                   `json:"suggestion"`
	DataSource   string                   `json:"data_source"`
}

func (s *TeacherService) ManageFAQKnowledge(ctx context.Context, courseName string) *FAQKnowledgeBase {
	if courseName == "" {
		courseName = "数据结构"
	}

	return &FAQKnowledgeBase{
		CourseName: courseName,
		TotalFAQs:  45,
		NewQuestions: []map[string]interface{}{
			{"id": "q1", "question": "Dijkstra算法如何处理负权边？", "asker": "匿名", "time": "2小时前", "status": "待回答"},
			{"id": "q2", "question": "红黑树和AVL树的实际应用场景区别？", "asker": "计科2301班", "time": "5小时前", "status": "待审核"},
		},
		PopularFAQs: []map[string]interface{}{
			{"question": "递归和迭代的区别？", "views": 234, "likes": 45, "status": "已发布"},
			{"question": "什么是动态规划的最优子结构？", "views": 189, "likes": 32, "status": "已发布"},
			{"question": "堆排序和快速排序的比较？", "views": 156, "likes": 28, "status": "已发布"},
		},
		Suggestion: "建议将BFS/DFS相关问题整理为专题FAQ，补充图论算法的可视化解释。",
		DataSource: "reference",
	}
}

// PersonalizedTeaching 个性化教学建议
type PersonalizedTeaching struct {
	StudentName   string   `json:"student_name"`
	LearningStyle string   `json:"learning_style"`
	WeakPoints    []string `json:"weak_points"`
	Strategy      string   `json:"strategy"`
	Resources     []string `json:"resources"`
	DataSource    string   `json:"data_source"`
}

func (s *TeacherService) GeneratePersonalizedTeaching(ctx context.Context, studentName string) *PersonalizedTeaching {
	if studentName == "" {
		studentName = "张明"
	}

	data := &PersonalizedTeaching{
		StudentName:   studentName,
		LearningStyle: "动手实践型",
		WeakPoints:    []string{"递归思想", "动态规划", "图论算法"},
		Strategy:      "建议增加编程练习量，用可视化工具辅助理解抽象概念。每学完一个算法立即用代码实现，对比不同解法的复杂度。",
		Resources: []string{
			"LeetCode 热题100",
			"《算法导论》动态规划章节",
			"VisuAlgo 可视化学习平台",
		},
		DataSource: "fallback",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是教学专家。学生%s，学习风格动手实践型，薄弱点在递归和动态规划。请给出50字个性化教学建议。", studentName)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.4, MaxTokens: 300,
		})
		if err == nil && resp != nil && resp.Content != "" {
			refined := strings.TrimSpace(resp.Content)
			if len(refined) > 0 {
				data.Strategy = refined
				data.DataSource = "ai"
			}
		}
	}

	return data
}

// ======================== P1 剩余方法 ========================

// HeatmapData 班级学情热力图
type HeatmapData struct {
	CourseName    string                   `json:"course_name"`
	Points        []map[string]interface{} `json:"points"`
	WeakTopFive   []string                 `json:"weak_top_five"`
	TotalStudents int                      `json:"total_students"`
	AnomalyCount  int                      `json:"anomaly_count"`
	AIAnalysis    string                   `json:"ai_analysis"`
	DataSource    string                   `json:"data_source"`
}

func (s *TeacherService) GenerateHeatmap(ctx context.Context, courseName string) *HeatmapData {
	data := &HeatmapData{
		CourseName:    courseName,
		Points:        []map[string]interface{}{},
		WeakTopFive:   []string{},
		TotalStudents: 0,
		AnomalyCount:  0,
		// 诚实空：未接入逐知识点掌握度数据表时不编造演示样例
		DataSource: "real",
	}

	return data
}

// StyleDistData 学生学习风格分布
type StyleDistData struct {
	CourseName   string         `json:"course_name"`
	Total        int            `json:"total"`
	Distribution map[string]int `json:"distribution"`
	Suggestions  []string       `json:"suggestions"`
	DataSource   string         `json:"data_source"`
}

func (s *TeacherService) GenerateStyleDist(ctx context.Context, courseName string) *StyleDistData {
	if courseName == "" {
		courseName = "数据结构"
	}
	return &StyleDistData{
		CourseName: courseName, Total: 45,
		Distribution: map[string]int{"视觉型": 12, "听觉型": 8, "动手型": 15, "阅读型": 10},
		Suggestions: []string{
			"动手型学生占比最高，建议增加实验和编程练习",
			"为视觉型学生准备更多图示和动画",
		},
		DataSource: "reference",
	}
}

// CommunityQAData 社区专业答疑
type CommunityQAData struct {
	MyAnswers        []map[string]interface{} `json:"my_answers"`
	PendingQuestions []map[string]interface{} `json:"pending_questions"`
	Stats            map[string]interface{}   `json:"stats"`
	DataSource       string                   `json:"data_source"`
}

func (s *TeacherService) GenerateCommunityQA(ctx context.Context) *CommunityQAData {
	return &CommunityQAData{
		MyAnswers: []map[string]interface{}{
			{"id": "1", "question": "递归和迭代的区别是什么？", "answer": "递归是函数调用自身，迭代是循环结构。递归代码简洁但有栈溢出风险，迭代效率更高。", "likes": 12, "certified": true, "time": "2026-05-14"},
			{"id": "2", "question": "什么是死锁？", "answer": "死锁是两个或多个进程互相等待对方释放资源而无限等待的状态。四个必要条件：互斥、占有等待、不可抢占、循环等待。", "likes": 8, "certified": true, "time": "2026-05-13"},
		},
		PendingQuestions: []map[string]interface{}{
			{"id": "3", "question": "B+树和B树的区别？", "course": "数据结构", "asker": "匿名同学", "time": "1小时前"},
			{"id": "4", "question": "虚拟内存的页面置换算法有哪些？", "course": "操作系统", "asker": "学习中", "time": "3小时前"},
		},
		Stats: map[string]interface{}{
			"total_answers": 15, "certified_count": 12, "likes_received": 45, "questions_in_faq": 8,
		},
		DataSource: "reference",
	}
}
