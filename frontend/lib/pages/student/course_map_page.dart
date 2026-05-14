import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class CourseMapPage extends StatefulWidget {
  const CourseMapPage({super.key});
  @override
  State<CourseMapPage> createState() => _CourseMapPageState();
}

class _CourseMapPageState extends State<CourseMapPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchCourseMap();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('课程地图')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchCourseMap(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchCourseMap())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final nodes = provider.courseNodes;
    if (nodes.isEmpty) return const Center(child: Text('暂无数据'));
    final semesters = <int, List>{};
    for (final n in nodes) {
      semesters.putIfAbsent(n.semester, () => []).add(n);
    }
    final sortedKeys = semesters.keys.toList()..sort();
    return ListView(
      padding: const EdgeInsets.all(16),
      children: sortedKeys.map((sem) {
        final courses = semesters[sem]!;
        return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text('第 $sem 学期', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          ),
          ...courses.map((c) => Card(
            child: ListTile(
              leading: Icon(
                c.status == 'completed' ? Icons.check_circle : c.status == 'current' ? Icons.play_circle : Icons.circle_outlined,
                color: c.status == 'completed' ? Colors.green : c.status == 'current' ? Colors.blue : Colors.grey,
              ),
              title: Text(c.name),
              subtitle: Text('${c.credits} 学分 · ${c.statusLabel}'),
              trailing: c.category.isNotEmpty ? Chip(label: Text(c.category, style: const TextStyle(fontSize: 11)), visualDensity: VisualDensity.compact) : null,
            ),
          )),
          const SizedBox(height: 8),
        ]);
      }).toList(),
    );
  }
}
