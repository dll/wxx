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
import '../../utils/role_utils.dart';
import '../../utils/storage.dart';
import '../../utils/date_utils.dart';
import '../../config/api_config.dart';
import '../../services/api_service.dart';
import '../../widgets/avatar_card.dart';
import '../../widgets/consent_dialog.dart';
import '../../widgets/datetime_banner.dart';
import '../../widgets/error_view.dart';
import '../../widgets/skeleton.dart';

// ── 学生专区卡片配置 ──
class _FeatureCard {
  final IconData icon;
  final String label;
  final Color color;
  final String route;
  const _FeatureCard(this.icon, this.label, this.color, this.route);
}

const _studentFeatures = [
  _FeatureCard(Icons.school_outlined, '新生指南', Color(0xFF00695C), '/student/freshmen-guide'),
  _FeatureCard(Icons.topic_outlined, '毕设选题', Color(0xFF1565C0), '/graduation'),
  _FeatureCard(
      Icons.emoji_events_outlined, '学科竞赛', Color(0xFFE65100), '/competition'),
  _FeatureCard(Icons.calendar_today, '大学规划', Color(0xFF2E7D32), '/plan'),
  _FeatureCard(
      Icons.flag_outlined, '入党教育', Color(0xFFC62828), '/party-education'),
  _FeatureCard(Icons.groups_outlined, '社团生活', Color(0xFF7B1FA2), '/club'),
];

