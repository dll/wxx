package repository

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
