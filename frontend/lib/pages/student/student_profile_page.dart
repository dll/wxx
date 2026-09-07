import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../providers/student_feature_provider.dart';
import '../../utils/storage.dart';
import '../../widgets/avatar_card.dart';
import '../../widgets/error_view.dart';

/// 学生个人信息档案 — 聚合展示基本信息/数字画像/性格/学业/竞赛/活动等
class StudentProfilePage extends StatefulWidget {
  const StudentProfilePage({super.key});

  @override
  State<StudentProfilePage> createState() => _StudentProfilePageState();
}

class _StudentProfilePageState extends State<StudentProfilePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudentFeatureProvider>();
      p.fetchPersonalProfile();
      p.fetchDigitalTwin();
      p.fetchAvatar(
          displayName: Storage.displayName ?? '同学',
          role: Storage.role ?? 'student');
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('个人信息档案')),
      body: RefreshIndicator(
        onRefresh: () async {
          final p = context.read<StudentFeatureProvider>();
          await p.fetchPersonalProfile();
          await p.fetchDigitalTwin();
          await p.fetchAvatar(
              displayName: Storage.displayName ?? '同学',
              role: Storage.role ?? 'student');
        },
        child: provider.profileLoading && provider.personalProfile == null
            ? const Center(child: CircularProgressIndicator())
            : provider.personalProfile == null
                ? ErrorView.error(
                    message: '暂无档案数据',
                    onRetry: () => provider.fetchPersonalProfile())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final data = provider.personalProfile ?? {};
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // 1. 数字人形象卡片（数据驱动）
        if (Storage.showAvatar && provider.avatar != null) ...[
          AvatarCard(config: provider.avatar!, height: 300),
          const SizedBox(height: 16),
        ],

        // 2. 基本信息
        _buildBasicInfo(theme, data),

        const SizedBox(height: 16),

        // 3. 成长画像入口：完整五维分析集中在数字孪生页，避免重复堆叠。
        _buildTwinLink(theme, provider),

        const SizedBox(height: 16),

        // 4. 学业记录
        _buildAcademicSection(theme, data),

        const SizedBox(height: 16),

        // 5. 竞赛 / 入党 / 社团
        _buildActivitySection(theme, data),

        const SizedBox(height: 16),

        // 6. 打卡 / 积分
        _buildStatsSection(theme, data),
      ],
    );
  }

  Widget _buildTwinLink(ThemeData theme, StudentFeatureProvider provider) {
    final twin = provider.twin;
    final available =
        twin?.dimensions.where((d) => d.dataAvailable).length ?? 0;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: ListTile(
        contentPadding: const EdgeInsets.fromLTRB(16, 10, 10, 10),
        leading: Icon(Icons.insights_rounded, color: theme.colorScheme.primary),
        title:
            const Text('成长画像', style: TextStyle(fontWeight: FontWeight.w700)),
        subtitle: Text(
          twin == null
              ? '查看我的五维成长、证据与建议'
              : '$available/${twin.dimensions.length} 个维度已有真实记录',
        ),
        trailing: FilledButton.tonalIcon(
          onPressed: () => context.push('/student/digital-twin'),
          icon: const Icon(Icons.arrow_forward_rounded, size: 17),
          label: const Text('查看'),
        ),
      ),
    );
  }

  // ── 基本信息 ──
  Widget _buildBasicInfo(ThemeData theme, Map<String, dynamic> data) {
    final basic = data['basic_info'] as Map<String, dynamic>? ?? {};
    final displayName = Storage.displayName ?? '同学';
    final username = data['username'] ?? '';
    final college = basic['college'] ?? '';
    final major = basic['major'] ?? '';
    final className = basic['class_name'] ?? '';
    final enrollmentYear = basic['enrollment_year'] ?? '';

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              // 头像（首字母；后续支持照片）
              CircleAvatar(
                radius: 28,
                backgroundColor: theme.colorScheme.primaryContainer,
                child: Text(
                  displayName.characters.first,
                  style: TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.bold,
                    color: theme.colorScheme.onPrimaryContainer,
                  ),
                ),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(displayName,
                        style: theme.textTheme.titleLarge
                            ?.copyWith(fontWeight: FontWeight.bold)),
                    const SizedBox(height: 2),
                    Text('学号 $username',
                        style: TextStyle(
                            fontSize: 13,
                            color: theme.colorScheme.onSurfaceVariant)),
                  ],
                ),
              ),
            ]),
            const SizedBox(height: 14),
            _infoRow(theme, Icons.account_balance_outlined, '学院', college),
            _infoRow(theme, Icons.menu_book_outlined, '专业', major),
            _infoRow(theme, Icons.groups_outlined, '班级', className),
            _infoRow(theme, Icons.event_outlined, '入学年份', enrollmentYear),
          ],
        ),
      ),
    );
  }

  Widget _infoRow(ThemeData theme, IconData icon, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Icon(icon, size: 18, color: theme.colorScheme.primary),
          const SizedBox(width: 10),
          SizedBox(
              width: 72,
              child: Text(label,
                  style: TextStyle(
                      fontSize: 13,
                      color: theme.colorScheme.onSurfaceVariant))),
          Expanded(
            child: Text(value.isNotEmpty ? value : '—',
                style: const TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }

  // ── 学业记录 ──
  Widget _buildAcademicSection(ThemeData theme, Map<String, dynamic> data) {
    final academic = data['academic'] as Map<String, dynamic>? ?? {};
    final courseCount = academic['course_count'] ?? 0;
    final credits = academic['total_credits'] ?? 0;
    final avgScore = academic['avg_score'] ?? 0;
    final avgGpa = academic['avg_gpa'] ?? 0;
    final passRate = academic['pass_rate'] ?? 0;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(Icons.school_outlined,
                  color: theme.colorScheme.primary, size: 18),
              const SizedBox(width: 8),
              Text('学业记录', style: theme.textTheme.titleMedium),
            ]),
            const SizedBox(height: 14),
            Row(
              children: [
                _statItem(theme, '课程数', '$courseCount', Icons.menu_book),
                _statItem(theme, '总学分', '$credits', Icons.star_outline),
                _statItem(theme, '平均成绩', (avgScore as num).toStringAsFixed(1),
                    Icons.analytics_outlined),
                _statItem(theme, '平均GPA', (avgGpa as num).toStringAsFixed(2),
                    Icons.trending_up),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              '通过率 ${(passRate as num).toStringAsFixed(1)}%',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.primary),
            ),
          ],
        ),
      ),
    );
  }

  Widget _statItem(ThemeData theme, String label, String value, IconData icon) {
    return Expanded(
      child: Column(
        children: [
          Icon(icon, size: 20, color: theme.colorScheme.primary),
          const SizedBox(height: 4),
          Text(value,
              style:
                  const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          const SizedBox(height: 2),
          Text(label,
              style: TextStyle(
                  fontSize: 11, color: theme.colorScheme.onSurfaceVariant)),
        ],
      ),
    );
  }

  // ── 竞赛 / 入党 / 社团 ──
  Widget _buildActivitySection(ThemeData theme, Map<String, dynamic> data) {
    final competitions =
        (data['competitions'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final party = data['party'] as Map<String, dynamic>? ?? {};
    final clubs = (data['clubs'] as List?)?.cast<Map<String, dynamic>>() ?? [];

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(Icons.emoji_events_outlined,
                  color: theme.colorScheme.primary, size: 18),
              const SizedBox(width: 8),
              Text('竞赛 · 入党 · 社团', style: theme.textTheme.titleMedium),
            ]),
            const SizedBox(height: 12),
            Text('学科竞赛',
                style: theme.textTheme.titleSmall
                    ?.copyWith(color: theme.colorScheme.primary)),
            const SizedBox(height: 6),
            if (competitions.isEmpty)
              _emptyText(theme, '暂无竞赛记录')
            else
              ...competitions.map((c) {
                final name = c['student_name'] ?? '';
                final status = c['status'] ?? '';
                final award = c['award_level'] ?? '';
                return ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.flag_outlined, size: 18),
                  title: Text(name.isEmpty ? '竞赛报名' : name,
                      style: theme.textTheme.bodySmall),
                  subtitle: Text(
                    award.isNotEmpty
                        ? '获奖：$award'
                        : '状态：${_competitionStatus(status)}',
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                  ),
                );
              }),
            const SizedBox(height: 10),
            Text('入党进度',
                style: theme.textTheme.titleSmall
                    ?.copyWith(color: theme.colorScheme.primary)),
            const SizedBox(height: 4),
            _emptyText(
                theme,
                party['status'] != null && party['status'] != ''
                    ? '${_partyStatus(party['status'])} · ${party['current_stage'] ?? ''}'
                    : '暂无入党记录'),
            const SizedBox(height: 10),
            Text('社团参与',
                style: theme.textTheme.titleSmall
                    ?.copyWith(color: theme.colorScheme.primary)),
            const SizedBox(height: 4),
            if (clubs.isEmpty)
              _emptyText(theme, '暂无社团')
            else
              ...clubs.map((c) => _emptyText(
                  theme, '社团 #${c['club_id']} · ${c['role'] ?? 'member'}')),
          ],
        ),
      ),
    );
  }

  String _competitionStatus(String s) {
    const map = {
      'registered': '已报名',
      'submitted': '已提交作品',
      'awarded': '已获奖',
      'not_awarded': '未获奖',
    };
    return map[s] ?? s;
  }

  String _partyStatus(String s) {
    const map = {
      'applicant': '申请人',
      'activist': '积极分子',
      'development': '发展对象',
      'probation': '预备党员',
      'member': '正式党员',
    };
    return map[s] ?? s;
  }

  Widget _emptyText(ThemeData theme, String text) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Text(text,
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
    );
  }

  // ── 打卡 / 积分 ──
  Widget _buildStatsSection(ThemeData theme, Map<String, dynamic> data) {
    final checkin = data['checkin'] as Map<String, dynamic>? ?? {};
    final points = data['points'] as Map<String, dynamic>? ?? {};
    final totalDays = checkin['total_days'] ?? 0;
    final totalPoints = points['total_points'] ?? 0;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(Icons.local_fire_department,
                  color: theme.colorScheme.primary, size: 18),
              const SizedBox(width: 8),
              Text('打卡 · 积分', style: theme.textTheme.titleMedium),
            ]),
            const SizedBox(height: 14),
            Row(
              children: [
                _statItem(
                    theme, '打卡天数', '$totalDays', Icons.local_fire_department),
                _statItem(theme, '累计积分', '$totalPoints', Icons.stars),
                _statItem(theme, '档案完整度', '完整', Icons.verified_outlined),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
