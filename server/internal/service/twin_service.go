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
	Key           string  `json:"key"`            // academic/ability/ideological/emotional/social
	Name          string  `json:"name"`           // 中文维度名
	Score         float64 `json:"score"`          // 0-100（无数据时为 0）
	Level         string  `json:"level"`          // 优秀/良好/待提升/数据积累中
	Desc          string  `json:"desc"`           // 该维度简述
	DataAvailable bool    `json:"data_available"` // 是否有足量数据支撑该维度（false 时前端显示「数据积累中」，不展示伪分数）
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

	// 情感：score 越高越负面（[-1,1]），反向线性映射到 0-100 全区间：
	// avg=0（中性）→ 50，avg=-1（正向）→ 100，avg=1（负面）→ 0；每次高风险额外扣 8 分。
	// 无情感日志时不硬编码默认分，DataAvailable=false 由前端显示「数据积累中」。
	emotionalDataAvailable := m.EmotionLogCount > 0
	emotional := 0.0
	if emotionalDataAvailable {
		emotional = clamp(50*(1-m.AvgEmotionScore) - float64(m.HighRiskCount)*8)
	}

	// 社交：社团(每个 20 分,上限 60) + 活动(每次 8 分,上限 40)
	social := clamp(math.Min(float64(m.ClubCount)*20, 60) + math.Min(float64(m.ActivityRegCount)*8, 40))

	mk := func(key, name string, score float64, desc string) TwinDimension {
		return TwinDimension{Key: key, Name: name, Score: score, Level: scoreLevel(score), Desc: desc, DataAvailable: true}
	}
	dims := []TwinDimension{
		mk("academic", "学业", academic, fmt.Sprintf("平均绩点 %.2f，修得学分 %.1f", m.AvgGPA, m.CreditsEarned)),
		mk("ability", "能力", ability, fmt.Sprintf("竞赛 %d 次，获奖 %d 次，完成规划 %d/%d", m.CompetitionCount, m.AwardCount, m.PlanDoneCount, m.PlanCount)),
		mk("ideological", "思想", ideological, fmt.Sprintf("党建阶段序 %d，学习记录 %d 条", m.PartyStageRank, m.PartyStudyCount)),
		mk("emotional", "情感", emotional, fmt.Sprintf("情感记录 %d 条，高风险 %d 次", m.EmotionLogCount, m.HighRiskCount)),
		mk("social", "社交", social, fmt.Sprintf("参与社团 %d 个，活动报名 %d 次", m.ClubCount, m.ActivityRegCount)),
	}
	if !emotionalDataAvailable {
		dims[3].Level = "数据积累中"
		dims[3].Desc = "暂无情感记录，完成每日心情打卡或与蔚小芯聊天后可生成"
		dims[3].DataAvailable = false
	}
	return dims
}

