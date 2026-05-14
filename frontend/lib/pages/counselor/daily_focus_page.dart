import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - AI 今日关注
class DailyFocusPage extends StatefulWidget {
  const DailyFocusPage({super.key});
  @override
  State<DailyFocusPage> createState() => _DailyFocusPageState();
}

class _DailyFocusPageState extends State<DailyFocusPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchDailyFocus();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 今日关注')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final data = provider.dailyFocus;
    if (data == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              gradient: LinearGradient(colors: [theme.colorScheme.primary.withOpacity(0.1), theme.colorScheme.secondary.withOpacity(0.05)]),
            ),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.visibility, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('今日关注摘要', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text('班级健康指数: ${data.classHealthScore.toStringAsFixed(1)}', style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        Text('重点关注学生', style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        ...data.topStudents.map((s) => Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: CircleAvatar(child: Text(s.name.isNotEmpty ? s.name[0] : '?')),
            title: Text(s.name),
            subtitle: Text(s.reason),
            trailing: Chip(label: Text(s.riskLevel, style: const TextStyle(fontSize: 12))),
          ),
        )),
      ],
    );
  }
}
