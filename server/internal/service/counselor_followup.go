package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// GenerateFollowUpReminders 生成谈心记录跟进提醒。
func (s *CounselorService) GenerateFollowUpReminders(ctx context.Context, counselorID int64) *FollowUpReminder {
	reminder := &FollowUpReminder{Tasks: []map[string]interface{}{}, Suggestion: "暂无待跟进的谈心记录。完成谈心谈话并保存记录后，这里会自动生成跟进提醒。", DataSource: "real"}
	if s.phase2 != nil && counselorID > 0 {
		if records, err := s.phase2.ListTalkRecords(counselorID, 100); err == nil {
			now := time.Now()
			var overdue, pending int
			for _, rec := range records {
				status, _ := rec["status"].(string)
				if status != "following" {
					continue
				}
				studentName, _ := rec["student_name"].(string)
				topic, _ := rec["topic"].(string)
				createdAt, _ := rec["created_at"].(string)
				if studentName == "" {
					continue
				}
				due := "待跟进"
				if ts, e := time.Parse("2006-01-02 15:04:05", createdAt); e == nil {
					ageDays := now.Sub(ts).Hours() / 24
					if ageDays >= 7 {
						due, overdue = "已逾期", overdue+1
					} else if ageDays >= 3 {
						due, pending = "临近截止", pending+1
					}
				}
				reminder.Tasks = append(reminder.Tasks, map[string]interface{}{"student": studentName, "type": topic, "due": due, "status": status, "priority": "high"})
			}
			reminder.OverdueCount, reminder.PendingCount = overdue, pending+len(reminder.Tasks)
			if len(reminder.Tasks) > 0 {
				reminder.Suggestion = fmt.Sprintf("当前有 %d 名学生的谈心记录待跟进（%d 项已逾期），请优先处理。", len(reminder.Tasks), overdue)
			}
		}
	}
	if s.llmClient != nil && len(reminder.Tasks) > 0 {
		prompt := fmt.Sprintf("你是辅导员助理。%d项待跟进谈话，%d项已逾期。请给出50字优先级建议。", reminder.PendingCount+reminder.OverdueCount, reminder.OverdueCount)
		if resp, e := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: prompt}}, Temperature: 0.3, MaxTokens: 200}); e == nil && resp != nil && resp.Content != "" {
			reminder.Suggestion += " | AI：" + strings.TrimSpace(resp.Content)
			reminder.DataSource = "ai"
		}
	}
	return reminder
}
