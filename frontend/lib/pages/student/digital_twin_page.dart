import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class DigitalTwinPage extends StatefulWidget {
  const DigitalTwinPage({super.key});
  @override
  State<DigitalTwinPage> createState() => _DigitalTwinPageState();
}

class _DigitalTwinPageState extends State<DigitalTwinPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchDigitalTwin();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('数字孪生画像')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchDigitalTwin(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchDigitalTwin())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final t = provider.twin;
    if (t == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        if (t.dimensions.isNotEmpty) ...[
          Text('能力维度', style: theme.textTheme.titleMedium),
          const SizedBox(height: 12),
          ...t.dimensions.map((d) {
            final normalized = d.score > 1 ? d.score / 100.0 : d.score;
            return Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                  Text(d.name),
                  Text(d.label.isNotEmpty ? d.label : '${(normalized * 100).toInt()}%', style: theme.textTheme.bodySmall),
                ]),
                const SizedBox(height: 4),
                LinearProgressIndicator(
                  value: normalized.clamp(0.0, 1.0),
                  backgroundColor: theme.colorScheme.surfaceContainerHighest,
                  color: normalized >= 0.8 ? Colors.green : normalized >= 0.5 ? Colors.orange : Colors.red,
                ),
              ]),
            );
          }),
        ],
        if (t.aiSummary.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            color: theme.colorScheme.primaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Icon(Icons.psychology, color: theme.colorScheme.onPrimaryContainer),
                  const SizedBox(width: 8),
                  Text('AI 分析', style: theme.textTheme.titleSmall),
                ]),
                const SizedBox(height: 8),
                Text(t.aiSummary),
              ]),
            ),
          ),
        ],
        if (t.suggestions.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('成长建议', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...t.suggestions.asMap().entries.map((e) => Card(
            child: ListTile(
              leading: CircleAvatar(backgroundColor: theme.colorScheme.secondaryContainer, child: Text('${e.key + 1}')),
              title: Text(e.value),
            ),
          )),
        ],
      ],
    );
  }
}
