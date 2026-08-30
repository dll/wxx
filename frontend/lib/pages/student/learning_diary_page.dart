import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../widgets/feature_page_scaffold.dart';
import '../../widgets/md_text.dart';

class LearningDiaryPage extends StatefulWidget {
  const LearningDiaryPage({super.key});
  @override
  State<LearningDiaryPage> createState() => _LearningDiaryPageState();
}

class _LearningDiaryPageState extends State<LearningDiaryPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchLearningDiary();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return FeaturePageScaffold(
      title: 'AI 学习日记',
      loading: provider.loading,
      error: provider.error.isEmpty ? null : provider.error,
      onRefresh: () => provider.fetchLearningDiary(),
      contentBuilder: (_) => _buildContent(theme, provider),
    );
  }

  Widget _buildContent(ThemeData theme, StudentFeatureProvider provider) {
    final d = provider.diary;
    if (d == null) return const Center(child: Text('暂无数据'));
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          color: theme.colorScheme.secondaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(children: [
              Icon(Icons.timer, color: theme.colorScheme.onSecondaryContainer),
              const SizedBox(width: 8),
              Text('今日学习 ${d.studyMinutes} 分钟', style: theme.textTheme.titleMedium),
            ]),
          ),
        ),
        if (d.coursesStudied.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('学习课程', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 8, children: d.coursesStudied.map((c) => Chip(label: Text(c))).toList()),
        ],
        if (d.keyPoints.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('重点知识', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...d.keyPoints.map((p) => Padding(
            padding: const EdgeInsets.only(bottom: 4),
            child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
              const Text('• ', style: TextStyle(fontSize: 16)),
              Expanded(child: MdText(p)),
            ]),
          )),
        ],
        if (d.quiz.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('随堂测验', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...d.quiz.asMap().entries.map((e) => Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('Q${e.key + 1}: ${e.value.question}', style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                ...e.value.options.asMap().entries.map((o) => Padding(
                  padding: const EdgeInsets.only(left: 8, bottom: 4),
                  child: Row(children: [
                    Icon(o.key == e.value.correctIndex ? Icons.check_circle : Icons.circle_outlined, size: 16, color: o.key == e.value.correctIndex ? Colors.green : null),
                    const SizedBox(width: 8),
                    Expanded(child: Text(o.value)),
                  ]),
                )),
                if (e.value.explanation.isNotEmpty) ...[
                  const Divider(),
                  MdText(e.value.explanation, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                ],
              ]),
            ),
          )),
        ],
        if (d.tomorrowPlan.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(child: ListTile(leading: const Icon(Icons.calendar_today), title: const Text('明日计划'), subtitle: MdText(d.tomorrowPlan))),
        ],
        if (d.encouragement.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            color: theme.colorScheme.tertiaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(children: [const Icon(Icons.emoji_events), const SizedBox(width: 8), Expanded(child: MdText(d.encouragement))]),
            ),
          ),
        ],
      ],
    );
  }
}
