import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../../providers/study_plan_provider.dart';
import '../../../widgets/error_view.dart';

/// 学习计划主页
///
/// 顶部展示当前学期校历横幅（学期名、当前教学周、近期事件），
/// 下方 TabBar 按 plan_type（本周/本月/本季度/本学期/本学年/大学四年）切换，
/// 每个 Tab 列出对应计划卡片（标题、日期范围、进度条、任务统计），
/// 右下角 FAB 弹出底部表单创建计划（含 AI 一键生成）。
class StudyPlanPage extends StatefulWidget {
  const StudyPlanPage({super.key});

  @override
  State<StudyPlanPage> createState() => _StudyPlanPageState();
}

class _StudyPlanPageState extends State<StudyPlanPage>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;

  /// Tab ↔ plan_type 映射（顺序与 TabBar 一致）
  static const List<_PlanTab> _tabs = [
    _PlanTab('本周', 'weekly', Color(0xFF1565C0)),
    _PlanTab('本月', 'monthly', Color(0xFFE65100)),
    _PlanTab('本季度', 'quarterly', Color(0xFF2E7D32)),
    _PlanTab('本学期', 'semester', Color(0xFF7B1FA2)),
    _PlanTab('本学年', 'yearly', Color(0xFFC62828)),
    _PlanTab('大学四年', 'four_year', Color(0xFF00695C)),
  ];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: _tabs.length, vsync: this);
    _tabController.addListener(_onTabChanged);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudyPlanProvider>();
      p.fetchCalendar();
      p.fetchOverview();
      p.fetchPlans(_tabs.first.type, force: true);
    });
  }

  @override
  void dispose() {
    _tabController.removeListener(_onTabChanged);
    _tabController.dispose();
    super.dispose();
  }

  void _onTabChanged() {
    if (_tabController.indexIsChanging) return;
    final type = _tabs[_tabController.index].type;
    final p = context.read<StudyPlanProvider>();
    p.setCurrentType(type);
    p.fetchPlans(type, force: false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyPlanProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('学习计划'),
        actions: [
          IconButton(
            icon: const Icon(Icons.table_view_outlined),
            tooltip: '我的课表',
            onPressed: () => context.go('/student/timetable'),
          ),
        ],
        bottom: TabBar(
          controller: _tabController,
          isScrollable: true,
          tabAlignment: TabAlignment.start,
          tabs: [
            for (final t in _tabs)
              Tab(text: t.label),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          for (final t in _tabs)
            _buildTabBody(theme, provider, t),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _showCreateSheet(context, _tabs[_tabController.index]),
        icon: const Icon(Icons.add),
        label: const Text('创建计划'),
      ),
    );
  }

  /// 单个 Tab 的内容：顶部校历横幅（仅第一项展示） + 计划列表
  Widget _buildTabBody(
    ThemeData theme,
    StudyPlanProvider provider,
    _PlanTab tab,
  ) {
    final plans = provider.plansOf(tab.type);
    final showBanner = tab.type == 'weekly';

    return RefreshIndicator(
      onRefresh: () async {
        if (showBanner) await provider.fetchCalendar();
        await Future.wait([
          provider.fetchPlans(tab.type, force: true),
          provider.fetchOverview(),
        ]);
      },
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 96),
        children: [
          if (showBanner) ...[
            _CalendarBanner(provider: provider),
            const SizedBox(height: 16),
          ],
          _OverviewStrip(provider: provider, activeType: tab.type),
          const SizedBox(height: 12),
          _buildPlanList(theme, provider, tab, plans),
        ],
      ),
    );
  }

  Widget _buildPlanList(
    ThemeData theme,
    StudyPlanProvider provider,
    _PlanTab tab,
    List<dynamic> plans,
  ) {
    if (provider.loading && plans.isEmpty) {
      return const SizedBox(
        height: 240,
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (provider.error.isNotEmpty && plans.isEmpty) {
      return SizedBox(
        height: 240,
        child: ErrorView.error(
          message: provider.error,
          onRetry: () => provider.fetchPlans(tab.type, force: true),
        ),
      );
    }
    if (plans.isEmpty) {
      return SizedBox(
        height: 240,
        child: ErrorView.empty(
          message: '暂无${tab.label}计划',
          subtitle: '点击右下角"创建计划"开始规划吧',
          icon: Icons.checklist_rtl_outlined,
        ),
      );
    }
    return Column(
      children: [
        for (final raw in plans)
          _PlanCard(
            plan: raw as Map<String, dynamic>,
            accent: tab.color,
            onTap: () {
              final id = (planIdOf(raw)).toString();
              if (id.isEmpty) return;
              context.go('/student/study-plan/$id');
            },
            onDelete: () => _confirmDelete(raw, tab.type),
          ),
      ],
    );
  }

  /// 删除二次确认
  Future<void> _confirmDelete(Map<String, dynamic> plan, String type) async {
    final id = planIdOf(plan);
    final title = (plan['title'] as String?) ?? '该计划';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除计划'),
        content: Text('确定要删除「$title」吗？该操作不可恢复。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton.tonal(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(
              foregroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (ok != true) return;
    if (!mounted) return;
    final provider = context.read<StudyPlanProvider>();
    final success = await provider.deletePlan(
      id.toString(),
      planType: type,
    );
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(success ? '已删除' : '删除失败，请重试')),
      );
    }
  }

  /// 创建计划底部表单
  void _showCreateSheet(BuildContext context, _PlanTab tab) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (_) => _CreatePlanSheet(tab: tab),
    );
  }
}

