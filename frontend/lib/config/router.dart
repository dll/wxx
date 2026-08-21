import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../utils/storage.dart';
import '../widgets/fab_menu.dart';
import '../pages/consent/consent_page.dart';
import '../pages/login/login_page.dart';
import '../pages/home/home_page.dart';
import '../pages/chat/chat_page.dart';
import '../pages/sessions/sessions_page.dart';
import '../pages/profile/profile_page.dart';
import '../pages/profile/my_logs_page.dart';
import '../pages/profile/portal_browse_page.dart';
import '../pages/enrollment/enrollment_page.dart';
import '../pages/browse/browse_page.dart';
import '../pages/emotion/emotion_dashboard_page.dart';
import '../pages/agent/agent_management_page.dart';
import '../pages/admin/admin_metrics_page.dart';
import '../pages/admin/admin_dashboard_page.dart';
import '../pages/admin/data_import_page.dart';
import '../pages/admin/admin_users_page.dart';
import '../pages/admin/admin_audit_page.dart';
import '../pages/admin/admin_content_page.dart';
import '../pages/admin/admin_settings_page.dart';
import '../pages/admin/review_page.dart';
import '../pages/admin/my_submissions_page.dart';
import '../pages/admin/feedback_page.dart';
import '../pages/profile/my_feedbacks_page.dart';
import '../pages/process/my_records_page.dart';
import '../pages/process/process_manage_page.dart';
import '../pages/process/process_review_page.dart';
import '../pages/bookmarks/bookmarks_page.dart';
import '../pages/notification/notification_page.dart';
import '../pages/about/about_page.dart';
import '../pages/help/help_page.dart';
// ── 学生 AI 功能页面 ──
import '../pages/student/daily_briefing_page.dart';
import '../pages/student/learning_diary_page.dart';
import '../pages/student/checkin_page.dart';
import '../pages/student/digital_twin_page.dart';
import '../pages/student/personality_page.dart';
import '../pages/student/student_profile_page.dart';
import '../pages/student/achievements_page.dart';
import '../pages/student/course_map_page.dart';
import '../pages/student/course_analytics_page.dart';
import '../pages/student/weekly_report_page.dart';
import '../pages/student/freshman_plan_page.dart';
import '../pages/student/freshmen_guide_page.dart';
import '../pages/student/freshman_onboarding_page.dart';
import '../pages/student/freshman_agenda_page.dart';
import '../pages/student/grade_growth_page.dart';
import '../pages/student/my_schedule_import_page.dart';
import '../pages/student/growth_path_page.dart';
import '../pages/student/political_study_page.dart';
import '../pages/student/ideological_record_page.dart';
import '../pages/student/party_progress_page.dart';
import '../pages/student/campus_life_page.dart';
import '../pages/student/schedule_page.dart';
import '../pages/student/competition_match_page.dart';
import '../pages/student/study_buddy_page.dart';
import '../pages/student/mental_health_page.dart';
import '../pages/student/digital_mentor_page.dart';
import '../pages/student/qa_plaza_page.dart';
import '../pages/student/hot_topics_page.dart';
import '../pages/student/qa_leaderboard_page.dart';
import '../pages/student/private_chat_page.dart';
// ── 辅导员 AI 功能页面 ──
import '../pages/counselor/daily_focus_page.dart';
import '../pages/counselor/class_report_page.dart';
import '../pages/counselor/twin_board_page.dart';
import '../pages/counselor/prediction_page.dart';
import '../pages/counselor/intervention_page.dart';
import '../pages/counselor/talk_record_page.dart';
import '../pages/counselor/talk_tips_page.dart';
import '../pages/counselor/ideological_page.dart';
import '../pages/counselor/class_profile_page.dart';
import '../pages/counselor/community_manage_page.dart';
import '../pages/counselor/hot_topic_sense_page.dart';
import '../pages/counselor/student_list_page.dart';
import '../pages/counselor/second_class_board_page.dart';
// ── 教师 AI 功能页面 ──
import '../pages/teacher/daily_overview_page.dart';
import '../pages/teacher/lesson_prep_page.dart';
import '../pages/teacher/exam_gen_page.dart';
import '../pages/teacher/class_interact_page.dart';
import '../pages/teacher/grading_page.dart';
import '../pages/teacher/heatmap_page.dart';
import '../pages/teacher/reflection_page.dart';
import '../pages/teacher/style_dist_page.dart';
import '../pages/teacher/community_qa_page.dart';
import '../pages/teacher/party_activity_register_page.dart';
import '../pages/teacher/teacher_grade_entry_page.dart';
import '../pages/teacher/teacher_course_apply_page.dart';
import '../pages/teacher/teacher_homework_page.dart';
// ── 教辅/学生会/学院管理员 AI 功能页面 ──
import '../pages/assistant_role/schedule_check_page.dart';
import '../pages/assistant_role/grad_audit_page.dart';
import '../pages/assistant_role/exam_arrange_page.dart';
import '../pages/assistant_role/teaching_calendar_page.dart';
import '../pages/assistant_role/student_info_page.dart';
import '../pages/assistant_role/notification_draft_page.dart';
import '../pages/assistant_role/facility_workbench_page.dart';
import '../pages/assistant_role/teacher_course_review_page.dart';
import '../pages/union/event_plan_page.dart';
import '../pages/union/poster_gen_page.dart';
import '../pages/union/activity_manage_page.dart';
import '../pages/union/union_workbench_page.dart';
import '../pages/college/twin_screen_page.dart';
import '../pages/college/data_analysis_page.dart';
import '../pages/secretary/secretary_outcome_dashboard_page.dart';
import '../pages/secretary/outcome_manage_page.dart';
import '../pages/secretary/party_dashboard_page.dart';
import '../pages/secretary/collab_dashboard_page.dart';
import '../pages/secretary/gov_ticket_manage_page.dart';
import '../pages/secretary/gov_ticket_mine_page.dart';
import '../pages/profile/model_config_page.dart';
import '../pages/culture/anthem_page.dart';
import '../pages/culture/radio_page.dart';
import '../pages/culture/lectures_page.dart';
import '../pages/culture/events_page.dart';
import '../pages/culture/volunteer_page.dart';
import '../pages/token_stats/token_stats_page.dart';
import '../pages/admin/forecast_page.dart';
import '../pages/admin/admin_guest_review_page.dart';
import '../pages/campus/campus_map_page.dart';
import '../pages/student/graduation_page.dart';
import '../pages/student/competition_page.dart';
import '../pages/student/plan_page.dart';
import '../pages/student/party_education_page.dart';
import '../pages/student/club_page.dart';
// ── 学生三大教育模块页面 ──
import '../pages/student/career/career_page.dart';
import '../pages/student/career/job_detail_page.dart';
import '../pages/student/career/policy_detail_page.dart';
import '../pages/student/study/study_page.dart';
import '../pages/student/study/course_detail_page.dart';
import '../pages/student/study/grades_page.dart';
import '../pages/student/study/resource_detail_page.dart';
import '../pages/student/study_plan/study_plan_page.dart';
import '../pages/student/study_plan/plan_detail_page.dart';
import '../pages/student/study_plan/timetable_page.dart';
import '../pages/student/mental/mental_page.dart';
import '../pages/student/mental/scale_detail_page.dart';
import '../pages/student/mental/counseling_page.dart';
import '../pages/student/mental/article_detail_page.dart';
import '../pages/student/health/health_page.dart';
import '../pages/student/mental/mood_diary_page.dart';
import '../pages/apps/app_center_page.dart';
import '../pages/services/service_center_page.dart';
import '../pages/vopc/vopc_page.dart';
import '../pages/ai_briefings/ai_briefings_page.dart';
import '../pages/ai_briefings/ai_briefing_admin_page.dart';
import '../utils/screenshot_capture.dart';

