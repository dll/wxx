package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

type DynamicMentorData struct {
	Name            string                   `json:"name"`
	AvatarStyle     string                   `json:"avatar_style"`
	Personality     string                   `json:"personality"`
	MemoryContext   []map[string]interface{} `json:"memory_context"`
	CurrentMood     string                   `json:"current_mood"`
	Greeting        string                   `json:"greeting"`
	Suggestions     []string                 `json:"suggestions"`
	InteractionTips []string                 `json:"interaction_tips"`
	DataSource      string                   `json:"data_source"`
}

func (s *StudentService) GenerateDynamicMentor(ctx context.Context, userID int64, style string) *DynamicMentorData {
	if style == "" {
		style = "温和"
	}
	styles := map[string]string{"温和": "耐心细致，循循善诱，善于鼓励", "严格": "要求严格，直指问题，督促进步", "幽默": "轻松风趣，用故事和比喻讲解知识", "思政": "注重价值引领，融入家国情怀与社会责任"}
	personality := styles[style]
	if personality == "" {
		personality = styles["温和"]
	}
	data := &DynamicMentorData{Name: "蔚小芯", AvatarStyle: style, Personality: personality, MemoryContext: []map[string]interface{}{{"date": "2026-05-15", "topic": "数据结构复习", "takeaway": "用户对图论部分存在畏难情绪，已推荐可视化学习工具"}, {"date": "2026-05-17", "topic": "学习动力", "takeaway": "用户期中成绩进步，信心增强，已设定ACM竞赛目标"}}, CurrentMood: "热情投入", Greeting: fmt.Sprintf("你好！我是你的%s风格AI导师蔚小芯。看到你最近的进步我很开心！今天我们继续加油吧。", style), Suggestions: []string{"本周重点攻克图的最短路径算法", "每天完成2道LeetCode中等题", "周末参加ACM训练赛"}, InteractionTips: []string{"可以随时问我学习问题", "我会记住你的学习偏好和薄弱点", "每周生成一份学习报告"}, DataSource: "reference"}
	if s.llmClient != nil {
		if resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: fmt.Sprintf("你是%s风格的AI数字导师，名叫蔚小芯。请根据学生的最近学习记录生成一段50字的个性化开场白。", style)}}, Temperature: 0.7, MaxTokens: 300}); err == nil && resp != nil && resp.Content != "" {
			data.Greeting = strings.TrimSpace(resp.Content)
			data.DataSource = "ai"
		}
	}
	return data
}
