package service

func fallbackQAPlaza() *QAPlazaData {
	return &QAPlazaData{HotQuestions: []map[string]interface{}{{"id": "1", "title": "转专业需要什么条件？", "author": "匿名同学", "answers": 5, "views": 128, "ai_answer": "转专业一般需要：1.大一第一学期结束后申请 2.绩点达到3.0以上 3.通过目标专业考核", "tags": []string{"政策", "学业"}}, {"id": "2", "title": "图书馆自习室怎么预约？", "author": "学习达人", "answers": 3, "views": 89, "ai_answer": "通过校园APP→图书馆→座位预约，每天22:00开放次日预约", "tags": []string{"生活", "图书馆"}}, {"id": "3", "title": "ACM竞赛如何入门？", "author": "编程新手", "answers": 8, "views": 256, "ai_answer": "建议从C++基础开始，刷LeetCode简单题，参加校内训练赛", "tags": []string{"竞赛", "学业"}}}, Categories: []string{"学业", "生活", "政策", "心理", "就业", "竞赛"}, MyPosts: 2, MyAnswers: 5, DataSource: "fallback"}
}
