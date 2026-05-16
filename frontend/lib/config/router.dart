import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../utils/storage.dart';
import '../pages/login/login_page.dart';
import '../pages/home/home_page.dart';
import '../pages/chat/chat_page.dart';
import '../pages/sessions/sessions_page.dart';
import '../pages/profile/profile_page.dart';
import '../pages/enrollment/enrollment_page.dart';
import '../pages/browse/browse_page.dart';
import '../pages/emotion/emotion_dashboard_page.dart';
import '../pages/agent/agent_management_page.dart';
import '../pages/admin/admin_metrics_page.dart';
import '../pages/admin/admin_users_page.dart';
import '../pages/admin/admin_audit_page.dart';
import '../pages/admin/admin_settings_page.dart';
import '../pages/admin/review_page.dart';
import '../pages/admin/my_submissions_page.dart';
import '../pages/admin/feedback_page.dart';
import '../pages/bookmarks/bookmarks_page.dart';
// ── 学生 AI 功能页面 ──
import '../pages/student/daily_briefing_page.dart';
import '../pages/student/learning_diary_page.dart';
import '../pages/student/checkin_page.dart';
import '../pages/student/digital_twin_page.dart';
import '../pages/student/personality_page.dart';
import '../pages/student/achievements_page.dart';
import '../pages/student/course_map_page.dart';
import '../pages/student/course_analytics_page.dart';
import '../pages/student/weekly_report_page.dart';
import '../pages/student/freshman_plan_page.dart';
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
import '../pages/student/process_enhanced_page.dart';
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
import '../pages/counselor/process_edit_page.dart';
import '../pages/counselor/student_list_page.dart';
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
// ── 教辅/学生会/学院管理员 AI 功能页面 ──
import '../pages/assistant_role/schedule_check_page.dart';
import '../pages/assistant_role/grad_audit_page.dart';
import '../pages/assistant_role/exam_arrange_page.dart';
import '../pages/union/event_plan_page.dart';
import '../pages/union/poster_gen_page.dart';
import '../pages/college/twin_screen_page.dart';
import '../pages/college/data_analysis_page.dart';
import '../pages/profile/model_config_page.dart';
import '../pages/culture/anthem_page.dart';
import '../pages/culture/radio_page.dart';
import '../pages/culture/lectures_page.dart';
import '../pages/culture/events_page.dart';
import '../pages/culture/volunteer_page.dart';
import '../utils/screenshot_capture.dart';
import '../widgets/fab_menu.dart';

/// 鉴权状态刷新通知 — 当 token 过期/退出登录时通知 GoRouter 重新评估 redirect
class AuthRefreshNotifier extends ChangeNotifier {
  void refresh() => notifyListeners();
}

final authRefreshNotifier = AuthRefreshNotifier();

