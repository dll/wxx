import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

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
                Text('班级画像', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text(data['summary'] ?? '暂无画像数据', style: theme.textTheme.bodyMedium),
              if (data['traits'] != null) ...[
                const SizedBox(height: 16),
                Wrap(
                  spacing: 8, runSpacing: 8,
                  children: (data['traits'] as List? ?? []).map<Widget>((t) => Chip(label: Text(t.toString()))).toList(),
                ),
              ],
            ]),
          ),
        ),
      ],
    );
  }
}
