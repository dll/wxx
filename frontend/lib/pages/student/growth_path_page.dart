import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';

class GrowthPathPage extends StatefulWidget {
  const GrowthPathPage({super.key});
  @override
  State<GrowthPathPage> createState() => _GrowthPathPageState();
}

class _GrowthPathPageState extends State<GrowthPathPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchGrowthPath();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    final gp = provider.growthPath;
    return Scaffold(
      appBar: AppBar(title: const Text('成长路径')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchGrowthPath(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Container(
                    padding: const EdgeInsets.all(20),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [theme.colorScheme.primary, theme.colorScheme.primary.withOpacity(0.7)],
                        begin: Alignment.topLeft, end: Alignment.bottomRight,
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(children: [
                      Icon(Icons.trending_up, color: theme.colorScheme.onPrimary, size: 32),
                      const SizedBox(width: 16),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('成长路径', style: TextStyle(color: theme.colorScheme.onPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
                        const SizedBox(height: 4),
                        Text('AI 个性化成长建议', style: TextStyle(color: theme.colorScheme.onPrimary.withOpacity(0.8), fontSize: 13)),
                      ])),
                    ]),
                  ),
                  const SizedBox(height: 16),
                  if (gp == null || (gp['milestones'] == null && gp['response'] == null && gp['content'] == null))
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Column(children: [
                          Icon(Icons.trending_up, size: 48, color: theme.colorScheme.primary.withOpacity(0.5)),
                          const SizedBox(height: 12),
                          Text('暂无成长路径数据，请先完善数字孪生画像', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                        ]),
                      ),
                    )
                  else if (gp['milestones'] != null)
                    ..._buildGrowthPathContent(theme, gp)
                  else
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: SelectableText(gp['response'] ?? gp['content'] ?? '', style: theme.textTheme.bodyMedium),
                      ),
                    ),
                ],
              ),
      ),
    );
  }

  List<Widget> _buildGrowthPathContent(ThemeData theme, Map<String, dynamic> gp) {
    final summary = (gp['summary'] ?? '').toString();
    final currentStage = (gp['current_stage'] ?? '').toString();
    final strongest = (gp['strongest_dim'] ?? '').toString();
    final weakest = (gp['weakest_dim'] ?? '').toString();
    final milestones = (gp['milestones'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    return [
      if (summary.isNotEmpty) ...[
        Card(
          color: theme.colorScheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.auto_awesome, color: theme.colorScheme.onPrimaryContainer, size: 18),
                const SizedBox(width: 6),
                Text('AI 成长总结', style: theme.textTheme.titleSmall),
              ]),
              const SizedBox(height: 8),
              Text(summary, style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        const SizedBox(height: 12),
      ],
      Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(children: [
            _stageChip(theme, '当前阶段', currentStage, Icons.school),
            const SizedBox(width: 12),
            _stageChip(theme, '优势维度', strongest, Icons.trending_up),
            const SizedBox(width: 12),
            _stageChip(theme, '待提升', weakest, Icons.trending_down),
          ]),
        ),
      ),
      const SizedBox(height: 16),
      Text('分阶段路线图', style: theme.textTheme.titleMedium),
      const SizedBox(height: 8),
      ...milestones.asMap().entries.map((e) {
        final m = e.value;
        return Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ExpansionTile(
            leading: CircleAvatar(
              backgroundColor: theme.colorScheme.secondaryContainer,
              child: Text('${e.key + 1}'),
            ),
            title: Text(m['stage'] ?? '', style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
            subtitle: Text(m['focus'] ?? ''),
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('关键行动：', style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.primary)),
                    const SizedBox(height: 4),
                    ...((m['actions'] as List?)?.map((a) => Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                            const Text('•  '),
                            Expanded(child: Text(a.toString(), style: theme.textTheme.bodySmall)),
                          ]),
                        )) ??
                        []),
                  ],
                ),
              ),
            ],
          ),
        );
      }),
    ];
  }

  Widget _stageChip(ThemeData theme, String label, String value, IconData icon) {
    return Expanded(
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(label, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.outline)),
        const SizedBox(height: 4),
        Row(children: [
          Icon(icon, size: 16, color: theme.colorScheme.primary),
          const SizedBox(width: 4),
          Flexible(
            child: Text(value.isEmpty ? '—' : value, style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600)),
          ),
        ]),
      ]),
    );
  }
}
