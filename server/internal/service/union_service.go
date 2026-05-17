package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

// UnionService 学生会角色 AI 功能服务
type UnionService struct {
	llmClient llm.ChatClient
}

func NewUnionService(llmClient llm.ChatClient) *UnionService {
	return &UnionService{llmClient: llmClient}
}

// EventPlan 活动策划方案
type EventPlan struct {
	Title        string   `json:"title"`
	Goal         string   `json:"goal"`
	Budget       string   `json:"budget"`
	Timeline     []map[string]string `json:"timeline"`
	Promotion    string   `json:"promotion"`
	PosterCopy   string   `json:"poster_copy"`
	RiskAssessment []string `json:"risk_assessment"`
	DataSource   string   `json:"data_source"`
}

func (s *UnionService) GenerateEventPlan(ctx context.Context, eventType, eventName string) *EventPlan {
	if eventName == "" {
		eventName = "校园科技文化节"
	}

	plan := &EventPlan{
		Title:      eventName,
		Goal:       "丰富校园文化生活，提升学生综合素质",
		Budget:     "预估经费：3000元（场地布置1000元 + 宣传物料500元 + 奖品1500元）",
		Timeline: []map[string]string{
			{"phase": "策划期（D-14天）", "tasks": "确定方案、申请场地、组建工作组"},
			{"phase": "筹备期（D-7天）", "tasks": "宣传物料制作、物资采购、节目排练"},
			{"phase": "执行期（D-Day）", "tasks": "场地布置、活动执行、现场协调"},
			{"phase": "收尾期（D+3天）", "tasks": "活动复盘、财务报销、新闻稿发布"},
		},
		Promotion:      "微信公众号推文 + 校园海报 + 班级群通知 + 朋友圈海报",
		PosterCopy:     "【标题】" + eventName + "重磅来袭！\n【副标题】精彩纷呈，等你来参与！\n【时间地点】详见海报",
		RiskAssessment: []string{"天气风险：准备室内备选场地", "参与度风险：提前一周预热宣传", "安全风险：安排志愿者维持秩序"},
		DataSource:     "fallback",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是学生会活动策划专家。请为「%s」活动生成完整策划方案。包括：目标、预算、时间线、宣传文案、风险评估。150字以内。", eventName)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.6, MaxTokens: 500,
		})
		if err == nil && resp != nil && resp.Content != "" {
			plan.Promotion += "\nAI补充：" + strings.TrimSpace(resp.Content)
			plan.DataSource = "ai"
		}
	}

	return plan
}

