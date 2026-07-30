package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/repository"
)

// PersonalityService 学生性格洞察业务服务
type PersonalityService struct {
	repo      *repository.PersonalityRepo
	userRepo  *repository.UserRepo
	twinRepo  *repository.TwinRepo
	llmClient llm.ChatClient
}

// NewPersonalityService 创建性格洞察服务
func NewPersonalityService(
	repo *repository.PersonalityRepo,
	userRepo *repository.UserRepo,
	twinRepo *repository.TwinRepo,
	llmClient llm.ChatClient,
) *PersonalityService {
	return &PersonalityService{repo: repo, userRepo: userRepo, twinRepo: twinRepo, llmClient: llmClient}
}

// PersonalityResult 返回给前端的性格洞察结果
type PersonalityResult struct {
	PersonalityType   string       `json:"type"`
	TypeLabel         string       `json:"label"`
	Description       string       `json:"description"`
	Strengths         []string     `json:"strengths"`
	Weaknesses        []string     `json:"weaknesses"`
	CareerSuggestions []string     `json:"career_suggestions"`
	LearningStyle     string       `json:"learning_style"`
	VARK              VARKScore    `json:"vark"`
	BigFive           BigFiveScore `json:"big_five"`
	DataSource        string       `json:"data_source"`
	ComputedAt        string       `json:"computed_at"`
}

// VARKScore VARK 学习风格分数
type VARKScore struct {
	Visual      float64 `json:"visual"`
	Auditory    float64 `json:"auditory"`
	Reading     float64 `json:"reading"`
	Kinesthetic float64 `json:"kinesthetic"`
	Dominant    string  `json:"dominant"` // 主导类型
}

// BigFiveScore 大五人格分数
type BigFiveScore struct {
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
}

// GetPersonality 获取用户性格洞察（有缓存则复用，否则 LLM 推断）
func (s *PersonalityService) GetPersonality(ctx context.Context, userID int64) (*PersonalityResult, error) {
	// 先尝试读取已有画像（7 天内视为有效）
	existing, _ := s.repo.GetByUserID(userID)
	if existing != nil {
		t, _ := time.Parse("2006-01-02 15:04:05", existing.ComputedAt)
		if time.Since(t) < 7*24*time.Hour {
			return s.profileToResult(existing), nil
		}
	}

	// 需要重新计算：收集行为数据 → LLM 推断
	result, err := s.inferPersonality(ctx, userID)
	if err != nil {
		// LLM 失败但有旧画像，返回旧的
		if existing != nil {
			return s.profileToResult(existing), nil
		}
		// 全无数据，返回规则兜底
		return s.fallbackResult(), nil
	}
	return result, nil
}

