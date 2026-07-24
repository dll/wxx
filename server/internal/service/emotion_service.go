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
	"github.com/dll/wxx/server/internal/temporal"
	"github.com/dll/wxx/server/internal/temporal/workflows"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
	sdkclient "go.temporal.io/sdk/client"
)

// EmotionService 情感预警业务服务
type EmotionService struct {
	emotionRepo    *repository.EmotionRepo
	llmClient      llm.ChatClient
	temporalClient *temporal.Client // 可选：Temporal 工作流客户端
}

// NewEmotionService 创建情感预警服务
func NewEmotionService(emotionRepo *repository.EmotionRepo, llmClient llm.ChatClient) *EmotionService {
	return &EmotionService{
		emotionRepo: emotionRepo,
		llmClient:   llmClient,
	}
}

// SetTemporalClient 设置 Temporal 客户端（nil = 走直接调用路径）
func (s *EmotionService) SetTemporalClient(tc *temporal.Client) {
	s.temporalClient = tc
}

// emotionAnalysisResult LLM 情感分析结果结构
type emotionAnalysisResult struct {
	Score        float64  `json:"score"`          // -1.0 ~ 1.0
	RiskLevel    string   `json:"risk_level"`     // low / medium / high / urgent
	Emotions     []string `json:"emotions"`       // 检测到的情绪
	Keywords     []string `json:"keywords"`       // 高风险关键词
	Reasoning    string   `json:"reasoning"`      // 分析理由
	NeedFollowUp bool     `json:"need_follow_up"` // 是否需要跟进
}

// AnalyzeAndLog 分析文本情感并记录
// 当 Temporal 已配置时，通过工作流引擎执行（获得重试/可观测性）
func (s *EmotionService) AnalyzeAndLog(ctx context.Context, userID int64, username, sessionID, messageText string) (*model.EmotionLog, error) {
	// 如果 Temporal 已启用，走工作流
	if s.temporalClient != nil {
		return s.analyzeViaTemporal(ctx, userID, username, sessionID, messageText)
	}

	// ── 阶段 0：关键词预筛 ──
	if preResult := prefilterEmotion(messageText); preResult != nil && preResult.RiskLevel == "low" {
		// 明显中性/积极消息，跳过 LLM 调用，直接记录
		return s.saveEmotionLog(userID, username, sessionID, messageText, preResult)
	}

	// ── 阶段 1：LLM 情感分析 ──
	analysis, err := s.analyzeEmotion(ctx, messageText)
	if err != nil {
		log.Printf("情感分析失败: %v，使用兜底策略", err)
		analysis = &emotionAnalysisResult{
			Score:     0,
			RiskLevel: "low",
			Emotions:  []string{},
			Reasoning: "分析失败，使用兜底策略",
		}
	}

	// ── 阶段 2：连续高风险升级 ──
	if analysis.RiskLevel == "high" || analysis.RiskLevel == "urgent" {
		if escalated := s.checkEscalation(userID, analysis); escalated {
			analysis.RiskLevel = "urgent"
			analysis.Reasoning += "（连续高风险，自动升级为紧急）"
		}
	}

	return s.saveEmotionLog(userID, username, sessionID, messageText, analysis)
}

