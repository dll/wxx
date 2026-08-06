import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';
import '../../widgets/md_text.dart';

/// 辅导员 - 班级性格画像
class ClassProfilePage extends StatefulWidget {
  const ClassProfilePage({super.key});
  @override
  State<ClassProfilePage> createState() => _ClassProfilePageState();
}

class _ClassProfilePageState extends State<ClassProfilePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchClassProfile();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('班级性格画像')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final data = provider.classProfile;
    if (data == null) return const Center(child: Text('暂无数据'));
    final distribution = (data['distribution'] as Map?)?.cast<String, int>() ?? {};
    final characteristics = (data['characteristics'] as List?)?.cast<String>() ?? [];
    final suggestions = (data['suggestions'] as List?)?.cast<String>() ?? [];
    final total = distribution.values.fold<int>(0, (s, v) => s + v);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              gradient: LinearGradient(colors: [theme.colorScheme.tertiary.withOpacity(0.1), theme.colorScheme.primary.withOpacity(0.05)]),
            ),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.groups, color: theme.colorScheme.tertiary),
                const SizedBox(width: 8),
                Text('${data['class_name'] ?? '班级'} 性格画像', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text('共 ${data['total'] ?? total} 名学生', style: theme.textTheme.bodySmall),
            ]),
          ),
        ),
        if (distribution.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('性格分布', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(children: distribution.entries.map((e) {
                final count = e.value;
                final ratio = total > 0 ? count / total : 0.0;
                return Padding(
                  padding: const EdgeInsets.only(bottom: 10),
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                      Text(e.key, style: theme.textTheme.bodyMedium),
                      Text('$count 人（${(ratio * 100).toInt()}%）', style: theme.textTheme.bodySmall),
                    ]),
                    const SizedBox(height: 4),
                    LinearProgressIndicator(value: ratio, minHeight: 7, borderRadius: BorderRadius.circular(4)),
                  ]),
                );
              }).toList()),
            ),
          ),
        ],
        if (characteristics.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('班级特点', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: characteristics.map((t) => Chip(label: Text(t))).toList(),
          ),
        ],
        if (suggestions.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('管理建议', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...suggestions.asMap().entries.map((e) => Card(
                child: ListTile(
                  leading: CircleAvatar(
                    backgroundColor: theme.colorScheme.secondaryContainer,
                    child: Text('${e.key + 1}'),
                  ),
                  title: MdText(e.value),
                ),
              )),
        ],
      ],
    );
  }
}
