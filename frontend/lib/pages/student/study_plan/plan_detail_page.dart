import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/study_plan_provider.dart';
import '../../../widgets/error_view.dart';

/// 学习计划详情页
///
/// 顶部展示计划标题、日期范围与进度环；
/// 中部展示目标列表（goals_json 解析）；
/// 下方为任务列表，每个任务卡片可勾选完成、可展开填写反思；
/// 底部提供"添加任务"按钮。
class PlanDetailPage extends StatefulWidget {
  final String planId;
  const PlanDetailPage({super.key, required this.planId});

  @override
  State<PlanDetailPage> createState() => _PlanDetailPageState();
}

class _PlanDetailPageState extends State<PlanDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudyPlanProvider>().fetchPlanDetail(widget.planId);
    });
  }

  @override
  void dispose() {
    context.read<StudyPlanProvider>().clearDetail();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyPlanProvider>();
    final plan = provider.currentPlan;

    return Scaffold(
      appBar: AppBar(
        title: const Text('计划详情'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: '刷新',
            onPressed: () => provider.fetchPlanDetail(widget.planId),
          ),
        ],
      ),
      body: provider.loading && plan == null
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && plan == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchPlanDetail(widget.planId),
                )
              : plan == null
                  ? ErrorView.empty(
                      message: '计划不存在',
                      icon: Icons.checklist_rtl_outlined,
                    )
                  : _buildBody(theme, provider, plan),
      floatingActionButton: plan == null
          ? null
          : FloatingActionButton.extended(
              onPressed: () => _showAddTaskSheet(context),
              icon: const Icon(Icons.add_task),
              label: const Text('添加任务'),
            ),
    );
  }

  Widget _buildBody(
    ThemeData theme,
    StudyPlanProvider provider,
    Map<String, dynamic> plan,
  ) {
    final title = (plan['title'] as String?) ?? '未命名计划';
    final start = (plan['start_date'] as String?) ?? '';
    final end = (plan['end_date'] as String?) ?? '';
    final progress = ((plan['progress'] as num?)?.toDouble() ?? 0.0).clamp(0.0, 100.0);
    final planType = (plan['plan_type'] as String?) ?? 'weekly';
    final status = (plan['status'] as String?) ?? 'active';
    final goals = _parseGoals(plan['goals_json']);

    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 96),
      children: [
        _PlanHeader(
          title: title,
          start: start,
          end: end,
          progress: progress,
          planType: planType,
          status: status,
        ),
        const SizedBox(height: 16),
        if (goals.isNotEmpty) ...[
          _GoalsCard(goals: goals),
          const SizedBox(height: 16),
        ],
        _TasksSection(provider: provider, planId: widget.planId),
      ],
    );
  }

  /// 兼容 goals_json 为 JSON 字符串 / List / null 的情况
  List<String> _parseGoals(dynamic raw) {
    if (raw == null) return const [];
    if (raw is List) {
      return raw.map((e) => e.toString()).where((e) => e.isNotEmpty).toList();
    }
    if (raw is String) {
      final s = raw.trim();
      if (s.isEmpty) return const [];
      // 尝试 JSON 解析
      try {
        final decoded = jsonDecode(s);
        if (decoded is List) {
          return decoded.map((e) => e.toString()).where((e) => e.isNotEmpty).toList();
        }
      } catch (_) {
        // 非 JSON，按行切分
      }
      return s
          .split('\n')
          .map((e) => e.trim().replaceAll(RegExp(r'^[\d\-\.\·\*]+\s*'), ''))
          .where((e) => e.isNotEmpty)
          .toList();
    }
    return const [];
  }

  void _showAddTaskSheet(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (_) => _AddTaskSheet(planId: widget.planId),
    );
  }
}

