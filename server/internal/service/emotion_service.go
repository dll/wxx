package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// EmotionService 情感预警业务服务
type EmotionService struct {
	emotionRepo *repository.EmotionRepo
	llmClient   llm.ChatClient
}

// NewEmotionService 创建情感预警服务
func NewEmotionService(emotionRepo *repository.EmotionRepo, llmClient llm.ChatClient) *EmotionService {
	return &EmotionService{
		emotionRepo: emotionRepo,
		llmClient:   llmClient,
	}
}

// emotionAnalysisResult LLM 情感分析结果结构
type emotionAnalysisResult struct {
	Score       float64  `json:"score"`        // -1.0 ~ 1.0
	RiskLevel   string   `json:"risk_level"`   // low / medium / high / urgent
	Emotions    []string `json:"emotions"`     // 检测到的情绪
	Keywords    []string `json:"keywords"`     // 高风险关键词
	Reasoning   string   `json:"reasoning"`    // 分析理由
	NeedFollowUp bool   `json:"need_follow_up"` // 是否需要跟进
}

// AnalyzeAndLog 分析文本情感并记录
func (s *EmotionService) AnalyzeAndLog(ctx context.Context, userID int64, username, sessionID, messageText string) (*model.EmotionLog, error) {
	// 调用 LLM 进行情感分析
	analysis, err := s.analyzeEmotion(ctx, messageText)
	if err != nil {
		// 分析失败时记录低风险兜底
		log.Printf("情感分析失败: %v，使用兜底策略", err)
		analysis = &emotionAnalysisResult{
			Score:     0,
			RiskLevel: "low",
			Emotions:  []string{},
			Reasoning: "分析失败，使用兜底策略",
		}
	}

	// 序列化完整分析结果
	analysisJSON, _ := json.Marshal(analysis)

	// 创建记录
	alertID := "alert-" + uuid.New().String()[:8]
	logEntry := &model.EmotionLog{
		AlertID:      alertID,
		UserID:       userID,
		Username:     username,
		SessionID:    sessionID,
		MessageText:  truncateText(messageText, 500),
		Score:        analysis.Score,
		RiskLevel:    analysis.RiskLevel,
		AnalysisJSON: string(analysisJSON),
		Notified:     boolToInt(analysis.RiskLevel == "high" || analysis.RiskLevel == "urgent"),
		Status:       "pending",
	}

	id, err := s.emotionRepo.Create(logEntry)
	if err != nil {
		return nil, fmt.Errorf("保存情感记录失败: %w", err)
	}

	logEntry.ID = id

	// 高风险时输出预警日志
	if analysis.RiskLevel == "high" || analysis.RiskLevel == "urgent" {
		log.Printf("情感预警: user=%s risk=%s score=%.2f emotions=%v keywords=%v",
			username, analysis.RiskLevel, analysis.Score, analysis.Emotions, analysis.Keywords)
	}

	return logEntry, nil
}

// analyzeEmotion 调用 LLM 分析文本情感
func (s *EmotionService) analyzeEmotion(ctx context.Context, text string) (*emotionAnalysisResult, error) {
	prompt := buildEmotionPrompt(text)

	// 使用 LLM API 分析（超时 10s）
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	response, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "system", Content: emotionSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1, // 情感分析用低温度，保持一致性
		MaxTokens:   512,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 情感分析失败: %w", err)
	}

	return parseEmotionResponse(response.Content)
}

// ListAlerts 分页查询告警
func (s *EmotionService) ListAlerts(riskLevel, status, ownerScope, ownerID, role string, page, pageSize int) ([]*model.EmotionLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.emotionRepo.ListAlerts(riskLevel, status, ownerScope, ownerID, role, page, pageSize)
}

// GetStats 获取告警统计（按角色过滤范围）
func (s *EmotionService) GetStats(ownerScope, ownerID, role string) (*model.EmotionStats, error) {
	return s.emotionRepo.GetStats(ownerScope, ownerID, role)
}

