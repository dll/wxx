package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/llm"
)

// UnionService 学生会角色 AI 功能服务
// db 用于接真实数据（活动/报名/到场统计），不依赖 LLM 也能返回真实统计。
type UnionService struct {
	db        *sql.DB
	llmClient llm.ChatClient
}

func NewUnionService(db *sql.DB, llmClient llm.ChatClient) *UnionService {
	return &UnionService{db: db, llmClient: llmClient}
}

// EventPlan 活动策划方案
type EventPlan struct {
	Title          string              `json:"title"`
	Goal           string              `json:"goal"`
	Budget         string              `json:"budget"`
	Timeline       []map[string]string `json:"timeline"`
	Promotion      string              `json:"promotion"`
	PosterCopy     string              `json:"poster_copy"`
	RiskAssessment []string            `json:"risk_assessment"`
	DataSource     string              `json:"data_source"`
}

func (s *UnionService) GenerateEventPlan(ctx context.Context, eventType, eventName string) *EventPlan {
	if eventName == "" {
		eventName = "校园科技文化节"
	}

	plan := &EventPlan{
		Title:  eventName,
		Goal:   "丰富校园文化生活，提升学生综合素质",
		Budget: "预估经费：3000元（场地布置1000元 + 宣传物料500元 + 奖品1500元）",
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
	"科技": {"colors": "蓝色系 #1565C0 + #42A5F5", "layout": "左侧科技图案 + 右侧文字排版"},
	"文艺": {"colors": "暖色系 #E65100 + #FFB74D", "layout": "居中对称 + 艺术字体"},
	"简约": {"colors": "黑白灰 #333 + #F5F5F5", "layout": "极简排版 + 大标题"},
	"学术": {"colors": "深蓝紫 #283593 + #7E57C2", "layout": "上标题 + 中内容 + 下信息"},
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
		Subtitle:    "滁州学院计算机学院",
		Copy:        "诚邀您的参与！\n时间：2026年5月\n地点：信息楼报告厅",
		ColorScheme: styleConfig["colors"],
		Layout:      styleConfig["layout"],
		DataSource:  "reference",
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
		DataSource: "reference",
	}
}

// MemberManagementData 成员管理
type MemberManagementData struct {
	Members    []map[string]interface{} `json:"members"`
	Stats      map[string]interface{}   `json:"stats"`
	DataSource string                   `json:"data_source"`
}

// ManageMembers 成员活跃度（真实数据）：按报名/到场聚合到人，避免编造姓名。
// 数据源：health_activity_signups JOIN users（真实报名与签到记录）。
func (s *UnionService) ManageMembers(ctx context.Context) *MemberManagementData {
	data := &MemberManagementData{
		Members:    []map[string]interface{}{},
		Stats:      map[string]interface{}{"total": 0, "active": 0, "excellent": 0, "needs_improve": 0, "with_signups": 0},
		DataSource: "reference",
	}
	if s.db == nil {
		return data
	}
	drv := dbutil.DriverOf(s.db)
	// 每个参与人：报名次数、到场次数（attended=1）、到场率；去重按 user_id
	stmt := `SELECT u.id, COALESCE(u.display_name, u.username, ?) AS name,
	                COUNT(s.id) AS signups,
	                COALESCE(SUM(CASE WHEN s.attended = 1 THEN 1 ELSE 0 END), 0) AS attends
	         FROM health_activity_signups s
	         LEFT JOIN users u ON u.id = s.user_id
	         WHERE s.status = 'registered'
	         GROUP BY u.id
	         ORDER BY signups DESC, attends DESC
	         LIMIT 50`
	rows, err := s.db.Query(dbutil.AdaptForDriver(stmt, drv), "未署名同学")
	if err != nil {
		return data
	}
	defer rows.Close()

	total := 0
	excellent := 0
	needsImprove := 0
	active := 0
	for rows.Next() {
		var uid int64
		var name string
		var signups, attends int
		if err := rows.Scan(&uid, &name, &signups, &attends); err != nil {
			continue
		}
		total++
		perf := "C"
		if attends >= 3 {
			perf = "A"
			excellent++
		} else if attends >= 1 {
			perf = "B"
			active++
		} else {
			needsImprove++
		}
		data.Members = append(data.Members, map[string]interface{}{
			"user_id":     uid,
			"name":        name,
			"signups":     signups,
			"attends":     attends,
			"performance": perf,
			"suggestion":  perfSuggest(perf),
		})
	}
	// 有真实记录即标记为真实数据
	if total > 0 {
		data.DataSource = "real"
	}
	data.Stats = map[string]interface{}{
		"total":         total,
		"active":        active,
		"excellent":     excellent,
		"needs_improve": needsImprove,
		"with_signups":  total,
	}
	return data
}

