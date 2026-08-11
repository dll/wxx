// Package auth 实现基于"角色继承 + 能力授权"的统一权限模型
//
// 设计要点：
// 1. Capability（能力）是端点的最小授权单元，命名规范为 domain.action（如 self.briefing.read）
// 2. 每个角色绑定一组直接 capability，并指定一个父角色实现线性继承
// 3. HasCapability 通过递归向上查询角色继承链，让高阶角色自动获得低阶能力
// 4. 与 owner_scope/owner_id 数据范围正交：能力解决"能不能做"，scope 解决"能看哪部分数据"
package auth

// Capability 能力标识符。命名规范：<domain>.<action>，全部小写下划线
type Capability string

// 个人能力（每个登录用户都该有）
const (
	SelfGuestRead       Capability = "self.guest.read"       // 游客浏览公开信息
	SelfBriefingRead    Capability = "self.briefing.read"    // 今日速览
	SelfDiaryRead       Capability = "self.diary.read"       // 学习日记
	SelfCheckinWrite    Capability = "self.checkin.write"    // 每日打卡
	SelfTwinRead        Capability = "self.twin.read"        // 个人数字孪生
	SelfTwinWrite       Capability = "self.twin.write"       // 数字孪生画像生成（文生图/图生图）
	SelfPersonalityRead Capability = "self.personality.read" // 性格洞察
	SelfFeedbackSubmit  Capability = "self.feedback.submit"  // 提交问题反馈
	SelfProfileWrite    Capability = "self.profile.write"    // 编辑个人资料
	SelfChat            Capability = "self.chat"             // AI 对话
	SelfKnowledgeRead   Capability = "self.knowledge.read"   // 浏览知识库
	SelfVoice           Capability = "self.voice"            // 语音功能
	SelfSessionRead     Capability = "self.session.read"     // 自己的会话
	SelfSessionDelete   Capability = "self.session.delete"   // 删除自己的会话
	SelfRecommendRead   Capability = "self.recommend.read"   // 个性化推荐
	SelfExportSelf      Capability = "self.export.self"      // 导出自己的回答
	SelfAchievements    Capability = "self.achievements"     // 个人积分/成就
	SelfCourseMapRead   Capability = "self.course_map.read"  // 课程地图
	SelfCourseAnalytics Capability = "self.course.analytics" // 个人课程学情
	SelfWeeklyReport    Capability = "self.weekly.report"    // 学习周报
	SelfGenericAI       Capability = "self.generic_ai"       // 通用 AI 功能（思政/成长/竞赛等）
	SelfCommunityRead   Capability = "self.community.read"   // 问答广场/热点/排行榜
	SelfPrivateChat     Capability = "self.private_chat"     // 站内私聊
	SelfProcessRead     Capability = "self.process.read"     // 流程办理增强查看
	SelfEmotionStats    Capability = "self.emotion.stats"    // 自身情感统计
	SelfEmotionConsent  Capability = "self.emotion.consent"  // 情感分析独立授权（区别于通用 consent）
	SelfTokenStats      Capability = "self.token.stats"      // 词元统计
	SelfAIBriefingRead  Capability = "self.ai_briefing.read" // 浏览 AI 简讯（首页资讯）

	// 校园文化（全员可见，骨架阶段直接归到 self.* 基线）
	SelfCultureAnthem    Capability = "self.culture.anthem"    // 校歌曲库
	SelfCultureRadio     Capability = "self.culture.radio"     // 校园广播
	SelfCultureLectures  Capability = "self.culture.lectures"  // 学术讲座
	SelfCultureEvents    Capability = "self.culture.events"    // 校园活动
	SelfCultureVolunteer Capability = "self.culture.volunteer" // 志愿服务

	// ── 学生功能模块 ──
	SelfGraduationRead   Capability = "self.graduation.read"   // 毕设选题查看
	SelfGraduationWrite  Capability = "self.graduation.write"  // 毕设选题操作
	SelfCompetitionRead  Capability = "self.competition.read"  // 学科竞赛查看
	SelfCompetitionWrite Capability = "self.competition.write" // 学科竞赛报名
	SelfPlanRead         Capability = "self.plan.read"         // 大学规划查看
	SelfPlanWrite        Capability = "self.plan.write"        // 大学规划操作
	SelfPartyRead        Capability = "self.party.read"        // 入党教育查看
	SelfPartyWrite       Capability = "self.party.write"       // 入党教育操作
	SelfClubRead         Capability = "self.club.read"         // 社团生活查看
	SelfClubWrite        Capability = "self.club.write"        // 社团生活操作

	// ── 就业指导模块 ──
	SelfCareerRead  Capability = "self.career.read"  // 就业指导查看（政策、招聘、宣讲会、面试题）
	SelfCareerWrite Capability = "self.career.write" // 就业指导操作

	// ── 学业学习模块 ──
	SelfStudyRead  Capability = "self.study.read"  // 学业学习查看（课程、成绩、资源、考试）
	SelfStudyWrite Capability = "self.study.write" // 学业学习操作

	// ── 心理健康模块 ──
	SelfMentalRead  Capability = "self.mental.read"  // 心理健康查看（量表、咨询师、文章、热线）
	SelfMentalWrite Capability = "self.mental.write" // 心理健康操作（测评、预约、情绪日记）

	// ── 身体健康模块 ──
	SelfHealthRead  Capability = "self.health.read"  // 身体健康查看（基本信息、体检记录、病历记录）
	SelfHealthWrite Capability = "self.health.write" // 身体健康操作（新增/编辑/删除）
)

