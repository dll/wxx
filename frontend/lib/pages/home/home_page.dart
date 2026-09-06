import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../config/release_config.dart';
import '../../providers/chat_provider.dart';
import '../../providers/emotion_provider.dart';
import '../../providers/session_provider.dart';
import '../../providers/notification_provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../providers/update_provider.dart';
import '../../providers/ai_briefing_provider.dart';
import '../../main.dart';
import '../../utils/role_utils.dart';
import '../../utils/capability_utils.dart';
import '../../utils/storage.dart';
import '../../utils/date_utils.dart';
import '../../config/api_config.dart';
import '../../services/api_service.dart';
import '../../widgets/avatar_card.dart';
import '../../widgets/consent_dialog.dart';
import '../../widgets/datetime_banner.dart';
import '../../widgets/error_view.dart';
import '../../widgets/skeleton.dart';
import '../../widgets/student_interest_pick_dialog.dart';
import '../teacher/daily_overview_page.dart';
import 'student_home_skeleton.dart';
import 'home_welcome_banner.dart';
import 'home_error_card.dart';
import 'home_calendar_bar.dart';
import 'home_overview_item.dart';
import 'home_empty_card.dart';
import 'home_alert_overview.dart';

// ── 学生专区卡片配置 ──
class _FeatureCard {
  final IconData icon;
  final String label;
  final Color color;
  final String route;

  /// 年级优先级（1~4 年级分别给 0~5 的权重，越小越靠前；无值视为 6）
  final List<int> gradePriority;

  /// 兴趣标签：与「关注内容」匹配，命中则整体提前
  final String interestKey;

  const _FeatureCard(this.icon, this.label, this.color, this.route,
      {this.gradePriority = const [], this.interestKey = ''});
}

// 学生专区卡片（含年级优先级 + 兴趣标签，供首页按年级/关注内容定制排序）
const _studentFeatures = [
  _FeatureCard(Icons.school_outlined, '新生指南', Color(0xFF00695C),
      '/student/freshmen-guide',
      gradePriority: [1, 9, 9, 9], interestKey: '校园生活'),
  _FeatureCard(Icons.topic_outlined, '毕设选题', Color(0xFF1565C0), '/graduation',
      gradePriority: [9, 9, 2, 1], interestKey: '竞赛科研'),
  _FeatureCard(
      Icons.emoji_events_outlined, '学科竞赛', Color(0xFFE65100), '/competition',
      gradePriority: [9, 1, 1, 3], interestKey: '竞赛科研'),
  _FeatureCard(Icons.calendar_today, '大学规划', Color(0xFF2E7D32), '/plan',
      gradePriority: [4, 4, 3, 2], interestKey: '职业就业'),
  _FeatureCard(
      Icons.flag_outlined, '入党教育', Color(0xFFC62828), '/party-education',
      gradePriority: [3, 2, 2, 9], interestKey: '思想政治'),
  _FeatureCard(Icons.groups_outlined, '社团生活', Color(0xFF7B1FA2), '/club',
      gradePriority: [2, 3, 9, 9], interestKey: '校园生活'),
];

const _educationFeatures = [
  _FeatureCard(
      Icons.work_outline, '就业服务', Color(0xFFE65100), '/student/career'),
  _FeatureCard(
      Icons.menu_book_outlined, '学业服务', Color(0xFF1565C0), '/student/study'),
  _FeatureCard(
      Icons.checklist_rtl, '学习计划', Color(0xFF00695C), '/student/study-plan'),
  _FeatureCard(
      Icons.favorite_outline, '心理健康', Color(0xFFC62828), '/student/mental'),
];

/// 角色工作台条目（能力门控后的一枚快捷卡片）
class _WorkbenchEntry {
  final IconData icon;
  final String label;
  final Color color;
  final String route;
  const _WorkbenchEntry(this.icon, this.label, this.color, this.route);
}

