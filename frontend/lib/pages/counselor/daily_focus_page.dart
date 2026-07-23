import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';
import '../../widgets/error_view.dart';

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
      body: RefreshIndicator(
        onRefresh: () => provider.fetchDailyFocus(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty
                ? ErrorView.error(
                    message: provider.error,
                    onRetry: () => provider.fetchDailyFocus(),
                  )
                : _buildContent(provider, theme),
      ),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final data = provider.dailyFocus;
    if (data == null) return const Center(child: Text('暂无数据'));

    final score = data.classHealthScore;
    final scoreColor = score >= 80
        ? Colors.green
        : score >= 60
            ? Colors.orange
            : Colors.red;

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // 班级健康度卡片
        Card(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              gradient: LinearGradient(
                colors: [
                  theme.colorScheme.primary.withOpacity( 0.08),
                  theme.colorScheme.secondary.withOpacity( 0.04),
                ],
              ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  Icon(Icons.favorite, color: scoreColor, size: 24),
                  const SizedBox(width: 8),
                  Text('班级健康指数',
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.bold)),
                  const Spacer(),
                  Text(
                    '${score.toStringAsFixed(0)}分',
                    style: TextStyle(
                      fontSize: 28,
                      fontWeight: FontWeight.bold,
                      color: scoreColor,
                    ),
                  ),
                ]),
                const SizedBox(height: 8),
                ClipRRect(
                  borderRadius: BorderRadius.circular(4),
                  child: LinearProgressIndicator(
                    value: score / 100,
                    backgroundColor:
                        theme.colorScheme.surfaceContainerHighest,
                    color: scoreColor,
                    minHeight: 6,
                  ),
                ),
              ],
            ),
          ),
        ),

        // 告警概览
        if (data.overview.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('告警概览',
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _buildStatChip('紧急', data.overview['urgent'] ?? 0, Colors.red),
                  _buildStatChip('高风险', data.overview['high'] ?? 0, Colors.orange),
                  _buildStatChip('中风险', data.overview['medium'] ?? 0, Colors.amber),
                  _buildStatChip('低风险', data.overview['low'] ?? 0, Colors.green),
                ],
              ),
            ),
          ),
        ],

        // AI 叙事
        if (data.aiNarrative.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            color: theme.colorScheme.tertiaryContainer.withOpacity( 0.5),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(Icons.auto_awesome,
                      color: theme.colorScheme.tertiary, size: 18),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      data.aiNarrative,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onTertiaryContainer,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],

        // 重点关注学生
        const SizedBox(height: 16),
        Text('重点关注学生 TOP${data.topStudents.length}',
            style: theme.textTheme.titleSmall
                ?.copyWith(fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),
        ...data.topStudents.map((s) {
          final riskColor = s.riskLevel == 'high'
              ? Colors.red
              : s.riskLevel == 'medium'
                  ? Colors.orange
                  : Colors.green;
          return Card(
            margin: const EdgeInsets.only(bottom: 8),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(children: [
                    CircleAvatar(
                      backgroundColor:
                          theme.colorScheme.primaryContainer,
                      radius: 16,
                      child: Text(
                        s.name.isNotEmpty ? s.name[0] : '?',
                        style: TextStyle(
                          color: theme.colorScheme.onPrimaryContainer,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(s.name,
                          style: const TextStyle(
                              fontWeight: FontWeight.w600)),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 3),
                      decoration: BoxDecoration(
                        color: riskColor.withOpacity( 0.12),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        s.riskLevel == 'high'
                            ? '高风险'
                            : s.riskLevel == 'medium'
                                ? '中风险'
                                : '低风险',
                        style: TextStyle(
                          fontSize: 11,
                          color: riskColor,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ]),
                  const SizedBox(height: 8),
                  Text('原因：${s.reason}',
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant)),
                  if (s.suggestion.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Icon(Icons.lightbulb_outline,
                          size: 14,
                          color: theme.colorScheme.primary),
                      const SizedBox(width: 4),
                      Expanded(
                        child: Text('建议：${s.suggestion}',
                            style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.primary)),
                      ),
                    ]),
                  ],
                ],
              ),
            ),
          );
        }),
      ],
    );
  }

  Widget _buildStatChip(String label, int count, Color color) {
    return Column(
      children: [
        Text('$count',
            style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.bold,
                color: count > 0 ? color : Colors.grey.shade400)),
        const SizedBox(height: 2),
        Text(label,
            style: TextStyle(fontSize: 11, color: Colors.grey.shade600)),
      ],
    );
  }
}
