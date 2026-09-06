package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// StudentService 学生角色 AI 功能服务
// 整合用户画像 + 历史会话 + 情感数据 + LLM，生成真实个性化的学生输出
// （区别于早期 handler 中的硬编码 mock）
type StudentService struct {
	userRepo    *repository.UserRepo
	sessionRepo *repository.SessionRepo
	messageRepo *repository.MessageRepo
	emotionRepo *repository.EmotionRepo
	kbRepo      *repository.KBRepo
	twinRepo    *repository.TwinRepo // 复用五维底座的成绩/班级基准查询，可为 nil（走兜底）
	llmClient   llm.ChatClient
}

// NewStudentService 创建学生服务（llmClient 可为 nil，对应 LLM 失败时走兜底）
func NewStudentService(
	userRepo *repository.UserRepo,
	sessionRepo *repository.SessionRepo,
	messageRepo *repository.MessageRepo,
	emotionRepo *repository.EmotionRepo,
	kbRepo *repository.KBRepo,
	twinRepo *repository.TwinRepo,
	llmClient llm.ChatClient,
) *StudentService {
	return &StudentService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		emotionRepo: emotionRepo,
		kbRepo:      kbRepo,
		twinRepo:    twinRepo,
		llmClient:   llmClient,
	}
}

// DailyBriefing 今日速览结构
type DailyBriefing struct {
	Date            string                   `json:"date"`
	Greeting        string                   `json:"greeting"`
	UserDisplayName string                   `json:"user_display_name"`
	RecentQuestions []string                 `json:"recent_questions"` // 最近 5 个提问
	SessionCount    int                      `json:"session_count"`    // 历史会话数
	EmotionRisk     string                   `json:"emotion_risk"`     // none/low/medium/high
	Courses         []map[string]interface{} `json:"courses"`
	Deadlines       []map[string]interface{} `json:"deadlines"`
	Activities      []map[string]interface{} `json:"activities"`
	Weather         string                   `json:"weather"`
	Motto           string                   `json:"motto"`       // LLM 生成的个性化激励语
	DataSource      string                   `json:"data_source"` // ai/fallback
}

// GenerateDailyBriefing 生成学生今日速览
// 真实数据流：用户信息 + 最近提问 + 情感风险 → LLM → 个性化激励语
func (s *StudentService) GenerateDailyBriefing(ctx context.Context, userID int64) (*DailyBriefing, error) {
	today := time.Now().Format("2006-01-02")

	// 防御性 nil 检查（测试或异常场景）
	if s.userRepo == nil {
		return s.fallbackBriefing(today, "同学"), nil
	}

	// 读取用户信息
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		// 用户不存在（Vercel 冷启动场景）→ 兜底
		return s.fallbackBriefing(today, "同学"), nil
	}

	// 读取最近提问（用于 LLM 个性化）
	var recentQs []string
	if s.messageRepo != nil {
		recentQs, _ = s.messageRepo.GetRecentQuestionsByUserID(userID, 5)
	}

	// 读取会话总数
	var sessionCount int
	if s.sessionRepo != nil {
		sessions, _ := s.sessionRepo.ListByUserID(userID, 100)
		sessionCount = len(sessions)
	}

	// 读取情感风险（最近 7 天高风险告警）
	emotionRisk := s.evaluateEmotionRisk(user.OwnerScope, user.OwnerID, user.Role)

	briefing := &DailyBriefing{
		Date:            today,
		UserDisplayName: user.DisplayName,
		RecentQuestions: recentQs,
		SessionCount:    sessionCount,
		EmotionRisk:     emotionRisk,
		Courses:         defaultCourses(),
		Deadlines:       defaultDeadlines(),
		Activities:      defaultActivities(),
		Weather:         "晴 26°C",
		DataSource:      "fallback",
	}

	// LLM 生成个性化问候语 + 激励语
	greeting, motto := s.generatePersonalizedGreeting(ctx, user, recentQs, emotionRisk)
	briefing.Greeting = greeting
	briefing.Motto = motto
	if s.llmClient != nil && motto != defaultMotto {
		briefing.DataSource = "ai"
	}

	return briefing, nil
}

// evaluateEmotionRisk 评估当前用户的情感风险水平
func (s *StudentService) evaluateEmotionRisk(ownerScope, ownerID, role string) string {
	if s.emotionRepo == nil {
		return "none"
	}
	stats, err := s.emotionRepo.GetStats(ownerScope, ownerID, role)
	if err != nil || stats == nil {
		return "none"
	}
	if stats.High > 0 {
		return "high"
	}
	if stats.Medium > 0 {
		return "medium"
	}
	if stats.Low > 0 {
		return "low"
	}
	return "none"
}

const defaultMotto = "学如逆水行舟，不进则退。"

// generatePersonalizedGreeting 用 LLM 生成符合学生当下情况的问候和激励语
func (s *StudentService) generatePersonalizedGreeting(ctx context.Context, user *model.User, recentQs []string, emotionRisk string) (greeting, motto string) {
	// 默认问候（按时段）
	greeting = greetingByHour(time.Now().Hour(), user.DisplayName)
	motto = defaultMotto

	if s.llmClient == nil {
		return
	}

	// 拼装提示词
	var b strings.Builder
	b.WriteString("你是一个温和的校园 AI 学工助手。请为以下学生生成今天的简短个性化问候和一句激励语。\n\n")
	b.WriteString(fmt.Sprintf("学生姓名：%s\n", user.DisplayName))
	b.WriteString(fmt.Sprintf("当前时段：%s\n", timeOfDay(time.Now().Hour())))
	if emotionRisk != "none" {
		b.WriteString(fmt.Sprintf("学生最近情绪风险：%s（请用更温和、更关怀的语气）\n", emotionRisk))
	}
	if len(recentQs) > 0 {
		b.WriteString("最近提问：\n")
		for _, q := range recentQs {
			b.WriteString("- " + q + "\n")
		}
	}
	b.WriteString("\n输出格式（严格遵守，两行）：\n问候语：xxx\n激励语：xxx\n要求：每行不超过 30 字，平实、不说教。")

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.6,
		MaxTokens:   200,
	})
	if err != nil || resp.Content == "" {
		return
	}

	g, m := parseGreetingAndMotto(resp.Content)
	if g != "" {
		greeting = g
	}
	if m != "" {
		motto = m
	}
	return
}

// parseGreetingAndMotto 解析 LLM 输出的"问候语：xxx\n激励语：xxx"格式
func parseGreetingAndMotto(text string) (greeting, motto string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "问候语：") || strings.HasPrefix(line, "问候语:") {
			greeting = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "问候语："), "问候语:"))
		} else if strings.HasPrefix(line, "激励语：") || strings.HasPrefix(line, "激励语:") {
			motto = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "激励语："), "激励语:"))
		}
	}
	return
}

func greetingByHour(hour int, name string) string {
	switch {
	case hour < 6:
		return fmt.Sprintf("夜深了，%s 注意休息哦", name)
	case hour < 11:
		return fmt.Sprintf("早上好，%s！新的一天开始了", name)
	case hour < 14:
		return fmt.Sprintf("中午好，%s，吃饭了吗？", name)
	case hour < 18:
		return fmt.Sprintf("下午好，%s，继续加油", name)
	default:
		return fmt.Sprintf("晚上好，%s，今天辛苦了", name)
	}
}