// 学生会能力
const (
	UnionKBSubmit      Capability = "union.kb.submit"      // 知识库提交
	UnionFeedbackList  Capability = "union.feedback.list"  // 反馈列表查看
	UnionFeedbackRead  Capability = "admin.feedback.read"  // 反馈统计查看（复用 admin 前缀，学生会及以上可用）
	UnionFeedbackWrite Capability = "admin.feedback.write" // 反馈关联知识等管理操作
	UnionEventPlan     Capability = "union.event.plan"     // 活动策划
	UnionPosterGen     Capability = "union.poster.gen"     // 海报生成
)

// 辅导员能力
const (
	CounselorDailyFocusRead    Capability = "counselor.daily_focus.read"   // 今日关注
	CounselorClassRead         Capability = "counselor.class.read"         // 班级看板
	CounselorAlertRead         Capability = "counselor.alert.read"         // 情感预警
	CounselorAlertHandle       Capability = "counselor.alert.handle"       // 处理告警
	CounselorAlertAnalyze      Capability = "counselor.alert.analyze"      // 触发情感分析
	CounselorEmotionTrends     Capability = "counselor.emotion.trends"     // 情感趋势
	CounselorKBWrite           Capability = "counselor.kb.write"           // 知识库 CRUD
	CounselorKBReview          Capability = "counselor.kb.review"          // 知识审核
	CounselorInterventionWrite Capability = "counselor.intervention.write" // 干预方案
	CounselorTalkRecord        Capability = "counselor.talk.record"        // 谈心记录
	CounselorPredictionRead    Capability = "counselor.prediction.read"    // 预测性预警
	CounselorIntegrationRead   Capability = "counselor.integration.read"   // 校外系统对接
	CounselorReviewPending     Capability = "counselor.review.pending"     // 待审核知识列表
	CounselorClassReport       Capability = "counselor.class.report"       // 班级学情日报
	CounselorTwinBoard         Capability = "counselor.twin.board"         // 学生数字孪生看板
	CounselorTalkTips          Capability = "counselor.talk.tips"          // 谈话话术
	CounselorIdeological       Capability = "counselor.ideological"        // 思想动态
	CounselorClassProfile      Capability = "counselor.class.profile"      // 班级画像
	CounselorCommunityManage   Capability = "counselor.community.manage"   // 社区问答管理
	CounselorHotTopicSense     Capability = "counselor.hot_topic.sense"    // 热点话题感知
	CounselorProcessEdit       Capability = "counselor.process.edit"       // 流程步骤编辑
	CounselorStudentList       Capability = "counselor.student.list"       // 学生列表
	CounselorTokenSubordinates Capability = "counselor.token.subordinates"
	CounselorImportStudent     Capability = "counselor.import.student" // 导入学生（学生会及以上角色继承）
	CounselorNotify            Capability = "counselor.notify"         // 通知管理（创建/发布/删除）
)

// 教师能力
const (
	TeacherLessonPrep    Capability = "teacher.lesson.prep"    // 备课助手
	TeacherExamGen       Capability = "teacher.exam.gen"       // 考试出题
	TeacherHeatmapRead   Capability = "teacher.heatmap.read"   // 班级学情热力图
	TeacherClassInteract Capability = "teacher.class.interact" // 课堂互动
	TeacherDailyOverview Capability = "teacher.daily.overview" // 今日授课概览
	TeacherGrading       Capability = "teacher.grading"        // 作业批改辅助
	TeacherReflection    Capability = "teacher.reflection"     // 教学反思
	TeacherStyleDist     Capability = "teacher.style.dist"     // 学生学习风格分布
	TeacherCommunityQA   Capability = "teacher.community.qa"   // 社区专业答疑
)

// 教辅能力
const (
	AssistantScheduleCheck Capability = "assistant.schedule.check" // 排课冲突
	AssistantGradAudit     Capability = "assistant.grad.audit"     // 毕业审核
	AssistantExamArrange   Capability = "assistant.exam.arrange"   // 考试安排
)

