import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../providers/teacher_feature_provider.dart';
import '../../widgets/error_view.dart';

/// 教师今日教学工作台。
class DailyOverviewPage extends StatefulWidget {
  final bool homeMode;

  const DailyOverviewPage({super.key, this.homeMode = false});

  @override
  State<DailyOverviewPage> createState() => _DailyOverviewPageState();
}

class _DailyOverviewPageState extends State<DailyOverviewPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TeacherFeatureProvider>().fetchDailyOverview();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.homeMode ? '今日教学' : '今日授课'),
        actions: [
          IconButton(
            onPressed: () => context.push('/notifications'),
            icon: const Icon(Icons.notifications_outlined),
            tooltip: '消息',
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: provider.fetchDailyOverview,
        child: _buildBody(provider),
      ),
    );
  }

  Widget _buildBody(TeacherFeatureProvider provider) {
    if (provider.loading && provider.dailyOverview == null) {
      return const _TeacherWorkspaceSkeleton();
    }
    if (provider.error.isNotEmpty && provider.dailyOverview == null) {
      return ListView(
        children: [
          SizedBox(
            height: 420,
            child: ErrorView.error(
              message: provider.error,
              onRetry: provider.fetchDailyOverview,
            ),
          ),
        ],
      );
    }

    final data = provider.dailyOverview;
    if (data == null) {
      return ListView(
        children: [
          SizedBox(
            height: 420,
            child: ErrorView.empty(
              message: '今天暂无教学安排',
              subtitle: '可以先使用 AI 工具准备后续课程',
              icon: Icons.event_available_outlined,
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(16),
            child: _buildToolSection(),
          ),
        ],
      );
    }
    return _buildWorkspace(data);
  }

  Widget _buildWorkspace(Map<String, dynamic> data) {
    final classes =
        (data['classes'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final pendingTasks =
        (data['pending_tasks'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final alerts = (data['alerts'] as List?)?.cast<String>() ?? [];

    return LayoutBuilder(
      builder: (context, constraints) {
        final desktop = constraints.maxWidth >= 920;
        return ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
          children: [
            _buildSummary(data, classes.length, pendingTasks.length),
            const SizedBox(height: 20),
            if (desktop)
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    flex: 7,
                    child: _buildScheduleSection(classes),
                  ),
                  const SizedBox(width: 20),
                  Expanded(
                    flex: 5,
                    child: Column(
                      children: [
                        _buildPendingSection(pendingTasks),
                        if (alerts.isNotEmpty) ...[
                          const SizedBox(height: 20),
                          _buildAlertSection(alerts),
                        ],
                      ],
                    ),
                  ),
                ],
              )
            else ...[
              _buildScheduleSection(classes),
              const SizedBox(height: 20),
              _buildPendingSection(pendingTasks),
              if (alerts.isNotEmpty) ...[
                const SizedBox(height: 20),
                _buildAlertSection(alerts),
              ],
            ],
            const SizedBox(height: 24),
            _buildToolSection(),
          ],
        );
      },
    );
  }

  Widget _buildSummary(
      Map<String, dynamic> data, int classCount, int taskCount) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.secondaryContainer.withOpacity(0.45),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: theme.colorScheme.outlineVariant),
      ),
      child: Wrap(
        spacing: 24,
        runSpacing: 12,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          Icon(Icons.cast_for_education,
              color: theme.colorScheme.secondary, size: 32),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(data['date']?.toString() ?? '今天',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  )),
              Text('共 $classCount 节课，$taskCount 项待处理',
                  style: theme.textTheme.titleLarge),
            ],
          ),
          FilledButton.tonalIcon(
            onPressed: () => context.push('/chat'),
            icon: const Icon(Icons.auto_awesome),
            label: const Text('使用教学助手'),
          ),
        ],
      ),
    );
  }

  Widget _buildScheduleSection(List<Map<String, dynamic>> classes) {
    return _WorkspaceSection(
      title: '课程安排',
      icon: Icons.schedule_outlined,
      emptyText: '今天暂无课程',
      children: [
        for (final course in classes)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: _CourseTile(course: course),
          ),
      ],
    );
  }

  Widget _buildPendingSection(List<Map<String, dynamic>> tasks) {
    return _WorkspaceSection(
      title: '待处理',
      icon: Icons.pending_actions_outlined,
      emptyText: '暂无待处理事项',
      children: [
        for (final task in tasks)
          ListTile(
            contentPadding: EdgeInsets.zero,
            leading: const Icon(Icons.task_alt_outlined),
            title: Text(task['task']?.toString() ?? ''),
            subtitle: Text(
              '${task['count'] ?? ''} 项 · 截止 ${task['deadline'] ?? ''}',
            ),
            trailing: const Icon(Icons.chevron_right),
          ),
      ],
    );
  }

  Widget _buildAlertSection(List<String> alerts) {
    final theme = Theme.of(context);
    return _WorkspaceSection(
      title: '需要确认',
      icon: Icons.notification_important_outlined,
      emptyText: '',
      children: [
        for (final alert in alerts)
          ListTile(
            contentPadding: EdgeInsets.zero,
            leading:
                Icon(Icons.info_outline, color: theme.colorScheme.tertiary),
            title: Text(alert),
            subtitle: const Text('请核对相关数据后再处理'),
          ),
      ],
    );
  }

  Widget _buildToolSection() {
    const tools = [
      _TeachingTool('AI 备课', Icons.auto_awesome, '/teacher/lesson-prep'),
      _TeachingTool('AI 出题', Icons.quiz_outlined, '/teacher/exam-gen'),
      _TeachingTool(
          '课堂互动', Icons.live_help_outlined, '/teacher/class-interact'),
      _TeachingTool('AI 批改', Icons.grading, '/teacher/grading'),
      _TeachingTool('学情分析', Icons.grid_on_outlined, '/teacher/heatmap'),
      _TeachingTool('教学反思', Icons.self_improvement, '/teacher/reflection'),
    ];
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('教学工具', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        LayoutBuilder(
          builder: (context, constraints) {
            final columns = constraints.maxWidth >= 1000
                ? 6
                : constraints.maxWidth >= 600
                    ? 3
                    : 2;
            const gap = 10.0;
            final width =
                (constraints.maxWidth - gap * (columns - 1)) / columns;
            return Wrap(
              spacing: gap,
              runSpacing: gap,
              children: [
                for (final tool in tools)
                  SizedBox(
                    width: width,
                    child: _TeachingToolTile(tool: tool),
                  ),
              ],
            );
          },
        ),
      ],
    );
  }
}