/// 鉴权状态刷新通知 — 当 token 过期/退出登录时通知 GoRouter 重新评估 redirect
class AuthRefreshNotifier extends ChangeNotifier {
  void refresh() => notifyListeners();
}

final authRefreshNotifier = AuthRefreshNotifier();

/// 全局导航 key：供 deep link（扫码登录唤起 APK）回调导航到登录页
final GlobalKey<NavigatorState> rootNavigatorKey = GlobalKey<NavigatorState>();

/// 应用路由配置
final GoRouter appRouter = GoRouter(
  navigatorKey: rootNavigatorKey,
  refreshListenable: authRefreshNotifier,
  initialLocation: '/home',
  redirect: (context, state) {
    final firstLaunchDone = Storage.firstLaunchDone;
    final loggedIn = Storage.isLoggedIn;
    final isConsentPage = state.matchedLocation == '/consent';
    final isLoginPage = state.matchedLocation == '/login';

    // 首次启动 → 必须先同意隐私政策
    if (!firstLaunchDone && !isConsentPage) return '/consent';
    // 未登录 → 允许访问首页（游客模式），但其他页面需登录
    if (!loggedIn) {
      final publicPaths = [
        '/home',
        '/login',
        '/consent',
        '/campus',
        '/browse',
        '/services',
      ];
      final isPublic =
          publicPaths.any((p) => state.matchedLocation.startsWith(p));
      if (!isPublic) return '/home';
    }
    // 已登录 → 不需要看登录页
    if (loggedIn && isLoginPage) return '/home';
    if (loggedIn && isConsentPage && firstLaunchDone) return '/home';

    // vOPC 与其他主菜单一致：入口和页面访问只依赖登录状态。
    // 具体创建、管理、评审等操作仍由能力按钮与后端授权控制，
    // 避免能力清单请求失败或缓存尚未刷新时误隐藏功能。
    return null;
  },
  routes: [
    GoRoute(
      path: '/consent',
      builder: (context, state) => const ConsentPage(),
    ),
    GoRoute(
      path: '/login',
      builder: (context, state) =>
          LoginPage(qrSessionId: state.uri.queryParameters['qr']),
    ),
    ShellRoute(
      builder: (context, state, child) => MainShell(child: child),
      routes: [
        GoRoute(
          path: '/home',
          builder: (context, state) => const HomePage(),
        ),
        GoRoute(
          path: '/chat',
          builder: (context, state) {
            final askQuery = state.uri.queryParameters['ask'];
            final autoVoice = state.uri.queryParameters['v'] == 'voice';
            return ChatPage(initialQuestion: askQuery, autoVoice: autoVoice);
          },
        ),
        GoRoute(
          path: '/browse',
          builder: (context, state) => const BrowsePage(),
        ),
        GoRoute(
          path: '/enrollment',
          builder: (context, state) => const EnrollmentPage(),
        ),
        GoRoute(
          path: '/sessions',
          builder: (context, state) => const SessionsPage(),
        ),
        GoRoute(
          path: '/emotion',
          builder: (context, state) => const EmotionDashboardPage(),
        ),
        GoRoute(
          path: '/agents',
          builder: (context, state) => const AgentManagementPage(),
        ),
        GoRoute(
          path: '/profile',
          builder: (context, state) => const ProfilePage(),
        ),
        GoRoute(
          path: '/profile/model-config',
          builder: (context, state) => const ModelConfigPage(),
        ),
        GoRoute(
          path: '/portal',
          builder: (context, state) => const PortalBrowsePage(),
        ),
        GoRoute(
          path: '/my-logs',
          builder: (context, state) => const MyLogsPage(),
        ),
        GoRoute(
          path: '/apps',
          builder: (context, state) => const AppCenterPage(),
        ),
        GoRoute(
          path: '/services',
          builder: (context, state) => const ServiceCenterPage(),
        ),
        GoRoute(
          path: '/vopc',
          builder: (context, state) => const VopcPage(),
        ),
        GoRoute(
          path: '/vopc/projects/:id',
          builder: (context, state) {
            final id = int.tryParse(state.pathParameters['id'] ?? '');
            if (id == null || id <= 0) return const VopcPage();
            return VopcProjectPage(projectId: id);
          },
        ),
        GoRoute(
          path: '/ai-briefings',
          builder: (context, state) => const AIBriefingsPage(),
        ),
        GoRoute(
          path: '/admin/ai-briefings',
          builder: (context, state) => const AIBriefingAdminPage(),
        ),
        // ── 校园文化智能体（全员可见）──
        GoRoute(
            path: '/culture/anthems', builder: (_, __) => const AnthemPage()),
        GoRoute(path: '/culture/radio', builder: (_, __) => const RadioPage()),
        GoRoute(
            path: '/culture/lectures',
            builder: (_, __) => const LecturesPage()),
        GoRoute(
            path: '/culture/events', builder: (_, __) => const EventsPage()),
        GoRoute(
            path: '/culture/volunteer',
            builder: (_, __) => const VolunteerPage()),
        GoRoute(
          path: '/token-stats',
          builder: (context, state) => const TokenStatsPage(),
        ),
        GoRoute(
          path: '/admin/dashboard',
          builder: (context, state) => const AdminDashboardPage(),
        ),
        GoRoute(
          path: '/admin/metrics',
          builder: (context, state) => const AdminMetricsPage(),
        ),
        GoRoute(
          path: '/admin/data-import',
          builder: (context, state) => const DataImportPage(),
        ),
        GoRoute(
          path: '/admin/users',
          builder: (context, state) => const AdminUsersPage(),
        ),
        GoRoute(
          path: '/admin/audit',
          builder: (context, state) => const AdminAuditPage(),
        ),
        GoRoute(
          path: '/admin/content',
          builder: (context, state) => const AdminContentPage(),
        ),
        GoRoute(
          path: '/admin/settings',
          builder: (context, state) => const AdminSettingsPage(),
        ),
        GoRoute(
          path: '/review',
          builder: (context, state) => const ReviewPage(),
        ),
        GoRoute(
          path: '/my-submissions',
          builder: (context, state) => const KnowledgeGovernancePage(),
        ),
        GoRoute(
          path: '/feedback',
          builder: (context, state) => const FeedbackPage(),
        ),
        GoRoute(
          path: '/forecast',
          builder: (context, state) => const ForecastPage(),
        ),
        GoRoute(
          path: '/admin/guests',
          builder: (context, state) => const AdminGuestReviewPage(),
        ),
        GoRoute(
          path: '/graduation',
          builder: (context, state) => const GraduationPage(),
        ),
        GoRoute(
          path: '/competition',
          builder: (context, state) => const CompetitionPage(),
        ),
        GoRoute(
          path: '/plan',
          builder: (context, state) => const PlanPage(),
        ),
        GoRoute(
          path: '/party-education',
          builder: (context, state) => const PartyEducationPage(),
        ),
        GoRoute(
          path: '/club',
          builder: (context, state) => const ClubPage(),
        ),
        // ── 学生三大教育模块路由 ──
        GoRoute(
            path: '/student/career', builder: (_, __) => const CareerPage()),
        GoRoute(
          path: '/student/career/job/:id',
          builder: (_, state) => JobDetailPage(
            jobId: state.pathParameters['id'] ?? '',
          ),
        ),
        GoRoute(
          path: '/student/career/policy/:id',
          builder: (_, state) => PolicyDetailPage(
            policyId: state.pathParameters['id'] ?? '',
          ),
        ),
        GoRoute(path: '/student/study', builder: (_, __) => const StudyPage()),
        GoRoute(
          path: '/student/study/course/:id',
          builder: (_, state) => CourseDetailPage(
            courseId: state.pathParameters['id'] ?? '',
          ),
        ),
        GoRoute(
            path: '/student/study/grades',
            builder: (_, __) => const GradesPage()),
        GoRoute(
          path: '/student/study/resource/:id',
          builder: (_, state) => ResourceDetailPage(
            resourceId: state.pathParameters['id'] ?? '',
          ),
        ),
        // ── 学习计划与校历课表 ──
        GoRoute(
          path: '/student/study-plan',
          builder: (_, __) => const StudyPlanPage(),
        ),
        GoRoute(
          path: '/student/study-plan/:id',
          builder: (_, state) => PlanDetailPage(
            planId: state.pathParameters['id'] ?? '',
          ),
        ),
        GoRoute(
          path: '/student/timetable',
          builder: (_, __) => const TimetablePage(),
        ),
        GoRoute(
            path: '/student/mental', builder: (_, __) => const MentalPage()),
        GoRoute(
            path: '/student/health', builder: (_, __) => const HealthPage()),
        GoRoute(
          path: '/student/mental/scale/:id',
          builder: (_, state) => ScaleDetailPage(
            scaleId: state.pathParameters['id'] ?? '',
          ),
        ),
        GoRoute(
            path: '/student/mental/counseling',
            builder: (_, __) => const CounselingPage()),
        GoRoute(
          path: '/student/mental/article/:id',
          builder: (_, state) => ArticleDetailPage(
            articleId: state.pathParameters['id'] ?? '',
          ),
        ),
        GoRoute(
            path: '/student/mental/mood',
            builder: (_, __) => const MoodDiaryPage()),
        GoRoute(
          path: '/my-feedbacks',
          builder: (context, state) => const MyFeedbacksPage(),
        ),
        GoRoute(
          path: '/my-records',
          builder: (context, state) => const MyRecordsPage(),
        ),
        GoRoute(
          path: '/process-manage',
          builder: (context, state) => const ProcessManagePage(),
        ),
        GoRoute(
          path: '/process-review',
          builder: (context, state) => const ProcessReviewPage(),
        ),
        GoRoute(
          path: '/bookmarks',
          builder: (context, state) => const BookmarksPage(),
        ),
        GoRoute(
          path: '/notifications',
          builder: (context, state) => const NotificationPage(),
        ),
        GoRoute(
          path: '/about',
          builder: (context, state) => const AboutPage(),
        ),
        GoRoute(
          path: '/help',
          builder: (context, state) => const HelpPage(),
        ),
        GoRoute(
          path: '/campus',
          builder: (context, state) => CampusMapPage(
            initialTab: state.uri.queryParameters['v'],
          ),
        ),
        // ── 学生 AI 功能路由 ──
        GoRoute(
            path: '/student/daily-briefing',
            builder: (_, __) => const DailyBriefingPage()),
        GoRoute(
            path: '/student/learning-diary',
            builder: (_, __) => const LearningDiaryPage()),
        GoRoute(
            path: '/student/checkin', builder: (_, __) => const CheckinPage()),
        GoRoute(
            path: '/student/digital-twin',
            builder: (_, __) => const DigitalTwinPage()),
        GoRoute(
            path: '/student/profile',
            builder: (_, __) => const StudentProfilePage()),
        GoRoute(
            path: '/student/personality',
            builder: (_, __) => const PersonalityInsightPage()),
        GoRoute(
            path: '/student/achievements',
            builder: (_, __) => const AchievementsPage()),
        GoRoute(
            path: '/student/course-map',
            builder: (_, __) => const CourseMapPage()),
        GoRoute(
            path: '/student/course-analytics',
            builder: (_, __) => const CourseAnalyticsPage()),
        GoRoute(
            path: '/student/weekly-report',
            builder: (_, __) => const WeeklyReportPage()),
        GoRoute(
            path: '/student/freshman-plan',
            builder: (_, __) => const FreshmanPlanPage()),
        GoRoute(
            path: '/student/freshmen-guide',
            builder: (_, __) => const FreshmenGuidePage()),
        GoRoute(
            path: '/student/freshman-onboarding',
            builder: (_, __) => const FreshmanOnboardingPage()),
        GoRoute(
            path: '/student/freshman-agenda',
            builder: (_, __) => const FreshmanAgendaPage()),
        GoRoute(
            path: '/student/grade-growth',
            builder: (_, __) => const GradeGrowthPage()),
        GoRoute(
            path: '/student/schedule-import',
            builder: (_, __) => const MyScheduleImportPage()),
        GoRoute(
            path: '/student/growth-path',
            builder: (_, __) => const GrowthPathPage()),
        GoRoute(
            path: '/student/political-study',
            builder: (_, __) => const PoliticalStudyPage()),
        GoRoute(
            path: '/student/ideological-record',
            builder: (_, __) => const IdeologicalRecordPage()),
        GoRoute(
            path: '/student/party-progress',
            builder: (_, __) => const PartyProgressPage()),
        GoRoute(
            path: '/student/campus-life',
            builder: (_, __) => const CampusLifePage()),
        GoRoute(
            path: '/student/schedule',
            builder: (_, __) => const ScheduleManagerPage()),
        GoRoute(
            path: '/student/competition-match',
            builder: (_, __) => const CompetitionMatchPage()),
        GoRoute(
            path: '/student/study-buddy',
            builder: (_, __) => const StudyBuddyPage()),
        GoRoute(
            path: '/student/mental-health',
            builder: (_, __) => const MentalHealthPage()),
        GoRoute(
            path: '/student/digital-mentor',
            builder: (_, __) => const DigitalMentorPage()),
        GoRoute(
            path: '/student/qa-plaza', builder: (_, __) => const QAPlazaPage()),
        GoRoute(
            path: '/student/hot-topics',
            builder: (_, __) => const HotTopicsPage()),
        GoRoute(
            path: '/student/qa-leaderboard',
            builder: (_, __) => const QALeaderboardPage()),
        GoRoute(
            path: '/student/private-chat',
            builder: (_, __) => const PrivateChatPage()),
        // ── 辅导员 AI 功能路由 ──
        GoRoute(
            path: '/counselor/daily-focus',
            builder: (_, __) => const DailyFocusPage()),
        GoRoute(
            path: '/counselor/class-report',
            builder: (_, __) => const ClassReportPage()),
        GoRoute(
            path: '/counselor/twin-board',
            builder: (_, __) => const TwinBoardPage()),
        GoRoute(
            path: '/counselor/prediction',
            builder: (_, __) => const PredictionPage()),
        GoRoute(
            path: '/counselor/intervention',
            builder: (_, __) => const InterventionPage()),
        GoRoute(
            path: '/counselor/talk-record',
            builder: (_, __) => const TalkRecordPage()),
        GoRoute(
            path: '/counselor/talk-tips',
            builder: (_, __) => const TalkTipsPage()),
        GoRoute(
            path: '/counselor/ideological',
            builder: (_, __) => const CounselorIdeologicalPage()),
        GoRoute(
            path: '/counselor/class-profile',
            builder: (_, __) => const ClassProfilePage()),
        GoRoute(
            path: '/counselor/community-manage',
            builder: (_, __) => const CommunityManagePage()),
        GoRoute(
            path: '/counselor/hot-topic-sense',
            builder: (_, __) => const HotTopicSensePage()),
        GoRoute(
            path: '/counselor/process-edit',
            builder: (_, __) => const ProcessManagePage()),
        GoRoute(
            path: '/counselor/student-list',
            builder: (_, __) => const StudentListPage()),
        GoRoute(
            path: '/counselor/second-class-board',
            builder: (_, __) => const SecondClassBoardPage()),
        // ── 教师 AI 功能路由 ──
        GoRoute(
            path: '/teacher/daily-overview',
            builder: (_, __) => const DailyOverviewPage()),
        GoRoute(
            path: '/teacher/lesson-prep',
            builder: (_, __) => const LessonPrepPage()),
        GoRoute(
            path: '/teacher/exam-gen', builder: (_, __) => const ExamGenPage()),
        GoRoute(
            path: '/teacher/class-interact',
            builder: (_, __) => const ClassInteractPage()),
        GoRoute(
            path: '/teacher/grading', builder: (_, __) => const GradingPage()),
        GoRoute(
            path: '/teacher/heatmap', builder: (_, __) => const HeatmapPage()),
        GoRoute(
            path: '/teacher/reflection',
            builder: (_, __) => const ReflectionPage()),
        GoRoute(
            path: '/teacher/style-dist',
            builder: (_, __) => const StyleDistPage()),
        GoRoute(
            path: '/teacher/community-qa',
            builder: (_, __) => const CommunityQAPage()),
        // ── 党课/活动登记（蓝图第3块，2026-08-16）──
        GoRoute(
            path: '/teacher/party-register',
            builder: (_, __) => const PartyActivityRegisterPage()),
        // ── 教师成绩录入（P0-1，2026-08-17，方案A：教师自主声明授课）──
        GoRoute(
            path: '/teacher/grades-entry',
            builder: (_, __) => const TeacherGradeEntryPage()),
        // ── 教师授课关系申报（R3，2026-08-17 越权边界升级）──
        GoRoute(
            path: '/teacher/course-apply',
            builder: (_, __) => const TeacherCourseApplyPage()),
        // ── 教师作业信息发布+成绩统计（P2 轻量版，2026-08-17）──
        GoRoute(
            path: '/teacher/homework',
            builder: (_, __) => const TeacherHomeworkPage()),
        // ── 教师作业信息发布+成绩统计（2026-08-17，P2 轻量版，teacher.grade.write）──
        GoRoute(
            path: '/teacher/homework',
            builder: (_, __) => const TeacherHomeworkPage()),
        // ── 教辅/学生会/学院管理员 AI 功能路由 ──
        GoRoute(
            path: '/assistant/schedule-check',
            builder: (_, __) => const ScheduleCheckPage()),
        GoRoute(
            path: '/assistant/grad-audit',
            builder: (_, __) => const GradAuditPage()),
        GoRoute(
            path: '/assistant/exam-arrange',
            builder: (_, __) => const ExamArrangePage()),
        GoRoute(
            path: '/assistant/teaching-calendar',
            builder: (_, __) => const TeachingCalendarPage()),
        GoRoute(
            path: '/assistant/student-info',
            builder: (_, __) => const StudentInfoPage()),
        GoRoute(
            path: '/assistant/notification-draft',
            builder: (_, __) => const NotificationDraftPage()),
        GoRoute(
            path: '/assistant/facility-workbench',
            builder: (_, __) => const FacilityWorkbenchPage()),
        // ── 教师授课申报审核（R3 补 H，2026-08-17）：教辅/教务审核 + 待审角标（teacher.course.review）──
        GoRoute(
            path: '/assistant/teacher-course-review',
            builder: (_, __) => const TeacherCourseReviewPage()),
        GoRoute(
            path: '/union/event-plan',
            builder: (_, __) => const EventPlanPage()),
        GoRoute(
            path: '/union/poster-gen',
            builder: (_, __) => const PosterGenPage()),
        GoRoute(
            path: '/union/activity-manage',
            builder: (_, __) => const UnionActivityManagePage()),
        GoRoute(
            path: '/union/workbench',
            builder: (_, __) => const UnionWorkbenchPage()),
        GoRoute(
            path: '/college/twin-screen',
            builder: (_, __) => const TwinScreenPage()),
        GoRoute(
            path: '/college/data-analysis',
            builder: (_, __) => const DataAnalysisPage()),
        GoRoute(
            path: '/secretary/education-outcome',
            builder: (_, __) => const SecretaryOutcomeDashboardPage()),
        // 书记 party/collab 专项可视化深链（D1-1 功能补齐，2026-08-16）
        GoRoute(
            path: '/secretary/party-dashboard',
            builder: (_, __) => const PartyDashboardPage()),
        GoRoute(
            path: '/secretary/collab-dashboard',
            builder: (_, __) => const CollabDashboardPage()),
        GoRoute(
            path: '/secretary/outcome-manage',
            builder: (_, __) => const OutcomeManagePage()),
        // 督办工单（D5-3「洞察→工单」治理回环，2026-08-16）
        GoRoute(
            path: '/secretary/ticket-manage',
            builder: (_, __) => const GovTicketManagePage()),
        GoRoute(
            path: '/secretary/ticket-mine',
            builder: (_, __) => const GovTicketMinePage()),
      ],
    ),
  ],
);