// saveEmotionLog 持久化情感记录
func (s *EmotionService) saveEmotionLog(userID int64, username, sessionID, messageText string, analysis *emotionAnalysisResult) (*model.EmotionLog, error) {
	analysisJSON, _ := json.Marshal(analysis)

	alertID := "alert-" + uuid.New().String()[:8]
	logEntry := &model.EmotionLog{
		AlertID:      alertID,
		UserID:       userID,
		Username:     username,
		SessionID:    sessionID,
		MessageText:  util.TruncateString(messageText, 500),
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

	if analysis.RiskLevel == "high" || analysis.RiskLevel == "urgent" {
		log.Printf("情感预警: user=%s risk=%s score=%.2f emotions=%v keywords=%v",
			username, analysis.RiskLevel, analysis.Score, analysis.Emotions, analysis.Keywords)
	}

	return logEntry, nil
}

// checkEscalation 检查用户最近是否有连续高风险记录，触发升级
func (s *EmotionService) checkEscalation(userID int64, current *emotionAnalysisResult) bool {
	recentAlerts, _, err := s.emotionRepo.ListAlerts("", "", "", "", "", 1, 5)
	if err != nil {
		return false
	}
	highCount := 0
	for _, alert := range recentAlerts {
		if alert.UserID == userID && (alert.RiskLevel == "high" || alert.RiskLevel == "urgent") {
			highCount++
		}
	}
	return highCount >= 3 // 含当前已是第 3 次及以上高风险
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

// ── 情感分析提示词 ──

const emotionSystemPrompt = `你是高校学生心理健康评估助手。分析学生消息中的情感状态，识别潜在的心理风险。

必须严格按以下 JSON 格式返回（不要返回其他内容）：
{
  "score": <float, -1.0到1.0>,
  "risk_level": "<low|medium|high|urgent>",
  "emotions": ["<情绪>"],
  "keywords": ["<关键词>"],
  "reasoning": "<简要分析，不超过一句话>",
  "need_follow_up": <bool>
}

风险等级判断标准：
- low: 正常交流（如"你好""怎么选课""奖学金什么时候发"）；轻度抱怨不算风险
- medium: 明显焦虑或困扰，但无紧迫风险（如"最近很焦虑""考研压力大""和室友有点矛盾"）
- high: 持续负面情绪需关注（如"失眠很久了""觉得生活无望""经常一个人哭"）
- urgent: 需立即干预（如明确自伤意图"不想活了""伤害自己"、暴力威胁、严重绝望"一切都完了"）

关键原则：
- 不要过度敏感。正常学业咨询（选课、考试、流程）即使表达轻微焦虑也评为 low
- 仅当出现自我伤害、伤害他人、严重绝望、极端孤立表述时评为 high/urgent
- 愤怒倾向（"想打人""报复"）评为 high
- 一次性的"好烦""郁闷"不构成风险，连续出现才值得关注`

// prefilterEmotion 关键词预筛：明显中性/积极的直接跳过 LLM，加速处理
func prefilterEmotion(text string) *emotionAnalysisResult {
	lower := strings.ToLower(text)

	// 积极/中性的高频短文本直接判定为 low
	neutralPatterns := []string{
		"你好", "谢谢", "好的", "收到", "明白了", "ok", "知道了",
		"怎么", "如何", "什么", "哪里", "什么时候", "请问",
		"选课", "考试时间", "成绩", "奖学金", "流程", "材料",
		"在吗", "hi", "hello", "帮我查", "我想问",
	}
	for _, p := range neutralPatterns {
		if strings.Contains(lower, p) && len([]rune(text)) <= 50 {
			return &emotionAnalysisResult{
				Score:     0.2,
				RiskLevel: "low",
				Emotions:  []string{"中立"},
				Reasoning: "信息咨询类消息，无情绪风险",
			}
		}
	}

	// 紧急关键词检测（快速标记高概率紧急）
	urgentKeywords := []string{
		"不想活", "自杀", "自残", "自伤", "去死", "死掉",
		"伤害自己", "结束生命", "活不下去", "没有意义",
		"想死", "杀", "割腕", "跳楼",
	}
	for _, kw := range urgentKeywords {
		if strings.Contains(lower, kw) {
			return &emotionAnalysisResult{
				Score:        -0.95,
				RiskLevel:    "urgent",
				Emotions:     []string{"绝望", "危机"},
				Keywords:     []string{kw},
				Reasoning:    "检测到自伤/自杀高风险关键词，立即评估为紧急",
				NeedFollowUp: true,
			}
		}
	}

	// 不确定的情况返回 nil，走 LLM 分析
	return nil
}

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
		return nil, fmt.Errorf("解析情感分析结果失败: %w, response: %s", err, util.TruncateString(response, 200))
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// analyzeViaTemporal 通过 Temporal 工作流引擎执行情感分析
func (s *EmotionService) analyzeViaTemporal(ctx context.Context, userID int64, username, sessionID, messageText string) (*model.EmotionLog, error) {
	workflowOpts := sdkclient.StartWorkflowOptions{
		ID:                       "emotion-" + uuid.New().String()[:8],
		TaskQueue:                s.temporalClient.TaskQueue(),
		WorkflowExecutionTimeout: 60 * time.Second,
	}

	input := workflows.EmotionAnalyzeInput{
		UserID:      userID,
		Username:    username,
		SessionID:   sessionID,
		MessageText: messageText,
	}

	run, err := s.temporalClient.SDKClient().ExecuteWorkflow(ctx, workflowOpts, workflows.EmotionAnalyzeWorkflow, input)
	if err != nil {
		log.Printf("启动情感分析工作流失败: %v，使用直接调用", err)
		return s.analyzeAndLogDirect(ctx, userID, username, sessionID, messageText)
	}

	var output workflows.EmotionAnalyzeOutput
	err = run.Get(ctx, &output)
	if err != nil {
		log.Printf("情感分析工作流执行失败: %v，使用直接调用", err)
		return s.analyzeAndLogDirect(ctx, userID, username, sessionID, messageText)
	}

	var logEntry model.EmotionLog
	if err := json.Unmarshal([]byte(output.EmotionLogJSON), &logEntry); err != nil {
		return nil, fmt.Errorf("反序列化情感记录失败: %w", err)
	}

	return &logEntry, nil
}

// analyzeAndLogDirect 情感分析直接调用（Temporal 失败时的降级路径）
func (s *EmotionService) analyzeAndLogDirect(ctx context.Context, userID int64, username, sessionID, messageText string) (*model.EmotionLog, error) {
	analysis, err := s.analyzeEmotion(ctx, messageText)
	if err != nil {
		log.Printf("情感分析失败: %v，使用兜底策略", err)
		analysis = &emotionAnalysisResult{
			Score:     0,
			RiskLevel: "low",
			Emotions:  []string{},
			Reasoning: "分析失败，使用兜底策略",
		}
	}

	analysisJSON, _ := json.Marshal(analysis)

	alertID := "alert-" + uuid.New().String()[:8]
	logEntry := &model.EmotionLog{
		AlertID:      alertID,
		UserID:       userID,
		Username:     username,
		SessionID:    sessionID,
		MessageText:  util.TruncateString(messageText, 500),
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

	if analysis.RiskLevel == "high" || analysis.RiskLevel == "urgent" {
		log.Printf("情感预警: user=%s risk=%s score=%.2f emotions=%v keywords=%v",
			username, analysis.RiskLevel, analysis.Score, analysis.Emotions, analysis.Keywords)
	}

	return logEntry, nil
}
