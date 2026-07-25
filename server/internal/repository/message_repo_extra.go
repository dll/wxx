package repository

import "fmt"

// GetRecentQuestionsByUserID 获取用户最近的提问文本（跨所有会话）
// 用于个性化推荐引擎提取兴趣关键词
func (r *MessageRepo) GetRecentQuestionsByUserID(userID int64, limit int) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT m.content FROM messages m
		 JOIN sessions s ON m.session_id = s.session_id
		 WHERE s.user_id = ? AND m.role = 'user'
		 ORDER BY m.id DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		questions = append(questions, content)
	}
	return questions, rows.Err()
}

// HotQuestion 平台热门提问聚合项
type HotQuestion struct {
	Title string // 提问文本（截断后）
	Count int    // 被提问次数
}

// GetHotQuestions 统计全平台被提问次数最多的问题（按归一化文本聚合）
// 用于问答广场热榜，数据来自真实 messages 表；无数据时返回空切片。
func (r *MessageRepo) GetHotQuestions(limit int) ([]HotQuestion, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(
		`SELECT TRIM(content) AS q, COUNT(*) AS cnt
		 FROM messages
		 WHERE role = 'user' AND TRIM(content) != ''
		 GROUP BY q
		 ORDER BY cnt DESC, q ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []HotQuestion
	for rows.Next() {
		var q string
		var cnt int
		if err := rows.Scan(&q, &cnt); err != nil {
			return nil, err
		}
		result = append(result, HotQuestion{Title: q, Count: cnt})
	}
	return result, rows.Err()
}

// WeeklyActivity 用户近 N 天的真实学习活跃度（来自 messages/sessions）
type WeeklyActivity struct {
	Questions  int // 提问次数
	Sessions   int // 涉及的会话数
	ActiveDays int // 有提问记录的去重天数
}

// GetWeeklyActivity 统计用户最近 sinceDays 天的真实交互活跃度
// 仅统计 user 角色消息；无数据时各计数为 0（前端据此判断是否展示）。
func (r *MessageRepo) GetWeeklyActivity(userID int64, sinceDays int) (*WeeklyActivity, error) {
	if sinceDays <= 0 {
		sinceDays = 7
	}
	wa := &WeeklyActivity{}
	err := r.db.QueryRow(
		`SELECT COUNT(*) AS questions,
		        COUNT(DISTINCT m.session_id) AS sessions,
		        COUNT(DISTINCT date(m.created_at)) AS active_days
		 FROM messages m
		 JOIN sessions s ON m.session_id = s.session_id
		 WHERE s.user_id = ? AND m.role = 'user'
		   AND m.created_at >= datetime('now', ?)`,
		userID, fmt.Sprintf("-%d days", sinceDays),
	).Scan(&wa.Questions, &wa.Sessions, &wa.ActiveDays)
	if err != nil {
		return nil, err
	}
	return wa, nil
}
