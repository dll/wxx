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
    final styles = data['styles'] as List? ?? [];
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
              const SizedBox(height: 16),
              if (styles.isEmpty) const Text('暂无风格数据'),
              ...styles.map((s) {
                final pct = (s['percentage'] ?? 0).toDouble() / 100;
                return Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                      Text(s['name'] ?? '', style: theme.textTheme.bodyMedium),
                      Text('${s['percentage'] ?? 0}%', style: theme.textTheme.bodySmall),
                    ]),
                    const SizedBox(height: 4),
                    LinearProgressIndicator(value: pct, minHeight: 8, borderRadius: BorderRadius.circular(4)),
                  ]),
                );
              }),
              if (data['summary'] != null) ...[
                const Divider(height: 24),
                Text(data['summary'], style: theme.textTheme.bodyMedium),
              ],
            ]),
          ),
        ),
      ],
    );
  }
}
