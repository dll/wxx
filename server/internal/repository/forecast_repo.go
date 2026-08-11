package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/model"
)

// ForecastRepo 问题预案数据访问
type ForecastRepo struct {
	db *sql.DB
}

// NewForecastRepo 创建问题预案 repo
func NewForecastRepo(db *sql.DB) *ForecastRepo {
	return &ForecastRepo{db: db}
}

// CreateForecast 创建问题预案
func (r *ForecastRepo) CreateForecast(forecast *model.IssueForecast) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO issue_forecasts
		 (forecast_id, college_id, category, subcategory, title, risk_level,
		  status, affected_count, root_cause, suggested_actions, data_summary,
		  sources, ai_analysis, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		forecast.ForecastID, forecast.CollegeID, forecast.Category,
		forecast.Subcategory, forecast.Title, forecast.RiskLevel,
		forecast.Status, forecast.AffectedCount, forecast.RootCause,
		forecast.SuggestedActions, forecast.DataSummary,
		forecast.Sources, forecast.AIAnalysis, forecast.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("创建问题预案失败: %w", err)
	}
	return result.LastInsertId()
}

// GetForecast 获取问题预案详情
func (r *ForecastRepo) GetForecast(forecastID string) (*model.IssueForecast, error) {
	forecast := &model.IssueForecast{}
	err := r.db.QueryRow(
		`SELECT id, forecast_id, college_id, category, subcategory, title,
		        risk_level, status, affected_count, root_cause, suggested_actions,
		        data_summary, sources, ai_analysis, created_by, created_at,
		        updated_at, resolved_at, resolved_by
		 FROM issue_forecasts WHERE forecast_id = ?`, forecastID,
	).Scan(
		&forecast.ID, &forecast.ForecastID, &forecast.CollegeID,
		&forecast.Category, &forecast.Subcategory, &forecast.Title,
		&forecast.RiskLevel, &forecast.Status, &forecast.AffectedCount,
		&forecast.RootCause, &forecast.SuggestedActions, &forecast.DataSummary,
		&forecast.Sources, &forecast.AIAnalysis, &forecast.CreatedBy,
		&forecast.CreatedAt, &forecast.UpdatedAt, &forecast.ResolvedAt,
		&forecast.ResolvedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("获取问题预案失败: %w", err)
	}
	return forecast, nil
}

// ListForecasts 分页查询问题预案
func (r *ForecastRepo) ListForecasts(collegeID string, category string, riskLevel string, status string, page int, pageSize int) ([]*model.IssueForecast, int, error) {
	offset := (page - 1) * pageSize

	where := []string{"1=1"}
	args := []interface{}{}

	if collegeID != "" {
		where = append(where, "(college_id = ? OR college_id = '')")
		args = append(args, collegeID)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if riskLevel != "" {
		where = append(where, "risk_level = ?")
		args = append(args, riskLevel)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}

	whereClause := strings.Join(where, " AND ")

	// 计数
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM issue_forecasts WHERE %s`, whereClause)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计问题预案数量失败: %w", err)
	}

	// 查询列表
	query := fmt.Sprintf(
		`SELECT id, forecast_id, college_id, category, subcategory, title,
		        risk_level, status, affected_count, root_cause, suggested_actions,
		        data_summary, sources, ai_analysis, created_by, created_at,
		        updated_at, resolved_at, resolved_by
		 FROM issue_forecasts
		 WHERE %s
		 ORDER BY
		   CASE risk_level WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
		   created_at DESC
		 LIMIT ? OFFSET ?`, whereClause)

	args = append(args, pageSize, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询问题预案列表失败: %w", err)
	}
	defer rows.Close()

	var forecasts []*model.IssueForecast
	for rows.Next() {
		f := &model.IssueForecast{}
		if err := rows.Scan(
			&f.ID, &f.ForecastID, &f.CollegeID, &f.Category, &f.Subcategory,
			&f.Title, &f.RiskLevel, &f.Status, &f.AffectedCount, &f.RootCause,
			&f.SuggestedActions, &f.DataSummary, &f.Sources, &f.AIAnalysis,
			&f.CreatedBy, &f.CreatedAt, &f.UpdatedAt, &f.ResolvedAt, &f.ResolvedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描问题预案记录失败: %w", err)
		}
		forecasts = append(forecasts, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历问题预案列表失败: %w", err)
	}

	return forecasts, total, nil
}

// UpdateForecastStatus 更新问题预案状态
func (r *ForecastRepo) UpdateForecastStatus(forecastID string, status string, resolvedBy int64) error {
	query := `UPDATE issue_forecasts SET status = ?, updated_at = CURRENT_TIMESTAMP`
	args := []interface{}{status}

	if status == "resolved" {
		query += `, resolved_at = CURRENT_TIMESTAMP, resolved_by = ?`
		args = append(args, resolvedBy)
	}

	query += ` WHERE forecast_id = ?`
	args = append(args, forecastID)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新问题预案状态失败: %w", err)
	}
	return nil
}

// CreateIssueDetail 创建问题详情
func (r *ForecastRepo) CreateIssueDetail(detail *model.IssueDetail) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO issue_details
		 (forecast_id, user_id, user_type, username, display_name,
		  college, class_name, detail_type, detail_data, risk_score)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		detail.ForecastID, detail.UserID, detail.UserType, detail.Username,
		detail.DisplayName, detail.College, detail.ClassName,
		detail.DetailType, detail.DetailData, detail.RiskScore,
	)
	if err != nil {
		return 0, fmt.Errorf("创建问题详情失败: %w", err)
	}
	return result.LastInsertId()
}

// ListIssueDetails 查询问题详情列表
func (r *ForecastRepo) ListIssueDetails(forecastID string, detailType string, page int, pageSize int) ([]*model.IssueDetail, int, error) {
	offset := (page - 1) * pageSize

	where := []string{"forecast_id = ?"}
	args := []interface{}{forecastID}

	if detailType != "" {
		where = append(where, "detail_type = ?")
		args = append(args, detailType)
	}

	whereClause := strings.Join(where, " AND ")

	// 计数
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM issue_details WHERE %s`, whereClause)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计问题详情数量失败: %w", err)
	}

	// 查询列表
	query := fmt.Sprintf(
		`SELECT id, forecast_id, user_id, user_type, username, display_name,
		        college, class_name, detail_type, detail_data, risk_score, created_at
		 FROM issue_details
		 WHERE %s
		 ORDER BY risk_score DESC, created_at DESC
		 LIMIT ? OFFSET ?`, whereClause)

	args = append(args, pageSize, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询问题详情列表失败: %w", err)
	}
	defer rows.Close()

	var details []*model.IssueDetail
	for rows.Next() {
		d := &model.IssueDetail{}
		if err := rows.Scan(
			&d.ID, &d.ForecastID, &d.UserID, &d.UserType, &d.Username,
			&d.DisplayName, &d.College, &d.ClassName, &d.DetailType,
			&d.DetailData, &d.RiskScore, &d.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描问题详情记录失败: %w", err)
		}
		details = append(details, d)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历问题详情列表失败: %w", err)
	}

	return details, total, nil
}

