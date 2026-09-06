package service

func referenceQALeaderboard() *QALeaderboardData {
	return &QALeaderboardData{TopAnswerers: []map[string]interface{}{{"rank": 1, "name": "知识达人", "answers": 23, "adopted": 15, "score": 95.0}, {"rank": 2, "name": "热心学长", "answers": 18, "adopted": 10, "score": 82.5}, {"rank": 3, "name": "编程高手", "answers": 12, "adopted": 8, "score": 78.0}}, Contributors: []map[string]interface{}{{"rank": 1, "name": "知识达人", "contributions": 15, "quality_score": 4.8}, {"rank": 2, "name": "热心学长", "contributions": 10, "quality_score": 4.5}, {"rank": 3, "name": "学霸笔记", "contributions": 8, "quality_score": 4.3}}, HotQuestions: []map[string]interface{}{{"rank": 1, "title": "ACM竞赛如何入门？", "count": 8}, {"rank": 2, "title": "转专业需要什么条件？", "count": 5}, {"rank": 3, "title": "考研还是就业？", "count": 12}}, Period: "本周", DataSource: "reference"}
}