// GetTrendReport 获取情感趋势报告（按天聚合，含范围过滤）
func (s *EmotionService) GetTrendReport(days int, ownerScope, ownerID, role string) (*model.EmotionTrendReport, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	points, err := s.emotionRepo.GetTrends(days, ownerScope, ownerID, role)
	if err != nil {
		return nil, fmt.Errorf("获取趋势数据失败: %w", err)
	}

	// 计算汇总指标
	report := &model.EmotionTrendReport{
		Days:   days,
		Points: points,
	}
	for _, p := range points {
		report.Summary.TotalAnalyses += p.Total
		report.Summary.TotalUrgent += p.Urgent
		report.Summary.TotalHigh += p.High
	}
	if len(points) > 0 {
		report.Summary.AvgDaily = report.Summary.TotalAnalyses / len(points)
	}

	return report, nil
}

// UpdateAlertStatus 更新告警状态（确认/已处理）
func (s *EmotionService) UpdateAlertStatus(alertID, status, acknowledgedBy string) (*model.EmotionLog, error) {
	if err := s.emotionRepo.UpdateStatus(alertID, status, acknowledgedBy); err != nil {
		return nil, err
	}

	logEntry, err := s.emotionRepo.GetByAlertID(alertID)
	if err != nil {
		return nil, err
	}
	if logEntry == nil {
		return nil, fmt.Errorf("告警不存在: %s", alertID)
	}

	log.Printf("告警状态已更新 alert_id=%s status=%s by=%s", alertID, status, acknowledgedBy)
	return logEntry, nil
}

// ── 情感分析提示词 ──

const emotionSystemPrompt = `你是高校学生心理健康评估助手。你的任务是分析学生消息中的情感状态，识别潜在的心理风险。

你必须严格按以下 JSON 格式返回分析结果（不要返回其他内容）：
{
  "score": <float, -1.0到1.0, -1=极度消极, 0=中性, 1=积极>,
  "risk_level": "<low|medium|high|urgent>",
  "emotions": ["<检测到的情绪>"],
  "keywords": ["<高风险关键词>"],
  "reasoning": "<简要分析理由>",
  "need_follow_up": <bool>
}

风险等级判断标准：
- low: 正常交流，无明显负面情绪
- medium: 有些焦虑或困扰，但无紧迫风险（如学业压力、人际困扰）
- high: 明显的负面情绪，需要关注（如持续的焦虑、沮丧、愤怒、无助感）
- urgent: 紧急情况，需要立即干预（如明确的自伤/伤人意图、严重绝望表述）

注意：不要过度敏感。学生对学业、考试、生活的正常抱怨不构成高风险。重点关注：
1. 自我伤害或伤害他人的意图
2. 严重的绝望、无助感
3. 极端的社会孤立表述
4. 突发的剧烈情绪变化`

func buildEmotionPrompt(text string) string {
	return fmt.Sprintf("请分析以下学生消息的情感状态：\n\n%s", text)
}

func parseEmotionResponse(response string) (*emotionAnalysisResult, error) {
	// 提取 JSON 部分（LLM 可能会包裹在 markdown 代码块中）
	jsonStr := response
	if idx := strings.Index(response, "{"); idx >= 0 {
		if endIdx := strings.LastIndex(response, "}"); endIdx > idx {
			jsonStr = response[idx : endIdx+1]
		}
	}

	var result emotionAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析情感分析结果失败: %w, response: %s", err, truncateText(response, 200))
	}

	// 校验 risk_level
	switch result.RiskLevel {
	case "low", "medium", "high", "urgent":
	default:
		result.RiskLevel = "low"
	}

	// 校验 score 范围
	if result.Score < -1.0 {
		result.Score = -1.0
	}
	if result.Score > 1.0 {
		result.Score = 1.0
	}

	return &result, nil
}

// ── 工具函数 ──

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