// GetRiskDistribution 获取风险等级分布
func (r *ForecastRepo) GetRiskDistribution(collegeID string, days int) (map[string]int, error) {
	where := []string{"created_at >= datetime('now', ?)"}
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		where = append(where, "(college_id = ? OR college_id = '')")
		args = append(args, collegeID)
	}

	whereClause := strings.Join(where, " AND ")
	query := fmt.Sprintf(
		`SELECT risk_level, COUNT(*) as count
		 FROM issue_forecasts
		 WHERE %s
		 GROUP BY risk_level`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询风险分布失败: %w", err)
	}
	defer rows.Close()

	distribution := map[string]int{
		"low":    0,
		"medium": 0,
		"high":   0,
		"urgent": 0,
	}

	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("扫描风险分布失败: %w", err)
		}
		distribution[level] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历风险分布失败: %w", err)
	}

	return distribution, nil
}

// GetCategoryDistribution 获取问题分类分布
func (r *ForecastRepo) GetCategoryDistribution(collegeID string, days int) (map[string]int, error) {
	where := []string{"created_at >= datetime('now', ?)"}
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		where = append(where, "(college_id = ? OR college_id = '')")
		args = append(args, collegeID)
	}

	whereClause := strings.Join(where, " AND ")
	query := fmt.Sprintf(
		`SELECT category, COUNT(*) as count
		 FROM issue_forecasts
		 WHERE %s
		 GROUP BY category`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询分类分布失败: %w", err)
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("扫描分类分布失败: %w", err)
		}
		distribution[category] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分类分布失败: %w", err)
	}

	return distribution, nil
}

// GetDailyTrend 获取每日趋势
func (r *ForecastRepo) GetDailyTrend(collegeID string, days int) ([]map[string]interface{}, error) {
	where := []string{"created_at >= datetime('now', ?)"}
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		where = append(where, "(college_id = ? OR college_id = '')")
		args = append(args, collegeID)
	}

	whereClause := strings.Join(where, " AND ")
	query := fmt.Sprintf(
		`SELECT date(created_at) as date, COUNT(*) as count
		 FROM issue_forecasts
		 WHERE %s
		 GROUP BY date(created_at)
		 ORDER BY date ASC`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询每日趋势失败: %w", err)
	}
	defer rows.Close()

	var trend []map[string]interface{}
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, fmt.Errorf("扫描每日趋势失败: %w", err)
		}
		trend = append(trend, map[string]interface{}{
			"date":  date,
			"count": count,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历每日趋势失败: %w", err)
	}

	return trend, nil
}