func perfSuggest(p string) string {
	switch p {
	case "A":
		return "积极参与、到场稳定，可考虑委以更重要的组织任务"
	case "B":
		return "有参与，建议持续保持，可增加组织协调机会"
	default:
		return "报名但到场较少，建议关注其参与情况"
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
		DataSource: "reference",
	}
}

// HotTopicTrackData 热点追踪
type HotTopicTrackData struct {
	Topics      []map[string]interface{} `json:"topics"`
	Suggestions []string                 `json:"suggestions"`
	DataSource  string                   `json:"data_source"`
}

func (s *UnionService) TrackHotTopics(ctx context.Context) *HotTopicTrackData {
	return &HotTopicTrackData{
		Topics: []map[string]interface{}{
			{"topic": "期末复习资料", "heat": 88, "trend": "rising", "related_events": "期末考试临近"},
			{"topic": "暑期社会实践", "heat": 75, "trend": "rising", "related_events": "报名即将截止"},
		},
		Suggestions: []string{"建议尽快发布期末复习资料整理通知", "暑期社会实践报名宣传可结合榜样案例"},
		DataSource:  "reference",
	}
}

// ActivityAnalysisData 活动数据分析
type ActivityAnalysisData struct {
	EventName   string                 `json:"event_name"`
	RegRate     float64                `json:"reg_rate"`
	AttendRate  float64                `json:"attend_rate"`
	Feedback    float64                `json:"feedback_score"`
	Demographic map[string]interface{} `json:"demographic"`
	Report      string                 `json:"report"`
	Suggestions []string               `json:"suggestions"`
	DataSource  string                 `json:"data_source"`
}

// AnalyzeActivity 活动数据分析（真实数据）：按活动查真实 报名数/名额/到场数。
// 数据源：health_activities + health_activity_signups。
func (s *UnionService) AnalyzeActivity(ctx context.Context, eventName string) *ActivityAnalysisData {
	data := &ActivityAnalysisData{
		EventName:   eventName,
		RegRate:     0,
		AttendRate:  0,
		Feedback:    0,
		Demographic: map[string]interface{}{},
		Report:      "暂无该活动的报名数据，可先新建活动并引导报名。",
		Suggestions: []string{},
		DataSource:  "reference",
	}
	if s.db == nil || eventName == "" {
		return data
	}
	drv := dbutil.DriverOf(s.db)
	var id string
	var title, startAt, venue, organizer string
	var capacity, creatorID int64
	err := s.db.QueryRow(dbutil.AdaptForDriver(
		`SELECT activity_id, title, COALESCE(start_at,''), COALESCE(venue,''), COALESCE(organizer,''), COALESCE(capacity,0), COALESCE(creator_id,0) FROM health_activities WHERE activity_id = ? OR title = ? LIMIT 1`, drv),
		eventName, eventName).Scan(&id, &title, &startAt, &venue, &organizer, &capacity, &creatorID)
	if err != nil {
		if err == sql.ErrNoRows {
			data.Report = fmt.Sprintf("未找到活动「%s」，请先用活动报名管理创建。", eventName)
		}
		return data
	}
	data.EventName = title

	var signups, attends int
	_ = s.db.QueryRow(dbutil.AdaptForDriver(
		`SELECT COUNT(*) FROM health_activity_signups WHERE activity_id = ? AND status = 'registered'`, drv), id).Scan(&signups)
	_ = s.db.QueryRow(dbutil.AdaptForDriver(
		`SELECT COALESCE(SUM(CASE WHEN attended = 1 THEN 1 ELSE 0 END),0) FROM health_activity_signups WHERE activity_id = ? AND status = 'registered'`, drv), id).Scan(&attends)

	if signups > 0 {
		data.DataSource = "real"
		reg := float64(signups)
		if capacity > 0 {
			data.RegRate = round2(reg / float64(capacity) * 100)
		} else {
			data.RegRate = round2(reg)
		}
		data.AttendRate = round2(float64(attends) / reg * 100)
		data.Suggestions = []string{
			fmt.Sprintf("该活动报名 %d 人，到场 %d 人，到场率 %.0f%%。", signups, attends, data.AttendRate),
		}
		if capacity > 0 && signups >= int(float64(capacity)*0.9) {
			data.Suggestions = append(data.Suggestions, "名额接近报满，可评估增加场次。")
		} else if capacity > 0 {
			data.Suggestions = append(data.Suggestions, fmt.Sprintf("距报满还差 %d 人，可加强宣传。", int(capacity)-signups))
		}
		if data.AttendRate < 70 {
			data.Suggestions = append(data.Suggestions, "到场率偏低，建议优化时间地点或增加到场激励。")
		}
		data.Report = fmt.Sprintf("%s：报名 %d 人（名额 %d），到场 %d 人，到场率 %.0f%%。", title, signups, capacity, attends, data.AttendRate)
	} else {
		data.Report = fmt.Sprintf("%s：暂无报名记录，可先引导报名再复盘。", title)
	}
	return data
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