/// 从计划对象中提取稳定的 id（兼容 id 是 int 或 string 的情况）
Object planIdOf(Map<String, dynamic> plan) {
  final id = plan['id'];
  return id ?? '';
}

class _PlanTab {
  final String label;
  final String type;
  final Color color;
  const _PlanTab(this.label, this.type, this.color);
}

/// 校历横幅 —— 学期名、当前教学周、近期事件
class _CalendarBanner extends StatelessWidget {
  final StudyPlanProvider provider;
  const _CalendarBanner({required this.provider});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cal = provider.calendarData;

    if (provider.loading && cal == null) {
      return Container(
        padding: const EdgeInsets.all(20),
        decoration: _bannerDecoration(theme),
        child: const Row(
          children: [
            SizedBox(
              width: 22,
              height: 22,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
            SizedBox(width: 12),
            Text('正在加载校历…'),
          ],
        ),
      );
    }
    if (cal == null) {
      return Container(
        padding: const EdgeInsets.all(16),
        decoration: _bannerDecoration(theme),
        child: Row(
          children: [
            Icon(Icons.event_busy_outlined, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                provider.error.isNotEmpty ? '校历加载失败' : '暂无校历信息',
                style: theme.textTheme.bodyMedium,
              ),
            ),
            TextButton(
              onPressed: () => provider.fetchCalendar(),
              child: const Text('重试'),
            ),
          ],
        ),
      );
    }

    final semester = (cal['semester'] as Map<String, dynamic>?) ?? {};
    final semesterName = (semester['semester_name'] as String?) ?? '本学期';
    final currentWeek = cal['current_week'] as int? ?? 0;
    final totalWeeks = semester['total_weeks'] as int? ?? 0;
    final events = (cal['upcoming_events'] as List?) ??
        (cal['events'] as List?) ??
        [];

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: _bannerDecoration(theme),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.school_rounded, color: theme.colorScheme.onPrimary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  semesterName,
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: theme.colorScheme.onPrimary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              if (currentWeek > 0)
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.22),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    totalWeeks > 0
                        ? '第 $currentWeek / $totalWeeks 教学周'
                        : '第 $currentWeek 教学周',
                    style: TextStyle(
                      color: theme.colorScheme.onPrimary,
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
            ],
          ),
          if (events.isNotEmpty) ...[
            const SizedBox(height: 12),
            Text(
              '近期事件',
              style: theme.textTheme.labelMedium?.copyWith(
                color: theme.colorScheme.onPrimary.withOpacity(0.85),
              ),
            ),
            const SizedBox(height: 6),
            Wrap(
              spacing: 8,
              runSpacing: 6,
              children: [
                for (final e in events.take(3))
                  _EventChip(event: e as Map<String, dynamic>),
              ],
            ),
          ],
        ],
      ),
    );
  }

  BoxDecoration _bannerDecoration(ThemeData theme) {
    return BoxDecoration(
      gradient: LinearGradient(
        colors: [
          theme.colorScheme.primary,
          theme.colorScheme.primary.withOpacity(0.75),
        ],
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
      ),
      borderRadius: BorderRadius.circular(16),
    );
  }
}