/// 应用路由配置
final GoRouter appRouter = GoRouter(
  refreshListenable: authRefreshNotifier,
  initialLocation: '/home',
  redirect: (context, state) {
    final loggedIn = Storage.isLoggedIn;
    final isLoginPage = state.matchedLocation == '/login';

    if (!loggedIn && !isLoginPage) return '/login';
    if (loggedIn && isLoginPage) return '/home';

    return null;
  },
  routes: [
    GoRoute(
      path: '/login',
      builder: (context, state) => const LoginPage(),
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
            return ChatPage(initialQuestion: askQuery);
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
        // ── 校园文化智能体（全员可见）──
        GoRoute(path: '/culture/anthems', builder: (_, __) => const AnthemPage()),
        GoRoute(path: '/culture/radio', builder: (_, __) => const RadioPage()),
        GoRoute(path: '/culture/lectures', builder: (_, __) => const LecturesPage()),
        GoRoute(path: '/culture/events', builder: (_, __) => const EventsPage()),
        GoRoute(path: '/culture/volunteer', builder: (_, __) => const VolunteerPage()),
        GoRoute(
          path: '/admin/metrics',
          builder: (context, state) => const AdminMetricsPage(),
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
          path: '/admin/settings',
          builder: (context, state) => const AdminSettingsPage(),
        ),
        GoRoute(
          path: '/review',
          builder: (context, state) => const ReviewPage(),
        ),
        GoRoute(
          path: '/my-submissions',
          builder: (context, state) => const MySubmissionsPage(),
        ),
        GoRoute(
          path: '/feedback',
          builder: (context, state) => const FeedbackPage(),
        ),
        GoRoute(
          path: '/bookmarks',
          builder: (context, state) => const BookmarksPage(),
        ),
        // ── 学生 AI 功能路由 ──
        GoRoute(path: '/student/daily-briefing', builder: (_, __) => const DailyBriefingPage()),
        GoRoute(path: '/student/learning-diary', builder: (_, __) => const LearningDiaryPage()),
        GoRoute(path: '/student/checkin', builder: (_, __) => const CheckinPage()),
        GoRoute(path: '/student/digital-twin', builder: (_, __) => const DigitalTwinPage()),
        GoRoute(path: '/student/personality', builder: (_, __) => const PersonalityInsightPage()),
        GoRoute(path: '/student/achievements', builder: (_, __) => const AchievementsPage()),
        GoRoute(path: '/student/course-map', builder: (_, __) => const CourseMapPage()),
        GoRoute(path: '/student/course-analytics', builder: (_, __) => const CourseAnalyticsPage()),
        GoRoute(path: '/student/weekly-report', builder: (_, __) => const WeeklyReportPage()),
        GoRoute(path: '/student/freshman-plan', builder: (_, __) => const FreshmanPlanPage()),
        GoRoute(path: '/student/growth-path', builder: (_, __) => const GrowthPathPage()),
        GoRoute(path: '/student/political-study', builder: (_, __) => const PoliticalStudyPage()),
        GoRoute(path: '/student/ideological-record', builder: (_, __) => const IdeologicalRecordPage()),
        GoRoute(path: '/student/party-progress', builder: (_, __) => const PartyProgressPage()),
        GoRoute(path: '/student/campus-life', builder: (_, __) => const CampusLifePage()),
        GoRoute(path: '/student/schedule', builder: (_, __) => const ScheduleManagerPage()),
        GoRoute(path: '/student/competition-match', builder: (_, __) => const CompetitionMatchPage()),
        GoRoute(path: '/student/study-buddy', builder: (_, __) => const StudyBuddyPage()),
        GoRoute(path: '/student/mental-health', builder: (_, __) => const MentalHealthPage()),
        GoRoute(path: '/student/digital-mentor', builder: (_, __) => const DigitalMentorPage()),
        GoRoute(path: '/student/qa-plaza', builder: (_, __) => const QAPlazaPage()),
        GoRoute(path: '/student/hot-topics', builder: (_, __) => const HotTopicsPage()),
        GoRoute(path: '/student/qa-leaderboard', builder: (_, __) => const QALeaderboardPage()),
        GoRoute(path: '/student/private-chat', builder: (_, __) => const PrivateChatPage()),
        GoRoute(path: '/student/process-enhanced', builder: (_, __) => const ProcessEnhancedPage()),
        // ── 辅导员 AI 功能路由 ──
        GoRoute(path: '/counselor/daily-focus', builder: (_, __) => const DailyFocusPage()),
        GoRoute(path: '/counselor/class-report', builder: (_, __) => const ClassReportPage()),
        GoRoute(path: '/counselor/twin-board', builder: (_, __) => const TwinBoardPage()),
        GoRoute(path: '/counselor/prediction', builder: (_, __) => const PredictionPage()),
        GoRoute(path: '/counselor/intervention', builder: (_, __) => const InterventionPage()),
        GoRoute(path: '/counselor/talk-record', builder: (_, __) => const TalkRecordPage()),
        GoRoute(path: '/counselor/talk-tips', builder: (_, __) => const TalkTipsPage()),
        GoRoute(path: '/counselor/ideological', builder: (_, __) => const CounselorIdeologicalPage()),
        GoRoute(path: '/counselor/class-profile', builder: (_, __) => const ClassProfilePage()),
        GoRoute(path: '/counselor/community-manage', builder: (_, __) => const CommunityManagePage()),
        GoRoute(path: '/counselor/hot-topic-sense', builder: (_, __) => const HotTopicSensePage()),
        GoRoute(path: '/counselor/process-edit', builder: (_, __) => const ProcessEditPage()),
        GoRoute(path: '/counselor/student-list', builder: (_, __) => const StudentListPage()),
        // ── 教师 AI 功能路由 ──
        GoRoute(path: '/teacher/daily-overview', builder: (_, __) => const DailyOverviewPage()),
        GoRoute(path: '/teacher/lesson-prep', builder: (_, __) => const LessonPrepPage()),
        GoRoute(path: '/teacher/exam-gen', builder: (_, __) => const ExamGenPage()),
        GoRoute(path: '/teacher/class-interact', builder: (_, __) => const ClassInteractPage()),
        GoRoute(path: '/teacher/grading', builder: (_, __) => const GradingPage()),
        GoRoute(path: '/teacher/heatmap', builder: (_, __) => const HeatmapPage()),
        GoRoute(path: '/teacher/reflection', builder: (_, __) => const ReflectionPage()),
        GoRoute(path: '/teacher/style-dist', builder: (_, __) => const StyleDistPage()),
        GoRoute(path: '/teacher/community-qa', builder: (_, __) => const CommunityQAPage()),
        // ── 教辅/学生会/学院管理员 AI 功能路由 ──
        GoRoute(path: '/assistant/schedule-check', builder: (_, __) => const ScheduleCheckPage()),
        GoRoute(path: '/assistant/grad-audit', builder: (_, __) => const GradAuditPage()),
        GoRoute(path: '/assistant/exam-arrange', builder: (_, __) => const ExamArrangePage()),
        GoRoute(path: '/union/event-plan', builder: (_, __) => const EventPlanPage()),
        GoRoute(path: '/union/poster-gen', builder: (_, __) => const PosterGenPage()),
        GoRoute(path: '/college/twin-screen', builder: (_, __) => const TwinScreenPage()),
        GoRoute(path: '/college/data-analysis', builder: (_, __) => const DataAnalysisPage()),

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

const _navItems = [
  _NavItem('首页', Icons.home_outlined, Icons.home, '/home'),
  _NavItem('对话', Icons.chat_bubble_outline, Icons.chat_bubble, '/chat'),
  _NavItem('知识', Icons.menu_book_outlined, Icons.menu_book, '/browse'),
  _NavItem('办事', Icons.assignment_outlined, Icons.assignment, '/enrollment'),
  _NavItem('我的', Icons.person_outline, Icons.person, '/profile'),
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
    final index = _currentIndex(context);
    return Scaffold(
      body: Row(
        children: [
          NavigationRail(
            selectedIndex: index,
            onDestinationSelected: (i) => _onTap(context, i),
            labelType: NavigationRailLabelType.all,
            leading: Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Icon(Icons.school,
                  size: 32, color: Theme.of(context).colorScheme.primary),
            ),
            destinations: _navItems
                .map((item) => NavigationRailDestination(
                      icon: Icon(item.icon),
                      selectedIcon: Icon(item.selectedIcon),
                      label: Text(item.label),
                    ))
                .toList(),
          ),
          const VerticalDivider(width: 1),
          Expanded(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 900),
              child: RepaintBoundary(key: screenshotKey, child: child),
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
    final index = _currentIndex(context);
    return Scaffold(
      body: RepaintBoundary(key: screenshotKey, child: child),
      floatingActionButton: const FabMenu(),
      bottomNavigationBar: ClipRRect(
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 12, sigmaY: 12),
          child: Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface.withValues(alpha: 0.7),
              border: Border(
                top: BorderSide(
                  color: theme.colorScheme.outlineVariant.withValues(alpha: 0.3),
                ),
              ),
            ),
            child: NavigationBar(
              selectedIndex: index,
              onDestinationSelected: (i) => _onTap(context, i),
              destinations: _navItems
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
    );
  }

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    for (int i = 0; i < _navItems.length; i++) {
      if (location.startsWith(_navItems[i].route)) return i;
    }
    return 0;
  }

  void _onTap(BuildContext context, int index) {
    context.go(_navItems[index].route);
  }
}