func timeOfDay(hour int) string {
	switch {
	case hour < 6:
		return "凌晨"
	case hour < 11:
		return "上午"
	case hour < 14:
		return "中午"
	case hour < 18:
		return "下午"
	default:
		return "晚上"
	}
}

// fallbackBriefing 兜底数据（用户不存在或异常时）
func (s *StudentService) fallbackBriefing(today, name string) *DailyBriefing {
	return &DailyBriefing{
		Date:            today,
		Greeting:        greetingByHour(time.Now().Hour(), name),
		UserDisplayName: name,
		Courses:         defaultCourses(),
		Deadlines:       defaultDeadlines(),
		Activities:      defaultActivities(),
		Weather:         "晴 26°C",
		Motto:           defaultMotto,
		DataSource:      "fallback",
		EmotionRisk:     "none",
	}
}

// LearningDiary 学习日记结构
type LearningDiary struct {
	Date           string                   `json:"date"`
	CoursesStudied []string                 `json:"courses_studied"`
	KeyPoints      []string                 `json:"key_points"`
	StudyMinutes   int                      `json:"study_minutes"`
	Quiz           []map[string]interface{} `json:"quiz"`
	TomorrowPlan   string                   `json:"tomorrow_plan"`
	Encouragement  string                   `json:"encouragement"`
	DataSource     string                   `json:"data_source"` // ai/fallback
}

// GenerateLearningDiary 用 LLM 生成个性化学习日记
func (s *StudentService) GenerateLearningDiary(ctx context.Context, userID int64) (*LearningDiary, error) {
	today := time.Now().Format("2006-01-02")

	// 读取用户最近提问用于个性化
	var recentQs []string
	if s.messageRepo != nil {
		recentQs, _ = s.messageRepo.GetRecentQuestionsByUserID(userID, 10)
	}

	// 尝试 LLM 生成
	if s.llmClient != nil {
		diary, err := s.generateDiaryWithLLM(ctx, today, recentQs)
		if err == nil && diary != nil {
			return diary, nil
		}
	}

	// 兜底
	return s.fallbackDiary(today), nil
}

func (s *StudentService) generateDiaryWithLLM(ctx context.Context, today string, recentQs []string) (*LearningDiary, error) {
	var b strings.Builder
	b.WriteString("你是一个校园 AI 学工助手。请根据学生在校情况生成今日学习日记。\n\n")
	b.WriteString(fmt.Sprintf("日期：%s\n", today))
	if len(recentQs) > 0 {
		b.WriteString("学生最近学习方向：\n")
		for _, q := range recentQs {
			b.WriteString("- " + q + "\n")
		}
	}
	b.WriteString("\n请按以下 JSON 格式输出（严格遵守）：\n")
	b.WriteString(`{
  "courses_studied": ["课程1", "课程2"],
  "key_points": ["知识点1", "知识点2", "知识点3"],
  "study_minutes": 185,
  "quiz": [
    {"question": "题目", "options": ["A选项", "B选项", "C选项", "D选项"], "correct_index": 0, "explanation": "解析"}
  ],
  "tomorrow_plan": "明日计划",
  "encouragement": "鼓励语"
}`)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.4,
		MaxTokens:   800,
	})
	if err != nil || resp.Content == "" {
		return nil, fmt.Errorf("LLM 调用失败")
	}

	diary, err := parseDiaryJSON(resp.Content)
	if err != nil {
		return nil, err
	}
	diary.Date = today
	diary.DataSource = "ai"
	return diary, nil
}

