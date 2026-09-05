package service

import "context"

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
	_ = ctx
	return &HeatmapData{CourseName: courseName, Points: []map[string]interface{}{}, WeakTopFive: []string{}, DataSource: "real"}
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
	_ = ctx
	if courseName == "" {
		courseName = "数据结构"
	}
	return &StyleDistData{CourseName: courseName, Total: 45, Distribution: map[string]int{"视觉型": 12, "听觉型": 8, "动手型": 15, "阅读型": 10}, Suggestions: []string{"动手型学生占比最高，建议增加实验和编程练习", "为视觉型学生准备更多图示和动画"}, DataSource: "reference"}
}

// CommunityQAData 社区专业答疑
type CommunityQAData struct {
	MyAnswers        []map[string]interface{} `json:"my_answers"`
	PendingQuestions []map[string]interface{} `json:"pending_questions"`
	Stats            map[string]interface{}   `json:"stats"`
	DataSource       string                   `json:"data_source"`
}

func (s *TeacherService) GenerateCommunityQA(ctx context.Context) *CommunityQAData {
	_ = ctx
	return &CommunityQAData{
		MyAnswers:        []map[string]interface{}{{"id": "1", "question": "递归和迭代的区别是什么？", "answer": "递归是函数调用自身，迭代是循环结构。递归代码简洁但有栈溢出风险，迭代效率更高。", "likes": 12, "certified": true, "time": "2026-05-14"}, {"id": "2", "question": "什么是死锁？", "answer": "死锁是两个或多个进程互相等待对方释放资源而无限等待的状态。四个必要条件：互斥、占有等待、不可抢占、循环等待。", "likes": 8, "certified": true, "time": "2026-05-13"}},
		PendingQuestions: []map[string]interface{}{{"id": "3", "question": "B+树和B树的区别？", "course": "数据结构", "asker": "匿名同学", "time": "1小时前"}, {"id": "4", "question": "虚拟内存的页面置换算法有哪些？", "course": "操作系统", "asker": "学习中", "time": "3小时前"}},
		Stats:            map[string]interface{}{"total_answers": 15, "certified_count": 12, "likes_received": 45, "questions_in_faq": 8}, DataSource: "reference",
	}
}