/// 顶部计划信息卡 —— 标题、日期范围、进度环
class _PlanHeader extends StatelessWidget {
  final String title;
  final String start;
  final String end;
  final double progress;
  final String planType;
  final String status;
  const _PlanHeader({
    required this.title,
    required this.start,
    required this.end,
    required this.progress,
    required this.planType,
    required this.status,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final typeLabel = _typeLabel(planType);
    final typeColor = _typeColor(planType, theme);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Text(
                    title,
                    style: theme.textTheme.titleLarge
                        ?.copyWith(fontWeight: FontWeight.bold),
                  ),
                ),
                const SizedBox(width: 12),
                SizedBox(
                  width: 64,
                  height: 64,
                  child: Stack(
                    alignment: Alignment.center,
                    children: [
                      CircularProgressIndicator(
                        value: progress / 100.0,
                        strokeWidth: 6,
                        backgroundColor:
                            theme.colorScheme.surfaceContainerHighest,
                        color: typeColor,
                      ),
                      Text(
                        '${progress.toStringAsFixed(0)}%',
                        style: theme.textTheme.labelLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: typeColor,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 6,
              children: [
                _MetaChip(
                  icon: Icons.category_outlined,
                  text: typeLabel,
                  color: typeColor,
                ),
                _MetaChip(
                  icon: Icons.flag_outlined,
                  text: _statusLabel(status),
                  color: _statusColor(status, theme),
                ),
                if (start.isNotEmpty || end.isNotEmpty)
                  _MetaChip(
                    icon: Icons.date_range_outlined,
                    text: _formatRange(start, end),
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _formatRange(String s, String e) {
    String fmt(String x) => x.length >= 10 ? x.substring(0, 10) : x;
    if (s.isEmpty && e.isEmpty) return '';
    if (e.isEmpty) return fmt(s);
    if (s.isEmpty) return '至 ${fmt(e)}';
    return '${fmt(s)} ~ ${fmt(e)}';
  }

  String _typeLabel(String type) => switch (type) {
        'weekly' => '本周计划',
        'monthly' => '本月计划',
        'quarterly' => '本季度计划',
        'semester' => '本学期计划',
        'yearly' => '本学年计划',
        'four_year' => '大学四年计划',
        _ => '学习计划',
      };

  Color _typeColor(String type, ThemeData theme) => switch (type) {
        'weekly' => const Color(0xFF1565C0),
        'monthly' => const Color(0xFFE65100),
        'quarterly' => const Color(0xFF2E7D32),
        'semester' => const Color(0xFF7B1FA2),
        'yearly' => const Color(0xFFC62828),
        'four_year' => const Color(0xFF00695C),
        _ => theme.colorScheme.primary,
      };

  String _statusLabel(String status) => switch (status) {
        'active' => '进行中',
        'done' || 'completed' => '已完成',
        'archived' => '已归档',
        _ => status,
      };

  Color _statusColor(String status, ThemeData theme) => switch (status) {
        'active' => theme.colorScheme.primary,
        'done' || 'completed' => const Color(0xFF2E7D32),
        'archived' => theme.colorScheme.outline,
        _ => theme.colorScheme.outline,
      };
}

class _MetaChip extends StatelessWidget {
  final IconData icon;
  final String text;
  final Color color;
  const _MetaChip({
    required this.icon,
    required this.text,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color.withOpacity(0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 14, color: color),
          const SizedBox(width: 4),
          Text(
            text,
            style: TextStyle(
              fontSize: 12,
              color: color,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

/// 目标列表卡片
class _GoalsCard extends StatelessWidget {
  final List<String> goals;
  const _GoalsCard({required this.goals});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.flag_outlined, color: theme.colorScheme.primary, size: 20),
                const SizedBox(width: 6),
                Text(
                  '计划目标',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w600),
                ),
              ],
            ),
            const SizedBox(height: 12),
            for (int i = 0; i < goals.length; i++) ...[
              if (i > 0) const SizedBox(height: 8),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    width: 22,
                    height: 22,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: theme.colorScheme.primaryContainer,
                      shape: BoxShape.circle,
                    ),
                    child: Text(
                      '${i + 1}',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onPrimaryContainer,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      goals[i],
                      style: theme.textTheme.bodyMedium,
                    ),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// 任务列表区
class _TasksSection extends StatelessWidget {
  final StudyPlanProvider provider;
  final String planId;
  const _TasksSection({required this.provider, required this.planId});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final tasks = provider.currentTasks;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(Icons.task_alt, color: theme.colorScheme.primary, size: 20),
            const SizedBox(width: 6),
            Text(
              '任务列表（${tasks.length}）',
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w600),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (tasks.isEmpty)
          Container(
            padding: const EdgeInsets.all(24),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
              borderRadius: BorderRadius.circular(12),
            ),
            child: ErrorView.empty(
              message: '暂无任务',
              subtitle: '点击下方"添加任务"开始拆解计划',
              icon: Icons.task_outlined,
            ),
          )
        else
          for (final raw in tasks)
            _TaskCard(
              task: raw as Map<String, dynamic>,
              planId: planId,
            ),
      ],
    );
  }
}

/// 单个任务卡片
class _TaskCard extends StatefulWidget {
  final Map<String, dynamic> task;
  final String planId;
  const _TaskCard({required this.task, required this.planId});

  @override
  State<_TaskCard> createState() => _TaskCardState();
}

class _TaskCardState extends State<_TaskCard> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final task = widget.task;
    final id = (task['id']?.toString()) ?? '';
    final title = (task['title'] as String?) ?? '任务';
    final desc = (task['description'] as String?) ?? '';
    final courseName = (task['course_name'] as String?) ?? '';
    final courseId = (task['course_id'] as String?) ?? '';
    final scheduledDate = (task['scheduled_date'] as String?) ?? '';
    final scheduledDuration = (task['scheduled_duration'] as num?)?.toInt() ?? 0;
    final actualDuration = (task['actual_duration'] as num?)?.toInt() ?? 0;
    final status = (task['status'] as String?) ?? 'pending';

    final isDone = status == 'done' || status == 'completed';

    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Checkbox(
                  value: isDone,
                  onChanged: providerBusy
                      ? null
                      : (v) => _toggleDone(v ?? false, status, isDone),
                  shape: const CircleBorder(),
                ),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      if (courseName.isNotEmpty || courseId.isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.only(left: 0, bottom: 4),
                          child: Wrap(
                            spacing: 6,
                            children: [
                              if (courseName.isNotEmpty)
                                _CourseChip(label: courseName),
                              if (courseId.isNotEmpty && courseName.isEmpty)
                                _CourseChip(label: courseId),
                            ],
                          ),
                        ),
                      Text(
                        title,
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                          decoration: isDone ? TextDecoration.lineThrough : null,
                          color: isDone ? theme.colorScheme.outline : null,
                        ),
                      ),
                      if (desc.isNotEmpty) ...[
                        const SizedBox(height: 4),
                        Text(
                          desc,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                      const SizedBox(height: 6),
                      Wrap(
                        spacing: 12,
                        runSpacing: 4,
                        children: [
                          if (scheduledDate.isNotEmpty)
                            _MetaLine(
                              icon: Icons.event_outlined,
                              text: scheduledDate.length >= 10
                                  ? scheduledDate.substring(0, 10)
                                  : scheduledDate,
                            ),
                          if (scheduledDuration > 0)
                            _MetaLine(
                              icon: Icons.timer_outlined,
                              text: '计划 $scheduledDuration 分钟',
                            ),
                          if (actualDuration > 0)
                            _MetaLine(
                              icon: Icons.history,
                              text: '实际 $actualDuration 分钟',
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
                _TaskStatusChip(status: status),
              ],
            ),
            // 展开反思区
            if (isDone || _expanded) ...[
              const Divider(height: 20),
              _ReflectionRow(
                taskId: id,
                planId: widget.planId,
                initialReflection: (task['reflection'] as String?) ?? '',
              ),
            ] else
              Align(
                alignment: Alignment.centerRight,
                child: TextButton.icon(
                  onPressed: () => setState(() => _expanded = true),
                  icon: const Icon(Icons.edit_note, size: 16),
                  label: const Text('反思', style: TextStyle(fontSize: 12)),
                ),
              ),
          ],
        ),
      ),
    );
  }

  StudyPlanProvider get provider => context.read<StudyPlanProvider>();
  bool get providerBusy =>
      context.read<StudyPlanProvider>().mutating;

  Future<void> _toggleDone(bool checked, String status, bool isDone) async {
    if (isDone && checked) return;
    if (!isDone && !checked) return;
    final next = checked ? 'done' : 'pending';
    final ok = await provider.updateTaskStatus(
      widget.planId,
      _taskId(),
      status: next,
    );
    if (!mounted) return;
    if (!ok && provider.error.isNotEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('更新失败：${provider.error}')),
      );
    }
  }

  String _taskId() {
    final id = widget.task['id']?.toString() ?? '';
    return id;
  }
}

