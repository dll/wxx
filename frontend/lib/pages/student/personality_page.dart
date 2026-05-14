import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';

class PersonalityInsightPage extends StatefulWidget {
  const PersonalityInsightPage({super.key});
  @override
  State<PersonalityInsightPage> createState() => _PersonalityInsightPageState();
}

class _PersonalityInsightPageState extends State<PersonalityInsightPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchPersonality();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('性格洞察')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchPersonality(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(message: provider.error, onRetry: () => provider.fetchPersonality())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final p = provider.personality;
    if (p == null || p.isEmpty) return const Center(child: Text('暂无数据'));
    final mbti = p['mbti'] as String? ?? '';
    final traits = (p['traits'] as List?)?.cast<String>() ?? [];
    final summary = p['summary'] as String? ?? '';
    final suggestions = (p['suggestions'] as List?)?.cast<String>() ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        if (mbti.isNotEmpty) Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(children: [
              Text(mbti, style: theme.textTheme.displaySmall?.copyWith(fontWeight: FontWeight.bold, color: theme.colorScheme.onPrimaryContainer)),
              const SizedBox(height: 8),
              Text('你的性格类型', style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        if (traits.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('性格特质', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 8, children: traits.map((t) => Chip(label: Text(t), avatar: const Icon(Icons.star, size: 16))).toList()),
        ],
        if (summary.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('AI 分析', style: theme.textTheme.titleSmall),
              const SizedBox(height: 8),
              Text(summary),
            ]),
          )),
        ],
        if (suggestions.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('发展建议', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...suggestions.map((s) => ListTile(leading: const Icon(Icons.lightbulb_outline, color: Colors.amber), title: Text(s))),
        ],
      ],
    );
  }
}
