import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';
import '../../widgets/md_text.dart';

/// 辅导员 - 学生数字孪生看板
class TwinBoardPage extends StatefulWidget {
  const TwinBoardPage({super.key});
  @override
  State<TwinBoardPage> createState() => _TwinBoardPageState();
}

class _TwinBoardPageState extends State<TwinBoardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchTwinBoard();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('学生数字孪生看板')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final list = provider.twinBoard;
    if (list.isEmpty) return const Center(child: Text('暂无数据'));
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: list.length,
      itemBuilder: (context, index) {
        final item = list[index];
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                CircleAvatar(child: Text(item['name']?.toString().isNotEmpty == true ? item['name'][0] : '?')),
                const SizedBox(width: 12),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(item['name'] ?? '未知', style: theme.textTheme.titleSmall),
                  Text(item['class'] ?? '', style: theme.textTheme.bodySmall),
                ])),
                if (item['risk_level'] != null) Chip(label: Text(item['risk_level'], style: const TextStyle(fontSize: 11))),
              ]),
              const SizedBox(height: 12),
              if (item['summary'] != null) MdText(item['summary'], style: theme.textTheme.bodyMedium),
            ]),
          ),
        );
      },
    );
  }
}
