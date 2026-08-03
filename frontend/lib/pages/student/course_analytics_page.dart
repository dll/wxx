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
    final summary = provider.courseAnalyticsSummary;
    if (list.isEmpty) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        if (summary != null) ...[
          _buildSummaryCard(theme, summary),
          const SizedBox(height: 12),
        ],
        ...list.map((c) => Card(
              margin: const EdgeInsets.only(bottom: 12),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                    Text(c.courseName, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
                    if (c.gradeLevel.isNotEmpty)
                      Chip(label: Text(c.gradeLevel, style: const TextStyle(fontSize: 11)), visualDensity: VisualDensity.compact),
                  ]),
                  if (c.semester.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text('${c.semester} · ${c.credits} 学分', style: theme.textTheme.bodySmall),
                  ],
                  const SizedBox(height: 8),
                  Row(children: [
                    Text('成绩', style: theme.textTheme.bodySmall),
                    const SizedBox(width: 8),
                    Expanded(
                        child: LinearProgressIndicator(
                            value: c.score.clamp(0.0, 100.0) / 100.0,
                            minHeight: 6,
                            borderRadius: BorderRadius.circular(3),
                            color: c.passed ? Colors.green : Colors.red)),
                    const SizedBox(width: 8),
                    Text('${c.score.toStringAsFixed(0)}分', style: theme.textTheme.bodySmall),
                  ]),
                  if (c.gpa > 0) ...[
                    const SizedBox(height: 4),
                    Text('GPA：${c.gpa.toStringAsFixed(2)}', style: theme.textTheme.bodySmall),
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
            )),
      ],
    );
  }

  Widget _buildSummaryCard(ThemeData theme, Map<String, dynamic> summary) {
    final overall = (summary['overall_gpa'] ?? 0).toDouble();
    final classAvg = (summary['class_avg_gpa'] ?? 0).toDouble();
    final advice = (summary['advice'] ?? '').toString();
    final weakCourses = (summary['weak_courses'] as List?)?.cast<String>() ?? [];
    return Card(
      color: theme.colorScheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(Icons.insights, color: theme.colorScheme.onPrimaryContainer, size: 20),
            const SizedBox(width: 6),
            Text('学情概览', style: theme.textTheme.titleSmall),
          ]),
          const SizedBox(height: 8),
          Text('整体 GPA：${overall.toStringAsFixed(2)}${classAvg > 0 ? '（班级均值 ${classAvg.toStringAsFixed(2)}）' : ''}',
              style: theme.textTheme.bodyMedium),
          if (weakCourses.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text('薄弱课程：${weakCourses.join('、')}', style: theme.textTheme.bodySmall),
          ],
          if (advice.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(advice, style: theme.textTheme.bodyMedium),
          ],
        ]),
      ),
    );
  }
}
