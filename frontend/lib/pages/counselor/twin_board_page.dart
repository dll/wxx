import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';
import '../../widgets/md_text.dart';

/// 辅导员 - 学生数字孪生看板
/// 后端 TwinBoardStudent 字段：student_id/name/academic/social/mental/practice/innovate/risk
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
        final risk = (item['risk'] ?? '').toString();
        final dims = <String, double>{
          '学业': _num(item['academic']),
          '社交': _num(item['social']),
          '心理': _num(item['mental']),
          '实践': _num(item['practice']),
          '创新': _num(item['innovate']),
        };
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
                  Text('学号 ${item['student_id'] ?? ''}', style: theme.textTheme.bodySmall),
                ])),
                if (risk.isNotEmpty) _riskChip(theme, risk),
              ]),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: dims.entries
                    .map((e) => _dimBadge(theme, e.key, e.value))
                    .toList(),
              ),
              const SizedBox(height: 12),
              if (item['summary'] != null) MdText(item['summary'], style: theme.textTheme.bodyMedium),
            ]),
          ),
        );
      },
    );
  }

  double _num(dynamic v) => (v is num) ? v.toDouble() : 0;

  Widget _riskChip(ThemeData theme, String risk) {
    final (color, label) = switch (risk) {
      'high' => (theme.colorScheme.error, '高风险'),
      'medium' => (Colors.orange, '中风险'),
      _ => (Colors.green, '低风险'),
    };
    return Chip(
      label: Text(label, style: TextStyle(fontSize: 11, color: color)),
      backgroundColor: color.withOpacity( 0.12),
      side: BorderSide(color: color.withOpacity( 0.5)),
    );
  }

  Widget _dimBadge(ThemeData theme, String label, double score) {
    final color = score >= 70
        ? Colors.green
        : score >= 50
            ? Colors.orange
            : theme.colorScheme.error;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withOpacity( 0.1),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color.withOpacity( 0.35)),
      ),
      child: Text(
        '$label $score',
        style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: color),
      ),
    );
  }
}
