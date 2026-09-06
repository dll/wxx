package service

import "time"

func fallbackHotTopics() *HotTopicsData {
	now := time.Now().Format("2006-01-02 15:04")
	return &HotTopicsData{Topics: []map[string]interface{}{{"id": "1", "title": "期中考试安排", "heat": 95, "trend": "rising", "posts": 23, "summary": "本学期期中考试集中在第10-11周"}, {"id": "2", "title": "暑期实习招聘", "heat": 82, "trend": "rising", "posts": 15, "summary": "多家互联网公司开放暑期实习岗位"}, {"id": "3", "title": "校园网升级", "heat": 68, "trend": "stable", "posts": 12, "summary": "校园网将于下周升级至千兆"}, {"id": "4", "title": "社团招新", "heat": 55, "trend": "falling", "posts": 8, "summary": "本学期第二轮社团招新已结束"}}, UpdatedAt: now, DataSource: "fallback"}
}
