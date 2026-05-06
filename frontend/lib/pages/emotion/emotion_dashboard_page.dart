import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/emotion_provider.dart';
import '../../models/models.dart';
import '../../widgets/error_view.dart';

/// 情感预警仪表盘（辅导员及以上角色可访问）
class EmotionDashboardPage extends StatefulWidget {
  const EmotionDashboardPage({super.key});

  @override
  State<EmotionDashboardPage> createState() => _EmotionDashboardPageState();
}

class _EmotionDashboardPageState extends State<EmotionDashboardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final provider = context.read<EmotionProvider>();
      if (provider.alerts.isEmpty) {
        provider.loadAlerts();
      }
      provider.fetchStats();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('情感预警'),
        actions: [
          Consumer<EmotionProvider>(
            builder: (_, provider, __) {
              final pending = provider.total;
              return pending > 0
                  ? Padding(
                      padding: const EdgeInsets.only(right: 16),
                      child: Badge(
                        label: Text('$pending'),
                        child: const Icon(Icons.notifications_outlined),
                      ),
                    )
                  : const Padding(
                      padding: EdgeInsets.only(right: 16),
                      child: Icon(Icons.notifications_outlined),
                    );
            },
          ),
        ],
      ),
      body: Column(
        children: [
          _buildStatsSummary(),
          _buildFilterBar(),
          Expanded(child: _buildAlertList()),
        ],
      ),
    );
  }

  /// 过滤栏：风险等级 + 状态
  Widget _buildFilterBar() {
    return Consumer<EmotionProvider>(
      builder: (_, provider, __) {
        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context).colorScheme.outlineVariant,
                width: 0.5,
              ),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // 风险等级过滤
              Row(
                children: [
                  const Icon(Icons.warning_amber_rounded, size: 18),
                  const SizedBox(width: 8),
                  Expanded(
                    child: SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      child: Row(
                        children: [
                          _buildRiskChip('', '全部', provider),
                          _buildRiskChip('low', '低风险', provider),
                          _buildRiskChip('medium', '中风险', provider),
                          _buildRiskChip('high', '高风险', provider),
                          _buildRiskChip('urgent', '紧急', provider),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              // 状态过滤
              Row(
                children: [
                  const Icon(Icons.filter_list, size: 18),
                  const SizedBox(width: 8),
                  Expanded(
                    child: SingleChildScrollView(
                      scrollDirection: Axis.horizontal,
                      child: Row(
                        children: [
                          _buildStatusChip('', '全部', provider),
                          _buildStatusChip('pending', '待处理', provider),
                          _buildStatusChip('acknowledged', '已确认', provider),
                          _buildStatusChip('resolved', '已处理', provider),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildRiskChip(String value, String label, EmotionProvider provider) {
    final selected = provider.riskFilter == value;
    return Padding(
      padding: const EdgeInsets.only(right: 6),
      child: FilterChip(
        label: Text(label, style: const TextStyle(fontSize: 13)),
        selected: selected,
        onSelected: (_) => provider.setRiskFilter(selected ? '' : value),
        materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        visualDensity: VisualDensity.compact,
        backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
        selectedColor: _riskColor(value).withValues(alpha: 0.15),
        side: BorderSide.none,
        showCheckmark: false,
      ),
    );
  }

  Widget _buildStatusChip(
      String value, String label, EmotionProvider provider) {
    final selected = provider.statusFilter == value;
    return Padding(
      padding: const EdgeInsets.only(right: 6),
      child: FilterChip(
        label: Text(label, style: const TextStyle(fontSize: 13)),
        selected: selected,
        onSelected: (_) => provider.setStatusFilter(selected ? '' : value),
        materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        visualDensity: VisualDensity.compact,
        backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
        selectedColor: Theme.of(context)
            .colorScheme
            .primaryContainer
            .withValues(alpha: 0.5),
        side: BorderSide.none,
        showCheckmark: false,
      ),
    );
  }

  /// 统计概览卡片
  Widget _buildStatsSummary() {
    return Consumer<EmotionProvider>(
      builder: (_, provider, __) {
        final stats = provider.stats;
        return Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
          child: Row(
            children: [
              _buildStatCard(
                label: '紧急',
                count: stats?.urgent ?? 0,
                color: const Color(0xFFC62828),
                icon: Icons.warning_rounded,
                loading: provider.statsLoading,
              ),
              const SizedBox(width: 8),
              _buildStatCard(
                label: '高风险',
                count: stats?.high ?? 0,
                color: const Color(0xFFE65100),
                icon: Icons.error_outline,
                loading: provider.statsLoading,
              ),
              const SizedBox(width: 8),
              _buildStatCard(
                label: '中风险',
                count: stats?.medium ?? 0,
                color: const Color(0xFFF9A825),
                icon: Icons.info_outline,
                loading: provider.statsLoading,
              ),
              const SizedBox(width: 8),
              _buildStatCard(
                label: '低风险',
                count: stats?.low ?? 0,
                color: const Color(0xFF4CAF50),
                icon: Icons.check_circle_outline,
                loading: provider.statsLoading,
              ),
              const SizedBox(width: 8),
              _buildStatCard(
                label: '待处理',
                count: stats?.pending ?? 0,
                color: const Color(0xFF1565C0),
                icon: Icons.pending_actions,
                loading: provider.statsLoading,
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildStatCard({
    required String label,
    required int count,
    required Color color,
    required IconData icon,
    required bool loading,
  }) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.06),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: color.withValues(alpha: 0.15)),
        ),
        child: loading
            ? const Center(
                child: SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              )
            : Column(
                children: [
                  Icon(icon, color: color, size: 20),
                  const SizedBox(height: 4),
                  Text(
                    '$count',
                    style: TextStyle(
                      fontSize: 22,
                      fontWeight: FontWeight.bold,
                      color: color,
                    ),
                  ),
                  Text(
                    label,
                    style: TextStyle(
                      fontSize: 11,
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
      ),
    );
  }

  /// 告警列表
  Widget _buildAlertList() {
    return Consumer<EmotionProvider>(
      builder: (_, provider, __) {
        if (provider.loading && provider.alerts.isEmpty) {
          return _buildLoadingState();
        }
        if (provider.error.isNotEmpty && provider.alerts.isEmpty) {
          return _buildErrorState(provider);
        }
        if (provider.alerts.isEmpty) {
          return _buildEmptyState();
        }

        return RefreshIndicator(
          onRefresh: provider.refresh,
          child: NotificationListener<ScrollNotification>(
            onNotification: (notification) {
              if (notification is ScrollEndNotification &&
                  notification.metrics.extentAfter < 200) {
                provider.loadAlerts();
              }
              return false;
            },
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: provider.alerts.length + (provider.hasMore ? 1 : 0),
              itemBuilder: (context, index) {
                if (index >= provider.alerts.length) {
                  return const Padding(
                    padding: EdgeInsets.all(16),
                    child: Center(child: CircularProgressIndicator()),
                  );
                }
                return _AlertCard(alert: provider.alerts[index]);
              },
            ),
          ),
        );
      },
    );
  }

  Widget _buildLoadingState() {
    return const Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircularProgressIndicator(),
          SizedBox(height: 16),
          Text('加载告警列表中...'),
        ],
      ),
    );
  }

  Widget _buildErrorState(EmotionProvider provider) {
    return ErrorView.error(
      message: provider.error,
      onRetry: provider.refresh,
    );
  }

  Widget _buildEmptyState() {
    return ErrorView.empty(
      message: '暂无预警信息',
      subtitle: '当前没有符合条件的情感告警',
      icon: Icons.sentiment_satisfied_alt,
    );
  }

  /// 根据风险等级获取颜色
  static Color _riskColor(String level) {
    switch (level) {
      case 'urgent':
        return const Color(0xFFD32F2F);
      case 'high':
        return const Color(0xFFE65100);
      case 'medium':
        return const Color(0xFFF9A825);
      default:
        return const Color(0xFF4CAF50);
    }
  }
}

/// 单条告警卡片
class _AlertCard extends StatefulWidget {
  final EmotionLog alert;
  const _AlertCard({required this.alert});

  @override
  State<_AlertCard> createState() => _AlertCardState();
}

class _AlertCardState extends State<_AlertCard> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final alert = widget.alert;
    final theme = Theme.of(context);
    final riskColor = _EmotionDashboardPageState._riskColor(alert.riskLevel);

    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      clipBehavior: Clip.antiAlias,
      child: IntrinsicHeight(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // 左侧风险色条
            Container(
              width: 4,
              color: riskColor,
            ),
            // 内容区
            Expanded(
              child: InkWell(
                onTap: () => setState(() => _expanded = !_expanded),
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // 第一行：用户名 + 风险徽章 + 时间
                      Row(
                        children: [
                          Icon(Icons.person, size: 16,
                              color: theme.colorScheme.onSurfaceVariant),
                          const SizedBox(width: 4),
                          Text(alert.username,
                              style: theme.textTheme.labelLarge),
                          const Spacer(),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: riskColor.withValues(alpha: 0.12),
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(alert.riskLabel,
                                style: TextStyle(
                                    color: riskColor,
                                    fontSize: 12,
                                    fontWeight: FontWeight.w600)),
                          ),
                          const SizedBox(width: 8),
                          Text(_formatTime(alert.createdAt),
                              style: theme.textTheme.bodySmall),
                        ],
                      ),
                      const SizedBox(height: 8),
                      // 消息预览
                      Text(
                        alert.messageText,
                        maxLines: _expanded ? 10 : 2,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodyMedium,
                      ),
                      if (_expanded) ...[
                        const SizedBox(height: 12),
                        const Divider(height: 1),
                        const SizedBox(height: 8),
                        // 分析详情
                        _buildAnalysisDetails(alert, theme),
                        const SizedBox(height: 8),
                        // 操作按钮
                        if (alert.status == 'pending' ||
                            alert.status == 'acknowledged')
                          _buildActionButtons(context, alert),
                      ],
                      const SizedBox(height: 4),
                      // 底部提示
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.surfaceContainerHighest,
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(alert.statusLabel,
                                style: const TextStyle(fontSize: 11)),
                          ),
                          Icon(Icons.expand_more,
                              size: 18,
                              color: _expanded
                                  ? theme.colorScheme.primary
                                  : theme.colorScheme.onSurfaceVariant),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// 分析详情：情绪、关键词、理由
  Widget _buildAnalysisDetails(EmotionLog alert, ThemeData theme) {
    final analysis = alert.analysis;
    if (analysis.isEmpty) return const SizedBox.shrink();

    final emotions = (analysis['emotions'] as List?)?.cast<String>() ?? [];
    final keywords = (analysis['keywords'] as List?)?.cast<String>() ?? [];
    final reasoning = analysis['reasoning'] as String? ?? '';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (emotions.isNotEmpty) ...[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('检测情绪：', style: theme.textTheme.labelMedium),
              const SizedBox(width: 4),
              Expanded(
                child: Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: emotions
                      .map((e) => Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.tertiaryContainer,
                              borderRadius: BorderRadius.circular(10),
                            ),
                            child: Text(e,
                                style: TextStyle(
                                    fontSize: 12,
                                    color: theme.colorScheme.onTertiaryContainer)),
                          ))
                      .toList(),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
        ],
        if (keywords.isNotEmpty) ...[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('关键词：', style: theme.textTheme.labelMedium),
              const SizedBox(width: 4),
              Expanded(
                child: Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: keywords
                      .map((k) => Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.errorContainer
                                  .withValues(alpha: 0.5),
                              borderRadius: BorderRadius.circular(10),
                            ),
                            child: Text(k,
                                style: TextStyle(
                                    fontSize: 12,
                                    color: theme.colorScheme.onErrorContainer)),
                          ))
                      .toList(),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
        ],
        if (reasoning.isNotEmpty)
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerLow,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(Icons.psychology, size: 16,
                    color: theme.colorScheme.primary),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(reasoning,
                      style: theme.textTheme.bodySmall),
                ),
              ],
            ),
          ),
      ],
    );
  }

  /// 操作按钮：确认 / 已处理
  Widget _buildActionButtons(BuildContext context, EmotionLog alert) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        if (alert.status == 'pending')
          OutlinedButton.icon(
            onPressed: () => _handleAction(context, alert, 'acknowledged'),
            icon: const Icon(Icons.check, size: 18),
            label: const Text('确认'),
            style: OutlinedButton.styleFrom(
              visualDensity: VisualDensity.compact,
              textStyle: const TextStyle(fontSize: 13),
            ),
          ),
        if (alert.status == 'pending' || alert.status == 'acknowledged') ...[
          const SizedBox(width: 8),
          FilledButton.icon(
            onPressed: () => _handleAction(context, alert, 'resolved'),
            icon: const Icon(Icons.done_all, size: 18),
            label: const Text('已处理'),
            style: FilledButton.styleFrom(
              visualDensity: VisualDensity.compact,
              textStyle: const TextStyle(fontSize: 13),
            ),
          ),
        ],
      ],
    );
  }

  Future<void> _handleAction(
      BuildContext context, EmotionLog alert, String newStatus) async {
    final provider = context.read<EmotionProvider>();
    final success = await provider.updateAlertStatus(alert.alertId, newStatus);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(success ? '操作成功' : '操作失败: ${provider.error}'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  String _formatTime(String raw) {
    if (raw.isEmpty) return '';
    // 后端格式: "2026-05-05 03:34:50"
    try {
      final dt = DateTime.parse(raw);
      final now = DateTime.now();
      final diff = now.difference(dt);
      if (diff.inMinutes < 1) return '刚刚';
      if (diff.inMinutes < 60) return '${diff.inMinutes}分钟前';
      if (diff.inHours < 24) return '${diff.inHours}小时前';
      if (diff.inDays < 7) return '${diff.inDays}天前';
      return '${dt.month}/${dt.day} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    } catch (_) {
      return raw;
    }
  }
}
