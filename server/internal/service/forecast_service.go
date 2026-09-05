package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// ForecastService 问题预案业务服务
type ForecastService struct {
	forecastRepo *repository.ForecastRepo
	emotionRepo  *repository.EmotionRepo
	feedbackRepo *repository.FeedbackRepo
	llmClient    llm.ChatClient
	db           *sql.DB
}

// NewForecastService 创建问题预案服务
func NewForecastService(
	db *sql.DB,
	forecastRepo *repository.ForecastRepo,
	emotionRepo *repository.EmotionRepo,
	feedbackRepo *repository.FeedbackRepo,
	llmClient llm.ChatClient,
) *ForecastService {
	return &ForecastService{
		forecastRepo: forecastRepo,
		emotionRepo:  emotionRepo,
		feedbackRepo: feedbackRepo,
		llmClient:    llmClient,
		db:           db,
	}
}

// AnalysisRequest 分析请求
type AnalysisRequest struct {
	CollegeID    string   `json:"college_id"`
	TimeRange    string   `json:"time_range"`    // last_7_days, last_30_days, last_90_days
	AnalysisType string   `json:"analysis_type"` // comprehensive, emotion, academic, attendance, complaint, discipline
	DataSources  []string `json:"data_sources"`  // 可选，指定数据源
}

// AnalysisResponse 分析响应
type AnalysisResponse struct {
	Summary   *model.ForecastSummary `json:"summary"`
	Issues    []*model.IssueForecast `json:"issues"`
	ReportURL string                 `json:"report_url,omitempty"`
}

// Analyze 执行问题分析
func (s *ForecastService) Analyze(req *AnalysisRequest, operatorID int64, operatorRole string) (*AnalysisResponse, error) {
	log.Printf("开始问题预案分析: college=%s, time=%s, type=%s", req.CollegeID, req.TimeRange, req.AnalysisType)

	// 1. 确定时间范围
	days := s.parseTimeRange(req.TimeRange)

	// 2. 聚合多源数据
	dataSummary, err := s.aggregateData(req.CollegeID, days, req.DataSources)
	if err != nil {
		return nil, fmt.Errorf("数据聚合失败: %w", err)
	}

	// 3. AI 分析生成问题
	issues, err := s.analyzeWithAI(dataSummary, req.CollegeID, operatorID)
	if err != nil {
		log.Printf("AI分析失败，使用规则引擎: %v", err)
		issues = s.analyzeWithRules(dataSummary, req.CollegeID, operatorID)
	}

	// 4. 保存分析结果
	for _, issue := range issues {
		if _, err := s.forecastRepo.CreateForecast(issue); err != nil {
			log.Printf("保存问题预案失败: %v", err)
		}
	}

	// 5. 统计摘要
	summary := s.buildSummary(issues, dataSummary, days)

	return &AnalysisResponse{
		Summary: summary,
		Issues:  issues,
	}, nil
}

// aggregateData 聚合多源数据
func (s *ForecastService) aggregateData(collegeID string, days int, dataSources []string) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// 1. 情感预警数据
	if s.containsSource(dataSources, "emotion") || len(dataSources) == 0 {
		emotionData, err := s.aggregateEmotionData(collegeID, days)
		if err != nil {
			log.Printf("聚合情感数据失败: %v", err)
		} else {
			summary["emotion"] = emotionData
		}
	}

	// 2. 反馈投诉数据
	if s.containsSource(dataSources, "feedback") || len(dataSources) == 0 {
		feedbackData, err := s.aggregateFeedbackData(collegeID, days)
		if err != nil {
			log.Printf("聚合反馈数据失败: %v", err)
		} else {
			summary["feedback"] = feedbackData
		}
	}

	// 3. 办事流程数据
	if s.containsSource(dataSources, "process") || len(dataSources) == 0 {
		processData, err := s.aggregateProcessData(collegeID, days)
		if err != nil {
			log.Printf("聚合流程数据失败: %v", err)
		} else {
			summary["process"] = processData
		}
	}

	// 4. 对话记录数据
	if s.containsSource(dataSources, "chat") || len(dataSources) == 0 {
		chatData, err := s.aggregateChatData(collegeID, days)
		if err != nil {
			log.Printf("聚合对话数据失败: %v", err)
		} else {
			summary["chat"] = chatData
		}
	}

	return summary, nil
}

