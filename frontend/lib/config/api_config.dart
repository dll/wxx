/// API 配置常量
class ApiConfig {
  // 后端基础地址（开发环境）
  // Web 端与后端同源时可用相对路径；跨域时需完整 URL
  static const String baseUrl = 'https://api.pydaydayup.xyz';

  // API 版本前缀
  static const String apiPrefix = '/api/v1';

  // 超时设置（毫秒）
  // Vercel 冷启动 + 跨境延迟较长，连接超时适当放大
  static const int connectTimeout = 30000;
  static const int receiveTimeout = 60000; // LLM 响应较慢，给足时间

  // ── 接口路径 ──
  static const String login = '$apiPrefix/auth/login';
  static const String profile = '$apiPrefix/user/profile';
  static const String consent = '$apiPrefix/user/consent';
  static const String chat = '$apiPrefix/chat';
  static const String sessions = '$apiPrefix/sessions';
  static String sessionMessages(String id) => '$apiPrefix/sessions/$id/messages';
  static String sessionDelete(String id) => '$apiPrefix/sessions/$id';
  static String sessionRename(String id) => '$apiPrefix/sessions/$id';

  // ── 语音接口 ──
  static const String voiceAsr = '$apiPrefix/voice/asr';
  static const String voiceTts = '$apiPrefix/voice/tts';

  // ── 知识大厅 ──
  static const String knowledge = '$apiPrefix/knowledge';

  // ── 情感预警 ──
  static const String emotionAnalyze = '$apiPrefix/emotion/analyze';
  static const String emotionAlerts = '$apiPrefix/emotion/alerts';
  static const String emotionStats = '$apiPrefix/emotion/stats';
  static String emotionAlertUpdate(String id) => '$apiPrefix/emotion/alerts/$id';

  // ── 导出 ──
  static const String export = '$apiPrefix/export';
  static const String exportAnswer = '$apiPrefix/export/answer';

  // ── 智能体管理 ──
  static const String agents = '$apiPrefix/agents';
  static String agentDetail(String id) => '$apiPrefix/agents/$id';

  // ── 密码管理 ──
  static const String changePassword = '$apiPrefix/user/password';
  static String resetPassword(int id) => '$apiPrefix/admin/users/$id/password';

  // ── 词元统计 ──
  static const String tokenStatsMy = '$apiPrefix/token-stats/my';
  static const String tokenStatsSubordinates = '$apiPrefix/token-stats/subordinates';

  // ── 管理端 ──
  static const String adminMetrics = '$apiPrefix/admin/metrics';
  static const String adminUsers = '$apiPrefix/admin/users';
  static String adminUserUpdate(String id) => '$apiPrefix/admin/users/$id';
  static const String adminAudit = '$apiPrefix/admin/audit';
  static const String adminSettings = '$apiPrefix/admin/settings';

  // ── 知识管理 ──
  static const String kbResources = '$apiPrefix/kb/resources';
  static String kbResource(String id) => '$apiPrefix/kb/resources/$id';
  static String kbApprove(String id) => '$apiPrefix/kb/resources/$id/approve';
  static String kbReject(String id) => '$apiPrefix/kb/resources/$id/reject';
  static String kbSubmitReview(String id) => '$apiPrefix/kb/resources/$id/submit';

  // ── 知识审核 ──
  static const String reviewPending = '$apiPrefix/review/pending';

  // ── 反馈 ──
  static const String feedback = '$apiPrefix/feedback';
  static const String feedbackScreenshot = '$apiPrefix/feedback/screenshot';
  static const String feedbackMine = '$apiPrefix/feedback/mine';
  static String feedbackResolve(String id) => '$apiPrefix/feedback/$id';

  // ── 办事流程办理记录 ──
  static const String processRecords = '$apiPrefix/process/records';
  static String processRecordStart(String flow) => '$apiPrefix/process/records/$flow/start';
  static String processRecordProgress(String flow) => '$apiPrefix/process/records/$flow/progress';

  // ── 语音配置 ──
  static const String voiceConfig = '$apiPrefix/user/voice-config';

  // ── 当前用户能力清单（基于角色继承自动展开）──
  static const String capabilities = '$apiPrefix/user/capabilities';