// 学院管理能力
const (
	CollegeUserRead        Capability = "college.user.read"        // 本院用户管理
	CollegeAuditRead       Capability = "college.audit.read"       // 本院审计日志
	CollegeMetricsRead     Capability = "college.metrics.read"     // 本院指标
	CollegeTwinScreen      Capability = "college.twin.screen"      // 学院数字孪生大屏
	CollegeDataAnalysis    Capability = "college.data.analysis"    // 学院数据分析
	CollegeGraduationRead  Capability = "college.graduation.read"  // 毕设选题管理查看
	CollegeGraduationWrite Capability = "college.graduation.write" // 毕设选题管理操作
	CollegeForecast        Capability = "college.forecast"         // 问题预案
)

// 学校管理能力
const (
	SchoolAgentWrite   Capability = "school.agent.write"    // 智能体管理
	SchoolUserWrite    Capability = "school.user.write"     // 用户管理（学校级）
	SchoolUserUpdate   Capability = "school.user.update"    // 修改用户信息（学校级）
	SchoolKBSyncExport Capability = "school.kb.sync.export" // 知识库全量/增量同步导出（学校级运维，区别于 self.export.self 导出本人回答）
)

// 系统管理能力
const (
	SystemSettingsWrite Capability = "system.settings.write" // 全局配置
	SystemAuditAll      Capability = "system.audit.all"      // 全局审计日志
	SystemPasswordReset Capability = "system.password.reset" // 重置任意用户密码
	SystemAIBriefing    Capability = "system.ai_briefing"    // AI 简讯管理（sys_admin 专属：CRUD/来源/导出/抓取）
)

// roleNode 角色继承节点
type roleNode struct {
	role         string
	parents      []string     // 父角色（继承所有父角色的 capability，多父支持 college_admin 同时继承 counselor+teacher+assistant）
	capabilities []Capability // 直接拥有的 capability（不含继承）
}

// roles 角色继承图（线性 + 例外多父）
//
// 继承链：
//
//	sys_admin → school_admin → college_admin → {counselor, teacher, assistant} → student_union → student
//
// counselor、teacher、assistant 三者平级（互不继承），但 college_admin 同时继承三者
//
// 修改方法：调整 parents 字段；新角色在此添加节点即可
var roles = map[string]*roleNode{
	"guest": {
		role:    "guest",
		parents: nil,
		capabilities: []Capability{
			// 仅浏览公开信息。修复 GPT56SOL v3 P0-01：
			// 此前 guest 持有 SelfChat/SelfKnowledgeRead/SelfProcessRead，
			// 即使中间件拦截失败也无业务能力可用（纵深防御）。
			SelfGuestRead,
		},
	},
	"student": {
		role:    "student",
		parents: nil,
		capabilities: []Capability{
			// 学生即"个人用户"基线，所有 self.* 能力的源头
			SelfBriefingRead, SelfDiaryRead, SelfCheckinWrite,
			SelfTwinRead, SelfTwinWrite, SelfPersonalityRead, SelfFeedbackSubmit,
			SelfProfileWrite, SelfChat, SelfKnowledgeRead, SelfVoice,
			SelfSessionRead, SelfSessionDelete, SelfRecommendRead,
			SelfExportSelf, SelfAchievements, SelfCourseMapRead,
			SelfCourseAnalytics, SelfWeeklyReport, SelfGenericAI,
			SelfCommunityRead, SelfPrivateChat, SelfProcessRead,
			SelfEmotionStats, SelfEmotionConsent,
			SelfTokenStats,
			// 校园文化（全员）
			SelfCultureAnthem, SelfCultureRadio, SelfCultureLectures,
			SelfCultureEvents, SelfCultureVolunteer,
			// 学生功能模块
			SelfGraduationRead, SelfGraduationWrite,
			SelfCompetitionRead, SelfCompetitionWrite,
			SelfPlanRead, SelfPlanWrite,
			SelfPartyRead, SelfPartyWrite,
			SelfClubRead, SelfClubWrite,
			// 就业指导、学业学习、心理健康、身体健康四大模块
			SelfCareerRead, SelfCareerWrite,
			SelfStudyRead, SelfStudyWrite,
			SelfMentalRead, SelfMentalWrite,
			SelfHealthRead, SelfHealthWrite,
			// AI 简讯（首页资讯）
			SelfAIBriefingRead,
		},
	},
	"student_union": {
		role:    "student_union",
		parents: []string{"student"},
		capabilities: []Capability{
			UnionKBSubmit, UnionFeedbackList, UnionFeedbackRead, UnionFeedbackWrite,
			UnionEventPlan, UnionPosterGen,
			CounselorImportStudent,
		},
	},
	"counselor": {
		role:    "counselor",
		parents: []string{"student_union"},
		capabilities: []Capability{
			CounselorDailyFocusRead, CounselorClassRead,
			CounselorAlertRead, CounselorAlertHandle,
			CounselorAlertAnalyze, CounselorEmotionTrends,
			CounselorKBWrite, CounselorKBReview,
			CounselorInterventionWrite, CounselorTalkRecord,
			CounselorPredictionRead, CounselorIntegrationRead,
			CounselorReviewPending, CounselorClassReport,
			CounselorTwinBoard, CounselorTalkTips,
			CounselorIdeological, CounselorClassProfile,
			CounselorCommunityManage, CounselorHotTopicSense,
			CounselorProcessEdit, CounselorStudentList,
			CounselorTokenSubordinates,
			CounselorNotify,
		},
	},
	"teacher": {
		role:    "teacher",
		parents: []string{"student_union"}, // 教师继承学生会的知识库提交，但不继承辅导员的预警
		capabilities: []Capability{
			TeacherLessonPrep, TeacherExamGen,
			TeacherHeatmapRead, TeacherClassInteract,
			TeacherDailyOverview, TeacherGrading,
			TeacherReflection, TeacherStyleDist, TeacherCommunityQA,
			CounselorTokenSubordinates,
		},
	},
	"assistant": {
		role:    "assistant",
		parents: []string{"student_union"},
		capabilities: []Capability{
			AssistantScheduleCheck, AssistantGradAudit, AssistantExamArrange,
		},
	},
	"college_admin": {
		role: "college_admin",
		// 学院管理员同时继承辅导员/教师/教辅三条线
		parents: []string{"counselor", "teacher", "assistant"},
		capabilities: []Capability{
			CollegeUserRead, CollegeAuditRead, CollegeMetricsRead,
			CollegeTwinScreen, CollegeDataAnalysis,
			CollegeGraduationRead, CollegeGraduationWrite, CollegeForecast,
		},
	},
	"school_admin": {
		role:    "school_admin",
		parents: []string{"college_admin"},
		capabilities: []Capability{
			SchoolAgentWrite, SchoolUserWrite, SchoolUserUpdate,
			SchoolKBSyncExport,
		},
	},
	"sys_admin": {
		role:    "sys_admin",
		parents: []string{"school_admin"},
		capabilities: []Capability{
			SystemSettingsWrite, SystemAuditAll, SystemPasswordReset,
			SystemAIBriefing,
		},
	},
}

