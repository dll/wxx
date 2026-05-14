import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class WeeklyReportPage extends StatefulWidget {
  const WeeklyReportPage({super.key});
  @override
  State<WeeklyReportPage> createState() => _WeeklyReportPageState();
}

class _WeeklyReportPageState extends State<WeeklyReportPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchWeeklyReport();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('学习周报')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchWeeklyReport(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchWeeklyReport())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final r = provider.weeklyReport;
    if (r == null || r.isEmpty) return const Center(child: Text('暂无数据'));
    final week = r['week'] as String? ?? '';
    final totalMinutes = r['total_minutes'] as int? ?? 0;
    final highlights = (r['highlights'] as List?)?.cast<String>() ?? [];
    final improvements = (r['improvements'] as List?)?.cast<String>() ?? [];
    final nextWeekGoals = (r['next_week_goals'] as List?)?.cast<String>() ?? [];
    final summary = r['summary'] as String? ?? '';
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              if (week.isNotEmpty) Text(week, style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              Row(children: [
                const Icon(Icons.timer_outlined),
                const SizedBox(width: 8),
                Text('本周学习 $totalMinutes 分钟', style: theme.textTheme.titleMedium),
              ]),
            ]),
          ),
        ),
        if (highlights.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('本周亮点', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...highlights.map((h) => ListTile(
            leading: const Icon(Icons.star, color: Colors.amber, size: 20),
            title: Text(h),
            dense: true,
          )),
        ],
        if (improvements.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('待改进', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...improvements.map((i) => ListTile(
            leading: const Icon(Icons.trending_up, color: Colors.orange, size: 20),
            title: Text(i),
            dense: true,
          )),
        ],
        if (nextWeekGoals.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('下周目标', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...nextWeekGoals.map((g) => ListTile(
            leading: const Icon(Icons.flag, color: Colors.green, size: 20),
            title: Text(g),
            dense: true,
          )),
        ],
        if (summary.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('AI 总结', style: theme.textTheme.titleSmall),
              const SizedBox(height: 8),
              Text(summary),
            ]),
          )),
        ],
      ],
    );
  }
}