  // ── AI 模型配置 ──
  static const String modelConfig = '$apiPrefix/user/model-config';

  // ── 学生 AI 功能 ──
  static const String dailyBriefing = '$apiPrefix/student/daily-briefing';
  static const String learningDiary = '$apiPrefix/student/learning-diary';
  static const String checkin = '$apiPrefix/student/checkin';
  static const String checkinHistory = '$apiPrefix/student/checkin/history';
  static const String digitalTwin = '$apiPrefix/student/digital-twin';
  static const String personalityInsight = '$apiPrefix/student/personality';
  static const String growthPath = '$apiPrefix/student/growth-path';
  static const String courseMap = '$apiPrefix/student/course-map';
  static const String courseAnalytics = '$apiPrefix/student/course-analytics';
  static const String weeklyReport = '$apiPrefix/student/weekly-report';
  static const String achievements = '$apiPrefix/student/achievements';
  static const String freshmanPlan = '$apiPrefix/student/freshman-plan';
  static const String ideologicalRecord = '$apiPrefix/student/ideological-record';
  static const String partyProgress = '$apiPrefix/student/party-progress';
  static const String politicalStudy = '$apiPrefix/student/political-study';
  static const String campusLife = '$apiPrefix/student/campus-life';
  static const String schedule = '$apiPrefix/student/schedule';
  static const String competitionMatch = '$apiPrefix/student/competition-match';
  static const String studyBuddy = '$apiPrefix/student/study-buddy';
  static const String mentalHealth = '$apiPrefix/student/mental-health';
  static const String digitalMentor = '$apiPrefix/student/digital-mentor';

  // ── 学生社区互动功能 ──
  static const String qaPlaza = '$apiPrefix/student/qa-plaza';
  static const String hotTopics = '$apiPrefix/student/hot-topics';
  static const String qaLeaderboard = '$apiPrefix/student/qa-leaderboard';
  static const String privateChat = '$apiPrefix/student/private-chat';
  static const String processEnhanced = '$apiPrefix/student/process-enhanced';

  // ── 辅导员 AI 功能 ──
  static const String counselorDailyFocus = '$apiPrefix/counselor/daily-focus';
  static const String counselorClassReport = '$apiPrefix/counselor/class-report';
  static const String counselorTwinBoard = '$apiPrefix/counselor/twin-board';
  static const String counselorPrediction = '$apiPrefix/counselor/prediction';
  static const String counselorIntervention = '$apiPrefix/counselor/intervention';
  static const String counselorTalkRecord = '$apiPrefix/counselor/talk-record';
  static const String counselorTalkTips = '$apiPrefix/counselor/talk-tips';
  static const String counselorIdeological = '$apiPrefix/counselor/ideological';
  static const String counselorClassProfile = '$apiPrefix/counselor/class-profile';

  // ── 辅导员社区功能 ──
  static const String counselorCommunityManage = '$apiPrefix/counselor/community-manage';
  static const String counselorHotTopicSense = '$apiPrefix/counselor/hot-topic-sense';
  static const String counselorProcessEdit = '$apiPrefix/counselor/process-edit';
  static const String counselorStudentList = '$apiPrefix/counselor/student-list';

  // ── 教师 AI 功能 ──
  static const String teacherDailyOverview = '$apiPrefix/teacher/daily-overview';
  static const String teacherLessonPrep = '$apiPrefix/teacher/lesson-prep';
  static const String teacherExamGen = '$apiPrefix/teacher/exam-gen';
  static const String teacherClassInteract = '$apiPrefix/teacher/class-interact';
  static const String teacherGrading = '$apiPrefix/teacher/grading';
  static const String teacherHeatmap = '$apiPrefix/teacher/heatmap';
  static const String teacherReflection = '$apiPrefix/teacher/reflection';
  static const String teacherStyleDist = '$apiPrefix/teacher/style-distribution';

  // ── 教师社区功能 ──
  static const String teacherCommunityQA = '$apiPrefix/teacher/community-qa';

