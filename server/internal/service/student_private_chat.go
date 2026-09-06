package service

import "context"

type PrivateChatData struct {
	Conversations       []map[string]interface{} `json:"conversations"`
	RecommendedContacts []map[string]interface{} `json:"recommended_contacts"`
	DataSource          string                   `json:"data_source"`
}

func (s *StudentService) GeneratePrivateChat(ctx context.Context) *PrivateChatData {
	_ = ctx
	return &PrivateChatData{Conversations: []map[string]interface{}{{"id": "1", "name": "李辅导员", "role": "counselor", "last_message": "明天下午来办公室聊聊", "time": "10:30", "unread": 1}, {"id": "2", "name": "张学长", "role": "student", "last_message": "ACM训练资料已发你邮箱", "time": "昨天", "unread": 0}, {"id": "3", "name": "AI学友-王同学", "role": "student", "last_message": "明天一起去图书馆复习吧", "time": "昨天", "unread": 0}}, RecommendedContacts: []map[string]interface{}{{"name": "赵学姐", "reason": "同专业大三，擅长算法", "match_score": 88}, {"name": "刘同学", "reason": "学习风格互补，可组队复习", "match_score": 82}}, DataSource: "reference"}
}