class _EventChip extends StatelessWidget {
  final Map<String, dynamic> event;
  const _EventChip({required this.event});

  @override
  Widget build(BuildContext context) {
    final name = (event['event_name'] as String?) ??
        (event['name'] as String?) ??
        '事件';
    final type = (event['event_type'] as String?) ?? '';
    final start = (event['start_date'] as String?) ?? '';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.18),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.25)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(_eventIcon(type), size: 14, color: Colors.white),
          const SizedBox(width: 4),
          Text(
            name,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 12,
              fontWeight: FontWeight.w500,
            ),
          ),
          if (start.isNotEmpty) ...[
            const SizedBox(width: 6),
            Text(
              start.substring(0, start.length >= 10 ? 10 : start.length),
              style: const TextStyle(color: Colors.white70, fontSize: 11),
            ),
          ],
        ],
      ),
    );
  }

  IconData _eventIcon(String type) {
    switch (type) {
      case 'exam':
        return Icons.assignment_outlined;
      case 'holiday':
        return Icons.beach_access_outlined;
      case 'activity':
        return Icons.celebration_outlined;
      default:
        return Icons.event_outlined;
    }
  }
}

/// 各维度概览条 —— 显示当前 Tab 高亮
class _OverviewStrip extends StatelessWidget {
  final StudyPlanProvider provider;
  final String activeType;
  const _OverviewStrip({required this.provider, required this.activeType});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final ov = provider.overview;
    if (ov == null) return const SizedBox.shrink();

    final items = <_OverviewItem>[
      _OverviewItem('本周', 'weekly', ov['weekly']),
      _OverviewItem('本月', 'monthly', ov['monthly']),
      _OverviewItem('本季度', 'quarterly', ov['quarterly']),
      _OverviewItem('本学期', 'semester', ov['semester']),
      _OverviewItem('本学年', 'yearly', ov['yearly']),
      _OverviewItem('大学四年', 'four_year', ov['four_year']),
    ];

