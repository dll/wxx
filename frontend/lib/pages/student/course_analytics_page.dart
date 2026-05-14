import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class CourseAnalyticsPage extends StatefulWidget {
  const CourseAnalyticsPage({super.key});
  @override
  State<CourseAnalyticsPage> createState() => _CourseAnalyticsPageState();
}

class _CourseAnalyticsPageState extends State<CourseAnalyticsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchCourseAnalytics();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('课程学情')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchCourseAnalytics(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchCourseAnalytics())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final list = provider.courseAnalytics;
    if (list.isEmpty) return const Center(child: Text('暂无数据'));
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: list.length,
      itemBuilder: (_, i) {
        final c = list[i];
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                Text(c.courseName, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
                Chip(label: Text('前 ${c.rankPercentile}%', style: const TextStyle(fontSize: 11)), visualDensity: VisualDensity.compact),
              ]),
              const SizedBox(height: 8),
              Row(children: [
                Text('进度', style: theme.textTheme.bodySmall),
                const SizedBox(width: 8),
                Expanded(child: LinearProgressIndicator(value: c.progress, minHeight: 6, borderRadius: BorderRadius.circular(3))),
                const SizedBox(width: 8),
                Text('${(c.progress * 100).toInt()}%', style: theme.textTheme.bodySmall),
              ]),
              if (c.knowledgePoints.isNotEmpty) ...[
                const SizedBox(height: 12),
                Text('知识点掌握', style: theme.textTheme.bodySmall),
                const SizedBox(height: 4),
                ...c.knowledgePoints.take(5).map((kp) => Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Row(children: [
                    Expanded(flex: 2, child: Text(kp.name, style: theme.textTheme.bodySmall)),
                    Expanded(flex: 3, child: LinearProgressIndicator(
                      value: kp.mastery,
                      backgroundColor: theme.colorScheme.surfaceContainerHighest,
                      color: kp.level == 'good' ? Colors.green : kp.level == 'medium' ? Colors.orange : Colors.red,
                    )),
                  ]),
                )),
              ],
              if (c.weakPoints.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(spacing: 4, children: c.weakPoints.map((w) => Chip(
                  label: Text(w, style: const TextStyle(fontSize: 10)),
                  backgroundColor: theme.colorScheme.errorContainer,
                  visualDensity: VisualDensity.compact,
                )).toList()),
              ],
            ]),
          ),
        );
      },
    );
  }
}
