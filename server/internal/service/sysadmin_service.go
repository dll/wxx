package service

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
)

// processStart 进程启动时刻，用于计算真实运行时长
var processStart = time.Now()

// SysAdminService 系统管理员角色 AI 功能服务
type SysAdminService struct {
	llmClient llm.ChatClient
	userRepo  *repository.UserRepo
	kbRepo    *repository.KBRepo
	auditRepo *repository.AuditRepo
}

func NewSysAdminService(llmClient llm.ChatClient, userRepo *repository.UserRepo, kbRepo *repository.KBRepo, auditRepo *repository.AuditRepo) *SysAdminService {
	return &SysAdminService{llmClient: llmClient, userRepo: userRepo, kbRepo: kbRepo, auditRepo: auditRepo}
}

// formatUptime 把时长格式化为「Xh Ym」
func formatUptime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	Status      string                   `json:"status"`
	Uptime      string                   `json:"uptime"`
	CPUUsage    float64                  `json:"cpu_usage"`
	MemoryUsage float64                  `json:"memory_usage"`
	DiskUsage   float64                  `json:"disk_usage"`
	TotalUsers  int                      `json:"total_users"`
	ActiveUsers int                      `json:"active_users"`
	APILatency  float64                  `json:"api_latency_ms"`
	Alerts      []map[string]interface{} `json:"alerts"`
	AIDiagnosis string                   `json:"ai_diagnosis"`
	DataSource  string                   `json:"data_source"`
}

func (s *SysAdminService) GetSystemHealth(ctx context.Context) *SystemHealth {
	// 真实运行时指标：内存来自 Go runtime，运行时长来自进程启动时刻
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	// 以 Sys（向 OS 申请的总内存）为分母估算堆占用比例，避免臆造宿主机指标
	memUsage := 0.0
	if mem.Sys > 0 {
		memUsage = float64(mem.HeapAlloc) / float64(mem.Sys)
	}

	health := &SystemHealth{
		Status:      "healthy",
		Uptime:      formatUptime(time.Since(processStart)),
		MemoryUsage: memUsage,
		Alerts:      []map[string]interface{}{},
		DataSource:  "real",
	}

	// 真实在册用户数
	if s.userRepo != nil {
		if n, err := s.userRepo.Count("", "", ""); err == nil {
			health.TotalUsers = n
		}
	}
	// 真实活跃用户数（近 7 天有审计操作的去重用户）
	if s.auditRepo != nil {
		if active, err := s.auditRepo.CountDistinctActiveUsers(7); err == nil {
			health.ActiveUsers = active
		}
	}

	// CPU/磁盘/API 延迟需接入宿主机监控，暂不提供臆造值（保持 0，前端据 data_source 决定展示）
	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是运维专家。当前进程堆内存占用约%.0f%%，运行时长%s，在册用户%d、近7日活跃%d。请用30字诊断系统健康状态。",
			memUsage*100, health.Uptime, health.TotalUsers, health.ActiveUsers)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.2, MaxTokens: 150,
		})
		if err == nil && resp != nil && resp.Content != "" {
			health.AIDiagnosis = strings.TrimSpace(resp.Content)
			health.DataSource = "real+ai"
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
	result := &KnowledgeQuality{
		Issues:      []map[string]interface{}{},
		Suggestions: []string{},
		DataSource:  "fallback",
	}

	if s.kbRepo == nil {
		return result
	}
	stats, err := s.kbRepo.GetStats()
	if err != nil || stats == nil {
		return result
	}

	result.TotalResources = stats.Total
	result.DataSource = "real"
	// 覆盖率 = 已发布 / 总量；冗余、时效需额外数据，暂不臆造
	if stats.Total > 0 {
		result.Coverage = float64(stats.Published) / float64(stats.Total)
	}

	// 基于真实状态分布生成问题项
	if stats.Retired > 0 {
		result.Issues = append(result.Issues, map[string]interface{}{
			"type": "已下线", "count": stats.Retired,
			"detail": "存在已下线资源，确认是否需要清理归档", "severity": "low",
		})
	}
	if stats.Pending > 0 {
		result.Issues = append(result.Issues, map[string]interface{}{
			"type": "待审核", "count": stats.Pending,
			"detail": fmt.Sprintf("%d 条资源待审核，建议及时处理", stats.Pending), "severity": "medium",
		})
	}
	if stats.Draft > 0 {
		result.Issues = append(result.Issues, map[string]interface{}{
			"type": "草稿", "count": stats.Draft,
			"detail": fmt.Sprintf("%d 条草稿未发布", stats.Draft), "severity": "low",
		})
	}

	// LLM 基于真实分布给运营建议
	if s.llmClient != nil {
		prompt := fmt.Sprintf(
			"你是知识库运营专家。真实统计：总计%d条，已发布%d，待审核%d，草稿%d，已下线%d。请给出3条精炼运营建议，每条不超过25字，只依据数据。",
			stats.Total, stats.Published, stats.Pending, stats.Draft, stats.Retired)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.4, MaxTokens: 220,
		})
		if err == nil && resp != nil && resp.Content != "" {
			result.Suggestions = splitSuggestions(resp.Content)
			result.DataSource = "real+ai"
		}
	}

	return result
}

// UserBehaviorAnalysis 用户行为分析
type UserBehaviorAnalysis struct {
	Period        string                   `json:"period"`
	TotalUsers    int                      `json:"total_users"`
	ActiveUsers   int                      `json:"active_users"`
	DAU           int                      `json:"dau"`
	RetentionRate float64                  `json:"retention_rate"`
	TopFeatures   []map[string]interface{} `json:"top_features"`
	UserJourneys  []string                 `json:"user_journeys"`
	Suggestions   []string                 `json:"suggestions"`
	DataSource    string                   `json:"data_source"`
}

func (s *SysAdminService) AnalyzeUserBehavior(ctx context.Context) *UserBehaviorAnalysis {
	result := &UserBehaviorAnalysis{
		Period:       time.Now().Format("2006年01月"),
		TopFeatures:  []map[string]interface{}{},
		UserJourneys: []string{},
		Suggestions:  []string{},
		DataSource:   "fallback",
	}

	// 真实在册用户总数
	if s.userRepo != nil {
		if n, err := s.userRepo.Count("", "", ""); err == nil {
			result.TotalUsers = n
			result.DataSource = "real"
		}
	}

	// 真实活跃用户与功能使用分布（来自审计日志聚合，非臆造埋点）
	if s.auditRepo != nil {
		if active, err := s.auditRepo.CountDistinctActiveUsers(7); err == nil {
			result.ActiveUsers = active // 近 7 天有操作记录的去重用户
			result.DataSource = "real"
		}
		if actions, err := s.auditRepo.TopActions(30, 5); err == nil && len(actions) > 0 {
			for _, a := range actions {
				result.TopFeatures = append(result.TopFeatures, map[string]interface{}{
					"feature": a.Action,
					"count":   a.Count,
				})
			}
		}
	}

	return result
}