// parseDiaryJSON 从 LLM 响应中提取日记 JSON（处理 markdown 代码块包裹）
func parseDiaryJSON(text string) (*LearningDiary, error) {
	jsonStr := extractJSON(text)

	var parsed struct {
		CoursesStudied []string                 `json:"courses_studied"`
		KeyPoints      []string                 `json:"key_points"`
		StudyMinutes   int                      `json:"study_minutes"`
		Quiz           []map[string]interface{} `json:"quiz"`
		TomorrowPlan   string                   `json:"tomorrow_plan"`
		Encouragement  string                   `json:"encouragement"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("学习日记 JSON 解析失败: %w", err)
	}

	diary := &LearningDiary{
		CoursesStudied: parsed.CoursesStudied,
		KeyPoints:      parsed.KeyPoints,
		StudyMinutes:   parsed.StudyMinutes,
		Quiz:           parsed.Quiz,
		TomorrowPlan:   parsed.TomorrowPlan,
		Encouragement:  parsed.Encouragement,
	}

	// 质量门槛：LLM 输出缺少核心内容时视为失败，交由兜底
	if len(diary.KeyPoints) == 0 && len(diary.Quiz) == 0 {
		return nil, fmt.Errorf("学习日记内容为空")
	}
	if len(diary.Quiz) == 0 {
		// 保证自测题可用
		diary.Quiz = []map[string]interface{}{
			{
				"question":      "二叉树的前序遍历顺序是？",
				"options":       []string{"根→左→右", "左→根→右", "左→右→根", "根→右→左"},
				"correct_index": 0,
				"explanation":   "前序遍历（Preorder）先访问根节点，再递归遍历左子树，最后右子树。",
			},
		}
	}
	return diary, nil
}

func (s *StudentService) fallbackDiary(today string) *LearningDiary {
	return fallbackLearningDiary(today)
}

func defaultCourses() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "数据结构", "subtitle": "第8周 · 二叉树遍历", "time": "08:00-09:40", "icon": "book"},
		{"title": "操作系统", "subtitle": "第8周 · 进程调度", "time": "10:00-11:40", "icon": "computer"},
	}
}

func defaultDeadlines() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "数据结构实验报告", "subtitle": "二叉树实现", "time": "今天 23:59", "icon": "assignment"},
	}
}

func defaultActivities() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "ACM 训练赛", "subtitle": "信息楼 301", "time": "19:00", "icon": "emoji_events"},
	}
}

// ─── 通用 AI 生成 ───

// GenerateAIResponse 为各类学生功能生成 AI 回复
// feature: campus-life, schedule, mental-health, competition-match,
//
//	freshman-plan, growth-path, political-study, ideological-record,
//	party-progress, digital-mentor
func (s *StudentService) GenerateAIResponse(ctx context.Context, feature string, userID int64) map[string]interface{} {
	// 读取用户信息用于个性化
	user, err := s.userRepo.GetByID(userID)
	userName := "同学"
	if err == nil && user != nil {
		userName = user.DisplayName
	}

	prompts := map[string]string{
		"campus-life":         fmt.Sprintf("你是校园生活助手。请为%s推荐今天校内的食堂特色菜品、图书馆空位情况、校车时刻和生活小贴士。约100字。", userName),
		"schedule":            fmt.Sprintf("你是日程管理助手。请为%s根据典型大学生课表生成今日日程安排建议，含课程提醒和空闲时段推荐。约80字。", userName),
		"mental-health":       fmt.Sprintf("你是心理健康关怀助手。请为%s提供一段温暖的心理关怀语和今日放松小建议。语气要温和、不judging。约80字。", userName),
		"competition-match":   fmt.Sprintf("你是竞赛推荐助手。请为%s根据计算机学院学生画像，推荐3个适合参加的竞赛并给出匹配度。约100字。", userName),
		"freshman-plan":       fmt.Sprintf("你是大学规划顾问。请为%s生成大一四阶段（适应/探索/提升/冲刺）的学习生活规划路线图。约120字。", userName),
		"growth-path":         fmt.Sprintf("你是成长路径规划师。请为%s分析当前学业阶段并给出学期里程碑和能力提升建议。约100字。", userName),
		"political-study":     fmt.Sprintf("你是思政学习助手。请为%s生成今日思政学习卡片，含当日学习主题和简短解读。约80字。", userName),
		"ideological-record":  fmt.Sprintf("你是思想档案助手。请为%s生成一段思想成长记录摘要，包含理论学习、志愿服务等方面。约80字。", userName),
		"party-progress":      fmt.Sprintf("你是入党进度追踪助手。请为%s说明入党全流程（申请书→积极分子→发展对象→预备党员→转正）的当前阶段和后续步骤。约100字。", userName),
		"classroom-extension": fmt.Sprintf("你是教学延伸助手。请为%s总结最近课堂的核心要点、生成复习提纲、推荐扩展阅读材料。约100字。", userName),
		"values-guidance":     fmt.Sprintf("你是价值观引导助手。请为%s生成一段自然融入正向价值观（诚信/责任/奉献/感恩）的引导语和建议。约100字。", userName),
	}

	prompt, ok := prompts[feature]
	if !ok {
		return map[string]interface{}{"content": feature, "response": "功能开发中，敬请期待。"}
	}

	if s.llmClient != nil {
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.6,
			MaxTokens:   400,
		})
		if err == nil && resp != nil && resp.Content != "" {
			return map[string]interface{}{
				"content":     feature,
				"response":    resp.Content,
				"data_source": "ai",
			}
		}
	}

	return fallbackAIResponse(feature)
}

func fallbackAIResponse(feature string) map[string]interface{} {
	responses := map[string]string{
		"campus-life":         "今日食堂推荐：一楼麻辣香锅(排队少)、二楼牛肉面(新品上市)。图书馆3楼靠窗区有空位。校车每15分钟一班。",
		"schedule":            "今日日程：8:00-9:40数据结构(信息楼301)、10:00-11:40操作系统(信息楼201)、下午自由学习、19:00ACM训练赛。",
		"mental-health":       "今天也辛苦了！记得给自己一个深呼吸的时间。学习之余可以做5分钟正念冥想，去操场散个步也是很好的放松方式。你不是一个人在努力。",
		"competition-match":   "推荐竞赛：1.ACM程序设计竞赛(匹配度95%) 2.全国大学生数学建模(匹配度85%) 3.蓝桥杯软件赛(匹配度90%)。建议优先准备ACM和蓝桥杯。",
		"freshman-plan":       "大一规划路线图：【适应期9-10月】熟悉校园、建立学习节奏；【探索期11-12月】参加社团、尝试竞赛；【提升期3-5月】重点突破核心课程；【冲刺期6-7月】暑期实习或项目实践。",
		"growth-path":         "你目前处于大二下学期，核心任务是提升算法能力和项目经验。建议本学期完成1个开源项目贡献，暑假争取获得实习机会。同时加强计算机网络和操作系统等核心课程。",
		"political-study":     "今日学习主题：新时代中国特色社会主义思想的核心要义。学习要点：坚持和发展中国特色社会主义的总任务，理解以中国式现代化全面推进中华民族伟大复兴的深刻内涵。",
		"ideological-record":  "思想成长摘要：本学期积极参与志愿服务活动2次，按时完成青年大学习，关注时事政治。整体表现良好，建议继续保持对时事的关注度，增加社会实践经历。",
		"party-progress":      "入党全流程追踪：\n1️⃣申请书已提交✅\n2️⃣确定为入党积极分子（当前阶段）→需完成：参加党校培训、定期思想汇报、志愿服务20小时\n3️⃣确定为发展对象→即将\n4️⃣接收为预备党员→待定\n5️⃣预备党员转正→待定",
		"digital-mentor":      "本周学习建议：重点突破数据结构的图论算法（最短路径、最小生成树）。这是目前你的薄弱环节，也是后续算法竞赛的基础。建议每天刷2道LeetCode图论题巩固。",
		"classroom-extension": "课后复习提纲：1.梳理本节课核心知识点和关键公式 2.完成课后习题并对比答案 3.标记不理解的内容，下次课前向老师提问。扩展推荐：《算法导论》对应章节和MIT公开课对应视频。",
		"values-guidance":     "价值观引导：诚信是做人之本，按时完成任务、考试不作弊是基本底线；责任是成长之基，主动承担班级和团队任务能锻炼能力；奉献是人生之乐，志愿服务让我们在帮助他人中成长；感恩是幸福之源，记住每一次来自师长和同学的帮助。",
	}
	if resp, ok := responses[feature]; ok {
		return map[string]interface{}{"content": feature, "response": resp, "data_source": "fallback"}
	}
	return map[string]interface{}{"content": feature, "response": "功能开发中，敬请期待。", "data_source": "fallback"}
}

// ─── P2 深度分析功能 ───

func (s *StudentService) generateAcademicWarningLegacy(ctx context.Context, userID int64) *AcademicWarning {
	user, err := s.userRepo.GetByID(userID)
	userName := "同学"
	if err == nil && user != nil {
		userName = user.DisplayName
	}

	warning := &AcademicWarning{
		StudentName: userName,
		RiskLevel:   "low",
		RiskScore:   0.12,
		Factors:     []string{"近两周出勤率下降5%", "最近一次作业成绩偏低"},
		Suggestions: []string{"建立每周学习计划", "参加学习小组", "定期与老师沟通学习进度"},
		Resources:   []string{"学习辅导中心", "图书馆自习室", "在线课程资源"},
		DataSource:  "fallback",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是学业预警分析师。学生%s最近出勤和作业有波动。请给出风险等级、风险因素和改进建议。80字以内。", userName)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 300,
		})
		if err == nil && resp != nil && resp.Content != "" {
			warning.Suggestions = append(warning.Suggestions, "AI建议："+resp.Content)
			warning.DataSource = "ai"
		}
	}

	return warning
}

func (s *StudentService) generateMockInterviewLegacy(ctx context.Context, position string) *MockInterview {
	if position == "" {
		position = "Java后端开发工程师"
	}

	interview := &MockInterview{
		Position: position,
		Questions: []map[string]interface{}{
			{"type": "自我介绍", "question": "请做一个简短的自我介绍（1分钟）", "tips": "突出技术栈和项目经验，保持自信"},
			{"type": "技术基础", "question": "请解释TCP三次握手和四次挥手的过程", "tips": "画出时序图，解释每个状态转换"},
			{"type": "项目经验", "question": "介绍一个你最熟悉的项目，遇到了什么技术挑战？", "tips": "使用STAR法则（情境/任务/行动/结果）"},
			{"type": "算法", "question": "实现一个LRU缓存，包含get和put操作", "tips": "使用HashMap+双向链表，O(1)时间复杂度"},
			{"type": "反问", "question": "你对我们团队有什么想问的吗？", "tips": "建议问技术栈/团队规模/成长空间"},
		},
		Tips:       []string{"提前了解公司业务和技术栈", "准备2-3个有亮点的项目经历", "练习白板编程，注意代码规范"},
		Score:      85,
		DataSource: "fallback",
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是大厂面试官。请为「%s」岗位生成5道模拟面试题(含简短答题提示)。输出JSON格式。", position)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.5, MaxTokens: 600,
		})
		if err == nil && resp != nil && resp.Content != "" {
			interview.Tips = append(interview.Tips, "AI补充："+resp.Content[:min(len(resp.Content), 100)])
			interview.DataSource = "ai"
		}
	}

	return interview
}

func (s *StudentService) generateStudyBuddyMatchesLegacy(ctx context.Context, userID int64) *StudyBuddyMatch {
	// 兜底：无 userRepo 时返回占位
	if s.userRepo == nil {
		return studyBuddyFallback()
	}

	me, err := s.userRepo.GetByID(userID)
	if err != nil || me == nil || (me.College == "" && me.Major == "" && me.ClassName == "") {
		// 无法定位本人或无院系资料，走兜底文案
		return studyBuddyFallback()
	}

	// 优先同专业，其次同学院，查询真实同学（排除自己、仅学生角色、活跃账号）
	q := &model.UserQuery{
		Role:   "student",
		Status: "active",
		Limit:  30,
	}
	if me.Major != "" {
		q.Major = me.Major
	} else if me.College != "" {
		q.College = me.College
	}
	peers, _, err := s.userRepo.ListAdvanced(q)
	if err != nil || len(peers) == 0 {
		return studyBuddyFallback()
	}

	matches := make([]map[string]interface{}, 0, 5)
	for _, p := range peers {
		if p.ID == userID {
			continue // 排除自己
		}
		// 打分：同班 +40，同专业 +30，同学院 +15，同年级 +15
		score := 50
		switch {
		case me.ClassName != "" && p.ClassName == me.ClassName:
			score += 40
		case me.Major != "" && p.Major == me.Major:
			score += 30
		case me.College != "" && p.College == me.College:
			score += 15
		}
		if me.EnrollmentYear != "" && p.EnrollmentYear == me.EnrollmentYear {
			score += 15
		}
		if score > 99 {
			score = 99
		}
		reason := "同学院"
		if me.ClassName != "" && p.ClassName == me.ClassName {
			reason = "同班同学"
		} else if me.Major != "" && p.Major == me.Major {
			reason = "同专业"
		}
		matches = append(matches, map[string]interface{}{
			"name":        maskDisplayName(p.DisplayName),
			"match_score": score,
			"major":       p.Major,
			"class_name":  p.ClassName,
			"reason":      reason,
		})
		if len(matches) >= 5 {
			break
		}
	}

	if len(matches) == 0 {
		return studyBuddyFallback()
	}

	// 按 match_score 降序
	sort.Slice(matches, func(i, j int) bool {
		return matches[i]["match_score"].(int) > matches[j]["match_score"].(int)
	})

	return &StudyBuddyMatch{
		Matches:     matches,
		MatchReason: fmt.Sprintf("基于你的专业「%s」与班级，为你匹配了 %d 位可结伴学习的同学（姓名已脱敏保护隐私）。", me.Major, len(matches)),
		DataSource:  "real",
	}
}

func legacyStudyBuddyFallback() *StudyBuddyMatch {
	return &StudyBuddyMatch{
		Matches: []map[string]interface{}{
			{"name": "张*", "match_score": 92, "reason": "同专业", "major": "计算机科学与技术"},
			{"name": "李*", "match_score": 85, "reason": "同学院"},
		},
		MatchReason: "暂无足够的同学资料用于精准匹配，以下为示例。完善院系班级信息后可获得真实推荐。",
		DataSource:  "fallback",
	}
}

// maskDisplayName 姓名脱敏：张* / 张**
func maskDisplayName(name string) string {
	r := []rune(name)
	if len(r) == 0 {
		return "同学"
	}
	if len(r) == 1 {
		return string(r[0]) + "*"
	}
	return string(r[0]) + "**"
}

// MentalHealthReport 心理健康评估报告
type MentalHealthReport struct {
	Date         string   `json:"date"`
	StressLevel  string   `json:"stress_level"`
	EmotionState string   `json:"emotion_state"`
	SocialLevel  string   `json:"social_level"`
	Resilience   string   `json:"resilience"`
	Suggestions  []string `json:"suggestions"`
	DataSource   string   `json:"data_source"`
}

func (s *StudentService) generateMentalHealthReportLegacy(ctx context.Context, userID int64) *MentalHealthReport {
	user, err := s.userRepo.GetByID(userID)
	userName := "同学"
	if err == nil && user != nil {
		userName = user.DisplayName
	}

	report := &MentalHealthReport{
		Date:         time.Now().Format("2006-01-02"),
		StressLevel:  "中等",
		EmotionState: "总体平稳",
		SocialLevel:  "良好",
		Resilience:   "较强",
		Suggestions: []string{
			"建议每天保持30分钟运动",
			"尝试正念冥想来缓解压力",
			"与朋友保持定期社交联系",
			"遇到困难及时向辅导员或心理咨询中心求助",
		},
		DataSource: "fallback",
	}

	// 读取本人最近情感分析记录，基于真实数据推断压力/情绪水平
	if s.emotionRepo != nil {
		logs, err := s.emotionRepo.ListRecentByUser(userID, 20)
		if err == nil && len(logs) > 0 {
			var sum float64
			var high, urgent int
			for _, l := range logs {
				sum += l.Score
				switch l.RiskLevel {
				case "high":
					high++
				case "urgent":
					urgent++
				}
			}
			avg := sum / float64(len(logs))
			// score 区间 -1.0~1.0：越低压力越大
			switch {
			case urgent > 0 || avg <= -0.5:
				report.StressLevel = "偏高"
				report.EmotionState = "近期情绪波动较大"
				report.Resilience = "需关注"
			case high > 0 || avg <= -0.2:
				report.StressLevel = "中等偏上"
				report.EmotionState = "存在一定压力"
				report.Resilience = "尚可"
			case avg >= 0.3:
				report.StressLevel = "较低"
				report.EmotionState = "情绪积极稳定"
				report.Resilience = "较强"
			default:
				report.StressLevel = "中等"
				report.EmotionState = "总体平稳"
				report.Resilience = "较强"
			}
			report.DataSource = "real"

			// 高风险时优先给出求助引导
			if urgent > 0 || high > 0 {
				report.Suggestions = append([]string{
					"近期检测到情绪压力信号，建议尽快联系学校心理咨询中心或辅导员当面沟通",
				}, report.Suggestions...)
			}

			// LLM 基于真实统计做个性化总结
			if s.llmClient != nil {
				prompt := fmt.Sprintf(
					"你是心理健康顾问。学生%s近期%d条情绪记录平均情感分%.2f（-1~1，越低压力越大），高风险%d条、紧急%d条。请用50字内给出温暖、可执行的建议，勿诊断、勿夸大。",
					userName, len(logs), avg, high, urgent,
				)
				resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
					Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
					Temperature: 0.5, MaxTokens: 200,
				})
				if err == nil && resp != nil && resp.Content != "" {
					report.Suggestions = append(report.Suggestions, "AI个性化建议："+resp.Content)
				}
			}
			return report
		}
	}

	// 无情感记录：给通用 LLM 文案（保持原兜底行为）
	if s.llmClient != nil {
		prompt := fmt.Sprintf("你是心理健康顾问。为%s生成简短的心理健康建议（50字）：保持规律作息，适当运动，积极社交。", userName)
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.5, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			report.Suggestions = append(report.Suggestions, "AI个性化建议："+resp.Content)
			report.DataSource = "ai"
		}
	}

	return report
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// mapFlowToResource 把流程类型映射到 KB resource_id
func mapFlowToResource(flowType string) (resourceID string, defaultTitle string) {
	switch flowType {
	case "graduation":
		return "process-graduation-2026", "毕业生离校流程"
	case "major-transfer", "major_transfer", "major_change":
		return "process-major-change-2026", "转专业流程"
	case "student-loan", "student_loan":
		return "process-student-loan-2026", "助学贷款申请流程"
	case "leave":
		return "process-leave-2026", "学生请假办理流程"
	case "scholarship":
		return "process-scholarship-2026", "奖学金申请流程"
	default:
		return "process-registration-2026", "新生入学报到流程"
	}
}

func processStep(order int, title, materials, entryURL, deadline, location, notes string) map[string]interface{} {
	return map[string]interface{}{
		"step":         order,
		"title":        title,
		"status":       "pending",
		"materials":    materials,
		"entry_url":    entryURL,
		"deadline":     deadline,
		"location":     location,
		"notes":        notes,
		"contact":      "",
		"phone":        "",
		"office_hours": "",
		"faq":          []map[string]interface{}{},
	}
}

func fallbackProcessSteps(flowType string) []map[string]interface{} {
	switch flowType {
	case "graduation":
		return []map[string]interface{}{
			processStep(1, "一表通在线申请", "[\"学生证\"]", "http://ybt.chzu.edu.cn/graduation", "6月初开放", "一表通线上系统", "提交离校申请"),
			processStep(2, "图书馆与财务清账", "[\"校园卡\",\"缴费凭证\"]", "", "6月20日前", "图书馆/财务处", "归还图书并结清欠费"),
			processStep(3, "宿舍退宿与校园卡清退", "[\"宿舍钥匙\",\"校园卡\"]", "", "6月25日前", "学生公寓/一卡通中心", "完成宿舍验收和余额清退"),
			processStep(4, "组织关系与档案确认", "[\"党员证\",\"档案确认单\"]", "", "6月25日前", "学院/党委组织部", "党员需转出组织关系"),
			processStep(5, "领取毕业证书", "[\"身份证\",\"学生证\"]", "", "毕业典礼后", "学院党政办", "领取毕业证、学位证和成绩单"),
		}
	case "major-transfer", "major_transfer", "major_change":
		return []map[string]interface{}{
			processStep(1, "了解接收条件", "[]", "http://jwc.chzu.edu.cn", "每年5月/11月", "教务处/学院官网", "查看转入专业条件和名额"),
			processStep(2, "提交申请材料", "[\"转专业申请表\",\"成绩单\",\"个人陈述\"]", "", "第12-14周", "所在学院教学办公室", "填写并提交申请表"),
			processStep(3, "学院与教务处审核", "[\"完整申请表\",\"成绩单\"]", "", "学期末", "所在学院/拟转入学院/教务处", "完成多级审批和公示"),
			processStep(4, "办理学籍变更", "[]", "http://jwc.chzu.edu.cn", "公示期满后", "教务处学籍科", "完成学籍信息变更"),
		}
	case "student-loan", "student_loan":
		return []map[string]interface{}{
			processStep(1, "网上申请", "[]", "https://sls.cdb.com.cn", "7月-9月", "国家开发银行学生在线系统", "注册并填写贷款申请"),
			processStep(2, "打印并认定申请表", "[\"申请表\"]", "", "7月-9月", "户籍地村居/乡镇", "完成家庭经济困难认定"),
			processStep(3, "现场签订合同", "[\"身份证\",\"录取通知书/学生证\",\"户口簿\"]", "", "7月-9月", "县区学生资助中心", "学生和共同借款人到场办理"),
			processStep(4, "学校回执录入", "[\"受理证明\"]", "", "开学后一周内", "学校学生资助中心", "提交回执并等待贷款发放"),
		}
	case "leave":
		return []map[string]interface{}{
			processStep(1, "提交请假申请", "[\"请假事由说明\",\"证明材料（如病假证明）\"]", "", "离校前提交", "辅导员/学院线上表单", "说明请假时间、去向和联系方式"),
			processStep(2, "辅导员审核", "[]", "", "提交后1个工作日内", "辅导员办公室", "辅导员核实请假原因和安全去向"),
			processStep(3, "学院审批", "[\"请假申请表\"]", "", "按学院要求", "学院学生工作办公室", "超过规定天数需学院审批"),
			processStep(4, "销假返校", "[]", "", "返校当日", "辅导员/班级群", "返校后及时销假并更新在校状态"),
		}
	case "scholarship":
		return []map[string]interface{}{
			processStep(1, "查看评选通知", "[]", "", "每学年评选期", "学院官网/班级群", "确认奖项类别、名额和申请条件"),
			processStep(2, "准备申请材料", "[\"申请表\",\"成绩单\",\"荣誉证明\",\"综测材料\"]", "", "通知规定时间内", "所在学院", "按奖项要求准备纸质或电子材料"),
			processStep(3, "班级评议与学院审核", "[\"完整申请材料\"]", "", "学院评审期", "班级/学院学生工作办公室", "完成民主评议、学院初审和排序"),
			processStep(4, "公示与学校审定", "[]", "", "公示期", "学院/学校官网", "公示无异议后报学校审定"),
			processStep(5, "发放与归档", "[\"银行卡信息\"]", "", "学校审定后", "财务处/学院", "奖助资金发放并完成材料归档"),
		}
	default:
		return []map[string]interface{}{
			processStep(1, "线上预报到", "[\"录取通知书\",\"身份证\"]", "https://yx.chzu.edu.cn", "报到前完成", "迎新系统", "完成个人信息确认和到校信息登记"),
			processStep(2, "缴纳学杂费", "[\"银行卡\",\"缴费凭证\"]", "http://cw.chzu.edu.cn", "报到前或报到日", "财务系统/现场缴费点", "助学贷款学生携带贷款回执"),
			processStep(3, "学院报到", "[\"录取通知书\",\"身份证\",\"档案\"]", "", "报到日", "计算机学院报到点", "领取班级、辅导员和校园卡信息"),
			processStep(4, "宿舍入住", "[\"校园卡\",\"身份证\"]", "", "报到日", "学生公寓", "按分配宿舍领取钥匙并入住"),
			processStep(5, "入学体检与学籍核验", "[\"身份证\",\"体检表\"]", "", "入学后两周内", "校医院/教务处", "按学院通知分批完成"),
		}
	}
}

// FreshmenSourceFile 新生指南引用的官方原文。
type FreshmenSourceFile struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Note  string `json:"note"`
}

// FreshmenGuide 新生指南聚合结果。
type FreshmenGuide struct {
	Guide       *model.KBResource    `json:"guide,omitempty"`
	Handbook    *model.KBResource    `json:"handbook,omitempty"`
	Zzsb        *model.KBResource    `json:"zzsb,omitempty"`
	Process     *model.KBResource    `json:"process,omitempty"`
	Steps       []*model.ProcessStep `json:"steps"`
	SourceFiles []FreshmenSourceFile `json:"source_files"`
}

// GetFreshmenGuide 聚合新生指南知识资源与报到步骤。
func (s *StudentService) GetFreshmenGuide() (*FreshmenGuide, error) {
	guide := &FreshmenGuide{
		Steps: []*model.ProcessStep{},
		SourceFiles: []FreshmenSourceFile{
			{
				Title: "2026级普通本科、对口本科新生入学指南",
				Path:  "data/滁州学院2026级普通本科、对口本科新生入学指南.pdf",
				Note:  "官方 PDF",
			},
			{
				Title: "2026级普通专升本新生入学须知",
				Path:  "data/滁州学院2026级普通专升本新生入学须知.pdf",
				Note:  "官方 PDF（扫描件）",
			},
			{
				Title: "2025年学生手册正文",
				Path:  "data/250827-2025年学生手册正文（定稿）终版.docx",
				Note:  "官方 DOCX",
			},
		},
	}
	if s.kbRepo == nil {
		return guide, nil
	}
	if kb, err := s.kbRepo.GetByResourceID("guide-freshmen-2026"); err == nil {
		guide.Guide = kb
	}
	if kb, err := s.kbRepo.GetByResourceID("policy-student-handbook-2025"); err == nil {
		guide.Handbook = kb
	}
	if kb, err := s.kbRepo.GetByResourceID("guide-freshmen-2026-zzsb"); err == nil {
		guide.Zzsb = kb
	}
	if kb, err := s.kbRepo.GetByResourceID("process-registration-2026"); err == nil {
		guide.Process = kb
	}
	if steps, err := s.kbRepo.GetProcessSteps("process-registration-2026"); err == nil {
		if steps == nil {
			steps = []*model.ProcessStep{}
		}
		guide.Steps = steps
	}
	return guide, nil
}

// GetProcessEnhanced AI 办事流程增强 — 按 type 参数从 KB + process_steps 拼装真实数据
// type: enrollment（入学）/ graduation（离校）/ major_change（转专业）/ student_loan（助学贷款）/ leave（请假）/ scholarship（奖学金）
func (svc *StudentService) GetProcessEnhanced(flowType string, userOwnerScope, userOwnerID string) (*model.KBResource, []map[string]interface{}, *model.AnswerCard, error) {
	resourceID, flowTitle := mapFlowToResource(flowType)

	var card *model.AnswerCard
	var kb *model.KBResource
	steps := []map[string]interface{}{}

	if svc.kbRepo != nil {
		var err error
		kb, err = svc.kbRepo.GetByResourceID(resourceID)
		if err == nil && kb != nil {
			flowTitle = kb.Title
			card = &model.AnswerCard{
				Conclusion: kb.Summary,
				Sources: []model.Source{{
					ResourceID:   kb.ResourceID,
					Title:        kb.Title,
					ResourceType: kb.ResourceType,
					Version:      kb.Version,
					SourceLink:   kb.SourceLink,
					EffectiveAt:  kb.EffectiveAt,
					Snippet:      kb.Summary,
				}},
			}
		}
		if rows, err := svc.kbRepo.GetProcessSteps(resourceID); err == nil {
			for _, ps := range rows {
				materials := ""
				if ps.Materials != "" && ps.Materials != "[]" {
					var parsed []string
					if err := json.Unmarshal([]byte(ps.Materials), &parsed); err == nil {
						b, _ := json.Marshal(parsed)
						materials = string(b)
					} else {
						materials = ps.Materials
					}
				}
				var faqList []map[string]interface{}
				if ps.FAQ != "" && ps.FAQ != "[]" {
					if err := json.Unmarshal([]byte(ps.FAQ), &faqList); err != nil {
						faqList = []map[string]interface{}{}
					}
				} else {
					faqList = []map[string]interface{}{}
				}
				// 媒体资源 JSON 数组解析（办理指引图/视频）
				mediaList := []string{}
				if ps.MediaURLs != "" && ps.MediaURLs != "[]" {
					if err := json.Unmarshal([]byte(ps.MediaURLs), &mediaList); err != nil {
						mediaList = []string{}
					}
				}
				stepMap := map[string]interface{}{
					"step":           ps.StepOrder,
					"title":          ps.Title,
					"status":         "pending",
					"materials":      materials,
					"entry_url":      ps.EntryURL,
					"deadline":       ps.Deadline,
					"location":       ps.Location,
					"notes":          ps.Notes,
					"contact":        ps.Contact,
					"phone":          ps.Phone,
					"contact_wechat": ps.ContactWechat,
					"office_hours":   ps.OfficeHours,
					"media_urls":     mediaList,
					"faq":            faqList,
				}
				// 地理坐标仅在已录入（非 0）时透出，避免前端把 0,0 当作有效坐标定位到几内亚湾
				if ps.GeoLat != 0 || ps.GeoLng != 0 {
					stepMap["geo_lat"] = ps.GeoLat
					stepMap["geo_lng"] = ps.GeoLng
				}
				steps = append(steps, stepMap)
			}
		}
	}
	if len(steps) == 0 {
		steps = fallbackProcessSteps(flowType)
	}
	if card == nil {
		card = &model.AnswerCard{
			Conclusion: flowTitle + "已整理，请按下列步骤办理。",
			Fallback:   true,
			Confidence: 0.5,
		}
	}

	return kb, steps, card, nil
}

// 注：成长路径已升级为真实五维数据版 GenerateGrowthPath（见下方 S1 功能7 实现，返回 *GrowthPathResult）

// ─── P2 专业知识图谱 + 笔记助手 + 简历生成 ───

// ======================== P1 剩余方法 ========================

func (s *StudentService) generateWeeklyReportLegacy(ctx context.Context, userID int64) *WeeklyReportData {
	weekNum := int(time.Now().YearDay()/7) + 1
	data := &WeeklyReportData{
		Week:          fmt.Sprintf("第%d周", weekNum),
		TotalHours:    22.5,
		CoursesCount:  5,
		Assignments:   3,
		RankChange:    2,
		Highlights:    []string{"数据结构实验满分", "英语演讲获得A"},
		Improvements:  []string{"操作系统作业需加强", "体育锻炼不足"},
		NextWeekGoals: []string{"完成算法作业", "准备期中考试"},
		TimeDistribution: map[string]float64{
			"上课": 15.0, "自习": 4.5, "实验": 2.0, "运动": 1.0,
		},
		KnowledgeChanges: []map[string]interface{}{
			{"course": "数据结构", "change": "+12%", "trend": "up", "detail": "树和图相关知识点掌握度提升"},
			{"course": "操作系统", "change": "-5%", "trend": "down", "detail": "内存管理章节理解不足"},
		},
		DataSource: "reference",
	}

	// 真实交互活跃度（近 7 天提问/会话/活跃天数）——覆盖模板值
	if s.messageRepo != nil {
		if wa, err := s.messageRepo.GetWeeklyActivity(userID, 7); err == nil && wa != nil && wa.Questions > 0 {
			data.QuestionsAsked = wa.Questions
			data.ActiveDays = wa.ActiveDays
			data.SessionsCount = wa.Sessions
			data.DataSource = "real"
		}
	}

	if s.llmClient != nil {
		prompt := fmt.Sprintf("学生本周与学工助手交互：提问%d次、活跃%d天、会话%d个。亮点：%s，不足：%s。请用40字做学习状态归因分析，只依据以上数据、不得编造。",
			data.QuestionsAsked, data.ActiveDays, data.SessionsCount,
			strings.Join(data.Highlights, "、"), strings.Join(data.Improvements, "、"))
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.3, MaxTokens: 200,
		})
		if err == nil && resp != nil && resp.Content != "" {
			data.Attribution = strings.TrimSpace(resp.Content)
			if data.DataSource == "real" {
				data.DataSource = "real+ai"
			} else {
				data.DataSource = "ai"
			}
		}
	}

	return data
}

func (s *StudentService) generateQAPlazaLegacy(ctx context.Context) *QAPlazaData {
	// 优先用真实已发布 FAQ 资源作为问答广场热门问题（结构化优先，可追溯）
	if s.kbRepo != nil {
		faqs, err := s.kbRepo.List("", "", "published", "FAQ", 0, 8)
		if err == nil && len(faqs) > 0 {
			hot := make([]map[string]interface{}, 0, len(faqs))
			for _, f := range faqs {
				ans := f.Summary
				if strings.TrimSpace(ans) == "" {
					ans = f.Content
				}
				if len([]rune(ans)) > 120 {
					ans = string([]rune(ans)[:120]) + "…"
				}
				hot = append(hot, map[string]interface{}{
					"id":          f.ResourceID,
					"title":       f.Title,
					"author":      "知识库",
					"answers":     1,
					"views":       0,
					"ai_answer":   ans,
					"tags":        parseKnowledgeTags(f.Tags),
					"source_link": f.SourceLink,
				})
			}
			return &QAPlazaData{
				HotQuestions: hot,
				Categories:   []string{"学业", "生活", "政策", "心理", "就业", "竞赛"},
				MyPosts:      0, MyAnswers: 0,
				DataSource: "real",
			}
		}
	}
	// 兜底：知识库暂无已发布 FAQ 时返回示例数据，保证前端可用
	return fallbackQAPlaza()
}

func (s *StudentService) generateHotTopicsLegacy(ctx context.Context) *HotTopicsData {
	// 优先用最近已发布的 Activity 资源作为校园热点（按 List 默认 updated_at 倒序取前若干）
	if s.kbRepo != nil {
		acts, err := s.kbRepo.List("", "", "published", "Activity", 0, 6)
		if err == nil && len(acts) > 0 {
			topics := make([]map[string]interface{}, 0, len(acts))
			// 热度按新近程度递减模拟（首条最热），趋势首两条 rising 其余 stable
			heat := 95
			for i, a := range acts {
				summary := a.Summary
				if strings.TrimSpace(summary) == "" {
					summary = a.Title
				}
				if len([]rune(summary)) > 60 {
					summary = string([]rune(summary)[:60]) + "…"
				}
				trend := "stable"
				if i < 2 {
					trend = "rising"
				}
				topics = append(topics, map[string]interface{}{
					"id":          a.ResourceID,
					"title":       a.Title,
					"heat":        heat,
					"trend":       trend,
					"summary":     summary,
					"source_link": a.SourceLink,
				})
				heat -= 10
				if heat < 40 {
					heat = 40
				}
			}
			return &HotTopicsData{
				Topics:     topics,
				UpdatedAt:  time.Now().Format("2006-01-02 15:04"),
				DataSource: "real",
			}
		}
	}
	// 兜底：知识库暂无已发布 Activity 时返回示例数据
	return fallbackHotTopics()
}

// QALeaderboardData 问答排行榜
type QALeaderboardData struct {
	HotQuestions []map[string]interface{} `json:"hot_questions"`
	TopAnswerers []map[string]interface{} `json:"top_answerers"`
	Contributors []map[string]interface{} `json:"contributors"`
	Period       string                   `json:"period"`
	DataSource   string                   `json:"data_source"`
}

func (s *StudentService) GenerateQALeaderboard(ctx context.Context) *QALeaderboardData {
	data := referenceQALeaderboard()

	// 热门提问：来自真实 messages 表的聚合统计
	if s.messageRepo != nil {
		if hot, err := s.messageRepo.GetHotQuestions(10); err == nil && len(hot) > 0 {
			questions := make([]map[string]interface{}, 0, len(hot))
			for i, h := range hot {
				title := h.Title
				if len([]rune(title)) > 40 {
					title = string([]rune(title)[:40]) + "…"
				}
				questions = append(questions, map[string]interface{}{
					"rank":  i + 1,
					"title": title,
					"count": h.Count, // 真实被提问次数
				})
			}
			data.HotQuestions = questions
			data.DataSource = "real" // 热榜为真实数据；答主榜仍为参考
			return data
		}
	}

	// 无真实提问数据时的参考样例
	return data
}

// ======================== P3 生态扩展 ========================

// EnhancedCareerSimulation 职业模拟器增强版（数据驱动仿真）
type EnhancedCareerSimulation struct {
	CareerPath          string                   `json:"career_path"`
	CurrentStage        string                   `json:"current_stage"`
	Stages              []map[string]interface{} `json:"stages"`
	SkillsGap           []map[string]interface{} `json:"skills_gap"`
	SalaryProjection    []map[string]interface{} `json:"salary_projection"`
	MarketTrends        []string                 `json:"market_trends"`
	AlternativePathways []string                 `json:"alternative_pathways"`
	AIAdvice            string                   `json:"ai_advice"`
	DataSource          string                   `json:"data_source"`
}

// ── 课程学情看板（S1 功能5）──

// CourseAnalyticsItem 单门课程学情
type CourseAnalyticsItem struct {
	CourseName     string  `json:"course_name"`
	Semester       string  `json:"semester"`
	Score          float64 `json:"score"`
	GPA            float64 `json:"gpa"`
	GradeLevel     string  `json:"grade_level"`
	Credits        float64 `json:"credits"`
	Passed         bool    `json:"passed"`
	RankPercentile int     `json:"rank_percentile"` // 班级匿名百分位（越小越靠前，0=未知）
}

// CourseAnalyticsResult 课程学情看板结果
type CourseAnalyticsResult struct {
	UserDisplayName string                 `json:"user_display_name"`
	ClassName       string                 `json:"class_name"`
	OverallGPA      float64                `json:"overall_gpa"`
	ClassAvgGPA     float64                `json:"class_avg_gpa"`
	ClassSize       int                    `json:"class_size"`
	CreditsEarned   float64                `json:"credits_earned"`
	Courses         []*CourseAnalyticsItem `json:"courses"`
	WeakCourses     []string               `json:"weak_courses"` // 未通过或分数偏低的课程
	Advice          string                 `json:"advice"`       // LLM 个性化学业建议
	DataSource      string                 `json:"data_source"`  // real/fallback
}

// GenerateCourseAnalytics 生成课程学情看板（真实成绩 + 班级匿名基准 + LLM 薄弱点建议）
// 无成绩数据时返回 (nil, nil)，由 handler 回落 mock。
func (s *StudentService) generateCourseAnalyticsLegacy(ctx context.Context, userID int64) (*CourseAnalyticsResult, error) {
	if s.twinRepo == nil || s.userRepo == nil {
		return nil, nil
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, nil
	}
	grades, err := s.twinRepo.ListCourseGrades(userID)
	if err != nil {
		return nil, err
	}
	if len(grades) == 0 {
		return nil, nil // 无真实成绩，交由 handler 兜底
	}

	basis, _ := s.twinRepo.GetClassBasis(user.ClassName)

	res := &CourseAnalyticsResult{
		UserDisplayName: user.DisplayName,
		ClassName:       user.ClassName,
		Courses:         make([]*CourseAnalyticsItem, 0, len(grades)),
		WeakCourses:     []string{},
		DataSource:      "real",
	}
	if basis != nil {
		res.ClassAvgGPA = basis.ClassAvgGPA
		res.ClassSize = basis.ClassSize
	}

	var gpaSum float64
	for _, g := range grades {
		item := &CourseAnalyticsItem{
			CourseName: g.CourseName,
			Semester:   g.Semester,
			Score:      g.Score,
			GPA:        g.GPA,
			GradeLevel: g.GradeLevel,
			Credits:    g.Credits,
			Passed:     g.Passed,
		}
		res.Courses = append(res.Courses, item)
		gpaSum += g.GPA
		if g.Passed {
			res.CreditsEarned += g.Credits
		}
		// 薄弱课程：未通过或分数 < 70
		if !g.Passed || g.Score < 70 {
			res.WeakCourses = append(res.WeakCourses, g.CourseName)
		}
	}
	if len(grades) > 0 {
		res.OverallGPA = gpaSum / float64(len(grades))
	}

	// LLM 生成薄弱点学业建议（失败不阻断，返回无 Advice 的真实数据）
	res.Advice = s.buildCourseAdvice(ctx, res)
	return res, nil
}

// ── 成长路径（S1 功能7）──

// GrowthMilestone 成长路径里程碑
type GrowthMilestone struct {
	Stage   string   `json:"stage"`   // 阶段名（如「本学期」「大三上」）
	Focus   string   `json:"focus"`   // 重点方向
	Actions []string `json:"actions"` // 关键行动
	Done    bool     `json:"done"`    // 是否已达成
}

// GrowthPathResult 成长路径结果
type GrowthPathResult struct {
	UserDisplayName string             `json:"user_display_name"`
	CurrentStage    string             `json:"current_stage"`  // 当前学业阶段
	AcademicScore   float64            `json:"academic_score"` // 五维·学业
	AbilityScore    float64            `json:"ability_score"`  // 五维·能力
	StrongestDim    string             `json:"strongest_dim"`  // 最强维度
	WeakestDim      string             `json:"weakest_dim"`    // 最弱维度
	Milestones      []*GrowthMilestone `json:"milestones"`     // 分阶段路线图
	Summary         string             `json:"summary"`        // LLM 个性化总结
	DataSource      string             `json:"data_source"`    // real/fallback
}

// GenerateGrowthPath 生成成长路径（基于数字孪生五维快照 + 学业阶段 → 分阶段里程碑 + LLM 总结）
// 无快照数据时返回 (nil, nil)，由 handler 回落通用 AI 文案。
func (s *StudentService) generateGrowthPathLegacy(ctx context.Context, userID int64) (*GrowthPathResult, error) {
	if s.twinRepo == nil || s.userRepo == nil {
		return nil, nil
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, nil
	}
	snap, err := s.twinRepo.GetSnapshot(userID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, nil // 无孪生快照，交 handler 兜底（可先访问 /student/digital-twin 生成）
	}

	// 找最强/最弱维度
	dims := map[string]float64{
		"学业": snap.AcademicScore, "能力": snap.AbilityScore, "思想": snap.IdeologicalScore,
		"情感": snap.EmotionalScore, "社交": snap.SocialScore,
	}
	strongest, weakest := "学业", "学业"
	for k, v := range dims {
		if v > dims[strongest] {
			strongest = k
		}
		if v < dims[weakest] {
			weakest = k
		}
	}

	res := &GrowthPathResult{
		UserDisplayName: user.DisplayName,
		CurrentStage:    inferAcademicStage(user.EnrollmentYear),
		AcademicScore:   snap.AcademicScore,
		AbilityScore:    snap.AbilityScore,
		StrongestDim:    strongest,
		WeakestDim:      weakest,
		DataSource:      "real",
	}
	res.Milestones = buildGrowthMilestones(weakest)
	res.Summary = s.buildGrowthSummary(ctx, user.DisplayName, res, weakest)
	return res, nil
}

// inferAcademicStage 根据入学年份推断当前学业阶段（简化：按自然年差）
func inferAcademicStage(enrollmentYear string) string {
	if enrollmentYear == "" {
		return "在读阶段"
	}
	stages := []string{"大一", "大二", "大三", "大四"}
	yr := time.Now().Year()
	var ey int
	fmt.Sscanf(enrollmentYear, "%d", &ey)
	if ey == 0 {
		return "在读阶段"
	}
	idx := yr - ey
	if idx < 0 {
		idx = 0
	}
	if idx >= len(stages) {
		return "毕业年级"
	}
	return stages[idx]
}

// buildGrowthMilestones 依据最弱维度给出分阶段路线图（规则模板，稳定可兜底）
func buildGrowthMilestones(weakest string) []*GrowthMilestone {
	base := []*GrowthMilestone{
		{Stage: "本学期", Focus: "夯实基础", Actions: []string{"稳定核心课程成绩", "确定一个能力提升方向"}, Done: false},
		{Stage: "下学期", Focus: "能力拓展", Actions: []string{"参加一项竞赛或项目", "积累实践/志愿经历"}, Done: false},
		{Stage: "长期", Focus: "目标冲刺", Actions: []string{"锁定升学或就业目标", "补齐关键短板"}, Done: false},
	}
	switch weakest {
	case "能力":
		base[0].Actions = append(base[0].Actions, "报名一项学科竞赛")
	case "思想":
		base[0].Actions = append(base[0].Actions, "参与思政学习与志愿服务")
	case "情感":
		base[0].Actions = append(base[0].Actions, "关注作息与心理调适，善用心理陪伴")
	case "社交":
		base[0].Actions = append(base[0].Actions, "加入一个社团或团队协作项目")
	}
	return base
}

// buildGrowthSummary 用真实五维数据生成个性化成长总结；LLM 不可用时规则兜底
func (s *StudentService) buildGrowthSummary(ctx context.Context, name string, res *GrowthPathResult, weakest string) string {
	fallback := fmt.Sprintf("%s当前处于%s，最强项是%s，建议本阶段重点补齐%s维度。", name, res.CurrentStage, res.StrongestDim, weakest)
	if s.llmClient == nil {
		return fallback
	}
	prompt := fmt.Sprintf(
		"你是成长规划师。学生%s当前%s，五维画像中最强为%s、最弱为%s（学业分%.0f、能力分%.0f）。请给出约90字的成长路径建议，聚焦补齐短板、鼓励语气。",
		name, res.CurrentStage, res.StrongestDim, weakest, res.AcademicScore, res.AbilityScore)
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.6, MaxTokens: 350,
	})
	if err == nil && resp != nil && strings.TrimSpace(resp.Content) != "" {
		return strings.TrimSpace(resp.Content)
	}
	return fallback
}

// buildCourseAdvice 基于薄弱课程生成个性化学业建议；LLM 不可用时给规则兜底文案
func (s *StudentService) buildCourseAdvice(ctx context.Context, res *CourseAnalyticsResult) string {
	if len(res.WeakCourses) == 0 {
		return "各科成绩稳定，继续保持。可尝试挑战竞赛或选修拓展课程。"
	}
	if s.llmClient == nil {
		return fmt.Sprintf("重点关注薄弱课程：%s。建议制定复习计划、多做真题并及时向任课教师答疑。", strings.Join(res.WeakCourses, "、"))
	}
	prompt := fmt.Sprintf(
		"你是学业辅导老师。一名学生当前均绩 %.2f（班级平均 %.2f），薄弱课程为：%s。请给出约80字的针对性提升建议，语气鼓励。",
		res.OverallGPA, res.ClassAvgGPA, strings.Join(res.WeakCourses, "、"))
	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.5, MaxTokens: 300,
	})
	if err == nil && resp != nil && strings.TrimSpace(resp.Content) != "" {
		return strings.TrimSpace(resp.Content)
	}
	return fmt.Sprintf("重点关注薄弱课程：%s。建议制定复习计划、多做真题并及时向任课教师答疑。", strings.Join(res.WeakCourses, "、"))
}