// aggregateEmotionData 聚合情感预警数据
func (s *ForecastService) aggregateEmotionData(collegeID string, days int) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// 查询风险分布
	distribution, err := s.forecastRepo.GetRiskDistribution(collegeID, days)
	if err != nil {
		return nil, err
	}
	data["risk_distribution"] = distribution

	// 查询总数
	var total int
	query := `SELECT COUNT(*) FROM emotion_logs WHERE created_at >= datetime('now', ?)`
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
		return nil, err
	}
	data["total"] = total

	// 查询高风险记录
	var highRiskCount int
	query = `SELECT COUNT(*) FROM emotion_logs WHERE risk_level IN ('high', 'urgent') AND created_at >= datetime('now', ?)`
	args = []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&highRiskCount); err != nil {
		return nil, err
	}
	data["high_risk_count"] = highRiskCount

	return data, nil
}

// aggregateFeedbackData 聚合反馈投诉数据
func (s *ForecastService) aggregateFeedbackData(collegeID string, days int) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// 查询反馈总数
	var total int
	query := `SELECT COUNT(*) FROM feedback WHERE created_at >= datetime('now', ?)`
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
		return nil, err
	}
	data["total"] = total

	// 查询待处理反馈
	var pendingCount int
	query = `SELECT COUNT(*) FROM feedback WHERE status = 'pending' AND created_at >= datetime('now', ?)`
	args = []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&pendingCount); err != nil {
		return nil, err
	}
	data["pending_count"] = pendingCount

	return data, nil
}

// aggregateProcessData 聚合办事流程数据
func (s *ForecastService) aggregateProcessData(collegeID string, days int) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// 查询流程记录总数
	var total int
	query := `SELECT COUNT(*) FROM process_records WHERE created_at >= datetime('now', ?)`
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
		return nil, err
	}
	data["total"] = total

	// 查询失败流程
	var failedCount int
	query = `SELECT COUNT(*) FROM process_records WHERE status = 'abandoned' AND created_at >= datetime('now', ?)`
	args = []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&failedCount); err != nil {
		return nil, err
	}
	data["failed_count"] = failedCount

	return data, nil
}

