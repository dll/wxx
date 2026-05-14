import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';

/// 教师 - 班级学情热力图
class HeatmapPage extends StatefulWidget {
  const HeatmapPage({super.key});
  @override
  State<HeatmapPage> createState() => _HeatmapPageState();
}

class _HeatmapPageState extends State<HeatmapPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TeacherFeatureProvider>().fetchHeatmap();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('班级学情热力图')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(TeacherFeatureProvider provider, ThemeData theme) {
    final data = provider.heatmap;
    if (data == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.grid_on, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('${data.courseName} 学情热力图', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text('共 ${data.totalStudents} 名学生，${data.anomalyCount} 人异常', style: theme.textTheme.bodyMedium),
              const SizedBox(height: 16),
              ...data.points.map((p) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                    Text(p.name, style: theme.textTheme.bodyMedium),
                    Text('${(p.mastery * 100).toInt()}%', style: theme.textTheme.bodySmall),
                  ]),
                  const SizedBox(height: 4),
                  LinearProgressIndicator(value: p.mastery, minHeight: 8, borderRadius: BorderRadius.circular(4)),
                ]),
              )),
              if (data.weakTopFive.isNotEmpty) ...[
                const Divider(height: 24),
                Text('薄弱知识点 Top5', style: theme.textTheme.titleSmall),
                const SizedBox(height: 8),
                Wrap(spacing: 8, runSpacing: 8, children: data.weakTopFive.map((w) => Chip(label: Text(w))).toList()),
              ],
            ]),
          ),
        ),
      ],
    );
  }
}