const _educationFeatures = [
  _FeatureCard(Icons.work_outline, '就业服务', Color(0xFFE65100), '/student/career'),
  _FeatureCard(Icons.menu_book_outlined, '学业服务', Color(0xFF1565C0), '/student/study'),
  _FeatureCard(Icons.checklist_rtl, '学习计划', Color(0xFF00695C), '/student/study-plan'),
  _FeatureCard(Icons.favorite_outline, '心理健康', Color(0xFFC62828), '/student/mental'),
];

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

  @override
  void initState() {
    super.initState();
    // 延迟加载数据，避免 build 中途触发 notifyListeners
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadData();
      _checkConsent();
      _checkAppUpdate();
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
        // 异步通知后端记录同意状态
        ApiService().post(ApiConfig.consent, data: {});
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
    // 学生角色加载个性化首页数据 + 数字人形象
    if (role == 'student' || role == 'student_union') {
      _loadStudentHome();
      context
          .read<StudentFeatureProvider>()
          .fetchAvatar(displayName: Storage.displayName ?? '同学');
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

  bool _canAccessAlerts(String? role) => RoleUtils.canAccessEmotion(role);

  @override
  Widget build(BuildContext context) {
    final role = Storage.role;
    final loggedIn = Storage.isLoggedIn;
    final theme = Theme.of(context);
    final isStudent = role == 'student' || role == 'student_union';
    final notificationProvider = context.watch<NotificationProvider>();
    final unreadCount = notificationProvider.unreadCount;

    return Scaffold(
      appBar: AppBar(
        title: const Text('蔚小芯'),
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
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: Colors.red,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      constraints: const BoxConstraints(minWidth: 16, minHeight: 16),
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
            // 安卓 APK 下载（紧凑卡片，无需滚动即可看到）
            GestureDetector(
              onTap: () => _showApkDownloadDialog(context, theme),
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
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
                    const Icon(Icons.android, color: Color(0xFF2E7D32), size: 20),
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
            // 学生个性化首页
            if (isStudent && loggedIn) ...[
              // 数字人形象卡片（可被系统设置隐藏）
              if (Storage.showAvatar) ...[
                _buildAvatarBanner(theme),
                const SizedBox(height: 20),
              ],
              _buildStudentHomeContent(theme),
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

  /// 安卓应用下载二维码
  Widget _buildApkDownloadCard(ThemeData theme) {
    final qrData = Uri.encodeComponent(ReleaseConfig.apkDownloadUrl);
    final qrUrl =
        'https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=$qrData&margin=10';
    final muted = theme.colorScheme.onSurfaceVariant;

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
            color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: Colors.black
                .withOpacity(theme.brightness == Brightness.dark ? 0.25 : 0.06),
            blurRadius: 24,
            offset: const Offset(0, 10),
          ),
        ],
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final compact = constraints.maxWidth < 620;
          final qr = _buildApkQr(theme, qrUrl);
          final info = _buildApkDownloadInfo(theme, muted);

          if (compact) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                info,
                const SizedBox(height: 16),
                Center(child: qr),
              ],
            );
          }

          return Row(
            children: [
              Expanded(child: info),
              const SizedBox(width: 18),
              qr,
            ],
          );
        },
      ),
    );
  }

  Widget _buildApkDownloadInfo(ThemeData theme, Color muted) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: const Color(0xFF2E7D32).withOpacity(0.12),
                borderRadius: BorderRadius.circular(14),
              ),
              child:
                  const Icon(Icons.android, color: Color(0xFF2E7D32), size: 28),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('安卓应用下载',
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 2),
                  Text('手机扫码下载 APK，主流 Android 手机可直接安装',
                      style: theme.textTheme.bodySmall?.copyWith(color: muted)),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 14),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            _buildReleaseChip(theme, '版本 v${ReleaseConfig.version}'),
            _buildReleaseChip(theme, '构建号 ${ReleaseConfig.buildNumber}'),
            _buildReleaseChip(theme, '发布 ${ReleaseConfig.releaseDate}'),
          ],
        ),
        const SizedBox(height: 12),
        Text(ReleaseConfig.apkFileName,
            style: theme.textTheme.bodyMedium
                ?.copyWith(fontWeight: FontWeight.w600)),
        const SizedBox(height: 14),
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            FilledButton.icon(
              onPressed: _openApkDownload,
              icon: const Icon(Icons.download_rounded, size: 18),
              label: const Text('下载 APK'),
            ),
            if (!Storage.isLoggedIn)
              OutlinedButton.icon(
                onPressed: () => context.go('/login'),
                icon: const Icon(Icons.login, size: 18),
                label: const Text('登录 Web 版'),
              ),
          ],
        ),
      ],
    );
  }

  Widget _buildApkQr(ThemeData theme, String qrUrl) {
    return Container(
      width: 156,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
            color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(10),
            child: Image.network(
              qrUrl,
              width: 132,
              height: 132,
              fit: BoxFit.cover,
              errorBuilder: (_, __, ___) => const SizedBox(
                width: 132,
                height: 132,
                child: Icon(Icons.qr_code_2, size: 96, color: Colors.black54),
              ),
            ),
          ),
          const SizedBox(height: 8),
          const Text('扫码安装',
              style: TextStyle(
                  color: Colors.black87, fontWeight: FontWeight.w700)),
        ],
      ),
    );
  }

  Widget _buildReleaseChip(ThemeData theme, String label) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: theme.colorScheme.primary.withOpacity(0.08),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label,
          style: theme.textTheme.labelMedium
              ?.copyWith(color: theme.colorScheme.primary)),
    );
  }

  Future<void> _openApkDownload() async {
    final uri = Uri.parse(ReleaseConfig.apkDownloadUrl);
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  void _showApkDownloadDialog(BuildContext context, ThemeData theme) {
    final qrData = Uri.encodeComponent(ReleaseConfig.apkDownloadUrl);
    final qrUrl =
        'https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=$qrData&margin=10';
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
                        color: theme.colorScheme.outlineVariant.withOpacity(0.3),
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
    final hour = DateTime.now().hour;
    final greeting = hour < 6
        ? '夜深了'
        : hour < 12
            ? '上午好'
            : hour < 14
                ? '中午好'
                : hour < 18
                    ? '下午好'
                    : '晚上好';

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            theme.colorScheme.primary,
            theme.colorScheme.primary.withOpacity(0.7),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '$greeting，$displayName',
            style: TextStyle(
              color: theme.colorScheme.onPrimary,
              fontSize: 20,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            '我是蔚小芯，你的学工智能助手',
            style: TextStyle(
              color: theme.colorScheme.onPrimary.withOpacity(0.85),
              fontSize: 14,
            ),
          ),
          const SizedBox(height: 16),
          // 快捷提问
          SizedBox(
            width: double.infinity,
            child: ElevatedButton.icon(
              onPressed: () => context.go('/chat'),
              icon: const Icon(Icons.chat, size: 18),
              label: const Text('开始对话'),
              style: ElevatedButton.styleFrom(
                backgroundColor: theme.colorScheme.onPrimary,
                foregroundColor: theme.colorScheme.primary,
                padding: const EdgeInsets.symmetric(vertical: 10),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(24),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// 告警统计概览（辅导员及以上可见）
  Widget _buildAlertOverview(ThemeData theme) {
    final stats = context.watch<EmotionProvider>();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('预警概览', style: theme.textTheme.titleMedium),
            TextButton.icon(
              onPressed: () => context.go('/emotion'),
              icon: const Icon(Icons.arrow_forward, size: 16),
              label: const Text('查看全部', style: TextStyle(fontSize: 13)),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (stats.statsLoading)
          const Center(child: CircularProgressIndicator())
        else
          Row(
            children: [
              _buildStatCard(
                theme,
                label: '紧急',
                count: stats.stats?.urgent ?? 0,
                color: const Color(0xFFC62828),
                icon: Icons.warning_rounded,
              ),
              const SizedBox(width: 10),
              _buildStatCard(
                theme,
                label: '高风险',
                count: stats.stats?.high ?? 0,
                color: const Color(0xFFE65100),
                icon: Icons.error_outline,
              ),
              const SizedBox(width: 10),
              _buildStatCard(
                theme,
                label: '待处理',
                count: stats.stats?.pending ?? 0,
                color: const Color(0xFF1565C0),
                icon: Icons.pending_actions,
              ),
            ],
          ),
      ],
    );
  }

  Widget _buildStatCard(
    ThemeData theme, {
    required String label,
    required int count,
    required Color color,
    required IconData icon,
  }) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 12),
        decoration: BoxDecoration(
          color: color.withOpacity(0.08),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: color.withOpacity(0.2)),
        ),
        child: Column(
          children: [
            Icon(icon, color: color, size: 24),
            const SizedBox(height: 6),
            Text(
              '$count',
              style: TextStyle(
                fontSize: 28,
                fontWeight: FontWeight.bold,
                color: color,
              ),
            ),
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 知识入口 — 四类快捷卡片
  Widget _buildKnowledgeEntry(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('知识大厅', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
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
        color: color.withOpacity(0.06),
        borderRadius: BorderRadius.circular(12),
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(12),
          child: Container(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              children: [
                Icon(icon, color: color, size: 28),
                const SizedBox(height: 6),
                Text(
                  label,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: color,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// 校园服务 — 地图/全景/招生抖音
  Widget _buildCampusService(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('校园服务', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
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
              icon: Icons.school,
              label: '学校首页',
              color: const Color(0xFF1565C0),
              onTap: () => context.go('/campus?v=home'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.music_note,
              label: '招生抖音',
              color: const Color(0xFFC62828),
              onTap: () => context.go('/campus?v=douyin'),
            ),
          ],
        ),
      ],
    );
  }

  /// 学生专区 — 毕设选题/学科竞赛/大学规划/入党教育/社团生活
  Widget _buildStudentFeatures(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('学生专区', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(
          children: [
            for (final f in _studentFeatures.take(3))
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
            for (final f in _studentFeatures.skip(3))
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

  /// 教育三大模块 — 就业/学业/学习计划/心理
  Widget _buildEducationFeatures(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('教育服务', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
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
      return const _StudentHomeSkeleton();
    }
    if (_studentHomeError != null) {
      return _buildErrorCard(theme);
    }
    if (_studentHomeData == null) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
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

  /// 错误提示卡片
  Widget _buildErrorCard(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer.withOpacity(0.3),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          Icon(Icons.error_outline, color: theme.colorScheme.error, size: 32),
          const SizedBox(height: 8),
          Text(
            '加载失败',
            style: TextStyle(
              color: theme.colorScheme.error,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            _studentHomeError ?? '未知错误',
            style: TextStyle(
              color: theme.colorScheme.onSurfaceVariant,
              fontSize: 12,
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 12),
          TextButton.icon(
            onPressed: _loadStudentHome,
            icon: const Icon(Icons.refresh, size: 18),
            label: const Text('重新加载'),
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

    return Container(
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
                    color: theme.colorScheme.onSecondaryContainer.withOpacity(0.8),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
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
            _buildOverviewItem(
              theme,
              icon: Icons.menu_book_outlined,
              label: '课程',
              count: todayCourses,
              unit: '节',
              color: const Color(0xFF1565C0),
            ),
            const SizedBox(width: 8),
            _buildOverviewItem(
              theme,
              icon: Icons.task_alt_outlined,
              label: '任务',
              count: todayTasks,
              unit: '个',
              color: const Color(0xFF2E7D32),
            ),
            const SizedBox(width: 8),
            _buildOverviewItem(
              theme,
              icon: Icons.notifications_outlined,
              label: '通知',
              count: unread,
              unit: '条',
              color: const Color(0xFFE65100),
            ),
            const SizedBox(width: 8),
            _buildOverviewItem(
              theme,
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

  Widget _buildOverviewItem(
    ThemeData theme, {
    required IconData icon,
    required String label,
    required int count,
    required String unit,
    required Color color,
  }) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 8),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: theme.colorScheme.outlineVariant.withOpacity(0.5),
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.04),
              blurRadius: 8,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          children: [
            Icon(icon, color: color, size: 22),
            const SizedBox(height: 6),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              children: [
                Text(
                  '$count',
                  style: TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.bold,
                    color: color,
                  ),
                ),
                const SizedBox(width: 2),
                Text(
                  unit,
                  style: TextStyle(
                    fontSize: 11,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 2),
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 今日课表
  Widget _buildTodayCourses(ThemeData theme) {
    final courses =
        (_studentHomeData?['today_courses'] as List?)?.cast<Map<String, dynamic>>() ?? [];

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
          _buildEmptyCard(theme, '今日没有课程', Icons.coffee_outlined)
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

  Widget _buildCourseItem(ThemeData theme, Map<String, dynamic> course) {
    final color = Color(int.parse(
            (course['color'] as String? ?? '#1565C0')
                .replaceAll('#', '0xFF')));
    final courseName = course['course_name'] ?? '未知课程';
    final time = course['time'] ?? '';
    final location = course['location'] ?? '';
    final teacher = course['teacher'] ?? '';

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
    final tasks =
        (_studentHomeData?['today_tasks'] as List?)?.cast<Map<String, dynamic>>() ?? [];

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
          _buildEmptyCard(theme, '今日没有任务安排', Icons.check_circle_outline)
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
    final quickEntries =
        (_studentHomeData?['quick_entries'] as List?)?.cast<Map<String, dynamic>>() ?? [];

    // 使用默认入口（如果后端没有返回）
    final entries = quickEntries.isEmpty
        ? [
            {'icon': 'chat', 'title': 'AI问答', 'route': '/chat'},
            {'icon': 'study_plan', 'title': '学习计划', 'route': '/student/study-plan'},
            {'icon': 'timetable', 'title': '我的课表', 'route': '/student/study-plan'},
            {'icon': 'career', 'title': '就业服务', 'route': '/student/career'},
            {'icon': 'study', 'title': '学业服务', 'route': '/student/study'},
            {'icon': 'mental', 'title': '心理健康', 'route': '/student/mental'},
          ]
        : quickEntries;

    final iconMap = <String, IconData>{
      'chat': Icons.chat_bubble_outline,
      'study_plan': Icons.fact_check_outlined,
      'timetable': Icons.calendar_month_outlined,
      'career': Icons.work_outline,
      'study': Icons.menu_book_outlined,
      'mental': Icons.favorite_outline,
    };

    final colorMap = <String, Color>{
      'chat': const Color(0xFF1565C0),
      'study_plan': const Color(0xFF2E7D32),
      'timetable': const Color(0xFF00695C),
      'career': const Color(0xFFE65100),
      'study': const Color(0xFF7B1FA2),
      'mental': const Color(0xFFC62828),
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
    final events =
        (_studentHomeData?['upcoming_events'] as List?)?.cast<Map<String, dynamic>>() ?? [];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('近期提醒', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        if (events.isEmpty)
          _buildEmptyCard(theme, '近期没有重要事件', Icons.event_available_outlined)
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

  /// 空状态卡片
  Widget _buildEmptyCard(ThemeData theme, String message, IconData icon) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.3),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          Icon(icon, size: 32, color: theme.colorScheme.onSurfaceVariant),
          const SizedBox(height: 8),
          Text(
            message,
            style: TextStyle(
              color: theme.colorScheme.onSurfaceVariant,
              fontSize: 14,
            ),
          ),
        ],
      ),
    );
  }
}

/// 学生首页骨架屏
class _StudentHomeSkeleton extends StatelessWidget {
  const _StudentHomeSkeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          height: 72,
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
            borderRadius: BorderRadius.circular(16),
          ),
        ),
        const SizedBox(height: 16),
        Text('今日概览', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(
          children: [
            for (var i = 0; i < 4; i++) ...[
              Expanded(
                child: Container(
                  height: 88,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
              ),
              if (i < 3) const SizedBox(width: 8),
            ],
          ],
        ),
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('今日课表', style: theme.textTheme.titleMedium),
            Container(
              width: 80,
              height: 16,
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
                borderRadius: BorderRadius.circular(8),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Container(
          height: 80,
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
            borderRadius: BorderRadius.circular(12),
          ),
        ),
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('今日任务', style: theme.textTheme.titleMedium),
            Container(
              width: 80,
              height: 16,
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
                borderRadius: BorderRadius.circular(8),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Container(
          height: 60,
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ],
    );
  }
}