  // ── 教辅 AI 功能 ──
  static const String assistantScheduleCheck = '$apiPrefix/assistant/schedule-check';
  static const String assistantGradAudit = '$apiPrefix/assistant/graduation-audit';
  static const String assistantExamArrange = '$apiPrefix/assistant/exam-arrange';

  // ── 学生会 AI 功能 ──
  static const String unionEventPlan = '$apiPrefix/union/event-plan';
  static const String unionPosterGen = '$apiPrefix/union/poster-gen';

  // ── 学院管理员 AI 功能 ──
  static const String collegeTwinScreen = '$apiPrefix/college/twin-screen';
  static const String collegeDataAnalysis = '$apiPrefix/college/data-analysis';

  // ── 校园文化智能体（全员可见）──
  static const String cultureAnthems = '$apiPrefix/culture/anthems';
  static const String cultureRadio = '$apiPrefix/culture/radio';
  static const String cultureLectures = '$apiPrefix/culture/lectures';
  static const String cultureEvents = '$apiPrefix/culture/events';
  static const String cultureVolunteer = '$apiPrefix/culture/volunteer';

  // ── 问题预案 ──
  static const String forecastAnalysis = '$apiPrefix/forecast/analysis';
  static const String forecastIssues = '$apiPrefix/forecast/issues';
  static String forecastIssueDetail(String id) => '$apiPrefix/forecast/issues/$id';
  static String forecastIssueStatus(String id) => '$apiPrefix/forecast/issues/$id/status';
  static const String forecastStatistics = '$apiPrefix/forecast/statistics';

  // ── 毕设选题 ──
  static const String graduationAdvisors = '$apiPrefix/graduation/advisors';
  static const String graduationTopics = '$apiPrefix/graduation/available-topics';
  static const String graduationSelect = '$apiPrefix/graduation/select-topic';
  static const String graduationMySelection = '$apiPrefix/graduation/my-selection';
  static const String graduationMilestones = '$apiPrefix/graduation/milestones';
  static const String graduationStats = '$apiPrefix/graduation/stats';
  static const String graduationSelections = '$apiPrefix/graduation/selections';
  static String graduationConfirm(String id) => '$apiPrefix/graduation/selections/$id/confirm';

  // ── 学科竞赛 ──
  static const String competitionList = '$apiPrefix/competition/list';
  static String competitionDetail(String id) => '$apiPrefix/competition/$id';
  static const String competitionRegister = '$apiPrefix/competition/register';
  static const String competitionMyRegistrations = '$apiPrefix/competition/my-registrations';
  static const String competitionSubmitWork = '$apiPrefix/competition/submit-work';
  static const String competitionStats = '$apiPrefix/competition/stats';

  // ── 大学规划 ──
  static const String planTemplates = '$apiPrefix/plan/templates';
  static const String planMyPlans = '$apiPrefix/plan/my-plans';
  static const String planCreate = '$apiPrefix/plan/create';
  static String planSubmit(String id) => '$apiPrefix/plan/$id/submit';
  static String planReview(String id) => '$apiPrefix/plan/$id/review';

  // ── 入党教育 ──
  static const String partyStages = '$apiPrefix/party/stages';
  static const String partyMyProgress = '$apiPrefix/party/my-progress';
  static const String partyStudyRecords = '$apiPrefix/party/my-study-records';
  static const String partyStudyRecordAdd = '$apiPrefix/party/study-record';
  static const String partyStats = '$apiPrefix/party/stats';

  // ── 游客注册 ──
  static const String guestRegister = '$apiPrefix/auth/guest-register';

  // ── 游客管理（管理员）──
  static const String adminGuestsPending = '$apiPrefix/admin/guests/pending';
  static String adminGuestApprove(String id) => '$apiPrefix/admin/guests/$id/approve';
  static String adminGuestReject(String id) => '$apiPrefix/admin/guests/$id/reject';

  // ── 社团生活 ──
  static const String clubList = '$apiPrefix/club/list';
  static String clubDetail(String id) => '$apiPrefix/club/$id';
  static const String clubJoin = '$apiPrefix/club/join';
  static const String clubMyClubs = '$apiPrefix/club/my-clubs';
  static const String clubActivities = '$apiPrefix/club/activities';
  static const String clubActivityRegister = '$apiPrefix/club/activity/register';
}
