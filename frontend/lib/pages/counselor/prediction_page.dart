import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - 预测性预警
class PredictionPage extends StatefulWidget {
  const PredictionPage({super.key});
  @override
  State<PredictionPage> createState() => _PredictionPageState();
}

class _PredictionPageState extends State<PredictionPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchPredictions();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('预测性预警')),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final list = provider.predictions;
    if (list.isEmpty) return const Center(child: Text('暂无预警'));
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: list.length,
      itemBuilder: (context, index) {
        final item = list[index];
        final level = item['level'] ?? 'low';
        final color = level == 'high' ? Colors.red : level == 'medium' ? Colors.orange : Colors.green;
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: ListTile(
            leading: CircleAvatar(backgroundColor: color.withOpacity(0.2), child: Icon(Icons.warning_amber, color: color)),
            title: Text(item['student_name'] ?? '未知学生'),
            subtitle: Text(item['reason'] ?? ''),
            trailing: Text('${item['probability'] ?? 0}%', style: TextStyle(color: color, fontWeight: FontWeight.bold)),
          ),
        );
      },
    );
  }
}