// inferPersonality 基于用户行为数据用 LLM 推断性格画像
func (s *PersonalityService) inferPersonality(ctx context.Context, userID int64) (*PersonalityResult, error) {
	// 收集行为特征
	var behaviorDesc strings.Builder
	behaviorDesc.WriteString("以下是一名大学生的行为数据摘要，请据此推断其性格画像：\n\n")

	// 用户基础信息
	if s.userRepo != nil {
		if u, err := s.userRepo.GetByID(userID); err == nil && u != nil {
			behaviorDesc.WriteString(fmt.Sprintf("专业：%s，年级：%s\n", u.Major, u.ClassName))
		}
	}

	// 五维画像数据
	if s.twinRepo != nil {
		if metrics, err := s.twinRepo.AggregateRawMetrics(userID); err == nil && metrics != nil {
			behaviorDesc.WriteString(fmt.Sprintf("学业：平均绩点 %.2f，通过率 %.0f%%，修 %d 门课\n",
				metrics.AvgGPA, metrics.PassRate*100, metrics.CourseCount))
			behaviorDesc.WriteString(fmt.Sprintf("竞赛参与 %d 次，获奖 %d 次\n", metrics.CompetitionCount, metrics.AwardCount))
			behaviorDesc.WriteString(fmt.Sprintf("学习规划完成率：%d/%d\n", metrics.PlanDoneCount, metrics.PlanCount))
			behaviorDesc.WriteString(fmt.Sprintf("社团 %d 个，活动报名 %d 次\n", metrics.ClubCount, metrics.ActivityRegCount))
			behaviorDesc.WriteString(fmt.Sprintf("情感记录 %d 条，高风险 %d 次\n", metrics.EmotionLogCount, metrics.HighRiskCount))
		}
	}

	if s.llmClient == nil {
		return nil, fmt.Errorf("LLM 不可用")
	}

	prompt := behaviorDesc.String() + `
请推断该学生的性格画像并以严格 JSON 格式返回（不要 markdown 代码块）：
{
  "type": "MBTI四字母如INTJ",
  "label": "中文类型名如建筑师型",
  "description": "50字以内的性格描述",
  "strengths": ["优势1","优势2","优势3"],
  "weaknesses": ["待改进1","待改进2"],
  "career_suggestions": ["职业1","职业2","职业3","职业4"],
  "learning_style": "30字以内学习风格描述",
  "vark": {"visual":0-100,"auditory":0-100,"reading":0-100,"kinesthetic":0-100},
  "big_five": {"openness":0-100,"conscientiousness":0-100,"extraversion":0-100,"agreeableness":0-100,"neuroticism":0-100}
}`

	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := s.llmClient.Chat(cctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.4,
		MaxTokens:   600,
	})
	if err != nil || resp == nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 推断失败: %v", err)
	}

	// 解析 JSON
	var parsed struct {
		Type              string   `json:"type"`
		Label             string   `json:"label"`
		Description       string   `json:"description"`
		Strengths         []string `json:"strengths"`
		Weaknesses        []string `json:"weaknesses"`
		CareerSuggestions []string `json:"career_suggestions"`
		LearningStyle     string   `json:"learning_style"`
		VARK              struct {
			Visual      float64 `json:"visual"`
			Auditory    float64 `json:"auditory"`
			Reading     float64 `json:"reading"`
			Kinesthetic float64 `json:"kinesthetic"`
		} `json:"vark"`
		BigFive struct {
			Openness          float64 `json:"openness"`
			Conscientiousness float64 `json:"conscientiousness"`
			Extraversion      float64 `json:"extraversion"`
			Agreeableness     float64 `json:"agreeableness"`
			Neuroticism       float64 `json:"neuroticism"`
		} `json:"big_five"`
	}

	jsonStr := extractJSONBlock(resp.Content)
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed.Type == "" {
		return nil, fmt.Errorf("解析性格画像 JSON 失败")
	}

	// 持久化
	strengthsJSON, _ := json.Marshal(parsed.Strengths)
	weaknessesJSON, _ := json.Marshal(parsed.Weaknesses)
	careerJSON, _ := json.Marshal(parsed.CareerSuggestions)

	_ = s.repo.Upsert(&repository.PersonalityProfile{
		UserID:            userID,
		VisualScore:       parsed.VARK.Visual,
		AuditoryScore:     parsed.VARK.Auditory,
		ReadingScore:      parsed.VARK.Reading,
		KinestheticScore:  parsed.VARK.Kinesthetic,
		Openness:          parsed.BigFive.Openness,
		Conscientiousness: parsed.BigFive.Conscientiousness,
		Extraversion:      parsed.BigFive.Extraversion,
		Agreeableness:     parsed.BigFive.Agreeableness,
		Neuroticism:       parsed.BigFive.Neuroticism,
		PersonalityType:   parsed.Type,
		TypeLabel:         parsed.Label,
		Description:       parsed.Description,
		LearningStyle:     parsed.LearningStyle,
		Strengths:         string(strengthsJSON),
		Weaknesses:        string(weaknessesJSON),
		CareerSuggestions: string(careerJSON),
		DataSource:        "llm",
	})

	// 构造返回
	dominant := dominantVARK(parsed.VARK.Visual, parsed.VARK.Auditory, parsed.VARK.Reading, parsed.VARK.Kinesthetic)

	return &PersonalityResult{
		PersonalityType:   parsed.Type,
		TypeLabel:         parsed.Label,
		Description:       parsed.Description,
		Strengths:         parsed.Strengths,
		Weaknesses:        parsed.Weaknesses,
		CareerSuggestions: parsed.CareerSuggestions,
		LearningStyle:     parsed.LearningStyle,
		VARK:              VARKScore{Visual: parsed.VARK.Visual, Auditory: parsed.VARK.Auditory, Reading: parsed.VARK.Reading, Kinesthetic: parsed.VARK.Kinesthetic, Dominant: dominant},
		BigFive:           BigFiveScore{Openness: parsed.BigFive.Openness, Conscientiousness: parsed.BigFive.Conscientiousness, Extraversion: parsed.BigFive.Extraversion, Agreeableness: parsed.BigFive.Agreeableness, Neuroticism: parsed.BigFive.Neuroticism},
		DataSource:        "ai",
		ComputedAt:        time.Now().Format(time.RFC3339),
	}, nil
}

