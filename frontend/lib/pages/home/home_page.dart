import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../config/release_config.dart';
import '../../providers/chat_provider.dart';
import '../../providers/emotion_provider.dart';
import '../../providers/session_provider.dart';
import '../../utils/role_utils.dart';
import '../../utils/storage.dart';
import '../../utils/date_utils.dart';
import '../../config/api_config.dart';
import '../../services/api_service.dart';
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
  @override
  void initState() {
    super.initState();
    // 延迟加载数据，避免 build 中途触发 notifyListeners
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadData();
      _checkConsent();
    });
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
  }

  bool _canAccessAlerts(String? role) => RoleUtils.canAccessEmotion(role);

  @override
  Widget build(BuildContext context) {
    final role = Storage.role;
    final loggedIn = Storage.isLoggedIn;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('蔚小芯'),
        actions: [
          IconButton(
            icon: const Icon(Icons.notifications_outlined),
            tooltip: '消息',
            onPressed: () {},
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
            // 日期时间 + 校历入口
            const DateTimeBanner(),
            const SizedBox(height: 20),
            // 辅导员及以上：告警概览
            if (_canAccessAlerts(role)) ...[
              _buildAlertOverview(theme),
              const SizedBox(height: 20),
            ],
            // 所有角色：知识入口
            _buildKnowledgeEntry(theme),
            const SizedBox(height: 20),
            // 所有角色：校园服务
            _buildCampusService(theme),
            const SizedBox(height: 20),
            // 学生专区（student/student_union）
            if (role == 'student' || role == 'student_union') ...[
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
              onTap: () => context.go('/campus'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.view_in_ar,
              label: 'VR全景',
              color: const Color(0xFF7B1FA2),
              onTap: () => context.go('/campus'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.school,
              label: '学校首页',
              color: const Color(0xFF1565C0),
              onTap: () => context.go('/campus'),
            ),
            const SizedBox(width: 10),
            _buildKnowledgeCard(
              theme,
              icon: Icons.music_note,
              label: '招生抖音',
              color: const Color(0xFFC62828),
              onTap: () => context.go('/campus'),
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
}
