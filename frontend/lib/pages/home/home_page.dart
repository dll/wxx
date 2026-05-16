import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
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
    final role = Storage.role;
    // 辅导员及以上角色加载告警统计
    if (_canAccessAlerts(role)) {
      context.read<EmotionProvider>().fetchStats();
    }
    // 加载最近会话
    context.read<SessionProvider>().fetchSessions();
  }

  bool _canAccessAlerts(String? role) => RoleUtils.canAccessEmotion(role);

  @override
  Widget build(BuildContext context) {
    final role = Storage.role;
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
            _buildWelcomeBanner(theme),
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
            // 最近对话
            _buildRecentSessions(theme),
          ],
        ),
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
            theme.colorScheme.primary.withValues(alpha: 0.7),
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
              color: theme.colorScheme.onPrimary.withValues(alpha: 0.85),
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
          color: color.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: color.withValues(alpha: 0.2)),
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
        color: color.withValues(alpha: 0.06),
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
              color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
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
                title: Text(s.title, maxLines: 1, overflow: TextOverflow.ellipsis),
                subtitle: Text(TimeFormatter.relative(s.updatedAt),
                    style: TextStyle(fontSize: 12, color: theme.colorScheme.outline)),
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  // 加载该会话的历史消息并跳转到对话页
                  context.read<ChatProvider>().loadSession(s.id);
                  context.go('/chat');
                },
                dense: true,
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              )),
      ],
    );
  }
}
