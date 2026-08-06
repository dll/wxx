import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';
import '../../widgets/md_text.dart';

/// 辅导员 - 学生思想档案
class CounselorIdeologicalPage extends StatefulWidget {
  const CounselorIdeologicalPage({super.key});
  @override
  State<CounselorIdeologicalPage> createState() => _CounselorIdeologicalPageState();
}

class _CounselorIdeologicalPageState extends State<CounselorIdeologicalPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchIdeological();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('学生思想档案')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final data = provider.ideologicalData;
    if (data == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.psychology, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('思想动态总览', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              MdText(data['summary'] ?? '暂无总结', style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
      ],
    );
  }
}