// PosterDesign 海报设计
type PosterDesign struct {
	Style       string `json:"style"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Copy        string `json:"copy"`
	ColorScheme string `json:"color_scheme"`
	Layout      string `json:"layout"`
	DataSource  string `json:"data_source"`
}

var posterStyles = map[string]map[string]string{
	"科技":   {"colors": "蓝色系 #1565C0 + #42A5F5", "layout": "左侧科技图案 + 右侧文字排版"},
	"文艺":   {"colors": "暖色系 #E65100 + #FFB74D", "layout": "居中对称 + 艺术字体"},
	"简约":   {"colors": "黑白灰 #333 + #F5F5F5", "layout": "极简排版 + 大标题"},
	"学术":   {"colors": "深蓝紫 #283593 + #7E57C2", "layout": "上标题 + 中内容 + 下信息"},
}

func (s *UnionService) GeneratePoster(ctx context.Context, title, style string) *PosterDesign {
	if title == "" {
		title = "校园活动"
	}
	if style == "" {
		style = "简约"
	}

	styleConfig, ok := posterStyles[style]
	if !ok {
		styleConfig = posterStyles["简约"]
	}

	return &PosterDesign{
		Style:       style,
		Title:       title,
		Subtitle:    "滁州学院信息学院",
		Copy:        "诚邀您的参与！\n时间：2026年5月\n地点：信息楼报告厅",
		ColorScheme: styleConfig["colors"],
		Layout:      styleConfig["layout"],
		DataSource:  "mock",
	}
}

// ======================== P2 深度分析功能 ========================

// RecruitmentData 招新方案
type RecruitmentData struct {
	Plan        string                   `json:"plan"`
	Stages      []map[string]interface{} `json:"stages"`
	ChannelPlan map[string]interface{}   `json:"channel_plan"`
	Questions   []string                 `json:"interview_questions"`
	MatchScore  []map[string]interface{} `json:"match_scores"`
	DataSource  string                   `json:"data_source"`
}

func (s *UnionService) GenerateRecruitment(ctx context.Context, deptName string) *RecruitmentData {
	return &RecruitmentData{
		Plan: fmt.Sprintf("%s招新方案", deptName),
		Stages: []map[string]interface{}{
			{"stage": "宣传期", "duration": "第1-2周", "actions": []string{"海报宣传", "线上推文", "线下宣讲会"}},
			{"stage": "报名期", "duration": "第3周", "actions": []string{"线上报名表", "简历收集"}},
			{"stage": "面试期", "duration": "第4周", "actions": []string{"一面(群面)", "二面(个面)", "综合评审"}},
		},
		Questions: []string{
			"你为什么想加入我们部门？",
			"描述一次你在团队中解决冲突的经历",
			"你觉得你能为部门带来什么？",
		},
		DataSource: "mock",
	}
}

// MemberManagementData 成员管理
type MemberManagementData struct {
	Members    []map[string]interface{} `json:"members"`
	Stats      map[string]interface{}   `json:"stats"`
	DataSource string                   `json:"data_source"`
}

func (s *UnionService) ManageMembers(ctx context.Context) *MemberManagementData {
	return &MemberManagementData{
		Members: []map[string]interface{}{
			{"name": "张明", "role": "干事", "hours": 45, "performance": "A", "suggestion": "推荐晋升副部长"},
			{"name": "李华", "role": "干事", "hours": 30, "performance": "B", "suggestion": "需加强主动性和责任心"},
		},
		Stats:      map[string]interface{}{"total": 20, "active": 18, "excellent": 5, "needs_improve": 2},
		DataSource: "mock",
	}
}

// QuestionnaireData 问卷生成
type QuestionnaireData struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	Questions   []map[string]interface{} `json:"questions"`
	DataSource  string                   `json:"data_source"`
}

func (s *UnionService) GenerateQuestionnaire(ctx context.Context, topic string) *QuestionnaireData {
	return &QuestionnaireData{
		Title:       topic + "调查问卷",
		Description: "为了更好地了解同学们的需求和意见",
		Questions: []map[string]interface{}{
			{"type": "single_choice", "q": "你对" + topic + "的满意度如何？", "options": []string{"非常满意", "满意", "一般", "不满意"}},
			{"type": "multiple_choice", "q": "你希望通过哪些渠道了解" + topic + "？", "options": []string{"公众号", "班级群", "海报", "线下活动"}},
			{"type": "text", "q": "你对" + topic + "有什么建议？"},
		},
		DataSource: "mock",
	}
}

// HotTopicTrackData 热点追踪
type HotTopicTrackData struct {
	Topics     []map[string]interface{} `json:"topics"`
	Suggestions []string                `json:"suggestions"`
	DataSource string                   `json:"data_source"`
}

func (s *UnionService) TrackHotTopics(ctx context.Context) *HotTopicTrackData {
	return &HotTopicTrackData{
		Topics: []map[string]interface{}{
			{"topic": "期末复习资料", "heat": 88, "trend": "rising", "related_events": "期末考试临近"},
			{"topic": "暑期社会实践", "heat": 75, "trend": "rising", "related_events": "报名即将截止"},
		},
		Suggestions: []string{"建议尽快发布期末复习资料整理通知", "暑期社会实践报名宣传可结合榜样案例"},
		DataSource:  "mock",
	}
}

// ActivityAnalysisData 活动数据分析
type ActivityAnalysisData struct {
	EventName    string                   `json:"event_name"`
	RegRate      float64                  `json:"reg_rate"`
	AttendRate   float64                  `json:"attend_rate"`
	Feedback     float64                  `json:"feedback_score"`
	Demographic  map[string]interface{}   `json:"demographic"`
	Report       string                   `json:"report"`
	Suggestions  []string                 `json:"suggestions"`
	DataSource   string                   `json:"data_source"`
}

func (s *UnionService) AnalyzeActivity(ctx context.Context, eventName string) *ActivityAnalysisData {
	return &ActivityAnalysisData{
		EventName:  eventName,
		RegRate:    0.85,
		AttendRate:  0.72,
		Feedback:   4.2,
		Demographic: map[string]interface{}{"大一": 40, "大二": 35, "大三": 20, "大四": 5},
		Report:     fmt.Sprintf("%s活动整体参与度良好，报名率达85%%，到场率72%%。大一大二学生为主要参与群体。建议优化活动时间安排以提升到场率。", eventName),
		Suggestions: []string{"优化时间安排，避开考试周", "增加线上参与渠道", "提前一周加大宣传力度"},
		DataSource:  "mock",
	}
}