class _WorkspaceSection extends StatelessWidget {
  final String title;
  final IconData icon;
  final String emptyText;
  final List<Widget> children;

  const _WorkspaceSection({
    required this.title,
    required this.icon,
    required this.emptyText,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(icon, size: 20, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text(title, style: theme.textTheme.titleMedium),
          ],
        ),
        const SizedBox(height: 10),
        Card(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: children.isEmpty
                ? Padding(
                    padding: const EdgeInsets.symmetric(vertical: 24),
                    child: Center(
                      child: Text(emptyText,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          )),
                    ),
                  )
                : Column(children: children),
          ),
        ),
      ],
    );
  }
}

class _CourseTile extends StatelessWidget {
  final Map<String, dynamic> course;

  const _CourseTile({required this.course});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.4),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          SizedBox(
            width: 76,
            child: Text(
              course['time']?.toString() ?? '',
              style: theme.textTheme.labelLarge?.copyWith(
                color: theme.colorScheme.primary,
              ),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${course['course'] ?? ''} · ${course['class_name'] ?? ''}',
                  style: theme.textTheme.titleMedium,
                ),
                const SizedBox(height: 2),
                Text(
                  '${course['room'] ?? ''} · ${course['students'] ?? 0} 人',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: () => context.push('/teacher/lesson-prep'),
            icon: const Icon(Icons.auto_awesome),
            tooltip: '为这节课备课',
          ),
        ],
      ),
    );
  }
}

class _TeachingToolTile extends StatelessWidget {
  final _TeachingTool tool;

  const _TeachingToolTile({required this.tool});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: InkWell(
        onTap: () => context.push(tool.route),
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 16),
          child: Column(
            children: [
              Icon(tool.icon, color: theme.colorScheme.secondary, size: 26),
              const SizedBox(height: 8),
              Text(tool.label,
                  style: theme.textTheme.labelLarge,
                  textAlign: TextAlign.center),
            ],
          ),
        ),
      ),
    );
  }
}

class _TeachingTool {
  final String label;
  final IconData icon;
  final String route;

  const _TeachingTool(this.label, this.icon, this.route);
}

class _TeacherWorkspaceSkeleton extends StatelessWidget {
  const _TeacherWorkspaceSkeleton();

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.surfaceContainerHighest;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Container(height: 112, color: color),
        const SizedBox(height: 20),
        Container(height: 280, color: color),
        const SizedBox(height: 20),
        Container(height: 140, color: color),
      ],
    );
  }
}