/// 首页仪表盘 — 按角色自适应布局
class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  // 学生首页数据
  Map<String, dynamic>? _studentHomeData;
  bool _studentHomeLoading = false;
  String? _studentHomeError;
  // 教师授课申报待审角标（R3 补 H）：教辅/教务可见
  int _teacherCoursePending = 0;

  @override
  void initState() {
    super.initState();
    // 延迟加载数据，避免 build 中途触发 notifyListeners
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadData();
      _checkConsent();
      _checkAppUpdate();
      _maybeShowOnboarding();
      _maybeShowInterestPick();
    });
  }

  /// 未采集「关注内容」的学生，弹出多选兴趣页（仅首次，可跳过）——
  /// 用于按关注内容定制首页学生专区排序。
  void _maybeShowInterestPick() {
    if (!Storage.isLoggedIn) return;
    final role = Storage.role;
    final isStudent = role == 'student' || role == 'student_union';
    if (!isStudent) return;
    if (Storage.studentInterestsCollected) return;
    // 首次新生引导未完成时，先完成引导再采集兴趣，避免连续弹窗
    if (!Storage.freshmanGuideSeen) return;
    Future.delayed(const Duration(milliseconds: 300), () {
      if (!mounted) return;
      _showInterestPickDialog();
    });
  }

  /// 关注内容多选对话框（学业/竞赛/就业/思政/心理/校园生活）
  Future<void> _showInterestPickDialog() async {
    final result = await pickupStudentInterests(context);
    if (result != null) {
      await Storage.setStudentInterests(result);
      if (mounted) setState(() {}); // 刷新学生专区排序
    }
  }

  /// 首次登录的学生，弹出大一新生应用内引导（仅一次，可跳过）
  /// 仅对大一新生自动弹出应用内引导（硬门控：年级=1 且未看过）。
  /// 老生/其他年级不自动弹窗、不写入 freshmanGuideSeen 数据；
  /// 想了解时仍可从学生专区「新生指南」等入口主动进入引导页。
  void _maybeShowOnboarding() {
    if (!Storage.isLoggedIn) return;
    final role = Storage.role;
    final isStudent = role == 'student' || role == 'student_union';
    if (!isStudent) return;
    // 年级硬门控：仅大一（grade == 1）自动弹窗。
    if (Storage.grade != 1) return;
    if (Storage.freshmanGuideSeen) return;
    // 让首页先渲染完，再轻量弹出，避免打扰正在进行的操作
    Future.delayed(const Duration(milliseconds: 600), () {
      if (!mounted) return;
      context.push('/student/freshman-onboarding');
    });
  }

  /// 检查应用更新
  Future<void> _checkAppUpdate() async {
    final updateProvider = context.read<UpdateProvider>();
    final hasUpdate = await updateProvider.checkUpdate(silent: true);
    if (hasUpdate && mounted) {
      updateProvider.showUpdateDialog(context);
    }
  }

  /// 检查是否已同意隐私政策与用户协议，未同意则弹出授权弹窗
  void _checkConsent() {
    if (!Storage.isLoggedIn) return;
    if (Storage.consented) return;

    ConsentDialog.show(context).then((agreed) {
      if (agreed == true) {
        Storage.setConsented(true);
        // 异步通知后端记录同意状态（失败静默，不影响使用）
        ApiService().post(ApiConfig.consent, data: {}).then((_) {},
            onError: (Object _) {});
      }
    });
  }

  void _loadData() {
    if (!Storage.isLoggedIn) return; // 游客无需加载数据
    final role = Storage.role;
    if (_canAccessAlerts(role)) {
      context.read<EmotionProvider>().fetchStats();
    }
    context.read<SessionProvider>().fetchSessions();
    // 加载未读通知数量
    context.read<NotificationProvider>().fetchUnreadCount();
    // R3 补 H：教辅/教务持有审核能力时拉取待审角标
    if (CapabilityUtils.has(Capability.teacherCourseReview)) {
      _loadTeacherCoursePending();
    }
    // 学生角色加载个性化首页数据 + 数字人形象
    if (role == 'student' || role == 'student_union') {
      _loadStudentHome();
      context.read<StudentFeatureProvider>().fetchAvatar(
          displayName: Storage.displayName ?? '同学',
          role: Storage.role ?? 'student');
    }
  }

  /// 加载学生首页数据
  Future<void> _loadStudentHome() async {
    setState(() {
      _studentHomeLoading = true;
      _studentHomeError = null;
    });
    try {
      final res = await ApiService().get(ApiConfig.studentHome);
      if (res.statusCode == 200 && res.data != null) {
        setState(() {
          _studentHomeData = res.data is Map<String, dynamic>
              ? res.data
              : (res.data['data'] as Map<String, dynamic>?);
        });
      }
    } catch (e) {
      if (e is DioException && e.response?.statusCode == 404) {
        _studentHomeData = null;
        _studentHomeError = null;
      } else {
        setState(() {
          _studentHomeError = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _studentHomeLoading = false;
        });
      }
    }
  }

  /// 切换任务状态
  Future<void> _toggleTaskStatus(Map<String, dynamic> task) async {
    final taskId = task['id']?.toString() ?? '';
    final planId = task['plan_id']?.toString() ?? '';
    final currentStatus = task['status'] ?? 'pending';
    if (taskId.isEmpty || planId.isEmpty) return;

    try {
      final newStatus = currentStatus == 'completed' ? 'pending' : 'completed';
      await ApiService().patch(
        ApiConfig.studyPlanTask(planId, taskId),
        data: {'status': newStatus},
      );
      // 重新加载首页数据
      _loadStudentHome();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('操作失败：${e.toString()}')),
        );
      }
    }
  }

  /// 加载教辅/教务授课申报待审角标（R3 补 H）；失败静默，角标保持 0 / 不显示
  Future<void> _loadTeacherCoursePending() async {
    if (!CapabilityUtils.has(Capability.teacherCourseReview)) return;
    try {
      final res = await ApiService().get(ApiConfig.teacherCoursesPendingCount);
      final pending = res.data is Map ? (res.data['pending'] ?? 0) : 0;
      if (!mounted) return;
      setState(
          () => _teacherCoursePending = pending is num ? pending.toInt() : 0);
    } catch (_) {
      // 静默：角标拉取失败不影响首页；审核页内仍有诚实空态
    }
  }

  bool _canAccessAlerts(String? role) => RoleUtils.canAccessEmotion(role);

  /// 首页标题：品牌 + 当前年级主题徽标（如「迎新」），让主题全站可感知
  Widget _buildHomeTitle(ThemeData theme) {
    final themeNotifier = context.watch<ThemeNotifier>();
    final accent = themeNotifier.gradeAccent;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.auto_awesome, size: 20, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        const Text('蔚小芯'),
        const SizedBox(width: 10),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
            color: accent.withOpacity(0.12),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: accent.withOpacity(0.4)),
          ),
          child: Text(
            '${themeNotifier.gradeThemeName}主题',
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w700,
              color: accent,
            ),
          ),
        ),
      ],
    );
  }

  /// 区块标题：图标 + 主标题 + 可选副标题，统一视觉层次
  Widget _buildSectionHeader(
    ThemeData theme, {
    required IconData icon,
    required String title,
    String? subtitle,
    Color? iconColor,
  }) {
    return Row(
      children: [
        Container(
          width: 34,
          height: 34,
          decoration: BoxDecoration(
            color: (iconColor ?? theme.colorScheme.primary).withOpacity(0.12),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Icon(icon,
              size: 19, color: iconColor ?? theme.colorScheme.primary),
        ),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title,
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
            if (subtitle != null)
              Text(subtitle,
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.outline,
                  )),
          ],
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final role = Storage.role;
    final loggedIn = Storage.isLoggedIn;
    if (loggedIn && role == 'teacher') {
      return const DailyOverviewPage(homeMode: true);
    }
    final theme = Theme.of(context);
    final isStudent = role == 'student' || role == 'student_union';
    final notificationProvider = context.watch<NotificationProvider>();
    final unreadCount = notificationProvider.unreadCount;

    return Scaffold(
      appBar: AppBar(
        title: _buildHomeTitle(theme),
        actions: [
          if (loggedIn)
            Stack(
              children: [
                IconButton(
                  icon: const Icon(Icons.notifications_outlined),
                  tooltip: '消息',
                  onPressed: () {
                    context.push('/notifications');
                  },
                ),
                if (unreadCount > 0)
                  Positioned(
                    right: 8,
                    top: 8,
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: Colors.red,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      constraints:
                          const BoxConstraints(minWidth: 16, minHeight: 16),
                      child: Text(
                        unreadCount > 99 ? '99+' : '$unreadCount',
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                        ),
                        textAlign: TextAlign.center,
                      ),
                    ),
                  ),
              ],
            ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async => _loadData(),
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (loggedIn)
              _buildWelcomeBanner(theme)
            else
              _buildGuestBanner(theme),
            const SizedBox(height: 12),
            // APK 下载只对 Web 游客展示，登录用户首页优先呈现任务。
            if (!loggedIn) ...[
              GestureDetector(
                onTap: () => _showApkDownloadDialog(context, theme),
                child: Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      colors: [
                        const Color(0xFF2E7D32).withOpacity(0.08),
                        const Color(0xFF2E7D32).withOpacity(0.04),
                      ],
                    ),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(
                      color: const Color(0xFF2E7D32).withOpacity(0.25),
                    ),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.android,
                          color: Color(0xFF2E7D32), size: 20),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('安卓应用下载',
                                style: theme.textTheme.labelLarge
                                    ?.copyWith(fontWeight: FontWeight.w600)),
                            const SizedBox(height: 2),
                            Text('扫码下载 APK • v${ReleaseConfig.version}',
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant,
                                )),
                          ],
                        ),
                      ),
                      Icon(Icons.arrow_forward_ios,
                          size: 16, color: theme.colorScheme.primary),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 20),
            ],
            // 角色工作台（能力门控：辅导员/教辅/学生会/书记/管理；学生/教师自动不显示）
            if (loggedIn) ...[
              _buildRoleWorkbench(theme),
            ],
            // 学生个性化首页
            if (isStudent && loggedIn) ...[
              _buildStudentHomeContent(theme),
              const SizedBox(height: 20),
            ],
            // AI 简讯（所有登录用户可见）
            if (loggedIn) ...[
              _buildAIBriefingCard(theme),
              const SizedBox(height: 20),
            ],
            // 非学生角色或未登录：显示日期时间 + 告警概览
            if (!isStudent || !loggedIn) ...[
              // 日期时间 + 校历入口
              const DateTimeBanner(),
              const SizedBox(height: 20),
              // 辅导员及以上：告警概览
              if (_canAccessAlerts(role)) ...[
                _buildAlertOverview(theme),
                const SizedBox(height: 20),
              ],
            ],
            // 所有角色：知识入口
            _buildKnowledgeEntry(theme),
            const SizedBox(height: 20),
            // 所有角色：校园服务
            _buildCampusService(theme),
            const SizedBox(height: 20),
            // 学生专区（student/student_union）
            if (isStudent) ...[
              _buildStudentFeatures(theme),
              const SizedBox(height: 20),
              _buildEducationFeatures(theme),
              const SizedBox(height: 20),
              // 数字画像降为次级入口，不再占据学生首屏。
              if (Storage.showAvatar) ...[
                _buildAvatarBanner(theme),
                const SizedBox(height: 20),
              ],
            ],
            // 管理专区（college_admin+）
            if (role == 'college_admin' ||
                role == 'school_admin' ||
                role == 'sys_admin') ...[
              _buildAdminFeatures(theme),
              const SizedBox(height: 20),
            ],
            // 最近对话
            if (loggedIn) _buildRecentSessions(theme),
          ],
        ),
      ),
    );
  }

  Future<void> _openApkDownload() async {
    final uri = Uri.parse(ReleaseConfig.apkDownloadUrl);
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  void _showApkDownloadDialog(BuildContext context, ThemeData theme) {
    final qrUrl =
        ReleaseConfig.qrCodeUrl(ReleaseConfig.apkDownloadUrl, size: 240);
    showDialog(
      context: context,
      barrierDismissible: true,
      builder: (ctx) => AlertDialog(
        contentPadding: EdgeInsets.zero,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  colors: [
                    const Color(0xFF2E7D32).withOpacity(0.08),
                    const Color(0xFF2E7D32).withOpacity(0.04),
                  ],
                ),
                borderRadius: const BorderRadius.only(
                  topLeft: Radius.circular(20),
                  topRight: Radius.circular(20),
                ),
              ),
              child: Row(
                children: [
                  const Icon(Icons.android, color: Color(0xFF2E7D32), size: 28),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('蔚小芯 APK 下载',
                            style: theme.textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.w700)),
                        Text('扫码或点击下方链接下载',
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            )),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(16),
                      border: Border.all(
                        color:
                            theme.colorScheme.outlineVariant.withOpacity(0.3),
                      ),
                    ),
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(10),
                      child: Image.network(
                        qrUrl,
                        width: 160,
                        height: 160,
                        fit: BoxFit.cover,
                        errorBuilder: (_, __, ___) => const SizedBox(
                          width: 160,
                          height: 160,
                          child: Icon(Icons.qr_code_2,
                              size: 96, color: Colors.black54),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    '版本 v${ReleaseConfig.version} • ${ReleaseConfig.apkFileName}',
                    style: theme.textTheme.labelMedium
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 16),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton.icon(
                      onPressed: _openApkDownload,
                      icon: const Icon(Icons.download_rounded, size: 18),
                      label: const Text('直接下载 APK'),
                    ),
                  ),
                  const SizedBox(height: 8),
                  SizedBox(
                    width: double.infinity,
                    child: OutlinedButton.icon(
                      onPressed: () => Navigator.pop(ctx),
                      icon: const Icon(Icons.close, size: 18),
                      label: const Text('关闭'),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 滁州学院快讯（游客横幅）
  Widget _buildGuestBanner(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [
            Color(0xFF1565C0),
            Color(0xFF7B1FA2),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.school,
                  color: Colors.white.withOpacity(0.9), size: 28),
              const SizedBox(width: 8),
              Text(
                '滁州学院 · 公开快讯',
                style: TextStyle(
                  color: Colors.white.withOpacity(0.9),
                  fontSize: 14,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          const Text(
            '欢迎来到滁州学院 👋',
            style: TextStyle(
              color: Colors.white,
              fontSize: 22,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            '26 级新生 · 学生家长 · 中学教师 · 社会访客',
            style: TextStyle(
              color: Colors.white.withOpacity(0.85),
              fontSize: 13,
            ),
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: ElevatedButton.icon(
                  onPressed: () => context.go('/login'),
                  icon: const Icon(Icons.login, size: 18),
                  label: const Text('登录 / 注册'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.white,
                    foregroundColor: const Color(0xFF1565C0),
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => context.go('/browse'),
                  icon: const Icon(Icons.explore_outlined, size: 18),
                  label: const Text('直接浏览'),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.white,
                    side: BorderSide(color: Colors.white.withOpacity(0.6)),
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  /// 欢迎横幅
  Widget _buildWelcomeBanner(ThemeData theme) {
    final displayName = Storage.displayName ?? '同学';
    final themeNotifier = context.watch<ThemeNotifier>();
    return HomeWelcomeBanner(
        displayName: displayName,
        gradeThemeName: themeNotifier.gradeThemeName,
        accent: themeNotifier.gradeAccent,
        onOpenChat: () => context.go('/chat'));
  }

  /// 告警统计概览（辅导员及以上可见）
  Widget _buildAlertOverview(ThemeData theme) {
    final stats = context.watch<EmotionProvider>();

    return HomeAlertOverview(
      loading: stats.statsLoading,
      urgent: stats.stats?.urgent ?? 0,
      high: stats.stats?.high ?? 0,
      pending: stats.stats?.pending ?? 0,
      onViewAll: () => context.go('/emotion'),
    );
  }

  /// 知识入口 — 四类快捷卡片
  Widget _buildKnowledgeEntry(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildSectionHeader(theme,
            icon: Icons.menu_book_outlined,
            title: '知识大厅',
            subtitle: '政策 · 流程 · 问答 · 活动'),
        const SizedBox(height: 14),
        Row(
          children: [
            _buildKnowledgeCard(
              theme,
              icon: Icons.gavel,
              label: '政策',
              color: const Color(0xFF1565C0),
              onTap: () => context.go('/browse'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.account_tree,
              label: '流程',
              color: const Color(0xFF2E7D32),
              onTap: () => context.go('/enrollment'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.help_outline,
              label: '问答',
              color: const Color(0xFFE65100),
              onTap: () => context.go('/chat'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.event,
              label: '活动',
              color: const Color(0xFF7B1FA2),
              onTap: () => context.go('/browse'),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildKnowledgeCard(
    ThemeData theme, {
    required IconData icon,
    required String label,
    required Color color,
    required VoidCallback onTap,
  }) {
    return Expanded(
      child: Material(
        color: color.withOpacity(0.07),
        borderRadius: BorderRadius.circular(14),
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(14),
          child: Container(
            padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 6),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(14),
              border: Border.all(color: color.withOpacity(0.12)),
            ),
            child: Column(
              children: [
                Container(
                  width: 44,
                  height: 44,
                  decoration: BoxDecoration(
                    color: color.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(icon, color: color, size: 24),
                ),
                const SizedBox(height: 8),
                Text(
                  label,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// AI 简讯卡片 — 点击进入 AI 简讯列表
  Widget _buildAIBriefingCard(ThemeData theme) {
    return Consumer<AIBriefingProvider>(
      builder: (context, provider, _) {
        // 首次进入时拉取最新资讯（取前 3 条展示）；失败后不自动重试，避免叠加限流
        if (!provider.userLoaded && !provider.userLoading) {
          Future.microtask(() {
            if (context.mounted) provider.fetchUserBriefings();
          });
        }
        final latest = provider.userBriefings.take(3).toList();
        return Material(
          color: theme.colorScheme.primaryContainer.withOpacity(0.5),
          borderRadius: BorderRadius.circular(16),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => context.go('/ai-briefings'),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.primary,
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Icon(Icons.newspaper,
                            color: Colors.white, size: 20),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('AI 简讯',
                                style: theme.textTheme.titleMedium
                                    ?.copyWith(fontWeight: FontWeight.w700)),
                            Text('AI 教学 / 工具 / 版本 / 行业热点',
                                style: theme.textTheme.bodySmall),
                          ],
                        ),
                      ),
                      const Icon(Icons.chevron_right),
                    ],
                  ),
                  if (latest.isNotEmpty) ...[
                    const SizedBox(height: 10),
                    ...latest.map((b) => Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Row(
                            children: [
                              const Icon(Icons.circle, size: 6),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text(
                                  b.topic,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: theme.textTheme.bodyMedium,
                                ),
                              ),
                            ],
                          ),
                        )),
                  ],
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  /// 校园服务 — 地图/全景/学院/官网
  Widget _buildCampusService(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildSectionHeader(theme,
            icon: Icons.location_city,
            title: '校园服务',
            subtitle: '导航 · 全景 · 学院入口'),
        const SizedBox(height: 14),
        Row(
          children: [
            _buildKnowledgeCard(
              theme,
              icon: Icons.map_outlined,
              label: '校园导航',
              color: const Color(0xFF1677FF),
              onTap: () => context.go('/campus?v=map'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.view_in_ar,
              label: 'VR全景',
              color: const Color(0xFF7B1FA2),
              onTap: () => context.go('/campus?v=vr'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.computer,
              label: '计算机学院',
              color: const Color(0xFF2E7D32),
              onTap: () => context.go('/campus?v=csci'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.school,
              label: '学校首页',
              color: const Color(0xFF1565C0),
              onTap: () => context.go('/campus?v=home'),
            ),
          ],
        ),
      ],
    );
  }

  /// 学生专区 — 毕设选题/学科竞赛/大学规划/入党教育/社团生活
  Widget _buildStudentFeatures(ThemeData theme) {
    final ordered = _orderedStudentFeatures();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildSectionHeader(theme,
            icon: Icons.school_outlined,
            title: '学生专区',
            subtitle: _studentSubtitle()),
        const SizedBox(height: 14),
        Row(
          children: [
            for (final f in ordered.take(3))
              _buildKnowledgeCard(
                theme,
                icon: f.icon,
                label: f.label,
                color: f.color,
                onTap: () => context.go(f.route),
              ),
          ],
        ),
        const SizedBox(height: 10),
        Row(
          children: [
            for (final f in ordered.skip(3))
              _buildKnowledgeCard(
                theme,
                icon: f.icon,
                label: f.label,
                color: f.color,
                onTap: () => context.go(f.route),
              ),
            // 第 3 个位置留空保持对齐
            const Expanded(child: SizedBox.shrink()),
          ],
        ),
      ],
    );
  }

  /// 学生专区按「年级 + 关注内容」定制排序：
  /// 1) 主键：当前年级的优先级（越小越靠前）
  /// 2) 次键：命中「关注内容」的卡片整体提前（同年级档次内优先）
  /// 3) 兜底：保持原有声明顺序
  List<_FeatureCard> _orderedStudentFeatures() {
    final g = Storage.grade; // 0=未知年级，仍按兴趣/声明顺序
    final interests = Storage.studentInterests.toSet();
    final list = List.of(_studentFeatures);
    list.sort((a, b) {
      final ap = _cardGradePriority(a, g);
      final bp = _cardGradePriority(b, g);
      if (ap != bp) return ap.compareTo(bp);
      // 年级档次相同，命中关注内容的优先
      final ai = interests.contains(a.interestKey) ? 0 : 1;
      final bi = interests.contains(b.interestKey) ? 0 : 1;
      if (ai != bi) return ai.compareTo(bi);
      return _studentFeatures.indexOf(a).compareTo(_studentFeatures.indexOf(b));
    });
    return list;
  }

  int _cardGradePriority(_FeatureCard f, int grade) {
    if (grade >= 1 && grade <= 4 && f.gradePriority.length >= grade) {
      final v = f.gradePriority[grade - 1];
      if (v >= 0) return v;
    }
    return 9; // 未知年级或未配置：排到较后，但仍按兴趣/声明排序
  }

  /// 学生专区副标题：展示年级阶段的定制说明
  String _studentSubtitle() {
    final g = Storage.grade;
    final phase = switch (g) {
      1 => '大一 · 报到 · 适应',
      2 => '大二 · 打基础 · 拓能力',
      3 => '大三 · 定方向 · 深专业',
      4 => '大四 · 毕业 · 就业',
      _ => '成长 · 竞赛 · 规划 · 组织',
    };
    if (Storage.studentInterestsCollected &&
        Storage.studentInterests.isNotEmpty) {
      return '$phase · 已按关注定制';
    }
    return phase;
  }

  /// 教育三大模块 — 就业/学业/学习计划/心理
  Widget _buildEducationFeatures(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildSectionHeader(theme,
            icon: Icons.auto_stories_outlined,
            title: '教育服务',
            subtitle: '就业 · 学业 · 计划 · 心理'),
        const SizedBox(height: 14),
        Row(
          children: [
            for (final f in _educationFeatures)
              _buildKnowledgeCard(
                theme,
                icon: f.icon,
                label: f.label,
                color: f.color,
                onTap: () => context.go(f.route),
              ),
          ],
        ),
      ],
    );
  }

  /// 角色工作台 — 按能力门控为各角色展示专属工作入口(卡片区)
  /// 学生/教师自动不显示(无对应能力)，返回空。复用 _buildKnowledgeCard 卡片样式。
  Widget _buildRoleWorkbench(ThemeData theme) {
    // 收集该用户有权访问的工作台入口
    final entries = <_WorkbenchEntry>[];

    // ── 辅导员工作台 ──
    if (CapabilityUtils.has(Capability.counselorAlertRead)) {
      entries.add(const _WorkbenchEntry(
          Icons.warning_amber_rounded, '情感预警', Color(0xFFC62828), '/emotion'));
    }
    if (CapabilityUtils.has(Capability.counselorDailyFocusRead)) {
      entries.add(const _WorkbenchEntry(Icons.today_outlined, '今日关注',
          Color(0xFF1565C0), '/counselor/daily-focus'));
    }
    if (CapabilityUtils.has(Capability.counselorClassReport)) {
      entries.add(const _WorkbenchEntry(Icons.assessment_outlined, '班级学情日报',
          Color(0xFFE65100), '/counselor/class-report'));
    }
    if (CapabilityUtils.has(Capability.counselorTalkRecord)) {
      entries.add(const _WorkbenchEntry(Icons.forum_outlined, '谈心记录',
          Color(0xFF2E7D32), '/counselor/talk-record'));
    }
    if (CapabilityUtils.has(Capability.counselorStudentList)) {
      entries.add(const _WorkbenchEntry(Icons.people_alt_outlined, '学生名单',
          Color(0xFF00695C), '/counselor/student-list'));
    }
    if (CapabilityUtils.has(Capability.counselorSecondClassBoard)) {
      entries.add(const _WorkbenchEntry(Icons.school_outlined, '第二课堂',
          Color(0xFF00838F), '/counselor/second-class-board'));
    }
    if (CapabilityUtils.has(Capability.counselorTwinBoard)) {
      entries.add(const _WorkbenchEntry(Icons.dashboard_outlined, '学生孪生看板',
          Color(0xFF1565C0), '/counselor/twin-board'));
    }
    if (CapabilityUtils.has(Capability.counselorIdeological)) {
      entries.add(const _WorkbenchEntry(Icons.flag_outlined, '思想动态',
          Color(0xFF7B1FA2), '/counselor/ideological'));
    }

    // ── 教辅工作台 ──
    if (CapabilityUtils.hasAny([
      Capability.outcomeRecordWrite,
      Capability.outcomeReview,
    ])) {
      entries.add(const _WorkbenchEntry(Icons.task_alt, '毕业去向登记',
          Color(0xFFE65100), '/secretary/outcome-manage'));
    }
    if (CapabilityUtils.has(Capability.assistantScheduleCheck)) {
      entries.add(const _WorkbenchEntry(Icons.calendar_month_outlined, '排课核查',
          Color(0xFF1565C0), '/assistant/schedule-check'));
    }
    if (CapabilityUtils.has(Capability.assistantGradAudit)) {
      entries.add(const _WorkbenchEntry(Icons.workspace_premium_outlined,
          '毕业审核', Color(0xFF2E7D32), '/assistant/grad-audit'));
    }
    if (CapabilityUtils.has(Capability.assistantExamArrange)) {
      entries.add(const _WorkbenchEntry(Icons.edit_calendar_outlined, '考试安排',
          Color(0xFF7B1FA2), '/assistant/exam-arrange'));
    }
    if (Storage.role == 'assistant') {
      entries.add(const _WorkbenchEntry(Icons.build, '后勤服务台', Color(0xFF00695C),
          '/assistant/facility-workbench'));
    }

    // ── 教师授课申报审核（R3 补 H，2026-08-17）：教辅/教务审核 + 待审角标（teacher.course.review）──
    // 带红色角标卡：展示 pending-count，无权限不显示入口
    if (CapabilityUtils.has(Capability.teacherCourseReview)) {
      entries.add(_WorkbenchEntry(
          Icons.fact_check_outlined,
          _teacherCoursePending > 0
              ? '授课申报审核·$_teacherCoursePending'
              : '授课申报审核',
          const Color(0xFFE65100),
          '/assistant/teacher-course-review'));
    }

    // ── 党课/活动登记（蓝图第3块，2026-08-16）：教师/教辅登记 → 书记党建看板 ──
    if (CapabilityUtils.has(Capability.partyRecordWrite)) {
      entries.add(const _WorkbenchEntry(
          Icons.flag, '党课/活动登记', Color(0xFFC62828), '/teacher/party-register'));
    }

    // ── 教师成绩录入（P0-1，2026-08-17，方案A：教师自主声明授课）──
    // 门控 teacher.grade.write：教师录入所授班级真实成绩
    if (CapabilityUtils.has(Capability.teacherGradeWrite)) {
      entries.add(const _WorkbenchEntry(Icons.grade_outlined, '成绩录入',
          Color(0xFF1565C0), '/teacher/grades-entry'));
    }

    // ── 教师作业信息发布+成绩统计（2026-08-17，P2 轻量版）：门控 teacher.grade.write
    // 蔚小芯侧重教育非教学：作业仅信息发布+成绩统计，不做学生提交/批改/内容流转
    if (CapabilityUtils.has(Capability.teacherGradeWrite)) {
      entries.add(const _WorkbenchEntry(Icons.assignment_outlined, '作业发布',
          Color(0xFF00695C), '/teacher/homework'));
    }

    // ── 教师作业信息发布+成绩统计（P2 轻量版，2026-08-17）：门控 teacher.grade.write ──
    // 作业仅信息发布+成绩统计，不做学生提交/批改；发布前强校验 approved 授课关系。
    if (CapabilityUtils.has(Capability.teacherGradeWrite)) {
      entries.add(const _WorkbenchEntry(Icons.assignment_outlined, '作业发布',
          Color(0xFF2E7D32), '/teacher/homework'));
    }

    // ── 学生会工作台 ──
    if (CapabilityUtils.has(Capability.unionEventPlan)) {
      entries.add(const _WorkbenchEntry(Icons.event_available, '活动策划',
          Color(0xFFE65100), '/union/event-plan'));
    }
    if (CapabilityUtils.has(Capability.unionFeedbackList)) {
      entries.add(const _WorkbenchEntry(
          Icons.feedback_outlined, '反馈处理', Color(0xFFC62828), '/feedback'));
    }
    if (CapabilityUtils.hasAny([
      Capability.unionKbSubmit,
      Capability.unionPosterGen,
    ])) {
      entries.add(const _WorkbenchEntry(Icons.workspaces_outlined, '学生会工作台',
          Color(0xFF7B1FA2), '/union/workbench'));
    }

    // ── 书记 / 学院管理 工作台 ──
    if (CapabilityUtils.has(Capability.outcomeDashboard)) {
      entries.add(const _WorkbenchEntry(Icons.auto_graph, '教育成果大屏',
          Color(0xFF1565C0), '/secretary/education-outcome'));
    }
    // 书记党建育人 / 协同育人专项可视化深链（D1-1 功能补齐，2026-08-16）
    if (CapabilityUtils.has(Capability.outcomeDashboard)) {
      entries.add(const _WorkbenchEntry(Icons.flag, '党建育人专项', Color(0xFFC62828),
          '/secretary/party-dashboard'));
    }
    if (CapabilityUtils.has(Capability.collabDashboard)) {
      entries.add(const _WorkbenchEntry(Icons.groups, '协同育人专项',
          Color(0xFF00695C), '/secretary/collab-dashboard'));
    }
    if (CapabilityUtils.has(Capability.collegeTwinScreen)) {
      entries.add(const _WorkbenchEntry(
          Icons.dashboard, '数字孪生', Color(0xFF2E7D32), '/college/twin-screen'));
    }
    if (CapabilityUtils.has(Capability.collegeDataAnalysis)) {
      entries.add(const _WorkbenchEntry(Icons.analytics, '数据分析',
          Color(0xFF7B1FA2), '/college/data-analysis'));
    }

    // ── 系统管理 工作台 ──
    if (CapabilityUtils.has(Capability.systemSettingsWrite)) {
      entries.add(const _WorkbenchEntry(Icons.settings_outlined, '系统配置',
          Color(0xFF455A64), '/admin/settings'));
    }
    if (CapabilityUtils.has(Capability.systemAuditAll)) {
      entries.add(const _WorkbenchEntry(
          Icons.history, '审计日志', Color(0xFFC62828), '/admin/audit'));
    }

    // 无任何工作台能力(纯学生/教师/游客)→ 不显示
    if (entries.isEmpty) return const SizedBox.shrink();

    // 标题名：优先按角色定制，兜底用「工作台」
    final title = _workbenchTitle(Storage.role);
    // 每行 4 个，超出换行；行间加间距
    final rowWidgets = <Widget>[];
    for (var i = 0; i < entries.length; i += 4) {
      final end = i + 4 > entries.length ? entries.length : i + 4;
      final chunk = entries.sublist(i, end);
      rowWidgets.add(Row(
        children: [
          for (final e in chunk)
            _buildKnowledgeCard(
              theme,
              icon: e.icon,
              label: e.label,
              color: e.color,
              onTap: () => context.go(e.route),
            ),
          for (var padIdx = chunk.length; padIdx < 4; padIdx++)
            const Expanded(child: SizedBox.shrink()),
        ],
      ));
      if (end < entries.length) rowWidgets.add(const SizedBox(height: 10));
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildSectionHeader(theme,
            icon: Icons.workspaces_outline,
            title: title,
            subtitle: '日常事务 · 一站式处理'),
        const SizedBox(height: 14),
        ...rowWidgets,
      ],
    );
  }

  /// 角色工作台标题映射（兜底泛用）
  String _workbenchTitle(String? role) {
    switch (role) {
      case 'counselor':
        return '辅导员工作台';
      case 'assistant':
        return '教辅工作台';
      case 'student_union':
        return '学生会工作台';
      case 'college_admin':
      case 'school_admin':
        return '书记工作台';
      case 'sys_admin':
        return '管理工作台';
      case 'teacher':
        return '教师工作台';
      default:
        return '工作台';
    }
  }

  /// 管理专区 — 问题预案
  Widget _buildAdminFeatures(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('管理专区', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(
          children: [
            _buildKnowledgeCard(
              theme,
              icon: Icons.warning_rounded,
              label: '问题预案',
              color: const Color(0xFFC62828),
              onTap: () => context.go('/forecast'),
            ),
            for (final _ in [1, 2, 3]) const Expanded(child: SizedBox.shrink()),
          ],
        ),
      ],
    );
  }

  /// 最近对话
  Widget _buildRecentSessions(ThemeData theme) {
    final sessionProv = context.watch<SessionProvider>();
    final sessions = sessionProv.sessions;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('最近对话', style: theme.textTheme.titleMedium),
            if (sessions.isNotEmpty)
              TextButton.icon(
                onPressed: () => context.go('/sessions'),
                icon: const Icon(Icons.arrow_forward, size: 16),
                label: const Text('全部', style: TextStyle(fontSize: 13)),
              ),
          ],
        ),
        const SizedBox(height: 8),
        if (sessionProv.loading && sessions.isEmpty)
          const SessionsSkeleton()
        else if (sessions.isEmpty)
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(24),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.3),
              borderRadius: BorderRadius.circular(12),
            ),
            child: ErrorView.empty(
              message: '暂无对话记录',
              subtitle: '去对话页开始提问吧',
              icon: Icons.chat_bubble_outline,
            ),
          )
        else
          ...sessions.take(3).map((s) => ListTile(
                leading: const Icon(Icons.chat_bubble_outline, size: 20),
                title:
                    Text(s.title, maxLines: 1, overflow: TextOverflow.ellipsis),
                subtitle: Text(TimeFormatter.relative(s.updatedAt),
                    style: TextStyle(
                        fontSize: 12, color: theme.colorScheme.outline)),
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  // 加载该会话的历史消息并跳转到对话页
                  context.read<ChatProvider>().loadSession(s.id);
                  context.go('/chat');
                },
                dense: true,
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              )),
      ],
    );
  }

  // ============================================================
  // 学生个性化首页
  // ============================================================

  /// 首页数字人形象横幅（数据驱动、可隐藏）
  Widget _buildAvatarBanner(ThemeData theme) {
    final provider = context.watch<StudentFeatureProvider>();
    final avatar = provider.avatar;
    return GestureDetector(
      onTap: () => context.go('/student/profile'),
      child: avatar != null
          ? AvatarCard(
              config: avatar,
              height: 220,
            )
          : Container(
              height: 220,
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  colors: [
                    theme.colorScheme.primary.withOpacity(0.10),
                    theme.colorScheme.tertiary.withOpacity(0.08),
                  ],
                ),
                borderRadius: BorderRadius.circular(24),
                border: Border.all(
                  color: theme.colorScheme.primary.withOpacity(0.2),
                ),
              ),
              child: Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(Icons.person_pin_circle,
                        size: 40, color: theme.colorScheme.primary),
                    const SizedBox(height: 8),
                    Text(
                      '我的数字画像',
                      style: TextStyle(
                        color: theme.colorScheme.onSurfaceVariant,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '点击查看个性化数字人',
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.outline,
                      ),
                    ),
                  ],
                ),
              ),
            ),
    );
  }

  /// 学生首页主内容
  Widget _buildStudentHomeContent(ThemeData theme) {
    if (_studentHomeLoading) {
      return const StudentHomeSkeleton();
    }
    if (_studentHomeError != null) {
      return HomeErrorCard(
          message: _studentHomeError, onRetry: _loadStudentHome);
    }
    if (_studentHomeData == null) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildGradeGrowthCard(theme),
        const SizedBox(height: 16),
        _buildCalendarBar(theme),
        const SizedBox(height: 16),
        _buildTodayOverview(theme),
        const SizedBox(height: 16),
        _buildTodayCourses(theme),
        const SizedBox(height: 16),
        _buildTodayTasks(theme),
        const SizedBox(height: 16),
        _buildQuickEntries(theme),
        const SizedBox(height: 16),
        _buildUpcomingEvents(theme),
      ],
    );
  }

  /// 分年级成长计划卡片（大二/大三/大四粘性增强）：首屏即见"本阶段该做什么"
  Widget _buildGradeGrowthCard(ThemeData theme) {
    final grade = context.watch<ThemeNotifier>().grade;
    if (grade < 2 || grade > 4) return const SizedBox.shrink(); // 大一走开学待办

    final color = context.watch<ThemeNotifier>().gradeAccent;
    final name = context.watch<ThemeNotifier>().gradeThemeName;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [color.withOpacity(0.16), color.withOpacity(0.06)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: color.withOpacity(0.25)),
      ),
      child: Row(
        children: [
          Icon(Icons.auto_awesome, color: color, size: 28),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '本阶段成长计划',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w800),
                ),
                const SizedBox(height: 2),
                Text(
                  '$name阶段专属：现在该做什么，一看就懂',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          TextButton(
            onPressed: () => context.go('/student/grade-growth'),
            child: const Text('查看'),
          ),
        ],
      ),
    );
  }

  /// 校历信息条
  Widget _buildCalendarBar(ThemeData theme) {
    final today = _studentHomeData?['today'] as Map<String, dynamic>? ?? {};
    final weekNo = today['week_no'] ?? 0;
    final weekday = today['weekday'] ?? '';
    final semesterName = today['semester_name'] ?? '';

    return HomeCalendarBar(
        weekNo: weekNo, weekday: weekday, semesterName: semesterName);
    /* return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            theme.colorScheme.secondaryContainer,
            theme.colorScheme.tertiaryContainer.withOpacity(0.7),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: theme.colorScheme.onSecondaryContainer.withOpacity(0.1),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(
              Icons.calendar_today,
              color: theme.colorScheme.onSecondaryContainer,
              size: 24,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '第 $weekNo 周 · $weekday',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                    color: theme.colorScheme.onSecondaryContainer,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  semesterName,
                  style: TextStyle(
                    fontSize: 12,
                    color:
                        theme.colorScheme.onSecondaryContainer.withOpacity(0.8),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    ); */
  }

  /// 今日概览统计
  Widget _buildTodayOverview(ThemeData theme) {
    final stats = _studentHomeData?['stats'] as Map<String, dynamic>? ?? {};
    final todayCourses =
        (_studentHomeData?['today_courses'] as List?)?.length ?? 0;
    final todayTasks = (_studentHomeData?['today_tasks'] as List?)?.length ?? 0;
    final unread = stats['unread_notifications'] ?? 0;
    final plansInProgress = stats['plans_in_progress'] ?? 0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('今日概览', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(
          children: [
            HomeOverviewItem(
              icon: Icons.menu_book_outlined,
              label: '课程',
              count: todayCourses,
              unit: '节',
              color: const Color(0xFF1565C0),
            ),
            const SizedBox(width: 8),
            HomeOverviewItem(
              icon: Icons.task_alt_outlined,
              label: '任务',
              count: todayTasks,
              unit: '个',
              color: const Color(0xFF2E7D32),
            ),
            const SizedBox(width: 8),
            HomeOverviewItem(
              icon: Icons.notifications_outlined,
              label: '通知',
              count: unread,
              unit: '条',
              color: const Color(0xFFE65100),
            ),
            const SizedBox(width: 8),
            HomeOverviewItem(
              icon: Icons.fact_check_outlined,
              label: '计划',
              count: plansInProgress,
              unit: '个',
              color: const Color(0xFF7B1FA2),
            ),
          ],
        ),
      ],
    );
  }

  /// 今日课表
  Widget _buildTodayCourses(ThemeData theme) {
    final courses = (_studentHomeData?['today_courses'] as List?)
            ?.cast<Map<String, dynamic>>() ??
        [];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('今日课表', style: theme.textTheme.titleMedium),
            TextButton.icon(
              onPressed: () => context.go('/student/study-plan'),
              icon: const Icon(Icons.arrow_forward, size: 16),
              label: const Text('查看全部', style: TextStyle(fontSize: 13)),
            ),
          ],
        ),
        const SizedBox(height: 8),
        if (courses.isEmpty)
          HomeEmptyCard(message: '今日没有课程', icon: Icons.coffee_outlined)
        else
          Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: theme.colorScheme.outlineVariant.withOpacity(0.5),
              ),
            ),
            child: Column(
              children: [
                for (int i = 0; i < courses.length; i++) ...[
                  _buildCourseItem(theme, courses[i]),
                  if (i < courses.length - 1)
                    Divider(
                      height: 1,
                      indent: 16,
                      endIndent: 16,
                      color: theme.colorScheme.outlineVariant.withOpacity(0.3),
                    ),
                ],
              ],
            ),
          ),
      ],
    );
  }

  /// 根据课程时间("HH:MM-HH:MM")判断当前状态，返回状态徽章(进行中/即将开始/已结束)
  Widget? _courseStatus(String time) {
    final parts = time.split('-');
    if (parts.length != 2) return null;
    final s = parts[0].trim().split(':'), e = parts[1].trim().split(':');
    if (s.length != 2 || e.length != 2) return null;
    final now = DateTime.now();
    final start = DateTime(now.year, now.month, now.day,
        int.tryParse(s[0]) ?? 0, int.tryParse(s[1]) ?? 0);
    final end = DateTime(now.year, now.month, now.day, int.tryParse(e[0]) ?? 0,
        int.tryParse(e[1]) ?? 0);
    String label;
    Color color;
    if (now.isBefore(start)) {
      final mins = start.difference(now).inMinutes;
      if (mins > 120) return null; // 距开始超过2小时不提示
      label = mins <= 0 ? '即将开始' : '$mins 分钟后开始';
      color = Colors.orange;
    } else if (now.isAfter(end)) {
      return null; // 已结束不提示
    } else {
      label = '进行中';
      color = Colors.green;
    }
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
        decoration: BoxDecoration(
          color: color.withOpacity(0.12),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Text(label,
            style: TextStyle(
                fontSize: 11, color: color, fontWeight: FontWeight.w700)),
      ),
    );
  }

  Widget _buildCourseItem(ThemeData theme, Map<String, dynamic> course) {
    final color = Color(int.parse(
        (course['color'] as String? ?? '#1565C0').replaceAll('#', '0xFF')));
    final courseName = course['course_name'] ?? '未知课程';
    final time = course['time'] ?? '';
    final location = course['location'] ?? '';
    final teacher = course['teacher'] ?? '';
    final status = _courseStatus(time);

    return Padding(
      padding: const EdgeInsets.all(14),
      child: Row(
        children: [
          Container(
            width: 4,
            height: 48,
            decoration: BoxDecoration(
              color: color,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  courseName,
                  style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                if (status != null) ...[const SizedBox(height: 4), status],
                const SizedBox(height: 4),
                Row(
                  children: [
                    Icon(Icons.access_time,
                        size: 14, color: theme.colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text(
                      time,
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Icon(Icons.location_on_outlined,
                        size: 14, color: theme.colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(
                        location,
                        style: TextStyle(
                          fontSize: 12,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
                if (teacher.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    '教师：$teacher',
                    style: TextStyle(
                      fontSize: 11,
                      color: theme.colorScheme.outline,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }

  /// 今日任务
  Widget _buildTodayTasks(ThemeData theme) {
    final tasks = (_studentHomeData?['today_tasks'] as List?)
            ?.cast<Map<String, dynamic>>() ??
        [];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('今日任务', style: theme.textTheme.titleMedium),
            TextButton.icon(
              onPressed: () => context.go('/student/study-plan'),
              icon: const Icon(Icons.arrow_forward, size: 16),
              label: const Text('查看全部', style: TextStyle(fontSize: 13)),
            ),
          ],
        ),
        const SizedBox(height: 8),
        if (tasks.isEmpty)
          HomeEmptyCard(message: '今日没有任务安排', icon: Icons.check_circle_outline)
        else
          Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: theme.colorScheme.outlineVariant.withOpacity(0.5),
              ),
            ),
            child: Column(
              children: [
                for (int i = 0; i < tasks.length; i++) ...[
                  _buildTaskItem(theme, tasks[i]),
                  if (i < tasks.length - 1)
                    Divider(
                      height: 1,
                      indent: 16,
                      endIndent: 16,
                      color: theme.colorScheme.outlineVariant.withOpacity(0.3),
                    ),
                ],
              ],
            ),
          ),
      ],
    );
  }

  Widget _buildTaskItem(ThemeData theme, Map<String, dynamic> task) {
    final title = task['title'] ?? '未命名任务';
    final status = task['status'] ?? 'pending';
    final duration = task['duration'] ?? 0;
    final isCompleted = status == 'completed';

    return InkWell(
      onTap: () => _toggleTaskStatus(task),
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Container(
              width: 24,
              height: 24,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: isCompleted
                    ? theme.colorScheme.primary
                    : Colors.transparent,
                border: Border.all(
                  color: isCompleted
                      ? theme.colorScheme.primary
                      : theme.colorScheme.outline,
                  width: 2,
                ),
              ),
              child: isCompleted
                  ? const Icon(Icons.check, size: 16, color: Colors.white)
                  : null,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      decoration:
                          isCompleted ? TextDecoration.lineThrough : null,
                      color: isCompleted
                          ? theme.colorScheme.onSurfaceVariant
                          : null,
                    ),
                  ),
                  if (duration > 0) ...[
                    const SizedBox(height: 2),
                    Text(
                      '预计 $duration 分钟',
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 功能入口
  Widget _buildQuickEntries(ThemeData theme) {
    final quickEntries = (_studentHomeData?['quick_entries'] as List?)
            ?.cast<Map<String, dynamic>>() ??
        [];

    // 使用默认入口（如果后端没有返回）
    final entries = quickEntries.isEmpty
        ? [
            {'icon': 'chat', 'title': 'AI问答', 'route': '/chat'},
            {
              'icon': 'study_plan',
              'title': '学习计划',
              'route': '/student/study-plan'
            },
            {
              'icon': 'timetable',
              'title': '我的课表',
              'route': '/student/study-plan'
            },
            {'icon': 'career', 'title': '就业服务', 'route': '/student/career'},
            {'icon': 'study', 'title': '学业服务', 'route': '/student/study'},
            {'icon': 'mental', 'title': '心理健康', 'route': '/student/mental'},
            // 按年级：大一→开学待办，大二以上→本阶段成长
            (context.read<ThemeNotifier>().grade <= 1
                ? {
                    'icon': 'agenda',
                    'title': '开学待办',
                    'route': '/student/freshman-agenda'
                  }
                : {
                    'icon': 'agenda',
                    'title': '本阶段成长',
                    'route': '/student/grade-growth'
                  }),
          ]
        : quickEntries;

    final iconMap = <String, IconData>{
      'chat': Icons.chat_bubble_outline,
      'study_plan': Icons.fact_check_outlined,
      'timetable': Icons.calendar_month_outlined,
      'career': Icons.work_outline,
      'study': Icons.menu_book_outlined,
      'mental': Icons.favorite_outline,
      'agenda': Icons.checklist,
    };

    final colorMap = <String, Color>{
      'chat': const Color(0xFF1565C0),
      'study_plan': const Color(0xFF2E7D32),
      'timetable': const Color(0xFF00695C),
      'career': const Color(0xFFE65100),
      'study': const Color(0xFF7B1FA2),
      'mental': const Color(0xFFC62828),
      'agenda': const Color(0xFF00695C),
    };

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('功能入口', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        GridView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 3,
            mainAxisSpacing: 10,
            crossAxisSpacing: 10,
            childAspectRatio: 1.1,
          ),
          itemCount: entries.length,
          itemBuilder: (context, index) {
            final entry = entries[index];
            final iconKey = entry['icon'] as String? ?? 'chat';
            final icon = iconMap[iconKey] ?? Icons.widgets_outlined;
            final color = colorMap[iconKey] ?? theme.colorScheme.primary;
            final title = entry['title'] ?? '';
            final route = entry['route'] ?? '/';

            return _buildQuickEntryCard(
              theme,
              icon: icon,
              label: title,
              color: color,
              onTap: () => context.go(route),
            );
          },
        ),
      ],
    );
  }

  Widget _buildQuickEntryCard(
    ThemeData theme, {
    required IconData icon,
    required String label,
    required Color color,
    required VoidCallback onTap,
  }) {
    return Material(
      color: color.withOpacity(0.08),
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: color.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(icon, color: color, size: 24),
              ),
              const SizedBox(height: 6),
              Text(
                label,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: color,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// 近期提醒
  Widget _buildUpcomingEvents(ThemeData theme) {
    final events = (_studentHomeData?['upcoming_events'] as List?)
            ?.cast<Map<String, dynamic>>() ??
        [];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('近期提醒', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        if (events.isEmpty)
          HomeEmptyCard(
              message: '近期没有重要事件', icon: Icons.event_available_outlined)
        else
          Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: theme.colorScheme.outlineVariant.withOpacity(0.5),
              ),
            ),
            child: Column(
              children: [
                for (int i = 0; i < events.length; i++) ...[
                  _buildEventItem(theme, events[i]),
                  if (i < events.length - 1)
                    Divider(
                      height: 1,
                      indent: 16,
                      endIndent: 16,
                      color: theme.colorScheme.outlineVariant.withOpacity(0.3),
                    ),
                ],
              ],
            ),
          ),
      ],
    );
  }

  Widget _buildEventItem(ThemeData theme, Map<String, dynamic> event) {
    final eventName = event['event_name'] ?? '';
    final eventType = event['event_type'] ?? '';
    final startDate = event['start_date'] ?? '';
    final daysLeft = event['days_left'] ?? 0;

    final iconMap = <String, IconData>{
      'holiday': Icons.celebration_outlined,
      'exam': Icons.edit_note_outlined,
      'registration': Icons.how_to_reg_outlined,
      'vacation': Icons.beach_access_outlined,
    };

    final colorMap = <String, Color>{
      'holiday': const Color(0xFFE65100),
      'exam': const Color(0xFFC62828),
      'registration': const Color(0xFF2E7D32),
      'vacation': const Color(0xFF1565C0),
    };

    final icon = iconMap[eventType] ?? Icons.event_outlined;
    final color = colorMap[eventType] ?? theme.colorScheme.primary;

    String daysText;
    if (daysLeft < 0) {
      daysText = '已过${-daysLeft}天';
    } else if (daysLeft == 0) {
      daysText = '今天';
    } else {
      daysText = '还有$daysLeft天';
    }

    return Padding(
      padding: const EdgeInsets.all(14),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: color.withOpacity(0.12),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(icon, color: color, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  eventName,
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  startDate,
                  style: TextStyle(
                    fontSize: 12,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: color.withOpacity(0.1),
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              daysText,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: color,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