class _ReflectionRow extends StatefulWidget {
  final String taskId;
  final String planId;
  final String initialReflection;
  const _ReflectionRow({
    required this.taskId,
    required this.planId,
    required this.initialReflection,
  });

  @override
  State<_ReflectionRow> createState() => _ReflectionRowState();
}

class _ReflectionRowState extends State<_ReflectionRow> {
  late final TextEditingController _ctrl;
  bool _editing = false;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.initialReflection);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(Icons.edit_note, size: 18, color: theme.colorScheme.outline),
        const SizedBox(width: 6),
        Expanded(
          child: _editing
              ? TextField(
                  controller: _ctrl,
                  minLines: 1,
                  maxLines: 3,
                  autofocus: true,
                  decoration: InputDecoration(
                    isDense: true,
                    hintText: '写下你的反思…',
                    border: const OutlineInputBorder(),
                    suffixIcon: IconButton(
                      icon: const Icon(Icons.check, size: 18),
                      onPressed: _save,
                    ),
                  ),
                )
              : InkWell(
                  onTap: () => setState(() => _editing = true),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 4),
                    child: Text(
                      _ctrl.text.isEmpty ? '点击添加反思…' : _ctrl.text,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: _ctrl.text.isEmpty
                            ? theme.colorScheme.outline
                            : theme.colorScheme.onSurfaceVariant,
                        fontStyle: _ctrl.text.isEmpty
                            ? FontStyle.italic
                            : FontStyle.normal,
                      ),
                    ),
                  ),
                ),
        ),
      ],
    );
  }

  Future<void> _save() async {
    final text = _ctrl.text.trim();
    final provider = context.read<StudyPlanProvider>();
    final ok = await provider.updateTaskStatus(
      widget.planId,
      widget.taskId,
      status: 'done',
      reflection: text,
    );
    if (!mounted) return;
    setState(() => _editing = false);
    if (!ok && provider.error.isNotEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('保存失败：${provider.error}')),
      );
    }
  }
}