// profileToResult 将持久化画像转换为返回结构
func (s *PersonalityService) profileToResult(p *repository.PersonalityProfile) *PersonalityResult {
	var strengths, weaknesses, career []string
	_ = json.Unmarshal([]byte(p.Strengths), &strengths)
	_ = json.Unmarshal([]byte(p.Weaknesses), &weaknesses)
	_ = json.Unmarshal([]byte(p.CareerSuggestions), &career)

	dominant := dominantVARK(p.VisualScore, p.AuditoryScore, p.ReadingScore, p.KinestheticScore)

	return &PersonalityResult{
		PersonalityType:   p.PersonalityType,
		TypeLabel:         p.TypeLabel,
		Description:       p.Description,
		Strengths:         strengths,
		Weaknesses:        weaknesses,
		CareerSuggestions: career,
		LearningStyle:     p.LearningStyle,
		VARK:              VARKScore{Visual: p.VisualScore, Auditory: p.AuditoryScore, Reading: p.ReadingScore, Kinesthetic: p.KinestheticScore, Dominant: dominant},
		BigFive:           BigFiveScore{Openness: p.Openness, Conscientiousness: p.Conscientiousness, Extraversion: p.Extraversion, Agreeableness: p.Agreeableness, Neuroticism: p.Neuroticism},
		DataSource:        p.DataSource,
		ComputedAt:        p.ComputedAt,
	}
}

// fallbackResult 无任何数据时的规则兜底
func (s *PersonalityService) fallbackResult() *PersonalityResult {
	return &PersonalityResult{
		PersonalityType:   "待评估",
		TypeLabel:         "数据不足",
		Description:       "暂无足够行为数据推断性格画像，请多使用系统积累数据后再查看。",
		Strengths:         []string{"持续使用中"},
		Weaknesses:        []string{"数据样本不足"},
		CareerSuggestions: []string{"继续探索中"},
		LearningStyle:     "需要更多学习记录来判断",
		VARK:              VARKScore{Visual: 50, Auditory: 50, Reading: 50, Kinesthetic: 50, Dominant: "均衡"},
		BigFive:           BigFiveScore{Openness: 50, Conscientiousness: 50, Extraversion: 50, Agreeableness: 50, Neuroticism: 50},
		DataSource:        "fallback",
		ComputedAt:        time.Now().Format(time.RFC3339),
	}
}

// dominantVARK 判断主导学习风格
func dominantVARK(v, a, r, k float64) string {
	max := v
	dom := "视觉型"
	if a > max {
		max, dom = a, "听觉型"
	}
	if r > max {
		max, dom = r, "阅读型"
	}
	if k > max {
		_, dom = k, "动觉型"
	}
	return dom
}

// extractJSONBlock 从可能带 markdown 代码块的文本中提取 JSON
func extractJSONBlock(s string) string {
	start := strings.Index(s, "{")
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
