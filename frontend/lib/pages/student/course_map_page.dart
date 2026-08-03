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
    final graph = provider.courseGraph;
    if (graph != null) {
      return _buildGraphContent(theme, graph);
    }
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

  Widget _buildGraphContent(ThemeData theme, Map<String, dynamic> graph) {
    final courseName = (graph['course_name'] ?? '课程知识图谱').toString();
    final nodes = (graph['nodes'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final edges = (graph['edges'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final nameById = {for (final n in nodes) n['id']: n['name'] ?? ''};
    if (nodes.isEmpty) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              colors: [theme.colorScheme.primary, theme.colorScheme.primary.withOpacity(0.7)],
            ),
            borderRadius: BorderRadius.circular(16),
          ),
          child: Row(children: [
            Icon(Icons.account_tree, color: theme.colorScheme.onPrimary, size: 32),
            const SizedBox(width: 16),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(courseName, style: TextStyle(color: theme.colorScheme.onPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Text('${nodes.length} 个知识点 · ${edges.length} 条关联', style: TextStyle(color: theme.colorScheme.onPrimary.withOpacity(0.8), fontSize: 13)),
            ])),
          ]),
        ),
        const SizedBox(height: 16),
        Text('知识点掌握度', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        ...nodes.map((n) {
          final mastery = (n['mastery'] ?? 0).toDouble();
          return Card(
            margin: const EdgeInsets.only(bottom: 8),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                  Text(n['name'] ?? '', style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500)),
                  Text('${(mastery * 100).toInt()}%', style: theme.textTheme.bodySmall),
                ]),
                const SizedBox(height: 6),
                LinearProgressIndicator(
                  value: mastery.clamp(0.0, 1.0),
                  minHeight: 5,
                  borderRadius: BorderRadius.circular(3),
                  color: mastery >= 0.8 ? Colors.green : mastery >= 0.5 ? Colors.orange : Colors.red,
                ),
              ]),
            ),
          );
        }),
        if (edges.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('知识点关联', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Wrap(
                spacing: 8,
                runSpacing: 6,
                children: edges.map((e) => Chip(
                  avatar: const Icon(Icons.link, size: 14),
                  label: Text('${nameById[e['from']] ?? e['from']} → ${nameById[e['to']] ?? e['to']}',
                      style: const TextStyle(fontSize: 11)),
                  visualDensity: VisualDensity.compact,
                )).toList(),
              ),
            ),
          ),
        ],
      ],
    );
  }
}