// HasCapability 判断角色是否拥有指定能力（含继承）
// 算法：DFS 向父角色递归查询，命中即返回 true
func HasCapability(role string, cap Capability) bool {
	visited := make(map[string]bool)
	return hasCapability(role, cap, visited)
}

func hasCapability(role string, cap Capability, visited map[string]bool) bool {
	if visited[role] {
		return false
	}
	visited[role] = true

	node, ok := roles[role]
	if !ok {
		return false
	}

	// 直接 capability 匹配
	for _, c := range node.capabilities {
		if c == cap {
			return true
		}
	}

	// 递归向父角色查询
	for _, parent := range node.parents {
		if hasCapability(parent, cap, visited) {
			return true
		}
	}
	return false
}

// CapabilitiesOf 返回角色拥有的所有能力（含继承），用于前端获取
// 结果去重，顺序无保证
func CapabilitiesOf(role string) []Capability {
	caps := make(map[Capability]bool)
	visited := make(map[string]bool)
	collectCapabilities(role, caps, visited)

	result := make([]Capability, 0, len(caps))
	for c := range caps {
		result = append(result, c)
	}
	return result
}

func collectCapabilities(role string, caps map[Capability]bool, visited map[string]bool) {
	if visited[role] {
		return
	}
	visited[role] = true

	node, ok := roles[role]
	if !ok {
		return
	}

	for _, c := range node.capabilities {
		caps[c] = true
	}
	for _, parent := range node.parents {
		collectCapabilities(parent, caps, visited)
	}
}

// IsKnownRole 判断是否为已注册角色
func IsKnownRole(role string) bool {
	_, ok := roles[role]
	return ok
}

// RoleMatches 判断当前角色是否命中目标角色（含继承：命中自己或任一祖先角色）
// 用于 visible_to.roles 这类"面向角色"的名单判断。
func RoleMatches(role, target string) bool {
	if role == target {
		return true
	}
	visited := make(map[string]bool)
	return roleReaches(role, target, visited)
}

func roleReaches(role, target string, visited map[string]bool) bool {
	if visited[role] {
		return false
	}
	visited[role] = true
	node, ok := roles[role]
	if !ok {
		return false
	}
	for _, parent := range node.parents {
		if parent == target || roleReaches(parent, target, visited) {
			return true
		}
	}
	return false
}
