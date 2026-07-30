package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
)

// TwinService 个人数字孪生业务服务
//
// 职责：聚合五维原始指标 → 归一化打分 → 差距分析 → LLM 状态解读（带兜底）。
// LLM 不可用时降级为规则化文本，保证功能始终可用（符合"低置信走兜底"约束）。
type TwinService struct {
	repo      *repository.TwinRepo
	userRepo  *repository.UserRepo
	llmClient llm.ChatClient // 可为 nil，此时全程走规则兜底
}

// NewTwinService 创建数字孪生服务
func NewTwinService(repo *repository.TwinRepo, userRepo *repository.UserRepo, llmClient llm.ChatClient) *TwinService {
	return &TwinService{repo: repo, userRepo: userRepo, llmClient: llmClient}
}

// TwinDimension 单维度结果
type TwinDimension struct {
	Key   string  `json:"key"`   // academic/ability/ideological/emotional/social
	Name  string  `json:"name"`  // 中文维度名
	Score float64 `json:"score"` // 0-100
	Level string  `json:"level"` // 优秀/良好/待提升
	Desc  string  `json:"desc"`  // 该维度简述
}

// TwinResult 数字孪生完整结果（返回给前端）
type TwinResult struct {
	UserID         int64           `json:"user_id"`
	DisplayName    string          `json:"display_name"`
	OverallScore   float64         `json:"overall_score"` // 五维加权总分
	Dimensions     []TwinDimension `json:"dimensions"`
	Interpretation string          `json:"interpretation"` // AI/规则状态解读
	GapAnalysis    []string        `json:"gap_analysis"`   // 差距点
	StageAdvice    []string        `json:"stage_advice"`   // 阶段建议
	ComputedAt     string          `json:"computed_at"`
	Fallback       bool            `json:"fallback"` // 是否为规则兜底（LLM 未参与解读）
}

// 五维权重（总和为 1）：学业最重，其次能力，情感与思想社交次之
var dimWeights = map[string]float64{
	"academic":    0.30,
	"ability":     0.25,
	"ideological": 0.15,
	"emotional":   0.15,
	"social":      0.15,
}

// clamp 限定分数在 [0,100]
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return math.Round(v*10) / 10
}

// scoreLevel 分数分档
func scoreLevel(s float64) string {
	switch {
	case s >= 80:
		return "优秀"
	case s >= 60:
		return "良好"
	default:
		return "待提升"
	}
}

// computeDimensions 把原始指标归一化为五维分数
func computeDimensions(m *repository.TwinRawMetrics) []TwinDimension {
	// 学业：GPA 满分按 4.0 折算 70%，通过率折算 30%
	academic := 0.0
	if m.CourseCount > 0 {
		academic = clamp(m.AvgGPA/4.0*70 + m.PassRate*30)
	}

	// 能力：竞赛参与(每次 8 分,上限 40) + 获奖(每次 15 分,上限 40) + 规划完成率 20
	ability := float64(m.CompetitionCount)*8 + float64(m.AwardCount)*15
	if ability > 80 {
		ability = 80
	}
	if m.PlanCount > 0 {
		ability += float64(m.PlanDoneCount) / float64(m.PlanCount) * 20
	}
	ability = clamp(ability)

	// 思想：党建阶段序 *16（满 5 阶段=80）+ 学习记录(每条 4 分,上限 20)
	ideological := float64(m.PartyStageRank)*16 + math.Min(float64(m.PartyStudyCount)*4, 20)
	ideological = clamp(ideological)

	// 情感：无日志默认 75（中性）；有日志时 score 越高越负面，故反向；高风险扣分
	emotional := 75.0
	if m.EmotionLogCount > 0 {
		emotional = clamp(100 - m.AvgEmotionScore*10 - float64(m.HighRiskCount)*8)
	}

	// 社交：社团(每个 20 分,上限 60) + 活动(每次 8 分,上限 40)
	social := clamp(math.Min(float64(m.ClubCount)*20, 60) + math.Min(float64(m.ActivityRegCount)*8, 40))

	mk := func(key, name string, score float64, desc string) TwinDimension {
		return TwinDimension{Key: key, Name: name, Score: score, Level: scoreLevel(score), Desc: desc}
	}
	return []TwinDimension{
		mk("academic", "学业", academic, fmt.Sprintf("平均绩点 %.2f，修得学分 %.1f", m.AvgGPA, m.CreditsEarned)),
		mk("ability", "能力", ability, fmt.Sprintf("竞赛 %d 次，获奖 %d 次，完成规划 %d/%d", m.CompetitionCount, m.AwardCount, m.PlanDoneCount, m.PlanCount)),
		mk("ideological", "思想", ideological, fmt.Sprintf("党建阶段序 %d，学习记录 %d 条", m.PartyStageRank, m.PartyStudyCount)),
		mk("emotional", "情感", emotional, fmt.Sprintf("情感记录 %d 条，高风险 %d 次", m.EmotionLogCount, m.HighRiskCount)),
		mk("social", "社交", social, fmt.Sprintf("参与社团 %d 个，活动报名 %d 次", m.ClubCount, m.ActivityRegCount)),
	}
}