// GetDigitalTwin 计算并返回某学生的数字孪生画像，同时刷新快照。
// refresh=false 且存在当日快照时可直接复用（此处始终重算保证实时性，快照仅作看板聚合缓存）。
func (s *TwinService) GetDigitalTwin(ctx context.Context, userID int64) (*TwinResult, error) {
	metrics, err := s.repo.AggregateRawMetrics(userID)
	if err != nil {
		return nil, fmt.Errorf("聚合五维指标失败: %w", err)
	}

	dims := computeDimensions(metrics)
	// 综合分：仅对 data_available 的维度加权，并对缺失维度权重重新归一化，避免无数据维度拉低总分
	var overall float64
	var weightSum float64
	for _, d := range dims {
		if !d.DataAvailable {
			continue
		}
		overall += d.Score * dimWeights[d.Key]
		weightSum += dimWeights[d.Key]
	}
	if weightSum > 0 {
		overall = clamp(overall / weightSum)
	}

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

// analyzeGaps 找出低于 60 分且数据可用的维度作为差距点（无数据维度不判弱项）
func analyzeGaps(dims []TwinDimension) []string {
	var gaps []string
	for _, d := range dims {
		if !d.DataAvailable {
			continue
		}
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
		if !d.DataAvailable {
			sb += fmt.Sprintf("- %s：数据积累中（暂无记录）\n", d.Name)
			continue
		}
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
		if !d.DataAvailable {
			continue
		}
		if d.Score > best.Score {
			best = d
		}
		if d.Score < worst.Score {
			worst = d
		}
	}
	if best.Score < 0 {
		return "五维数据仍在积累中，完成更多学习、活动与心情记录后即可生成画像。",
			[]string{"持续记录成长数据，保持画像新鲜度", "结合辅导员建议制定下一阶段成长目标"},
			true
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

// ─────────────────────────────────────────────────────────────
// 教辅/教师绩效画像（GetStaffTwin）
//
// 与 GetDigitalTwin（学生五维）不同：绩效画像把「我完成了多少工作」
// 汇聚成数字孪生画像，并把 教师 / 学生 / 蔚小芯 三方绑定到同一画像上。
//
// 打分规则（透明、可解释、不编造）：
//   Score = min(100, 真实次数)，次数即分数（1 次 = 1 分，封顶 100）；
//   DataAvailable = 次数 > 0（无记录时前端显示「数据积累中」）。
//   协作教师绑定取真实排课/课程数据（course_schedules.teacher 关联学生数），
//   无排课导入时诚实返回 DataAvailable=false。

// computeStaffDimensions 把绩效原始指标转成维度列表（含三方绑定块）
func computeStaffDimensions(m *repository.StaffTwinMetrics) []TwinDimension {
	mk := func(key, name string, count int, desc string) TwinDimension {
		return TwinDimension{
			Key:           key,
			Name:          name,
			Score:         clamp(float64(count)),
			Level:         "",
			Desc:          desc,
			DataAvailable: count > 0,
		}
	}
	dims := []TwinDimension{
		mk("help", "帮扶咨询", m.TalkCount, fmt.Sprintf("开展谈心帮扶 %d 次", m.TalkCount)),
		mk("schedule", "排课处理", m.ScheduleCount, fmt.Sprintf("处理排课冲突 %d 次", m.ScheduleCount)),
		mk("exam", "考试编排", m.ExamCount, fmt.Sprintf("编排考试 %d 次", m.ExamCount)),
		mk("notify", "通知发布", m.NotifyCount, fmt.Sprintf("发布通知 %d 次", m.NotifyCount)),
		mk("material", "材料产出", m.MaterialCount, fmt.Sprintf("产出材料模板/文档 %d 次", m.MaterialCount)),
		mk("facility", "后勤服务", m.FacilityCount, fmt.Sprintf("完成后勤服务 %d 次(实验/保洁/热水/查岗/环卫/借阅)", m.FacilityCount)),
		mk("wxx_use", "蔚小芯使用", m.WxxUseCount, fmt.Sprintf("调用蔚小芯功能 %d 次", m.WxxUseCount)),
		// 三方绑定块
		mk("bind_student", "服务学生", m.StudentBindCount, fmt.Sprintf("已服务 %d 名学生", m.StudentBindCount)),
		mk("bind_wxx", "蔚小芯能力", m.WxxBindCount, fmt.Sprintf("使用过 %d 项蔚小芯能力", m.WxxBindCount)),
		mk("bind_teacher", "协作教师", m.TeacherStuCount, fmt.Sprintf("关联 %d 名学生（任课/排课，来自课程数据）", m.TeacherStuCount)),
	}
	return dims
}

// GetStaffTwin 计算教辅/教师绩效画像（含三方绑定）。
// 供教辅/教师角色访问自己的数字孪生画像。
func (s *TwinService) GetStaffTwin(ctx context.Context, userID int64) (*TwinResult, error) {
	// 先取 displayName（用于匹配课程任课老师以统计师生关联）
	displayName := ""
	if s.userRepo != nil {
		if u, uerr := s.userRepo.GetByID(userID); uerr == nil && u != nil {
			displayName = u.DisplayName
		}
	}

	metrics, err := s.repo.AggregateStaffMetrics(userID, displayName)
	if err != nil {
		return nil, fmt.Errorf("聚合绩效画像指标失败: %w", err)
	}

	dims := computeStaffDimensions(metrics)

	// 综合绩效分：仅对有数据维度求平均（无数据跳过，避免拉低）
	var sum, n float64
	for _, d := range dims {
		if d.DataAvailable {
			sum += d.Score
			n++
		}
	}
	var overall float64
	if n > 0 {
		overall = clamp(sum / n)
	}

	// 绩效解读（规则文本，避免编造情感化表述）
	interpretation := "你的绩效画像数据尚在积累中，完成谈心帮扶、排课、考试编排、通知发布、后勤服务等教辅工作后会自动汇聚。"
	var advice []string
	if n > 0 {
		interpretation = fmt.Sprintf("当前已汇聚 %d 项绩效维度：累计 %d 次蔚小芯功能调用、服务 %d 名学生。",
			int(n), metrics.WxxUseCount, metrics.StudentBindCount)
		if metrics.TalkCount == 0 {
			advice = append(advice, "创建谈心帮扶记录可将「帮扶咨询」维度纳入绩效画像。")
		}
		if metrics.ScheduleCount == 0 && metrics.ExamCount == 0 {
			advice = append(advice, "使用排课冲突检测与考试编排功能可完善教务绩效。")
		}
	}

	return &TwinResult{
		UserID:         userID,
		DisplayName:    displayName,
		OverallScore:   overall,
		Dimensions:     dims,
		Interpretation: interpretation,
		GapAnalysis:    advice,
		StageAdvice:    advice,
		ComputedAt:     time.Now().Format(time.RFC3339),
		Fallback:       true, // 绩效为规则聚合，非 LLM 生成
	}, nil
}
