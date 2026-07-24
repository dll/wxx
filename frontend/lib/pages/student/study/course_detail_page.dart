import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/education_provider.dart';
import '../../../widgets/error_view.dart';

class CourseDetailPage extends StatefulWidget {
  final String courseId;
  const CourseDetailPage({super.key, required this.courseId});

  @override
  State<CourseDetailPage> createState() => _CourseDetailPageState();
}

class _CourseDetailPageState extends State<CourseDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudyProvider>().fetchCourseDetail(widget.courseId);
    });
  }

  @override
  void dispose() {
    context.read<StudyProvider>().clearDetail();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyProvider>();
    final course = provider.courseDetail;

    return Scaffold(
      appBar: AppBar(title: const Text('课程详情')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && course == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchCourseDetail(widget.courseId),
                )
              : course == null
                  ? ErrorView.empty(
                      message: '课程不存在',
                      icon: Icons.menu_book_outlined,
                    )
                  : _buildDetail(theme, course),
    );
  }

  Widget _buildDetail(ThemeData theme, Map<String, dynamic> course) {
    final name = course['name'] as String? ?? course['title'] as String? ?? '';
    final teacher = course['teacher'] as String? ?? course['instructor'] as String? ?? '';
    final credit = course['credit'] as num? ?? 0;
    final description = course['description'] as String? ?? course['intro'] as String? ?? '';
    final outline = course['outline'] as String? ?? '';
    final schedule = course['schedule'] as String? ?? '';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    name,
                    style: theme.textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Icon(Icons.person_outline, size: 18, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 6),
                      Text(teacher, style: theme.textTheme.bodyMedium),
                      const SizedBox(width: 20),
                      Icon(Icons.credit_card_outlined, size: 18, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 6),
                      Text('${credit}学分', style: theme.textTheme.bodyMedium),
                    ],
                  ),
                  if (schedule.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        Icon(Icons.access_time, size: 18, color: theme.colorScheme.onSurfaceVariant),
                        const SizedBox(width: 6),
                        Text(schedule, style: theme.textTheme.bodyMedium),
                      ],
                    ),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          if (description.isNotEmpty) ...[
            Text('课程简介', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(description, style: theme.textTheme.bodyMedium),
              ),
            ),
            const SizedBox(height: 16),
          ],
          if (outline.isNotEmpty) ...[
            Text('课程大纲', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(outline, style: theme.textTheme.bodyMedium),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
