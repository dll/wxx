import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/admin_provider.dart';
import '../../widgets/error_view.dart';

/// 质量看板页面（college_admin 及以上可访问）
class AdminMetricsPage extends StatefulWidget {
  const AdminMetricsPage({super.key});

  @override
  State<AdminMetricsPage> createState() => _AdminMetricsPageState();
}

class _AdminMetricsPageState extends State<AdminMetricsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchMetrics();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('质量看板')),
      body: Consumer<AdminProvider>(
        builder: (_, provider, __) {
          if (provider.metricsLoading && provider.metrics == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.metrics == null) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchMetrics(),
            );
          }
          final m = provider.metrics;
          if (m == null) {
            return ErrorView.empty(message: '暂无数据');
          }
          return RefreshIndicator(
            onRefresh: () => provider.fetchMetrics(),
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _MetricGrid(children: [
                  _MetricCard(
                    label: '问答命中率',
                    value: '${(m.hitRate * 100).toStringAsFixed(1)}%',
                    icon: Icons.check_circle_outline,
                    color: Colors.green,
                  ),
                  _MetricCard(
                    label: '兜底率',
                    value: '${(m.fallbackRate * 100).toStringAsFixed(1)}%',
                    icon: Icons.help_outline,
                    color: Colors.orange,
                  ),
                  _MetricCard(
                    label: '引用覆盖率',
                    value: '${(m.sourceCoverage * 100).toStringAsFixed(1)}%',
                    icon: Icons.link,
                    color: Colors.blue,
                  ),
                  _MetricCard(
                    label: 'P95 延迟',
                    value: '${m.p95LatencyMs} ms',
                    icon: Icons.speed,
                    color: Colors.purple,
                  ),
                ]),
                const SizedBox(height: 16),
                _MetricGrid(children: [
                  _MetricCard(
                    label: '总提问数',
                    value: '${m.totalQuestions}',
                    icon: Icons.question_answer_outlined,
                    color: Colors.teal,
                  ),
                  _MetricCard(
                    label: '总会话数',
                    value: '${m.totalSessions}',
                    icon: Icons.forum_outlined,
                    color: Colors.indigo,
                  ),
                  _MetricCard(
                    label: '今日活跃用户',
                    value: '${m.activeUsersToday}',
                    icon: Icons.people_outline,
                    color: Colors.deepOrange,
                  ),
                ]),
                const SizedBox(height: 20),
                _buildFallbackGovernance(context, provider),
              ],
            ),
          );
        },
      ),
    );
  }

  /// 高频兜底问题（知识治理）：展示真实兜底问题，提示补录知识库压降兜底率
  Widget _buildFallbackGovernance(BuildContext context, AdminProvider provider) {
    final theme = Theme.of(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (provider.fallbackQuestions.isEmpty && !provider.fallbackLoading) {
        provider.fetchFallbackQuestions();
      }
    });
    final qs = provider.fallbackQuestions;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(children: [
          Icon(Icons.contact_support_outlined, color: theme.colorScheme.tertiary),
          const SizedBox(width: 8),
          Text('高频兜底问题（知识治理）',
              style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
        ]),
        const SizedBox(height: 4),
        Text('以下问题命中知识库失败（兜底），建议补录进知识库以压降兜底率。',
            style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
        const SizedBox(height: 10),
        if (provider.fallbackLoading)
          const Padding(padding: EdgeInsets.all(16), child: Center(child: CircularProgressIndicator()))
        else if (qs.isEmpty)
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(children: [
              const Icon(Icons.check_circle_outline, color: Colors.green),
              const SizedBox(width: 8),
              Expanded(
                child: Text('暂无兜底问题记录。（若有问答数据，此处会显示高频未命中问题以便治理）',
                    style: theme.textTheme.bodySmall),
              ),
            ]),
          )
        else
          ...qs.asMap().entries.map((e) => Card(
                margin: const EdgeInsets.only(bottom: 6),
                child: ListTile(
                  dense: true,
                  leading: CircleAvatar(
                    radius: 13,
                    child: Text('${e.key + 1}', style: const TextStyle(fontSize: 12)),
                  ),
                  title: Text(e.value['question'] ?? '', maxLines: 1, overflow: TextOverflow.ellipsis),
                  trailing: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: Colors.orange.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text('兜底 ${e.value['count']} 次',
                        style: const TextStyle(fontSize: 12, color: Colors.orange, fontWeight: FontWeight.w600)),
                  ),
                ),
              )),
      ],
    );
  }
}

class _MetricGrid extends StatelessWidget {
  final List<Widget> children;
  const _MetricGrid({required this.children});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (_, constraints) {
        final crossAxisCount = constraints.maxWidth > 600 ? 4 : 2;
        return GridView.extent(
          maxCrossAxisExtent: constraints.maxWidth / crossAxisCount,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          children: children,
        );
      },
    );
  }
}

class _MetricCard extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;
  final Color color;
  const _MetricCard({
    required this.label,
    required this.value,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 28, color: color),
            const SizedBox(height: 8),
            Text(value,
                style: theme.textTheme.headlineSmall
                    ?.copyWith(fontWeight: FontWeight.bold, color: color)),
            const SizedBox(height: 4),
            Text(label,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }
}
