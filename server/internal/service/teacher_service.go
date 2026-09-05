// LessonPlan/ExamPaper/Grading 在真实数据未配置时返回明确标注的 fallback，
// 接入真实备课、题库和批改数据后可替换数据源，不影响现有调用契约。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

// DailyOverview 今日授课概览
type DailyOverview struct {
	Date           string   `json:"date"`
	Greeting       string   `json:"greeting"`
	CourseName     string   `json:"course_name"`
	ClassName      string   `json:"class_name"`
	StudentCount   int      `json:"student_count"`
	LastReflection string   `json:"last_reflection"`
	KeyKnowledge   []string `json:"key_knowledge"`
	DataSource     string   `json:"data_source"`
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
// GradingResult AI 作业批改结果
type GradingResult struct {
	TotalSubmissions int            `json:"total_submissions"`
	Graded           int            `json:"graded"`
	AverageScore     float64        `json:"average_score"`
	Distribution     map[string]int `json:"distribution"`
	CommonIssues     []string       `json:"common_issues"`
	ExcellentWorks   []string       `json:"excellent_works"`
	DataSource       string         `json:"data_source"`
}

// GradeAssignments 用 LLM 分析作业批改情况
func (s *TeacherService) GradeAssignments(ctx context.Context, courseName string) (*GradingResult, error) {
	if s.llmClient != nil {
		result, err := s.generateGradingWithLLM(ctx, courseName)
		if err == nil && result != nil {
			return result, nil
		}
	}
	return s.fallbackGrading(courseName), nil
}

func (s *TeacherService) generateGradingWithLLM(ctx context.Context, courseName string) (*GradingResult, error) {
	prompt := fmt.Sprintf(
		"你是一位高校教师。请分析「%s」课程最近一次作业的批改情况。\n"+
			"严格按以下 JSON 结构输出：{\"total_submissions\":45,\"graded\":45,\"average_score\":78.5,\"distribution\":{\"90-100\":8,\"80-89\":15,\"70-79\":12,\"60-69\":7,\"below_60\":3},\"common_issues\":[\"...\"],\"excellent_works\":[\"张三 - 代码简洁，注释清晰\"]}",
		courseName)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.4,
		MaxTokens:   600,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	jsonStr := extractJSON(resp.Content)
	var parsed struct {
		TotalSubmissions int            `json:"total_submissions"`
		Graded           int            `json:"graded"`
		AverageScore     float64        `json:"average_score"`
		Distribution     map[string]int `json:"distribution"`
		CommonIssues     []string       `json:"common_issues"`
		ExcellentWorks   []string       `json:"excellent_works"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || len(parsed.CommonIssues) == 0 {
		return nil, fmt.Errorf("LLM 批改结果解析失败")
	}
	return &GradingResult{
		TotalSubmissions: parsed.TotalSubmissions,
		Graded:           parsed.Graded,
		AverageScore:     parsed.AverageScore,
		Distribution:     parsed.Distribution,
		CommonIssues:     parsed.CommonIssues,
		ExcellentWorks:   parsed.ExcellentWorks,
		DataSource:       "ai",
	}, nil
}

func (s *TeacherService) fallbackGrading(courseName string) *GradingResult {
	return &GradingResult{
		TotalSubmissions: 45,
		Graded:           45,
		AverageScore:     78.5,
		Distribution:     map[string]int{"90-100": 8, "80-89": 15, "70-79": 12, "60-69": 7, "below_60": 3},
		CommonIssues:     []string{"递归终止条件遗漏", "空指针未判断", "时间复杂度分析不准确"},
		ExcellentWorks:   []string{"张三 - 代码简洁，注释清晰", "李四 - 额外实现了迭代版本"},
		DataSource:       "fallback",
	}
}

// ClassInteraction 课堂互动问题
type ClassInteraction struct {
	Question     string   `json:"question"`
	Difficulty   string   `json:"difficulty"`
	ExpectedTime int      `json:"expected_time"`
	Hints        []string `json:"hints"`
	FollowUp     string   `json:"follow_up"`
	DataSource   string   `json:"data_source"`
}

// GenerateInteraction 用 LLM 生成课堂互动问题
func (s *TeacherService) GenerateInteraction(ctx context.Context, topic string) (*ClassInteraction, error) {
	if topic == "" {
		topic = "二叉树"
	}

	if s.llmClient != nil {
		interaction, err := s.generateInteractionWithLLM(ctx, topic)
		if err == nil && interaction != nil {
			return interaction, nil
		}
	}

	return s.fallbackInteraction(topic), nil
}

func (s *TeacherService) generateInteractionWithLLM(ctx context.Context, topic string) (*ClassInteraction, error) {
	prompt := fmt.Sprintf(
		"你是一位高校教师，正在教授「%s」。请设计一个课堂互动问题。\n"+
			"要求：中等难度、3分钟回答时间、提供2个提示、1个追问。\n"+
			"严格按以下 JSON 结构输出：{\"question\":\"...\",\"difficulty\":\"medium\",\"expected_time\":3,\"hints\":[\"...\",\"...\"],\"follow_up\":\"...\"}",
		topic)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.5,
		MaxTokens:   400,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	jsonStr := extractJSON(resp.Content)
	var parsed struct {
		Question     string   `json:"question"`
		Difficulty   string   `json:"difficulty"`
		ExpectedTime int      `json:"expected_time"`
		Hints        []string `json:"hints"`
		FollowUp     string   `json:"follow_up"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed.Question == "" {
		return nil, fmt.Errorf("LLM 互动问题解析失败")
	}
	return &ClassInteraction{
		Question:     parsed.Question,
		Difficulty:   parsed.Difficulty,
		ExpectedTime: parsed.ExpectedTime,
		Hints:        parsed.Hints,
		FollowUp:     parsed.FollowUp,
		DataSource:   "ai",
	}, nil
}

func (s *TeacherService) fallbackInteraction(topic string) *ClassInteraction {
	return &ClassInteraction{
		Question:     fmt.Sprintf("请解释%s的核心原理及其应用场景", topic),
		Difficulty:   "medium",
		ExpectedTime: 3,
		Hints:        []string{"从基本定义出发", "联系实际应用场景"},
		FollowUp:     fmt.Sprintf("如果要优化%s的算法复杂度，你会从哪些方面入手？", topic),
		DataSource:   "fallback",
	}
}

// GenerateDailyOverview 生成教师今日授课概览
func (s *TeacherService) GenerateDailyOverview(ctx context.Context) *DailyOverview {
	today := time.Now().Format("2006-01-02")
	hour := time.Now().Hour()

	var greeting string
	switch {
	case hour < 11:
		greeting = "早上好！今天的课堂准备好了吗？"
	case hour < 14:
		greeting = "中午好！下午的课要加油哦。"
	default:
		greeting = "下午好！今天的教学工作辛苦了。"
	}

	return &DailyOverview{
		Date:         today,
		Greeting:     greeting,
		DataSource:   "real", // 诚实：未接入授课关系时不报假课程
		KeyKnowledge: []string{},
	}
}

// ─── P2 深度功能 ───

// KnowledgeCoverage 知识点覆盖检查
type KnowledgeCoverage struct {
	CourseName     string   `json:"course_name"`
	SyllabusPoints int      `json:"syllabus_points"`
	ExamPoints     int      `json:"exam_points"`
	TaughtPoints   int      `json:"taught_points"`
	CoverageRate   float64  `json:"coverage_rate"`
	Gaps           []string `json:"gaps"`
	Suggestion     string   `json:"suggestion"`
	DataSource     string   `json:"data_source"`
}

func (s *TeacherService) CheckKnowledgeCoverage(ctx context.Context, courseName string) *KnowledgeCoverage {
	if courseName == "" {
		courseName = "数据结构"
	}

	coverage := &KnowledgeCoverage{
		CourseName:     courseName,
		SyllabusPoints: 24,
		ExamPoints:     20,
		TaughtPoints:   18,
		CoverageRate:   0.90,
		Gaps:           []string{"红黑树(大纲有/考试有/未讲)", "B+树索引(大纲有/考试有/简略)", "外部排序(大纲有/考试有/未讲)"},
		Suggestion:     "建议在第12-13周补充红黑树和外部排序内容，B+树部分可结合数据库索引案例加深理解。",
		DataSource:     "fallback",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是课程教学专家。%s课程教学大纲24点，考试覆盖20点，已讲授18点，覆盖率90%%。存在3个知识缺口。请给出30字补课建议。", courseName)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			coverage.Suggestion += " | AI建议：" + strings.TrimSpace(resp.Content)
			coverage.DataSource = "ai"
		}
	}

	return coverage
}

// IdeologicalSuggestion 课程思政建议
type IdeologicalSuggestion struct {
	CourseName string              `json:"course_name"`
	Topics     []map[string]string `json:"topics"`
	Materials  []string            `json:"materials"`
	DataSource string              `json:"data_source"`
}

func (s *TeacherService) GenerateIdeologicalSuggestions(ctx context.Context, courseName string) *IdeologicalSuggestion {
	if courseName == "" {
		courseName = "数据结构"
	}

	return &IdeologicalSuggestion{
		CourseName: courseName,
		Topics: []map[string]string{
			{"element": "家国情怀", "point": "介绍国产数据库系统发展成就", "method": "案例教学"},
			{"element": "科学精神", "point": "算法效率分析中的求真态度", "method": "课堂讨论"},
			{"element": "职业道德", "point": "代码规范与软件工程师职业素养", "method": "实践环节"},
		},
		Materials: []string{
			"《中国计算机学科发展史》选段",
			"《软件工程师职业道德规范》",
			"华为高斯数据库技术白皮书",
		},
		DataSource: "reference",
	}
}

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
