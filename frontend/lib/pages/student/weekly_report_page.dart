import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/md_text.dart';

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
    final totalHours = (r['total_hours'] as num?)?.toDouble() ?? 0;
    final coursesCount = r['courses_count'] as int? ?? 0;
    final assignments = r['assignments'] as int? ?? 0;
    final rankChange = r['rank_change'] as int? ?? 0;
    final highlights = (r['highlights'] as List?)?.cast<String>() ?? [];
    final improvements = (r['improvements'] as List?)?.cast<String>() ?? [];
    final nextWeekGoals = (r['next_week_goals'] as List?)?.cast<String>() ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              if (week.isNotEmpty) Text(week, style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 12),
              Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                _metric(theme, '学习', '${totalHours.toStringAsFixed(1)}h', Icons.timer),
                _metric(theme, '课程', '$coursesCount门', Icons.school),
                _metric(theme, '作业', '$assignments份', Icons.assignment),
                _metric(theme, '排名', rankChange > 0 ? '↑$rankChange' : rankChange < 0 ? '↓${-rankChange}' : '-', Icons.trending_up),
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
            title: MdText(h), dense: true,
          )),
        ],
        if (improvements.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('待改进', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...improvements.map((i) => ListTile(
            leading: const Icon(Icons.trending_up, color: Colors.orange, size: 20),
            title: MdText(i), dense: true,
          )),
        ],
        if (nextWeekGoals.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('下周目标', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...nextWeekGoals.map((g) => ListTile(
            leading: const Icon(Icons.flag, color: Colors.green, size: 20),
            title: MdText(g), dense: true,
          )),
        ],
      ],
    );
  }

  Widget _metric(ThemeData theme, String label, String value, IconData icon) {
    return Column(children: [
      Icon(icon, size: 20, color: theme.colorScheme.onPrimaryContainer),
      const SizedBox(height: 4),
      Text(value, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
      Text(label, style: theme.textTheme.bodySmall),
    ]);
  }
}
