package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// TokenUsageRepo 词元使用记录数据访问
type TokenUsageRepo struct {
	db *sql.DB
}

// NewTokenUsageRepo 创建词元使用记录 repo
func NewTokenUsageRepo(db *sql.DB) *TokenUsageRepo {
	return &TokenUsageRepo{db: db}
}

// Create 保存一条词元使用记录
func (r *TokenUsageRepo) Create(u *model.TokenUsage) error {
	_, err := r.db.Exec(
		`INSERT INTO token_usage (user_id, session_id, prompt_tokens, output_tokens, model_provider)
		 VALUES (?, ?, ?, ?, ?)`,
		u.UserID, u.SessionID, u.PromptTokens, u.OutputTokens, u.ModelProvider,
	)
	return err
}

// GetStatsByUserID 获取指定用户在指定天数内的词元统计
func (r *TokenUsageRepo) GetStatsByUserID(userID int64, days int) (*model.TokenStatsData, error) {
	data := &model.TokenStatsData{}

	err := r.db.QueryRow(
		`SELECT
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(prompt_tokens + output_tokens), 0)
		 FROM token_usage
		 WHERE user_id = ? AND created_at >= datetime('now', '-' || ? || ' days')`,
		userID, days,
	).Scan(&data.Summary.TotalPromptTokens, &data.Summary.TotalOutputTokens, &data.Summary.TotalTokens)
	if err != nil {
		return nil, err
	}

	_ = r.db.QueryRow(
		`SELECT COALESCE(SUM(prompt_tokens + output_tokens), 0)
		 FROM token_usage
		 WHERE user_id = ? AND date(created_at) = date('now')`,
		userID,
	).Scan(&data.Summary.TodayTokens)

	rows, err := r.db.Query(
		`SELECT
			date(created_at) as date,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(prompt_tokens + output_tokens) as total_tokens
		 FROM token_usage
		 WHERE user_id = ? AND created_at >= datetime('now', '-' || ? || ' days')
		 GROUP BY date(created_at)
		 ORDER BY date(created_at) ASC`,
		userID, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p model.TokenDailyPoint
		if err := rows.Scan(&p.Date, &p.PromptTokens, &p.OutputTokens, &p.TotalTokens); err != nil {
			return nil, err
		}
		data.Daily = append(data.Daily, p)
	}

	return data, rows.Err()
}

// GetSubordinateStats 获取下级用户在指定天数内的词元统计
func (r *TokenUsageRepo) GetSubordinateStats(userIDs []int64, days int) ([]model.SubordinateTokenStats, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	placeholders := ""
	args := make([]interface{}, 0, len(userIDs)+1)
	for i, id := range userIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}
	args = append(args, days)

	query := `SELECT
		u.id, u.username, u.display_name,
		COALESCE(SUM(t.prompt_tokens), 0) as prompt_tokens,
		COALESCE(SUM(t.output_tokens), 0) as output_tokens,
		COALESCE(SUM(t.prompt_tokens + t.output_tokens), 0) as total_tokens
	 FROM users u
	 LEFT JOIN token_usage t ON u.id = t.user_id AND t.created_at >= datetime('now', '-' || ? || ' days')
	 WHERE u.id IN (` + placeholders + `)
	 GROUP BY u.id, u.username, u.display_name
	 ORDER BY total_tokens DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.SubordinateTokenStats
	for rows.Next() {
		var s model.SubordinateTokenStats
		if err := rows.Scan(&s.UserID, &s.Username, &s.DisplayName, &s.PromptTokens, &s.OutputTokens, &s.TotalTokens); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}
