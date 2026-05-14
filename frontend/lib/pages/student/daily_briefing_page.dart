import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class DailyBriefingPage extends StatefulWidget {
  const DailyBriefingPage({super.key});
  @override
  State<DailyBriefingPage> createState() => _DailyBriefingPageState();
}

class _DailyBriefingPageState extends State<DailyBriefingPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchDailyBriefing();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('AI 今日速览')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchDailyBriefing(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchDailyBriefing())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final b = provider.briefing;
    if (b == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(b.greeting.isNotEmpty ? b.greeting : '今日好！', style: theme.textTheme.titleLarge),
                const SizedBox(height: 4),
                Text(b.date, style: theme.textTheme.bodySmall),
                if (b.weather.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Row(children: [const Icon(Icons.cloud, size: 16), const SizedBox(width: 4), Text(b.weather)]),
                ],
              ],
            ),
          ),
        ),
        if (b.courses.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('今日课程', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...b.courses.map((c) => ListTile(
            leading: const Icon(Icons.school),
            title: Text(c.title),
            subtitle: c.subtitle.isNotEmpty ? Text(c.subtitle) : null,
            trailing: c.time.isNotEmpty ? Text(c.time, style: theme.textTheme.bodySmall) : null,
          )),
        ],
        if (b.deadlines.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('待办事项', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...b.deadlines.map((d) => ListTile(
            leading: const Icon(Icons.assignment_late, color: Colors.orange),
            title: Text(d.title),
            subtitle: d.subtitle.isNotEmpty ? Text(d.subtitle) : null,
            trailing: d.time.isNotEmpty ? Text(d.time, style: theme.textTheme.bodySmall) : null,
          )),
        ],
        if (b.activities.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('校园活动', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...b.activities.map((a) => ListTile(
            leading: const Icon(Icons.event, color: Colors.green),
            title: Text(a.title),
            subtitle: a.subtitle.isNotEmpty ? Text(a.subtitle) : null,
          )),
        ],
        if (b.motto.isNotEmpty) ...[
          const SizedBox(height: 24),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  const Icon(Icons.format_quote, color: Colors.grey),
                  const SizedBox(width: 8),
                  Expanded(child: Text(b.motto, style: theme.textTheme.bodyMedium?.copyWith(fontStyle: FontStyle.italic))),
                ],
              ),
            ),
          ),
        ],
      ],
    );
  }
}
