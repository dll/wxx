import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';

/// 教师 - 学生学习风格分布
class StyleDistPage extends StatefulWidget {
  const StyleDistPage({super.key});
  @override
  State<StyleDistPage> createState() => _StyleDistPageState();
}

class _StyleDistPageState extends State<StyleDistPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TeacherFeatureProvider>().fetchStyleDistribution();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('学习风格分布')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(TeacherFeatureProvider provider, ThemeData theme) {
    final data = provider.styleDist;
    if (data == null) return const Center(child: Text('暂无数据'));
    final distribution = (data['distribution'] as Map?)?.cast<String, int>() ?? {};
    final suggestions = (data['suggestions'] as List?)?.cast<String>() ?? [];
    final total = distribution.values.fold<int>(0, (s, v) => s + v);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.pie_chart, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('学习风格分布', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 4),
              Text('${data['course_name'] ?? ''} · 共 ${data['total'] ?? total} 人', style: theme.textTheme.bodySmall),
              const SizedBox(height: 16),
              if (distribution.isEmpty)
                const Text('暂无风格数据')
              else
                ...distribution.entries.map((e) {
                  final ratio = total > 0 ? e.value / total : 0.0;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                        Text(e.key, style: theme.textTheme.bodyMedium),
                        Text('${e.value} 人（${(ratio * 100).toInt()}%）', style: theme.textTheme.bodySmall),
                      ]),
                      const SizedBox(height: 4),
                      LinearProgressIndicator(value: ratio, minHeight: 8, borderRadius: BorderRadius.circular(4)),
                    ]),
                  );
                }),
              if (suggestions.isNotEmpty) ...[
                const Divider(height: 24),
                Text('教学建议', style: theme.textTheme.titleSmall),
                const SizedBox(height: 8),
                ...suggestions.map((s) => Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        const Text('•  '),
                        Expanded(child: Text(s, style: theme.textTheme.bodyMedium)),
                      ]),
                    )),
              ],
            ]),
          ),
        ),
      ],
    );
  }
}
