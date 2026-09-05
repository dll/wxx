package service

func fallbackLearningDiary(today string) *LearningDiary {
	return &LearningDiary{Date: today, CoursesStudied: []string{"数据结构", "操作系统"}, KeyPoints: []string{"二叉树前序/中序/后序遍历", "进程状态转换图", "死锁四个必要条件"}, StudyMinutes: 185, Quiz: []map[string]interface{}{{"question": "二叉树的前序遍历顺序是？", "options": []string{"根→左→右", "左→根→右", "左→右→根", "根→右→左"}, "correct_index": 0, "explanation": "前序遍历（Preorder）先访问根节点，再递归遍历左子树，最后右子树。"}}, TomorrowPlan: "复习二叉树算法，准备数据结构实验报告", Encouragement: "坚持就是胜利，今天的努力是明天的基石！", DataSource: "fallback"}
}
