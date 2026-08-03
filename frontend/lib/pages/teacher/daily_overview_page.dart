import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';

/// 教师 - AI 今日授课概览
class DailyOverviewPage extends StatefulWidget {
  const DailyOverviewPage({super.key});
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
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 今日授课概览')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(TeacherFeatureProvider provider, ThemeData theme) {
    final data = provider.dailyOverview;
    if (data == null) return const Center(child: Text('暂无数据'));
    final classes = (data['classes'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final pendingTasks = (data['pending_tasks'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final alerts = (data['alerts'] as List?)?.cast<String>() ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              gradient: LinearGradient(colors: [theme.colorScheme.primary.withOpacity(0.1), theme.colorScheme.secondary.withOpacity(0.05)]),
            ),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.school, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('今日授课', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text(data['date'] ?? '', style: theme.textTheme.bodySmall),
              const SizedBox(height: 4),
              Text('共 ${classes.length} 节课 · ${pendingTasks.length} 项待办', style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        if (classes.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('课程安排', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          ...classes.map((c) => Card(
            margin: const EdgeInsets.only(bottom: 8),
            child: ListTile(
              leading: Icon(Icons.book, color: theme.colorScheme.secondary),
              title: Text('${c['course'] ?? ''}（${c['class_name'] ?? ''}）'),
              subtitle: Text('${c['time'] ?? ''} | ${c['room'] ?? ''}'),
              trailing: Text('${c['students'] ?? 0} 人', style: theme.textTheme.bodySmall),
            ),
          )),
        ],
        if (pendingTasks.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('待办事项', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          ...pendingTasks.map((t) => Card(
            margin: const EdgeInsets.only(bottom: 8),
            child: ListTile(
              leading: Icon(Icons.task_alt, color: theme.colorScheme.tertiary),
              title: Text(t['task'] ?? ''),
              subtitle: Text('${t['count'] ?? ''} 项 · 截止 ${t['deadline'] ?? ''}', style: theme.textTheme.bodySmall),
            ),
          )),
        ],
        if (alerts.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('提醒', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          ...alerts.map((a) => Card(
            color: theme.colorScheme.errorContainer.withOpacity(0.4),
            child: ListTile(
              leading: const Icon(Icons.notifications_active, color: Colors.red, size: 20),
              title: Text(a, style: const TextStyle(fontSize: 14)),
            ),
          )),
        ],
      ],
    );
  }
}