// aggregateChatData 聚合对话记录数据
func (s *ForecastService) aggregateChatData(collegeID string, days int) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// 查询会话总数
	var total int
	query := `SELECT COUNT(*) FROM sessions WHERE created_at >= datetime('now', ?)`
	args := []interface{}{fmt.Sprintf("-%d days", days)}

	if collegeID != "" {
		query += ` AND user_id IN (SELECT id FROM users WHERE owner_scope = ?)`
		args = append(args, collegeID)
	}

	if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
		return nil, err
	}
	data["total_sessions"] = total

	// 查询常见问题类型（通过用户消息内容中的稳定业务词频统计）。
	// 仅统计受控词表，避免把姓名、学号等任意文本泄露到分析结果中。
	query = `SELECT m.content FROM messages m
		JOIN sessions s ON s.session_id = m.session_id
		JOIN users u ON u.id = s.user_id
		WHERE m.role = 'user' AND m.created_at >= datetime('now', ?)`
	args = []interface{}{fmt.Sprintf("-%d days", days)}
	if collegeID != "" {
		query += ` AND u.owner_scope = ?`
		args = append(args, collegeID)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var counts = make(map[string]int)
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		for _, topic := range forecastTopicKeywords {
			if strings.Contains(content, topic) {
				counts[topic]++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	data["common_topics"] = rankForecastTopics(counts, 10)

	return data, nil
}

var forecastTopicKeywords = []string{
	"奖学金", "选课", "课程", "成绩", "考试", "毕业", "请假", "宿舍", "竞赛",
	"入党", "心理", "就业", "实习", "转专业", "报到", "课表", "绩点", "社团",
}

type forecastTopicCount struct {
	Topic string
	Count int
}

func rankForecastTopics(counts map[string]int, limit int) []string {
	items := make([]forecastTopicCount, 0, len(counts))
	for topic, count := range counts {
		if count > 0 {
			items = append(items, forecastTopicCount{Topic: topic, Count: count})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Topic < items[j].Topic
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	topics := make([]string, 0, len(items))
	for _, item := range items {
		topics = append(topics, item.Topic)
	}
	return topics
}

// analyzeWithAI 使用AI分析数据生成问题
func (s *ForecastService) analyzeWithAI(dataSummary map[string]interface{}, collegeID string, operatorID int64) ([]*model.IssueForecast, error) {
	if s.llmClient == nil {
		return nil, fmt.Errorf("LLM客户端未配置")
	}

	// 构建分析提示词
	prompt := s.buildAnalysisPrompt(dataSummary, collegeID)

	// 调用LLM
	response, err := s.llmClient.Chat(context.Background(), &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "system", Content: "你是一个教育问题分析专家，负责分析学校数据并识别潜在问题。请根据提供的数据，识别问题并给出原因分析和解决预案。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析LLM响应
	issues, err := s.parseAIResponse(response.Content, collegeID, operatorID)
	if err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	return issues, nil
}

// buildAnalysisPrompt 构建分析提示词
func (s *ForecastService) buildAnalysisPrompt(dataSummary map[string]interface{}, collegeID string) string {
	dataJSON, _ := json.MarshalIndent(dataSummary, "", "  ")

	prompt := fmt.Sprintf(`请分析以下学校数据，识别潜在问题并给出解决预案：

数据摘要：
%s

学院：%s

请按以下格式返回分析结果（JSON数组）：
[
  {
    "category": "问题分类（academic/emotion/attendance/complaint/process/discipline）",
    "subcategory": "子分类",
    "title": "问题标题",
    "risk_level": "风险等级（low/medium/high/urgent）",
    "affected_count": "影响人数",
    "root_cause": "原因分析",
    "suggested_actions": ["建议措施1", "建议措施2"],
    "sources": ["数据来源1", "数据来源2"]
  }
]

要求：
1. 基于数据事实分析，不要编造
2. 每个问题给出具体的原因和可执行的建议
3. 风险等级根据影响范围和紧急程度判断
4. 只返回JSON数组，不要其他内容`, string(dataJSON), collegeID)

	return prompt
}

// parseAIResponse 解析AI响应
func (s *ForecastService) parseAIResponse(response string, collegeID string, operatorID int64) ([]*model.IssueForecast, error) {
	// 提取JSON部分
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("无法解析AI响应")
	}
	jsonStr := response[start : end+1]

	// 解析JSON
	var aiIssues []struct {
		Category         string   `json:"category"`
		Subcategory      string   `json:"subcategory"`
		Title            string   `json:"title"`
		RiskLevel        string   `json:"risk_level"`
		AffectedCount    int      `json:"affected_count"`
		RootCause        string   `json:"root_cause"`
		SuggestedActions []string `json:"suggested_actions"`
		Sources          []string `json:"sources"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &aiIssues); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 转换为IssueForecast
	var issues []*model.IssueForecast
	for _, ai := range aiIssues {
		actionsJSON, _ := json.Marshal(ai.SuggestedActions)
		sourcesJSON, _ := json.Marshal(ai.Sources)

		issue := &model.IssueForecast{
			ForecastID:       "forecast-" + uuid.New().String()[:8],
			CollegeID:        collegeID,
			Category:         ai.Category,
			Subcategory:      ai.Subcategory,
			Title:            ai.Title,
			RiskLevel:        ai.RiskLevel,
			Status:           "pending",
			AffectedCount:    ai.AffectedCount,
			RootCause:        ai.RootCause,
			SuggestedActions: string(actionsJSON),
			Sources:          string(sourcesJSON),
			AIAnalysis:       "AI生成",
			CreatedBy:        &operatorID,
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// analyzeWithRules 使用规则引擎分析数据
func (s *ForecastService) analyzeWithRules(dataSummary map[string]interface{}, collegeID string, operatorID int64) []*model.IssueForecast {
	var issues []*model.IssueForecast

	// 分析情感数据
	if emotionData, ok := dataSummary["emotion"].(map[string]interface{}); ok {
		if highRisk, ok := emotionData["high_risk_count"].(int); ok && highRisk > 5 {
			issue := &model.IssueForecast{
				ForecastID:       "forecast-" + uuid.New().String()[:8],
				CollegeID:        collegeID,
				Category:         "emotion",
				Subcategory:      "high_risk_alerts",
				Title:            fmt.Sprintf("近期高风险情感预警较多（%d条）", highRisk),
				RiskLevel:        "high",
				Status:           "pending",
				AffectedCount:    highRisk,
				RootCause:        "可能存在学生心理问题集中爆发的情况",
				SuggestedActions: `["加强心理辅导资源投入","开展心理健康普查","组织心理健康主题活动"]`,
				Sources:          `["情感预警系统"]`,
				CreatedBy:        &operatorID,
			}
			issues = append(issues, issue)
		}
	}

	// 分析反馈数据
	if feedbackData, ok := dataSummary["feedback"].(map[string]interface{}); ok {
		if pendingCount, ok := feedbackData["pending_count"].(int); ok && pendingCount > 10 {
			issue := &model.IssueForecast{
				ForecastID:       "forecast-" + uuid.New().String()[:8],
				CollegeID:        collegeID,
				Category:         "complaint",
				Subcategory:      "pending_feedback",
				Title:            fmt.Sprintf("待处理反馈积压（%d条）", pendingCount),
				RiskLevel:        "medium",
				Status:           "pending",
				AffectedCount:    pendingCount,
				RootCause:        "反馈处理机制可能需要优化",
				SuggestedActions: `["加快反馈处理流程","增加反馈处理人员","优化反馈分类机制"]`,
				Sources:          `["反馈投诉系统"]`,
				CreatedBy:        &operatorID,
			}
			issues = append(issues, issue)
		}
	}

	// 分析流程数据
	if processData, ok := dataSummary["process"].(map[string]interface{}); ok {
		if failedCount, ok := processData["failed_count"].(int); ok && failedCount > 3 {
			issue := &model.IssueForecast{
				ForecastID:       "forecast-" + uuid.New().String()[:8],
				CollegeID:        collegeID,
				Category:         "process",
				Subcategory:      "failed_processes",
				Title:            fmt.Sprintf("办事流程失败率偏高（%d条）", failedCount),
				RiskLevel:        "medium",
				Status:           "pending",
				AffectedCount:    failedCount,
				RootCause:        "可能存在流程设计不合理或指引不清晰的情况",
				SuggestedActions: `["优化办事流程设计","完善流程指引文档","加强流程咨询服务"]`,
				Sources:          `["办事流程系统"]`,
				CreatedBy:        &operatorID,
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// buildSummary 构建统计摘要
func (s *ForecastService) buildSummary(issues []*model.IssueForecast, dataSummary map[string]interface{}, days int) *model.ForecastSummary {
	summary := &model.ForecastSummary{
		TotalIssues:       len(issues),
		RiskDistribution:  make(map[string]int),
		CategoryDistribut: make(map[string]int),
		KeyFindings:       []string{},
	}

	// 统计风险分布
	for _, issue := range issues {
		summary.RiskDistribution[issue.RiskLevel]++
		summary.CategoryDistribut[issue.Category]++
	}

	// 判断趋势（简化：基于问题数量）
	if summary.TotalIssues > 5 {
		summary.Trend = "increasing"
	} else if summary.TotalIssues > 0 {
		summary.Trend = "stable"
	} else {
		summary.Trend = "decreasing"
	}

	// 生成关键发现
	for _, issue := range issues {
		if issue.RiskLevel == "high" || issue.RiskLevel == "urgent" {
			summary.KeyFindings = append(summary.KeyFindings, issue.Title)
		}
	}

	return summary
}

// parseTimeRange 解析时间范围
func (s *ForecastService) parseTimeRange(timeRange string) int {
	switch timeRange {
	case "last_7_days":
		return 7
	case "last_30_days":
		return 30
	case "last_90_days":
		return 90
	default:
		return 30
	}
}

// containsSource 检查是否包含指定数据源
func (s *ForecastService) containsSource(sources []string, target string) bool {
	for _, src := range sources {
		if src == target {
			return true
		}
	}
	return false
}

// GetForecast 获取问题预案详情
func (s *ForecastService) GetForecast(forecastID string) (*model.IssueForecast, error) {
	return s.forecastRepo.GetForecast(forecastID)
}

// ListForecasts 分页查询问题预案
func (s *ForecastService) ListForecasts(collegeID string, category string, riskLevel string, status string, page int, pageSize int) ([]*model.IssueForecast, int, error) {
	return s.forecastRepo.ListForecasts(collegeID, category, riskLevel, status, page, pageSize)
}

// UpdateStatus 更新问题预案状态
func (s *ForecastService) UpdateStatus(forecastID string, status string, operatorID int64) error {
	return s.forecastRepo.UpdateForecastStatus(forecastID, status, operatorID)
}

// GetRiskDistribution 获取风险等级分布
func (s *ForecastService) GetRiskDistribution(collegeID string, days int) (map[string]int, error) {
	return s.forecastRepo.GetRiskDistribution(collegeID, days)
}

// GetCategoryDistribution 获取问题分类分布
func (s *ForecastService) GetCategoryDistribution(collegeID string, days int) (map[string]int, error) {
	return s.forecastRepo.GetCategoryDistribution(collegeID, days)
}

// GetDailyTrend 获取每日趋势
func (s *ForecastService) GetDailyTrend(collegeID string, days int) ([]map[string]interface{}, error) {
	return s.forecastRepo.GetDailyTrend(collegeID, days)
}
