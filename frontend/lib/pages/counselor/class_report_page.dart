import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - 班级学情日报
class ClassReportPage extends StatefulWidget {
  const ClassReportPage({super.key});
  @override
  State<ClassReportPage> createState() => _ClassReportPageState();
}

class _ClassReportPageState extends State<ClassReportPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchClassReport();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('班级学情日报')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final data = provider.classReport;
    if (data == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.assessment, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('日报概览', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text(data.aiNarrative, style: theme.textTheme.bodyMedium),
              const SizedBox(height: 16),
              Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                _buildStat('出勤率', '${(data.checkinRate * 100).toInt()}%', theme),
                _buildStat('作业提交', '${(data.homeworkRate * 100).toInt()}%', theme),
                _buildStat('预警人数', '${data.emotionAlertCount}', theme),
              ]),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        if (data.anomalies.isNotEmpty) ...[Text('异常情况', style: theme.textTheme.titleSmall), const SizedBox(height: 8), ...data.anomalies.map((a) => Card(child: ListTile(leading: const Icon(Icons.info_outline), title: Text(a))))],
      ],
    );
  }

  Widget _buildStat(String label, String value, ThemeData theme) {
    return Column(children: [
      Text(value, style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold, color: theme.colorScheme.primary)),
      const SizedBox(height: 4),
      Text(label, style: theme.textTheme.bodySmall),
    ]);
  }
}
