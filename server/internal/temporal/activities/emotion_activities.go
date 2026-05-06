package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
)

// EmotionActivities 情感分析相关活动集合
type EmotionActivities struct {
	EmotionRepo *repository.EmotionRepo
	LLMClient   llm.ChatClient // 复用接口
}

// EmotionAnalyzeInput 情感分析活动输入
type EmotionAnalyzeInput struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	SessionID   string `json:"session_id"`
	MessageText string `json:"message_text"`
}

// EmotionAnalyzeOutput 情感分析活动输出
type EmotionAnalyzeOutput struct {
	EmotionLogJSON string `json:"emotion_log_json"`
}

// EmotionAnalyzeActivity 执行情感分析并记录
func (a *EmotionActivities) EmotionAnalyzeActivity(ctx context.Context, input EmotionAnalyzeInput) (*EmotionAnalyzeOutput, error) {
	// 调用 LLM 分析情感（由 emotion_service 调用，此处简化：直接使用 llmClient）
	analysis, err := analyzeEmotionViaLLM(ctx, a.LLMClient, input.MessageText)
	if err != nil {
		// 分析失败时使用兜底
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
		UserID:       input.UserID,
		Username:     input.Username,
		SessionID:    input.SessionID,
		MessageText:  util.TruncateString(input.MessageText, 500),
		Score:        analysis.Score,
		RiskLevel:    analysis.RiskLevel,
		AnalysisJSON: string(analysisJSON),
		Notified:     boolToInt(analysis.RiskLevel == "high" || analysis.RiskLevel == "urgent"),
		Status:       "pending",
	}

	_, err = a.EmotionRepo.Create(logEntry)
	if err != nil {
		return nil, fmt.Errorf("保存情感记录失败: %w", err)
	}

	logJSON, _ := json.Marshal(logEntry)
	return &EmotionAnalyzeOutput{EmotionLogJSON: string(logJSON)}, nil
}

// ── 以下为 emotion_service 逻辑的本地副本（避免循环依赖 service 包）──

type emotionAnalysisResult struct {
	Score        float64  `json:"score"`
	RiskLevel    string   `json:"risk_level"`
	Emotions     []string `json:"emotions"`
	Keywords     []string `json:"keywords"`
	Reasoning    string   `json:"reasoning"`
	NeedFollowUp bool     `json:"need_follow_up"`
}

func analyzeEmotionViaLLM(ctx context.Context, client llm.ChatClient, text string) (*emotionAnalysisResult, error) {
	resp, err := client.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "system", Content: emotionSystemPrompt},
			{Role: "user", Content: "请分析以下学生消息的情感状态：\n\n" + text},
		},
		Temperature: 0.1,
		MaxTokens:   512,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 情感分析失败: %w", err)
	}

	return parseEmotionResponse(resp.Content)
}

func parseEmotionResponse(response string) (*emotionAnalysisResult, error) {
	jsonStr := response
	if idx := strings.Index(response, "{"); idx >= 0 {
		if endIdx := strings.LastIndex(response, "}"); endIdx > idx {
			jsonStr = response[idx : endIdx+1]
		}
	}

	var result emotionAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析情感分析结果失败: %w", err)
	}

	switch result.RiskLevel {
	case "low", "medium", "high", "urgent":
	default:
		result.RiskLevel = "low"
	}

	if result.Score < -1.0 {
		result.Score = -1.0
	}
	if result.Score > 1.0 {
		result.Score = 1.0
	}

	return &result, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

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
- medium: 有些焦虑或困扰，但无紧迫风险
- high: 明显的负面情绪，需要关注
- urgent: 紧急情况，需要立即干预

注意：不要过度敏感。重点关注自我伤害意图、严重绝望表述、极端社会孤立、突发剧烈情绪变化。`