// GetDigitalTwin 计算并返回某学生的数字孪生画像，同时刷新快照。
// refresh=false 且存在当日快照时可直接复用（此处始终重算保证实时性，快照仅作看板聚合缓存）。
func (s *TwinService) GetDigitalTwin(ctx context.Context, userID int64) (*TwinResult, error) {
	metrics, err := s.repo.AggregateRawMetrics(userID)
	if err != nil {
		return nil, fmt.Errorf("聚合五维指标失败: %w", err)
	}

	dims := computeDimensions(metrics)
	var overall float64
	for _, d := range dims {
		overall += d.Score * dimWeights[d.Key]
	}
	overall = clamp(overall)

	gaps := analyzeGaps(dims)

	// 取用户基础信息用于快照归属与展示
	displayName := ""
	ownerScope, ownerID, college, major, className := "college", "default", "", "", ""
	if s.userRepo != nil {
		if u, uerr := s.userRepo.GetByID(userID); uerr == nil && u != nil {
			displayName = u.DisplayName
			ownerScope, ownerID = u.OwnerScope, u.OwnerID
			college, major, className = u.College, u.Major, u.ClassName
		}
	}

	// LLM 解读（失败降级为规则文本）
	interpretation, advice, fallback := s.interpret(ctx, displayName, dims, gaps)

	result := &TwinResult{
		UserID:         userID,
		DisplayName:    displayName,
		OverallScore:   overall,
		Dimensions:     dims,
		Interpretation: interpretation,
		GapAnalysis:    gaps,
		StageAdvice:    advice,
		ComputedAt:     time.Now().Format(time.RFC3339),
		Fallback:       fallback,
	}

	// 落快照（失败不阻断返回，仅影响看板聚合）
	gapJSON, _ := json.Marshal(gaps)
	adviceJSON, _ := json.Marshal(advice)
	_ = s.repo.UpsertSnapshot(&repository.TwinSnapshot{
		UserID: userID, OwnerScope: ownerScope, OwnerID: ownerID,
		College: college, Major: major, ClassName: className,
		AcademicScore: dims[0].Score, AbilityScore: dims[1].Score,
		IdeologicalScore: dims[2].Score, EmotionalScore: dims[3].Score, SocialScore: dims[4].Score,
		AIInterpretation: interpretation, GapAnalysis: string(gapJSON), StageAdvice: string(adviceJSON),
	})

	return result, nil
}

// analyzeGaps 找出低于 60 分的维度作为差距点
func analyzeGaps(dims []TwinDimension) []string {
	var gaps []string
	for _, d := range dims {
		if d.Score < 60 {
			gaps = append(gaps, fmt.Sprintf("%s维度偏弱（%.1f 分）：%s", d.Name, d.Score, d.Desc))
		}
	}
	if len(gaps) == 0 {
		gaps = []string{"五维发展较为均衡，暂无明显短板"}
	}
	return gaps
}

// interpret 生成状态解读与阶段建议；LLM 不可用则规则兜底
func (s *TwinService) interpret(ctx context.Context, name string, dims []TwinDimension, gaps []string) (string, []string, bool) {
	if s.llmClient == nil {
		return s.ruleInterpret(dims, gaps)
	}

	var sb string
	sb = "你是高校学工助手，请依据学生五维画像给出简短中肯的状态解读（80字内）与三条可执行的阶段建议。仅返回 JSON：{\"interpretation\":\"...\",\"advice\":[\"...\",\"...\",\"...\"]}。\n学生画像：\n"
	for _, d := range dims {
		sb += fmt.Sprintf("- %s：%.1f 分（%s）%s\n", d.Name, d.Score, d.Level, d.Desc)
	}

	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	resp, err := s.llmClient.Chat(cctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: sb}},
		Temperature: 0.5,
		MaxTokens:   400,
	})
	if err != nil || resp == nil || resp.Content == "" {
		return s.ruleInterpret(dims, gaps)
	}

	var parsed struct {
		Interpretation string   `json:"interpretation"`
		Advice         []string `json:"advice"`
	}
	if jerr := json.Unmarshal([]byte(extractJSON(resp.Content)), &parsed); jerr != nil || parsed.Interpretation == "" {
		return s.ruleInterpret(dims, gaps)
	}
	if len(parsed.Advice) == 0 {
		parsed.Advice = []string{"保持当前节奏，持续记录成长数据"}
	}
	return parsed.Interpretation, parsed.Advice, false
}

// ruleInterpret 规则兜底解读
func (s *TwinService) ruleInterpret(dims []TwinDimension, gaps []string) (string, []string, bool) {
	var best, worst TwinDimension
	best.Score, worst.Score = -1, 101
	for _, d := range dims {
		if d.Score > best.Score {
			best = d
		}
		if d.Score < worst.Score {
			worst = d
		}
	}
	interp := fmt.Sprintf("你的优势维度是%s（%.1f 分），相对薄弱的是%s（%.1f 分）。建议在保持优势的同时补齐短板。",
		best.Name, best.Score, worst.Name, worst.Score)
	advice := []string{
		fmt.Sprintf("重点提升「%s」：%s", worst.Name, worst.Desc),
		"定期更新成绩、竞赛与活动记录，保持画像新鲜度",
		"结合辅导员建议制定下一阶段成长目标",
	}
	return interp, advice, true
}

// extractJSON 从可能带 markdown 代码块的文本中提取 JSON 主体
func extractJSON(s string) string {
	start := -1
	for i, c := range s {
		if c == '{' {
			start = i
			break
		}
	}
	if start < 0 {
		return s
	}
	end := start
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}
	return s[start : end+1]
}