/// 导航项定义
class _NavItem {
  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final String route;
  const _NavItem(this.label, this.icon, this.selectedIcon, this.route);
}

List<_NavItem> _navItemsForRole(String? role) => [
      _NavItem(
        role == 'teacher' ? '教学' : '首页',
        role == 'teacher'
            ? Icons.cast_for_education_outlined
            : Icons.home_outlined,
        role == 'teacher' ? Icons.cast_for_education : Icons.home,
        '/home',
      ),
      const _NavItem(
          '问芯', Icons.chat_bubble_outline, Icons.chat_bubble, '/chat'),
      _NavItem(
        role == 'teacher' ? '事务' : '办事',
        Icons.assignment_outlined,
        Icons.assignment,
        '/enrollment',
      ),
      const _NavItem(
          '服务', Icons.grid_view_outlined, Icons.grid_view, '/services'),
      const _NavItem('我的', Icons.person_outline, Icons.person, '/profile'),
      const _NavItem(
          'vOPC', Icons.rocket_launch_outlined, Icons.rocket_launch, '/vopc'),
    ];

/// 主页面外壳 — 响应式布局 + 磨砂玻璃导航
class MainShell extends StatelessWidget {
  final Widget child;
  const MainShell({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    final isDesktop = width > 900;

    if (isDesktop) {
      return _buildDesktopLayout(context);
    }
    return _buildMobileLayout(context);
  }

  /// 桌面布局：左侧 NavigationRail + 右侧内容区（限制最大宽度）
  Widget _buildDesktopLayout(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    final items = _navItemsForRole(Storage.role);
    final index = _currentIndex(context, items);
    final extended = width >= 1280;
    final maxWidth = _contentMaxWidth(context);
    return Scaffold(
      body: Row(
        children: [
          NavigationRail(
            selectedIndex: index,
            onDestinationSelected: (i) => _onTap(context, items, i),
            extended: extended,
            minExtendedWidth: 216,
            labelType: extended
                ? NavigationRailLabelType.none
                : NavigationRailLabelType.all,
            leading: Padding(
              padding: const EdgeInsets.symmetric(vertical: 16),
              child: extended
                  ? Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.school,
                            size: 30,
                            color: Theme.of(context).colorScheme.primary),
                        const SizedBox(width: 10),
                        Text('蔚小芯',
                            style: Theme.of(context).textTheme.titleLarge),
                      ],
                    )
                  : Icon(Icons.school,
                      size: 32, color: Theme.of(context).colorScheme.primary),
            ),
            destinations: items
                .map((item) => NavigationRailDestination(
                      icon: Icon(item.icon),
                      selectedIcon: Icon(item.selectedIcon),
                      label: Text(item.label),
                    ))
                .toList(),
          ),
          const VerticalDivider(width: 1),
          Expanded(
            child: Center(
              child: ConstrainedBox(
                constraints: BoxConstraints(maxWidth: maxWidth),
                child: RepaintBoundary(key: screenshotKey, child: child),
              ),
            ),
          ),
        ],
      ),
      floatingActionButton: const FabMenu(),
    );
  }

  /// 移动端布局：磨砂玻璃底部导航
  Widget _buildMobileLayout(BuildContext context) {
    final theme = Theme.of(context);
    final items = _navItemsForRole(Storage.role);
    final index = _currentIndex(context, items);
    return Scaffold(
      body: RepaintBoundary(key: screenshotKey, child: child),
      bottomNavigationBar: ClipRRect(
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 12, sigmaY: 12),
          child: Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface.withOpacity(0.7),
              border: Border(
                top: BorderSide(
                  color: theme.colorScheme.outlineVariant.withOpacity(0.3),
                ),
              ),
            ),
            child: NavigationBar(
              selectedIndex: index,
              onDestinationSelected: (i) => _onTap(context, items, i),
              destinations: items
                  .map((item) => NavigationDestination(
                        icon: Icon(item.icon),
                        selectedIcon: Icon(item.selectedIcon),
                        label: item.label,
                      ))
                  .toList(),
            ),
          ),
        ),
      ),
      floatingActionButton: const FabMenu(),
    );
  }

  int _currentIndex(BuildContext context, List<_NavItem> items) {
    final location = GoRouterState.of(context).matchedLocation;
    for (int i = 0; i < items.length; i++) {
      if (location.startsWith(items[i].route)) return i;
    }
    if (location.startsWith('/browse') ||
        location.startsWith('/apps') ||
        location.startsWith('/culture') ||
        location.startsWith('/campus') ||
        location.startsWith('/student') ||
        location.startsWith('/assistant') ||
        location.startsWith('/union') ||
        (location.startsWith('/teacher') &&
            !location.startsWith('/teacher/daily-overview')) ||
        (location.startsWith('/counselor') &&
            !location.startsWith('/counselor/daily-focus'))) {
      return 3;
    }
    return 0;
  }

  double _contentMaxWidth(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith('/teacher') ||
        location.startsWith('/counselor') ||
        location.startsWith('/admin') ||
        location.startsWith('/college')) {
      return 1440;
    }
    if (location.startsWith('/chat')) return 1280;
    return 1120;
  }

  void _onTap(BuildContext context, List<_NavItem> items, int index) {
    context.go(items[index].route);
  }
}