    return SizedBox(
      height: 76,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        itemCount: items.length,
        separatorBuilder: (_, __) => const SizedBox(width: 8),
        itemBuilder: (_, i) {
          final it = items[i];
          final active = it.type == activeType;
          final data = (it.raw as Map<String, dynamic>?) ?? {};
          final count = (data['count'] as num?)?.toInt() ?? 0;
          final avg = (data['avg_progress'] as num?)?.toDouble() ?? 0.0;
          return Container(
            width: 96,
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            decoration: BoxDecoration(
              color: active
                  ? theme.colorScheme.primaryContainer
                  : theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
              borderRadius: BorderRadius.circular(10),
              border: active
                  ? Border.all(color: theme.colorScheme.primary, width: 1.2)
                  : null,
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  it.label,
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: active
                        ? theme.colorScheme.onPrimaryContainer
                        : theme.colorScheme.onSurfaceVariant,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '$count 个计划',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  '均进 ${avg.toStringAsFixed(0)}%',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _OverviewItem {
  final String label;
  final String type;
  final Object? raw;
  const _OverviewItem(this.label, this.type, this.raw);
}

/// 单个计划卡片
class _PlanCard extends StatelessWidget {
  final Map<String, dynamic> plan;
  final Color accent;
  final VoidCallback onTap;
  final VoidCallback onDelete;
  const _PlanCard({
    required this.plan,
    required this.accent,
    required this.onTap,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title = (plan['title'] as String?) ?? '未命名计划';
    final start = (plan['start_date'] as String?) ?? '';
    final end = (plan['end_date'] as String?) ?? '';
    final progress = ((plan['progress'] as num?)?.toDouble() ?? 0.0).clamp(0.0, 100.0);
    final status = (plan['status'] as String?) ?? 'active';
    final stats = (plan['task_stats'] as Map<String, dynamic>?) ?? {};
    final total = (stats['total'] as num?)?.toInt() ?? 0;
    final done = (stats['done'] as num?)?.toInt() ?? 0;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        onLongPress: onDelete,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 6,
                    height: 22,
                    margin: const EdgeInsets.only(right: 10),
                    decoration: BoxDecoration(
                      color: accent,
                      borderRadius: BorderRadius.circular(3),
                    ),
                  ),
                  Expanded(
                    child: Text(
                      title,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  _StatusChip(status: status),
                ],
              ),
              if (start.isNotEmpty || end.isNotEmpty) ...[
                const SizedBox(height: 8),
                Row(
                  children: [
                    Icon(Icons.date_range_outlined,
                        size: 14, color: theme.colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text(
                      _formatRange(start, end),
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ),
              ],
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(8),
                      child: LinearProgressIndicator(
                        value: progress / 100.0,
                        minHeight: 8,
                        backgroundColor:
                            theme.colorScheme.surfaceContainerHighest,
                        color: accent,
                      ),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Text(
                    '${progress.toStringAsFixed(0)}%',
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: accent,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Icon(Icons.task_alt_outlined,
                      size: 14, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(
                    '任务 $done / $total',
                    style: theme.textTheme.bodySmall,
                  ),
                  const Spacer(),
                  Icon(Icons.chevron_right,
                      size: 18, color: theme.colorScheme.onSurfaceVariant),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  String _formatRange(String start, String end) {
    String fmt(String s) =>
        s.length >= 10 ? s.substring(0, 10) : s;
    if (start.isEmpty && end.isEmpty) return '';
    if (end.isEmpty) return fmt(start);
    if (start.isEmpty) return '至 ${fmt(end)}';
    return '${fmt(start)} ~ ${fmt(end)}';
  }
}

class _StatusChip extends StatelessWidget {
  final String status;
  const _StatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (label, color) = switch (status) {
      'active' => ('进行中', theme.colorScheme.primary),
      'done' || 'completed' => ('已完成', const Color(0xFF2E7D32)),
      'archived' => ('已归档', theme.colorScheme.outline),
      _ => (status, theme.colorScheme.outline),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: color,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

/// 创建计划底部表单（含 AI 一键生成）
class _CreatePlanSheet extends StatefulWidget {
  final _PlanTab tab;
  const _CreatePlanSheet({required this.tab});

  @override
  State<_CreatePlanSheet> createState() => _CreatePlanSheetState();
}

class _CreatePlanSheetState extends State<_CreatePlanSheet> {
  late final TextEditingController _titleCtrl;
  late final TextEditingController _goalsCtrl;
  late final TextEditingController _hintCtrl;
  DateTimeRange? _range;

  @override
  void initState() {
    super.initState();
    _titleCtrl = TextEditingController();
    _goalsCtrl = TextEditingController();
    _hintCtrl = TextEditingController();
    // 默认日期范围按 plan_type 推断
    _range = _defaultRange(widget.tab.type);
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _goalsCtrl.dispose();
    _hintCtrl.dispose();
    super.dispose();
  }

  DateTimeRange _defaultRange(String type) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    switch (type) {
      case 'weekly':
        final weekday = today.weekday; // 周一=1
        final monday = today.subtract(Duration(days: weekday - 1));
        return DateTimeRange(
          start: monday,
          end: monday.add(const Duration(days: 6)),
        );
      case 'monthly':
        final first = DateTime(today.year, today.month, 1);
        final last = DateTime(today.year, today.month + 1, 0);
        return DateTimeRange(start: first, end: last);
      case 'quarterly':
        final qStartMonth = ((today.month - 1) ~/ 3) * 3 + 1;
        final first = DateTime(today.year, qStartMonth, 1);
        final last = DateTime(today.year, qStartMonth + 3, 0);
        return DateTimeRange(start: first, end: last);
      case 'semester':
      case 'yearly':
        final first = DateTime(today.year, 9, 1);
        final last = DateTime(today.year + 1, 7, 31);
        return DateTimeRange(start: first, end: last);
      case 'four_year':
        final first = DateTime(today.year, 9, 1);
        final last = DateTime(today.year + 4, 7, 31);
        return DateTimeRange(start: first, end: last);
      default:
        return DateTimeRange(start: today, end: today.add(const Duration(days: 7)));
    }
  }

  String _fmt(DateTime d) =>
      '${d.year}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyPlanProvider>();
    final padBottom = MediaQuery.of(context).viewInsets.bottom;

    return Padding(
      padding: EdgeInsets.only(bottom: padBottom),
      child: SingleChildScrollView(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Container(
                width: 40,
                height: 4,
                margin: const EdgeInsets.only(bottom: 12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.outlineVariant,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              Row(
                children: [
                  Icon(Icons.add_circle, color: widget.tab.color),
                  const SizedBox(width: 8),
                  Text(
                    '创建${widget.tab.label}计划',
                    style: theme.textTheme.titleLarge
                        ?.copyWith(fontWeight: FontWeight.w600),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _titleCtrl,
                decoration: const InputDecoration(
                  labelText: '计划标题',
                  hintText: '例如：本周学习计划',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.title),
                ),
              ),
              const SizedBox(height: 12),
              InkWell(
                onTap: () async {
                  final picked = await showDateRangePicker(
                    context: context,
                    firstDate: DateTime(2020),
                    lastDate: DateTime(2100),
                    initialDateRange: _range,
                  );
                  if (picked != null) setState(() => _range = picked);
                },
                child: InputDecorator(
                  decoration: const InputDecoration(
                    labelText: '日期范围',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.date_range_outlined),
                  ),
                  child: Text(
                    _range == null
                        ? '请选择日期范围'
                        : '${_fmt(_range!.start)} ~ ${_fmt(_range!.end)}',
                  ),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _goalsCtrl,
                maxLines: 3,
                decoration: const InputDecoration(
                  labelText: '目标（每行一条）',
                  hintText: '例如：\n1. 完成高数第8章习题\n2. 英语单词背诵300个',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.flag_outlined),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _hintCtrl,
                maxLines: 2,
                decoration: const InputDecoration(
                  labelText: 'AI 生成提示（可选）',
                  hintText: '告诉 AI 你的侧重点，例如：重点关注薄弱科目',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.lightbulb_outline),
                ),
              ),
              const SizedBox(height: 16),
              FilledButton.tonalIcon(
                onPressed: provider.aiGenerating ? null : _onAIGenerate,
                icon: provider.aiGenerating
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.auto_awesome),
                label: Text(provider.aiGenerating ? 'AI 生成中…' : 'AI 一键生成'),
              ),
              if (provider.aiGenerating)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(
                    '正在根据日期范围与目标生成计划草稿…',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
              const SizedBox(height: 8),
              FilledButton.icon(
                onPressed: provider.mutating ? null : _onSubmit,
                icon: provider.mutating
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.check),
                label: Text(provider.mutating ? '提交中…' : '保存计划'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _onAIGenerate() async {
    final provider = context.read<StudyPlanProvider>();
    final hint = _hintCtrl.text.trim();
    final goals = _goalsCtrl.text.trim();
    final body = <String, dynamic>{
      'plan_type': widget.tab.type,
      if (_range != null) ...{
        'start_date': _fmt(_range!.start),
        'end_date': _fmt(_range!.end),
      },
      if (hint.isNotEmpty) 'hint': hint,
      if (goals.isNotEmpty) 'goals': goals.split('\n').map((e) => e.trim()).where((e) => e.isNotEmpty).toList(),
    };
    final draft = await provider.aiGeneratePlan(body);
    if (!mounted) return;
    if (draft != null) {
      final title = (draft['title'] as String?) ?? '';
      final goalsJson = draft['goals_json'];
      if (title.isNotEmpty) _titleCtrl.text = title;
      if (goalsJson is String && goalsJson.isNotEmpty) {
        _goalsCtrl.text = goalsJson;
      } else if (goalsJson is List && goalsJson.isNotEmpty) {
        _goalsCtrl.text = goalsJson.map((e) => '· $e').join('\n');
      }
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('AI 已生成草稿，请确认后保存')),
      );
    } else if (provider.error.isNotEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('AI 生成失败：${provider.error}')),
      );
    }
  }

  Future<void> _onSubmit() async {
    final title = _titleCtrl.text.trim();
    if (title.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请填写计划标题')),
      );
      return;
    }
    if (_range == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请选择日期范围')),
      );
      return;
    }
    final goalsRaw = _goalsCtrl.text
        .split('\n')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    final body = <String, dynamic>{
      'title': title,
      'plan_type': widget.tab.type,
      'start_date': _fmt(_range!.start),
      'end_date': _fmt(_range!.end),
      'goals_json': goalsRaw,
      'status': 'active',
    };
    final provider = context.read<StudyPlanProvider>();
    final ok = await provider.createPlan(body);
    if (!mounted) return;
    Navigator.pop(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(ok ? '计划已创建' : '创建失败，请重试')),
    );
  }
}
