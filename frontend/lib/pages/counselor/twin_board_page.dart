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
              ? Center(
                  child: Text(provider.error,
                      style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final list = [...provider.twinBoard]
      ..sort((a, b) => _riskRank(b['risk']) - _riskRank(a['risk']));
    if (list.isEmpty) return _buildEmpty(theme);
    final highRisk = list.where((e) => e['risk'] == 'high').length;
    final attention = list.where((e) => e['risk'] == 'medium').length;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 28),
      children: [
        _buildHeader(theme, list.length, highRisk, attention),
        const SizedBox(height: 12),
        ...list.map((item) {
          final risk = (item['risk'] ?? '').toString();
          final dims = <String, double>{
            '学业': _num(item['academic']),
            '社交': _num(item['social']),
            '心理': _num(item['mental']),
            '实践': _num(item['practice']),
            '创新': _num(item['innovate']),
          };
          return Card(
            margin: const EdgeInsets.only(bottom: 10),
            elevation: 0,
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(18),
                side: BorderSide(color: theme.colorScheme.outlineVariant)),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      CircleAvatar(
                          child: Text(
                              item['name']?.toString().isNotEmpty == true
                                  ? item['name'][0]
                                  : '?')),
                      const SizedBox(width: 12),
                      Expanded(
                          child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                            Text(item['name'] ?? '未知',
                                style: theme.textTheme.titleSmall),
                            Text('学号 ${item['student_id'] ?? ''}',
                                style: theme.textTheme.bodySmall),
                          ])),
                      if (risk.isNotEmpty) _riskChip(theme, risk),
                    ]),
                    const SizedBox(height: 12),
                    ...dims.entries
                        .map((e) => _dimensionRow(theme, e.key, e.value)),
                    const SizedBox(height: 12),
                    if (item['summary'] != null)
                      MdText(item['summary'],
                          style: theme.textTheme.bodyMedium),
                  ]),
            ),
          );
        }),
      ],
    );
  }

  Widget _buildHeader(ThemeData theme, int total, int highRisk, int attention) {
    final cs = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(22),
        gradient: LinearGradient(colors: [cs.primaryContainer, cs.surface]),
        border: Border.all(color: cs.outlineVariant),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('学生成长看板',
            style: theme.textTheme.titleLarge
                ?.copyWith(fontWeight: FontWeight.w800)),
        const SizedBox(height: 4),
        Text('优先关注需要行动的学生，画像仅基于已有真实记录。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: cs.onSurfaceVariant)),
        const SizedBox(height: 14),
        Row(children: [
          _summaryMetric(theme, '$total', '有画像学生', cs.primary),
          _summaryMetric(theme, '$highRisk', '高风险', cs.error),
          _summaryMetric(theme, '$attention', '需关注', Colors.orange),
        ]),
      ]),
    );
  }

  Widget _summaryMetric(
      ThemeData theme, String value, String label, Color color) {
    return Expanded(
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text(value,
          style: theme.textTheme.headlineSmall
              ?.copyWith(fontWeight: FontWeight.w800, color: color)),
      Text(label,
          style: theme.textTheme.labelSmall
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
    ]));
  }

  Widget _buildEmpty(ThemeData theme) => Center(
          child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(Icons.insights_outlined,
              size: 48, color: theme.colorScheme.outline),
          const SizedBox(height: 12),
          Text('还没有可展示的学生画像', style: theme.textTheme.titleMedium),
          const SizedBox(height: 6),
          Text('学生产生学习、活动或情绪记录后，这里会自动更新。',
              textAlign: TextAlign.center, style: theme.textTheme.bodySmall),
        ]),
      ));

  int _riskRank(dynamic risk) =>
      switch (risk) { 'high' => 3, 'medium' => 2, _ => 1 };

  Widget _dimensionRow(ThemeData theme, String label, double score) {
    final color = score >= 70
        ? Colors.green
        : score >= 50
            ? Colors.orange
            : theme.colorScheme.error;
    return Padding(
      padding: const EdgeInsets.only(bottom: 7),
      child: Row(children: [
        SizedBox(
            width: 36, child: Text(label, style: theme.textTheme.labelSmall)),
        Expanded(
            child: ClipRRect(
                borderRadius: BorderRadius.circular(99),
                child: LinearProgressIndicator(
                    value: (score / 100).clamp(0, 1),
                    minHeight: 6,
                    color: color,
                    backgroundColor: color.withOpacity(0.10)))),
        const SizedBox(width: 8),
        SizedBox(
            width: 30,
            child: Text(score.toStringAsFixed(0),
                textAlign: TextAlign.right,
                style: theme.textTheme.labelSmall
                    ?.copyWith(fontWeight: FontWeight.w700))),
      ]),
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
      backgroundColor: color.withOpacity(0.12),
      side: BorderSide(color: color.withOpacity(0.5)),
    );
  }
}
