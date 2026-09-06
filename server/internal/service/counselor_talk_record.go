package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateTalkRecord 用 LLM 从对话中提取结构化摘要。
func (s *CounselorService) GenerateTalkRecord(ctx context.Context, req *TalkRecordRequest) (*TalkRecord, error) {
	now := time.Now().Format("2006-01-02 15:04")
	record := &TalkRecord{StudentName: req.StudentName, Topic: "日常交流", Emotion: "平稳", Demand: "无特殊诉求", FollowUp: "持续关注", Summary: req.Content, CreatedAt: now}
	if s.llmClient != nil && req.Content != "" {
		if summary, err := s.generateTalkSummary(ctx, req); err == nil && summary != nil {
			record.Topic, record.Emotion, record.Demand, record.Promise, record.FollowUp, record.Summary = summary.Topic, summary.Emotion, summary.Demand, summary.Promise, summary.FollowUp, summary.Summary
		}
	}
	return record, nil
}

func (s *CounselorService) generateTalkSummary(ctx context.Context, req *TalkRecordRequest) (*talkSummary, error) {
	prompt := fmt.Sprintf("你是一位辅导员助理。请从以下谈话内容中提取结构化信息。\n\n学生：%s\n谈话内容：%s\n\n请按以下格式输出（每行一个字段）：\n主题：xxx\n情绪：xxx（平稳/低落/焦虑/愤怒/积极）\n诉求：xxx\n承诺：xxx\n跟进事项：xxx\n摘要：xxx（50字以内）", req.StudentName, req.Content)
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.3, MaxTokens: 400})
	if err != nil || resp == nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}
	ts := &talkSummary{}
	for _, line := range strings.Split(resp.Content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "主题："):
			ts.Topic = strings.TrimPrefix(line, "主题：")
		case strings.HasPrefix(line, "情绪："):
			ts.Emotion = strings.TrimPrefix(line, "情绪：")
		case strings.HasPrefix(line, "诉求："):
			ts.Demand = strings.TrimPrefix(line, "诉求：")
		case strings.HasPrefix(line, "承诺："):
			ts.Promise = strings.TrimPrefix(line, "承诺：")
		case strings.HasPrefix(line, "跟进事项："):
			ts.FollowUp = strings.TrimPrefix(line, "跟进事项：")
		case strings.HasPrefix(line, "摘要："):
			ts.Summary = strings.TrimPrefix(line, "摘要：")
		}
	}
	return ts, nil
}
