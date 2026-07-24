package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
)

// SysAdminService 系统管理员角色 AI 功能服务
type SysAdminService struct {
	llmClient llm.ChatClient
}

func NewSysAdminService(llmClient llm.ChatClient) *SysAdminService {
	return &SysAdminService{llmClient: llmClient}
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	Status      string                   `json:"status"`
	Uptime      string                   `json:"uptime"`
	CPUUsage    float64                  `json:"cpu_usage"`
	MemoryUsage float64                  `json:"memory_usage"`
	DiskUsage   float64                  `json:"disk_usage"`
	ActiveUsers int                      `json:"active_users"`
	APILatency  float64                  `json:"api_latency_ms"`
	Alerts      []map[string]interface{} `json:"alerts"`
	AIDiagnosis string                   `json:"ai_diagnosis"`
	DataSource  string                   `json:"data_source"`
}

func (s *SysAdminService) GetSystemHealth(ctx context.Context) *SystemHealth {
	health := &SystemHealth{
		Status:      "healthy",
		Uptime:      "72h 35m",
		CPUUsage:    0.35,
		MemoryUsage: 0.62,
		DiskUsage:   0.45,
		ActiveUsers: 128,
		APILatency:  85.5,
		Alerts: []map[string]interface{}{
			{"type": "warning", "message": "磁盘空间预计7天后达到80%", "time": time.Now().Format("15:04")},
			{"type": "info", "message": "LLM API调用量较昨日增长15%", "time": time.Now().Format("15:04")},
		},
		DataSource: "mock",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是运维专家。CPU%.0f%%，内存%.0f%%，API延迟%.0fms。请用30字诊断系统健康状态。",
			health.CPUUsage*100, health.MemoryUsage*100, health.APILatency)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.2, MaxTokens: 150,
		})
		if err == nil && resp != nil && resp.Content != "" {
			health.AIDiagnosis = strings.TrimSpace(resp.Content)
			health.DataSource = "ai"
		}
	}

	return health
}

// KnowledgeQuality 知识质量评估
type KnowledgeQuality struct {
	TotalResources int                      `json:"total_resources"`
	Coverage       float64                  `json:"coverage"`
	Accuracy       float64                  `json:"accuracy"`
	Freshness      float64                  `json:"freshness"`
	Redundancy     float64                  `json:"redundancy"`
	Issues         []map[string]interface{} `json:"issues"`
	Suggestions    []string                 `json:"suggestions"`
	DataSource     string                   `json:"data_source"`
}

func (s *SysAdminService) EvaluateKnowledgeQuality(ctx context.Context) *KnowledgeQuality {
	return &KnowledgeQuality{
		TotalResources: 1250,
		Coverage:       0.82,
		Accuracy:       0.91,
		Freshness:      0.78,
		Redundancy:     0.12,
		Issues: []map[string]interface{}{
			{"type": "过时", "resource": "2024年奖学金评定标准", "detail": "已被2025年新标准替代，建议下线", "severity": "high"},
			{"type": "冗余", "resource": "转专业流程说明v1/v2", "detail": "存在2个版本，建议合并保留最新版", "severity": "medium"},
			{"type": "缺失", "resource": "国际交流项目申请", "detail": "学生查询量高但无对应知识条目", "severity": "medium"},
		},
		Suggestions: []string{
			"建议更新：奖学金政策(3条)、入党流程(2条)",
			"建议合并：转专业FAQ重复条目",
			"建议补充：国际交流项目、创新创业学分",
		},
		DataSource: "mock",
	}
}

// UserBehaviorAnalysis 用户行为分析
type UserBehaviorAnalysis struct {
	Period        string                   `json:"period"`
	ActiveUsers   int                      `json:"active_users"`
	DAU           int                      `json:"dau"`
	RetentionRate float64                  `json:"retention_rate"`
	TopFeatures   []map[string]interface{} `json:"top_features"`
	UserJourneys  []string                 `json:"user_journeys"`
	Suggestions   []string                 `json:"suggestions"`
	DataSource    string                   `json:"data_source"`
}

func (s *SysAdminService) AnalyzeUserBehavior(ctx context.Context) *UserBehaviorAnalysis {
	return &UserBehaviorAnalysis{
		Period:        "2026年5月",
		ActiveUsers:   3200,
		DAU:           450,
		RetentionRate: 0.68,
		TopFeatures: []map[string]interface{}{
			{"feature": "智能问答", "usage": 2800, "growth": "+15%", "satisfaction": 4.2},
			{"feature": "今日速览", "usage": 1800, "growth": "+22%", "satisfaction": 4.5},
			{"feature": "办事流程", "usage": 950, "growth": "+8%", "satisfaction": 3.8},
			{"feature": "AI备课助手", "usage": 420, "growth": "+35%", "satisfaction": 4.6},
		},
		UserJourneys: []string{
			"高频路径: 登录→今日速览→智能问答→课程查询",
			"新用户路径: 登录→办事流程→智能问答",
			"教师路径: 登录→备课助手→学情热力图→课堂互动",
		},
		Suggestions: []string{
			"智能问答是核心功能，建议持续优化知识库质量",
			"AI备课助手增长最快(35%)，建议重点推广给更多教师",
			"办事流程满意度偏低(3.8)，需优化交互体验",
		},
		DataSource: "mock",
	}
}
