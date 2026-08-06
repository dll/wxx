import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/md_text.dart';

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
    final mbtiType = p['type'] as String? ?? '';
    final label = p['label'] as String? ?? '';
    final description = p['description'] as String? ?? '';
    final strengths = (p['strengths'] as List?)?.cast<String>() ?? [];
    final weaknesses = (p['weaknesses'] as List?)?.cast<String>() ?? [];
    final careers = (p['career_suggestions'] as List?)?.cast<String>() ?? [];
    final learningStyle = p['learning_style'] as String? ?? '';
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        if (mbtiType.isNotEmpty) Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(children: [
              Text(mbtiType, style: theme.textTheme.displaySmall?.copyWith(fontWeight: FontWeight.bold, color: theme.colorScheme.onPrimaryContainer)),
              if (label.isNotEmpty) ...[const SizedBox(height: 4), Text(label, style: theme.textTheme.titleMedium)],
              if (description.isNotEmpty) ...[const SizedBox(height: 8), MdText(description, style: theme.textTheme.bodyMedium, inline: true), ],
            ]),
          ),
        ),
        if (strengths.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('优势特质', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 8, children: strengths.map((t) => Chip(label: Text(t), avatar: const Icon(Icons.star, size: 16, color: Colors.amber))).toList()),
        ],
        if (weaknesses.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('注意事项', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 8, children: weaknesses.map((t) => Chip(label: Text(t), avatar: const Icon(Icons.info_outline, size: 16))).toList()),
        ],
        if (careers.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('职业建议', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...careers.map((s) => ListTile(leading: const Icon(Icons.work_outline, color: Colors.blue), title: Text(s), dense: true)),
        ],
        if (learningStyle.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('学习风格', style: theme.textTheme.titleSmall),
              const SizedBox(height: 8),
              MdText(learningStyle),
            ]),
          )),
        ],
      ],
    );
  }
}