class _MetaLine extends StatelessWidget {
  final IconData icon;
  final String text;
  const _MetaLine({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 13, color: theme.colorScheme.onSurfaceVariant),
        const SizedBox(width: 3),
        Text(
          text,
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _CourseChip extends StatelessWidget {
  final String label;
  const _CourseChip({required this.label});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: theme.colorScheme.tertiaryContainer,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: theme.colorScheme.onTertiaryContainer,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _TaskStatusChip extends StatelessWidget {
  final String status;
  const _TaskStatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (label, color) = switch (status) {
      'done' || 'completed' => ('已完成', const Color(0xFF2E7D32)),
      'pending' => ('待完成', theme.colorScheme.primary),
      'in_progress' => ('进行中', const Color(0xFFE65100)),
      'skipped' => ('已跳过', theme.colorScheme.outline),
      _ => (status, theme.colorScheme.outline),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
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

/// 添加任务底部表单
class _AddTaskSheet extends StatefulWidget {
  final String planId;
  const _AddTaskSheet({required this.planId});

  @override
  State<_AddTaskSheet> createState() => _AddTaskSheetState();
}

class _AddTaskSheetState extends State<_AddTaskSheet> {
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  final _courseCtrl = TextEditingController();
  DateTime? _scheduledDate;
  int _duration = 60;

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    _courseCtrl.dispose();
    super.dispose();
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
                  Icon(Icons.add_task, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text(
                    '添加任务',
                    style: theme.textTheme.titleLarge
                        ?.copyWith(fontWeight: FontWeight.w600),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _titleCtrl,
                decoration: const InputDecoration(
                  labelText: '任务标题',
                  hintText: '例如：复习微积分第8章',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.title),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _courseCtrl,
                decoration: const InputDecoration(
                  labelText: '关联课程（可选）',
                  hintText: '例如：高等数学',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.menu_book_outlined),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _descCtrl,
                maxLines: 3,
                decoration: const InputDecoration(
                  labelText: '任务描述（可选）',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.description_outlined),
                ),
              ),
              const SizedBox(height: 12),
              InkWell(
                onTap: () async {
                  final picked = await showDatePicker(
                    context: context,
                    initialDate: _scheduledDate ?? DateTime.now(),
                    firstDate: DateTime(2020),
                    lastDate: DateTime(2100),
                  );
                  if (picked != null) setState(() => _scheduledDate = picked);
                },
                child: InputDecorator(
                  decoration: const InputDecoration(
                    labelText: '计划日期',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.event_outlined),
                  ),
                  child: Text(
                    _scheduledDate == null
                        ? '请选择日期'
                        : _fmt(_scheduledDate!),
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Icon(Icons.timer_outlined, size: 20),
                  const SizedBox(width: 8),
                  Text('计划时长（分钟）',
                      style: theme.textTheme.bodyMedium),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Slider(
                      value: _duration.toDouble(),
                      min: 15,
                      max: 240,
                      divisions: 15,
                      label: '$_duration 分钟',
                      onChanged: (v) => setState(() => _duration = v.round()),
                    ),
                  ),
                  SizedBox(
                    width: 60,
                    child: Text(
                      '$_duration',
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                      textAlign: TextAlign.end,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: provider.mutating ? null : _onSubmit,
                icon: provider.mutating
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.check),
                label: Text(provider.mutating ? '提交中…' : '添加任务'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _onSubmit() async {
    final title = _titleCtrl.text.trim();
    if (title.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请填写任务标题')),
      );
      return;
    }
    final body = <String, dynamic>{
      'title': title,
      if (_descCtrl.text.trim().isNotEmpty) 'description': _descCtrl.text.trim(),
      if (_courseCtrl.text.trim().isNotEmpty)
        'course_name': _courseCtrl.text.trim(),
      if (_scheduledDate != null) 'scheduled_date': _fmt(_scheduledDate!),
      'scheduled_duration': _duration,
      'status': 'pending',
    };
    final provider = context.read<StudyPlanProvider>();
    final ok = await provider.addTask(widget.planId, body);
    if (!mounted) return;
    Navigator.pop(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(ok ? '任务已添加' : '添加失败，请重试')),
    );
  }
}
