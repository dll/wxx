import 'storage.dart';

/// 能力 ID 常量集（与后端 server/internal/auth/capabilities.go 同步）
///
/// 命名规范：domain.action，全部小写下划线
/// 修改时务必两端同步更新
class Capability {
  Capability._();

  // ── 个人能力（每个登录用户都有，所有角色继承自 student）──
  static const selfBriefingRead = 'self.briefing.read';
  static const selfDiaryRead = 'self.diary.read';
  static const selfCheckinWrite = 'self.checkin.write';
  static const selfTwinRead = 'self.twin.read';
  static const selfPersonalityRead = 'self.personality.read';
  static const selfFeedbackSubmit = 'self.feedback.submit';
  static const selfProfileWrite = 'self.profile.write';
  static const selfChat = 'self.chat';
  static const selfKnowledgeRead = 'self.knowledge.read';
  static const selfVoice = 'self.voice';
  static const selfSessionRead = 'self.session.read';
  static const selfSessionDelete = 'self.session.delete';
  static const selfRecommendRead = 'self.recommend.read';
  static const selfExportSelf = 'self.export.self';
  static const selfAchievements = 'self.achievements';
  static const selfCourseMapRead = 'self.course_map.read';
  static const selfCourseAnalytics = 'self.course.analytics';
  static const selfWeeklyReport = 'self.weekly.report';
  static const selfGenericAi = 'self.generic_ai';
  static const selfCommunityRead = 'self.community.read';
  static const selfPrivateChat = 'self.private_chat';
  static const selfProcessRead = 'self.process.read';
  static const selfEmotionStats = 'self.emotion.stats';
  static const selfTokenStats = 'self.token.stats';

  // ── 学生会能力 ──
  static const unionKbSubmit = 'union.kb.submit';
  static const unionFeedbackList = 'union.feedback.list';
  static const unionEventPlan = 'union.event.plan';
  static const unionPosterGen = 'union.poster.gen';

  // ── 辅导员能力 ──
  static const counselorDailyFocusRead = 'counselor.daily_focus.read';
  static const counselorClassRead = 'counselor.class.read';
  static const counselorAlertRead = 'counselor.alert.read';
  static const counselorAlertHandle = 'counselor.alert.handle';
  static const counselorAlertAnalyze = 'counselor.alert.analyze';
  static const counselorEmotionTrends = 'counselor.emotion.trends';
  static const counselorKbWrite = 'counselor.kb.write';
  static const counselorKbReview = 'counselor.kb.review';
  static const counselorInterventionWrite = 'counselor.intervention.write';
  static const counselorTalkRecord = 'counselor.talk.record';
  static const counselorPredictionRead = 'counselor.prediction.read';
  static const counselorIntegrationRead = 'counselor.integration.read';
  static const counselorReviewPending = 'counselor.review.pending';
  static const counselorClassReport = 'counselor.class.report';
  static const counselorTwinBoard = 'counselor.twin.board';
  static const counselorTalkTips = 'counselor.talk.tips';
  static const counselorIdeological = 'counselor.ideological';
  static const counselorClassProfile = 'counselor.class.profile';
  static const counselorCommunityManage = 'counselor.community.manage';
  static const counselorHotTopicSense = 'counselor.hot_topic.sense';
  static const counselorProcessEdit = 'counselor.process.edit';
  static const counselorImportStudent = 'counselor.import.student';
  static const batchScheduleImport = 'college.schedule.import';

  static const counselorStudentList = 'counselor.student.list';
  static const counselorTokenSubordinates = 'counselor.token.subordinates';
  static const counselorSecondClassBoard = 'counselor.secondclass.board';

  // ── 教师能力 ──
  static const teacherLessonPrep = 'teacher.lesson.prep';
  static const teacherExamGen = 'teacher.exam.gen';
  static const teacherHeatmapRead = 'teacher.heatmap.read';
  static const teacherClassInteract = 'teacher.class.interact';
  static const teacherDailyOverview = 'teacher.daily.overview';
  static const teacherGrading = 'teacher.grading';
  static const teacherReflection = 'teacher.reflection';
  static const teacherStyleDist = 'teacher.style.dist';
  static const teacherCommunityQa = 'teacher.community.qa';
  // 教师录入所授班级成绩（2026-08-17，P0-1）：教师自主声明授课，审计 created_by
  static const teacherGradeWrite = 'teacher.grade.write';
  // 教辅/教务审核教师授课申报（2026-08-17，R3 越权边界升级；不授 teacher 杜绝自审）
  static const teacherCourseReview = 'teacher.course.review';

  // ── 教辅能力 ──
  static const assistantScheduleCheck = 'assistant.schedule.check';
  static const assistantGradAudit = 'assistant.grad.audit';
  static const assistantExamArrange = 'assistant.exam.arrange';

  // ── 学院管理 ──
  static const collegeUserRead = 'college.user.read';
  static const collegeAuditRead = 'college.audit.read';
  static const collegeMetricsRead = 'college.metrics.read';
  static const collegeTwinScreen = 'college.twin.screen';
  static const collegeDataAnalysis = 'college.data.analysis';

  // ── 毕业去向 / 书记教育成果（2026-08-15）──
  static const outcomeRecordWrite = 'outcome.record.write';
  static const outcomeRecordRead = 'outcome.record.read';
  static const outcomeReview = 'outcome.review';
  static const outcomeDashboard = 'outcome.dashboard';
  // 党课/活动登记 + 协同育人总览（2026-08-16）
  static const partyRecordWrite = 'party.record.write';
  static const partyRecordRead = 'party.record.read';
  static const collabDashboard = 'college.collab.dashboard';

  // ── 督办工单（2026-08-16，D5-3「洞察→工单」治理回环）──
  static const govTicketManage = 'college.ticket.manage';       // 书记：创建/分派/督办
  static const govTicketAssignee = 'college.ticket.assignee';   // 责任人：查看/推进本人分派

  // ── 学校管理 ──
  static const schoolAgentWrite = 'school.agent.write';
  static const schoolUserWrite = 'school.user.write';
  static const schoolUserUpdate = 'school.user.update';

  // ── 系统管理 ──
  static const systemSettingsWrite = 'system.settings.write';
  static const systemAuditAll = 'system.audit.all';
  static const systemPasswordReset = 'system.password.reset';
}

/// 能力判断工具（前端单一来源，由 Storage 持有当前用户能力清单）
///
/// 用法：
///   if (CapabilityUtils.has(Capability.counselorAlertRead)) { ... }
///
/// 能力清单由后端 GET /api/v1/user/capabilities 返回，登录后由 AuthProvider 拉取并写入 Storage
class CapabilityUtils {
  CapabilityUtils._();

  /// 当前用户是否拥有指定能力
  static bool has(String capability) =>
      Storage.capabilities.contains(capability);

  /// 当前用户是否拥有任一能力（OR 语义）
  static bool hasAny(Iterable<String> caps) {
    final mine = Storage.capabilities;
    return caps.any(mine.contains);
  }

  /// 当前用户是否同时拥有所有能力（AND 语义）
  static bool hasAll(Iterable<String> caps) {
    final mine = Storage.capabilities;
    return caps.every(mine.contains);
  }
}
