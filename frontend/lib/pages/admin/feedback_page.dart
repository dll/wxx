import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import '../../providers/feedback_provider.dart';
import '../../models/models.dart';
import '../../widgets/error_view.dart';
import '../../widgets/feedback_screenshot.dart';
import '../../widgets/feedback_repair.dart';

/// 反馈管理页面（student_union 及以上可访问）
class FeedbackPage extends StatefulWidget {
  const FeedbackPage({super.key});

  @override
  State<FeedbackPage> createState() => _FeedbackPageState();
}

class _FeedbackPageState extends State<FeedbackPage> {
  bool _showDashboard = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<FeedbackProvider>().fetchStats();
      context.read<FeedbackProvider>().fetchFeedbacks();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('反馈管理'),
        actions: [
          IconButton(
            icon: Icon(_showDashboard ? Icons.list : Icons.dashboard),
            tooltip: _showDashboard ? '切换到列表' : '切换到仪表盘',
            onPressed: () {
              setState(() {
                _showDashboard = !_showDashboard;
              });
            },
          ),
        ],
      ),
      body: _showDashboard ? _buildDashboard() : _buildListView(),
    );
  }

  Widget _buildDashboard() {
    return Consumer<FeedbackProvider>(
      builder: (_, provider, __) {
        if (provider.statsLoading && provider.stats == null) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.stats == null) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.fetchStats(),
          );
        }
        final stats = provider.stats;
        if (stats == null) {
          return const SizedBox.shrink();
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchStats(),
          child: ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _buildStatCards(stats),
              const SizedBox(height: 16),
              _buildWeekTrendCard(stats),
              const SizedBox(height: 16),
              _buildTopIssuesCard(stats),
              const SizedBox(height: 16),
              _buildAvgResolveCard(stats),
            ],
          ),
        );
      },
    );
  }

  Widget _buildStatCards(FeedbackStats stats) {
    final theme = Theme.of(context);
    return GridView.count(
      crossAxisCount: 2,
      crossAxisSpacing: 12,
      mainAxisSpacing: 12,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      children: [
        _StatCard(
          title: '反馈总数',
          value: '${stats.total}',
          icon: Icons.feedback_outlined,
          color: theme.colorScheme.primary,
        ),
        _StatCard(
          title: '待处理',
          value: '${stats.byStatus['pending'] ?? 0}',
          icon: Icons.pending_actions_outlined,
          color: Colors.orange,
        ),
        _StatCard(
          title: '处理中',
          value: '${stats.byStatus['processing'] ?? 0}',
          icon: Icons.autorenew_outlined,
          color: Colors.blue,
        ),
        _StatCard(
          title: '已解决',
          value: '${stats.byStatus['resolved'] ?? 0}',
          icon: Icons.check_circle_outline,
          color: Colors.green,
        ),
      ],
    );
  }

  Widget _buildWeekTrendCard(FeedbackStats stats) {
    final theme = Theme.of(context);
    if (stats.weekTrend.isEmpty) {
      return Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('近 7 天趋势', style: theme.textTheme.titleMedium),
              const SizedBox(height: 12),
              const Center(child: Text('暂无数据')),
            ],
          ),
        ),
      );
    }
    final maxCount = stats.weekTrend
        .map((e) => e.count)
        .fold<int>(0, (a, b) => a > b ? a : b);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('近 7 天趋势', style: theme.textTheme.titleMedium),
            const SizedBox(height: 16),
            SizedBox(
              height: 160,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.end,
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: stats.weekTrend.map((item) {
                  final height =
                      maxCount == 0 ? 0.0 : (item.count / maxCount) * 120;
                  return _BarItem(
                    date: item.date.substring(5),
                    count: item.count,
                    height: height,
                    color: theme.colorScheme.primary,
                  );
                }).toList(),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTopIssuesCard(FeedbackStats stats) {
    final theme = Theme.of(context);
    if (stats.topIssues.isEmpty) {
      return Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('热门问题分类', style: theme.textTheme.titleMedium),
              const SizedBox(height: 12),
              const Center(child: Text('暂无数据')),
            ],
          ),
        ),
      );
    }
    final maxCount = stats.topIssues
        .map((e) => e.count)
        .fold<int>(0, (a, b) => a > b ? a : b);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('热门问题分类', style: theme.textTheme.titleMedium),
            const SizedBox(height: 12),
            ...stats.topIssues.asMap().entries.map((entry) {
              final index = entry.key;
              final issue = entry.value;
              return Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  children: [
                    SizedBox(
                      width: 24,
                      child: Text(
                        '${index + 1}',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                    Expanded(
                      child: Text(issue.keyword,
                          style: theme.textTheme.bodyMedium),
                    ),
                    const SizedBox(width: 8),
                    SizedBox(
                      width: 100,
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(4),
                        child: LinearProgressIndicator(
                          value: maxCount == 0 ? 0 : issue.count / maxCount,
                          minHeight: 8,
                          backgroundColor: theme.colorScheme.surfaceVariant,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    SizedBox(
                      width: 32,
                      child: Text(
                        '${issue.count}',
                        style: theme.textTheme.bodySmall,
                        textAlign: TextAlign.right,
                      ),
                    ),
                  ],
                ),
              );
            }).toList(),
          ],
        ),
      ),
    );
  }

  Widget _buildAvgResolveCard(FeedbackStats stats) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Icon(Icons.schedule_outlined,
                size: 32, color: theme.colorScheme.primary),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('平均解决时长', style: theme.textTheme.titleMedium),
                  const SizedBox(height: 4),
                  Text(
                    stats.avgResolveHours > 0
                        ? '${stats.avgResolveHours.toStringAsFixed(1)} 小时'
                        : '暂无数据',
                    style: theme.textTheme.headlineSmall?.copyWith(
                      color: theme.colorScheme.primary,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildListView() {
    return Column(
      children: [
        _buildFilterBar(),
        Expanded(child: _buildFeedbackList()),
      ],
    );
  }

  Widget _buildFilterBar() {
    return Consumer<FeedbackProvider>(
      builder: (_, provider, __) {
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: Row(
            children: [
              Expanded(
                child: SegmentedButton<String>(
                  selected: {
                    provider.statusFilter.isEmpty ? '' : provider.statusFilter
                  },
                  onSelectionChanged: (v) {
                    provider.setStatusFilter(v.first == '' ? '' : v.first);
                  },
                  segments: const [
                    ButtonSegment(value: '', label: Text('全部')),
                    ButtonSegment(value: 'pending', label: Text('待处理')),
                    ButtonSegment(value: 'processing', label: Text('处理中')),
                    ButtonSegment(value: 'resolved', label: Text('已解决')),
                    ButtonSegment(value: 'dismissed', label: Text('已驳回')),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildFeedbackList() {
    return Consumer<FeedbackProvider>(
      builder: (_, provider, __) {
        if (provider.loading && provider.feedbacks.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.feedbacks.isEmpty) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.fetchFeedbacks(),
          );
        }
        if (provider.feedbacks.isEmpty) {
          return ErrorView.empty(
              message: '暂无反馈', icon: Icons.feedback_outlined);
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchFeedbacks(refresh: true),
          child: ListView.builder(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            itemCount: provider.feedbacks.length + 1,
            itemBuilder: (context, index) {
              if (index == provider.feedbacks.length) {
                if (provider.feedbacks.length < provider.total) {
                  provider.fetchFeedbacks();
                  return const Padding(
                    padding: EdgeInsets.all(16),
                    child: Center(child: CircularProgressIndicator()),
                  );
                }
                return const SizedBox.shrink();
              }
              return _FeedbackCard(feedback: provider.feedbacks[index]);
            },
          ),
        );
      },
    );
  }
}

class _StatCard extends StatelessWidget {
  final String title;
  final String value;
  final IconData icon;
  final Color color;

  const _StatCard({
    required this.title,
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
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, color: color, size: 28),
            const SizedBox(height: 8),
            Text(
              value,
              style: theme.textTheme.headlineSmall?.copyWith(
                fontWeight: FontWeight.bold,
                color: color,
              ),
            ),
            const SizedBox(height: 2),
            Text(title, style: theme.textTheme.bodySmall),
          ],
        ),
      ),
    );
  }
}

class _BarItem extends StatelessWidget {
  final String date;
  final int count;
  final double height;
  final Color color;

  const _BarItem({
    required this.date,
    required this.count,
    required this.height,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        Text('$count', style: theme.textTheme.bodySmall),
        const SizedBox(height: 4),
        Container(
          width: 24,
          height: height > 0 ? height : 2,
          decoration: BoxDecoration(
            color: color.withOpacity(0.8),
            borderRadius: BorderRadius.circular(4),
          ),
        ),
        const SizedBox(height: 4),
        Text(date, style: theme.textTheme.bodySmall),
      ],
    );
  }
}

class _FeedbackCard extends StatelessWidget {
  final FeedbackEntry feedback;
  const _FeedbackCard({required this.feedback});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isPending = feedback.status == 'pending';
    final isProcessing = feedback.status == 'processing';

    Color statusColor;
    switch (feedback.status) {
      case 'resolved':
        statusColor = Colors.green;
        break;
      case 'dismissed':
        statusColor = theme.colorScheme.error;
        break;
      case 'processing':
        statusColor = Colors.blue;
        break;
      default:
        statusColor = Colors.orange;
    }

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        onTap: () => _openDetail(context),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: statusColor.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      '${feedback.categoryLabel} · ${feedback.statusLabel}',
                      style: TextStyle(
                          fontSize: 11,
                          color: statusColor,
                          fontWeight: FontWeight.w600),
                    ),
                  ),
                  if (feedback.module.isNotEmpty) ...[
                    const SizedBox(width: 6),
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.secondaryContainer,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        feedback.module,
                        style: TextStyle(
                            fontSize: 11,
                            color: theme.colorScheme.onSecondaryContainer,
                            fontWeight: FontWeight.w600),
                      ),
                    ),
                  ],
                  const Spacer(),
                  Text(feedback.createdAt, style: theme.textTheme.labelSmall),
                  const SizedBox(width: 4),
                  IconButton(
                    tooltip: '复制反馈内容',
                    visualDensity: VisualDensity.compact,
                    iconSize: 18,
                    onPressed: () => _copyText(context, feedback.content),
                    icon: const Icon(Icons.copy_outlined),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              SelectableText(feedback.content,
                  maxLines: 2, style: theme.textTheme.bodyMedium),
              const SizedBox(height: 4),
              Text('— ${feedback.username}',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              // 满意度评分
              if (feedback.rating > 0) ...[
                const SizedBox(height: 6),
                Row(
                  children: [
                    const Icon(Icons.star, size: 14, color: Colors.amber),
                    const SizedBox(width: 2),
                    Text(
                      '${feedback.rating} 星',
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: Colors.amber),
                    ),
                  ],
                ),
              ],
              // 回复内容
              if (feedback.reply.isNotEmpty) ...[
                const SizedBox(height: 8),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.primaryContainer.withOpacity(0.3),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    '回复：${feedback.reply}',
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onPrimaryContainer,
                    ),
                  ),
                ),
              ],
              if (isPending || isProcessing) ...[
                const SizedBox(height: 10),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    OutlinedButton.icon(
                      onPressed: () => _doResolve(context, 'dismissed'),
                      icon: const Icon(Icons.close, size: 16),
                      label: const Text('驳回'),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: theme.colorScheme.error,
                        visualDensity: VisualDensity.compact,
                      ),
                    ),
                    const SizedBox(width: 8),
                    if (isPending)
                      OutlinedButton.icon(
                        onPressed: () => _doResolve(context, 'processing'),
                        icon: const Icon(Icons.autorenew, size: 16),
                        label: const Text('处理中'),
                        style: OutlinedButton.styleFrom(
                          foregroundColor: Colors.blue,
                          visualDensity: VisualDensity.compact,
                        ),
                      ),
                    if (isPending) const SizedBox(width: 8),
                    FilledButton.icon(
                      onPressed: () => _showReplyDialog(context),
                      icon: const Icon(Icons.check, size: 16),
                      label: const Text('解决'),
                      style: FilledButton.styleFrom(
                        visualDensity: VisualDensity.compact,
                      ),
                    ),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  void _openDetail(BuildContext context) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (ctx) => FeedbackDetailPage(feedbackId: feedback.feedbackId),
      ),
    );
  }

  void _copyText(BuildContext context, String text) {
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context)
        .showSnackBar(const SnackBar(content: Text('已复制反馈内容')));
  }

  void _showReplyDialog(BuildContext context) {
    final replyCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('处理反馈'),
        content: TextField(
          controller: replyCtrl,
          maxLines: 3,
          decoration: const InputDecoration(
            hintText: '可选：给用户回复...',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () async {
              Navigator.pop(ctx);
              final ok = await context.read<FeedbackProvider>().resolveFeedback(
                  feedback.feedbackId, 'resolved',
                  reply: replyCtrl.text);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '操作成功' : '操作失败')),
                );
              }
            },
            child: const Text('确认解决'),
          ),
        ],
      ),
    );
  }

  Future<void> _doResolve(BuildContext context, String status) async {
    final ok = await context
        .read<FeedbackProvider>()
        .resolveFeedback(feedback.feedbackId, status);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '操作成功' : '操作失败')),
      );
    }
  }
}

/// 反馈详情页
class FeedbackDetailPage extends StatefulWidget {
  final String feedbackId;
  const FeedbackDetailPage({super.key, required this.feedbackId});

  @override
  State<FeedbackDetailPage> createState() => _FeedbackDetailPageState();
}

class _FeedbackDetailPageState extends State<FeedbackDetailPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<FeedbackProvider>().fetchFeedbackDetail(widget.feedbackId);
      context.read<FeedbackProvider>().fetchFeedbackLogs(widget.feedbackId);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('反馈详情'),
        actions: [
          IconButton(
            tooltip: '复制完整反馈',
            onPressed: () => _copyFullFeedback(context),
            icon: const Icon(Icons.copy_all_outlined),
          ),
        ],
      ),
      body: Consumer<FeedbackProvider>(
        builder: (_, provider, __) {
          if (provider.detailLoading) {
            return const Center(child: CircularProgressIndicator());
          }
          final fb = provider.currentFeedback;
          if (fb == null) {
            return ErrorView.error(
              message: '加载失败',
              onRetry: () => provider.fetchFeedbackDetail(widget.feedbackId),
            );
          }
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _buildBasicInfo(fb, context),
              const SizedBox(height: 12),
              _buildContentSection(fb, context),
              const SizedBox(height: 12),
              _buildLinkedResource(fb, context),
              const SizedBox(height: 12),
              _buildReplySection(fb, context),
              const SizedBox(height: 12),
              _buildRatingSection(fb, context),
              const SizedBox(height: 12),
              _buildTimelineSection(context),
              const SizedBox(height: 20),
              // 在线修复：任何状态下都可用（已解决/已驳回的反馈也能查看代码定位）
              FilledButton.icon(
                onPressed: () => showOnlineRepair(context, fb),
                icon: const Icon(Icons.build_circle_outlined),
                label: const Text('在线修复'),
                style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.tertiary,
                  foregroundColor: Theme.of(context).colorScheme.onTertiary,
                ),
              ),
              const SizedBox(height: 10),
              if (fb.status == 'pending' || fb.status == 'processing')
                _buildActionButtons(fb, context),
            ],
          );
        },
      ),
    );
  }

  Widget _buildBasicInfo(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    Color statusColor;
    switch (fb.status) {
      case 'resolved':
        statusColor = Colors.green;
        break;
      case 'dismissed':
        statusColor = theme.colorScheme.error;
        break;
      case 'processing':
        statusColor = Colors.blue;
        break;
      default:
        statusColor = Colors.orange;
    }
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: statusColor.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    fb.statusLabel,
                    style: TextStyle(
                        color: statusColor, fontWeight: FontWeight.w600),
                  ),
                ),
                const SizedBox(width: 8),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.secondaryContainer,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    fb.categoryLabel,
                    style: TextStyle(
                        color: theme.colorScheme.onSecondaryContainer),
                  ),
                ),
                if (fb.module.isNotEmpty) ...[
                  const SizedBox(width: 8),
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.tertiaryContainer,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      fb.module,
                      style: TextStyle(
                          color: theme.colorScheme.onTertiaryContainer,
                          fontWeight: FontWeight.w500),
                    ),
                  ),
                ],
                const Spacer(),
                Text(fb.feedbackId, style: theme.textTheme.labelSmall),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                CircleAvatar(
                  radius: 16,
                  child: Text(fb.username.isNotEmpty ? fb.username[0] : '?'),
                ),
                const SizedBox(width: 8),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(fb.username,
                        style: theme.textTheme.bodyMedium
                            ?.copyWith(fontWeight: FontWeight.w600)),
                    Text('提交于 ${fb.createdAt}',
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant)),
                  ],
                ),
              ],
            ),
            if (fb.resolvedBy.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                '处理人：${fb.resolvedBy}'
                '${fb.resolvedAt != null ? "  ·  ${fb.resolvedAt}" : ""}',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildContentSection(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('反馈内容', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            SelectableText(fb.content, style: theme.textTheme.bodyLarge),
            if (fb.screenshotUrl.isNotEmpty) ...[
              const SizedBox(height: 12),
              GestureDetector(
                onTap: () => _showFullScreenshot(context, fb.screenshotUrl),
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: FeedbackScreenshot(
                    url: fb.screenshotUrl,
                    height: 200,
                    width: double.infinity,
                    fit: BoxFit.cover,
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildLinkedResource(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text('关联知识资源', style: theme.textTheme.titleMedium),
                const Spacer(),
                TextButton.icon(
                  onPressed: () => _showLinkDialog(context),
                  icon: const Icon(Icons.link, size: 16),
                  label: Text(fb.resourceId.isEmpty ? '关联' : '修改'),
                ),
              ],
            ),
            const SizedBox(height: 8),
            if (fb.resourceId.isEmpty)
              Text('暂无关联的知识资源',
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
            else
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(Icons.menu_book_outlined,
                          size: 18, color: theme.colorScheme.primary),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(fb.resourceId,
                            style: theme.textTheme.bodyMedium
                                ?.copyWith(fontWeight: FontWeight.w500)),
                      ),
                    ],
                  ),
                  if (fb.linkedResourceNote.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text('备注：${fb.linkedResourceNote}',
                        style: theme.textTheme.bodySmall),
                  ],
                  if (fb.linkedBy.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      '关联人：${fb.linkedBy}'
                      '${fb.linkedAt != null ? "  ·  ${fb.linkedAt}" : ""}',
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                    ),
                  ],
                ],
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildReplySection(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('处理回复', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            if (fb.reply.isEmpty)
              Text('暂无回复',
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
            else
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer.withOpacity(0.3),
                  borderRadius: BorderRadius.circular(8),
                ),
                child:
                    SelectableText(fb.reply, style: theme.textTheme.bodyMedium),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildRatingSection(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('用户满意度', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            if (fb.rating == 0)
              Text('用户尚未评价',
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
            else
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: List.generate(5, (i) {
                      return Icon(
                        i < fb.rating ? Icons.star : Icons.star_border,
                        color: Colors.amber,
                        size: 24,
                      );
                    }),
                  ),
                  if (fb.ratingComment.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    SelectableText('评价：${fb.ratingComment}',
                        style: theme.textTheme.bodyMedium),
                  ],
                  if (fb.ratedAt != null) ...[
                    const SizedBox(height: 4),
                    Text('评价时间：${fb.ratedAt}',
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant)),
                  ],
                ],
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildTimelineSection(BuildContext context) {
    final theme = Theme.of(context);
    return Consumer<FeedbackProvider>(
      builder: (_, provider, __) {
        if (provider.logsLoading && provider.logs.isEmpty) {
          return const Card(
            child: Padding(
              padding: EdgeInsets.all(16),
              child: Center(child: CircularProgressIndicator()),
            ),
          );
        }
        final logs = provider.logs;
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('处理记录', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                if (logs.isEmpty)
                  Text('暂无处理记录',
                      style: theme.textTheme.bodyMedium
                          ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
                else
                  ...logs.asMap().entries.map((entry) {
                    final index = entry.key;
                    final log = entry.value;
                    final isLast = index == logs.length - 1;
                    return _TimelineItem(
                      log: log,
                      isLast: isLast,
                    );
                  }).toList(),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildActionButtons(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () => _doResolve(context, fb, 'dismissed'),
                icon: const Icon(Icons.close),
                label: const Text('驳回'),
                style: OutlinedButton.styleFrom(
                  foregroundColor: theme.colorScheme.error,
                ),
              ),
            ),
            const SizedBox(width: 8),
            if (fb.status == 'pending')
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => _doResolve(context, fb, 'processing'),
                  icon: const Icon(Icons.autorenew),
                  label: const Text('标记处理中'),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.blue,
                  ),
                ),
              ),
            if (fb.status == 'pending') const SizedBox(width: 8),
            Expanded(
              child: FilledButton.icon(
                onPressed: () => _showReplyDialog(context, fb),
                icon: const Icon(Icons.check),
                label: const Text('标记解决'),
              ),
            ),
          ],
        ),
      ],
    );
  }

  void _showFullScreenshot(BuildContext context, String url) {
    showDialog(
      context: context,
      builder: (ctx) => Dialog(
        child: InteractiveViewer(
          child: FeedbackScreenshot(url: url, fit: BoxFit.contain),
        ),
      ),
    );
  }

  void _showLinkDialog(BuildContext context) {
    final resourceIdCtrl = TextEditingController();
    final noteCtrl = TextEditingController();
    final fb = context.read<FeedbackProvider>().currentFeedback;
    if (fb != null) {
      resourceIdCtrl.text = fb.resourceId;
      noteCtrl.text = fb.linkedResourceNote;
    }
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('关联知识资源'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: resourceIdCtrl,
              decoration: const InputDecoration(
                labelText: '资源 ID',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: noteCtrl,
              maxLines: 2,
              decoration: const InputDecoration(
                labelText: '备注（可选）',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () async {
              Navigator.pop(ctx);
              final ok = await context.read<FeedbackProvider>().linkResource(
                  widget.feedbackId, resourceIdCtrl.text,
                  note: noteCtrl.text);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '关联成功' : '关联失败')),
                );
              }
            },
            child: const Text('确认关联'),
          ),
        ],
      ),
    );
  }

  void _showReplyDialog(BuildContext context, FeedbackEntry fb) {
    final replyCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('处理反馈'),
        content: TextField(
          controller: replyCtrl,
          maxLines: 3,
          decoration: const InputDecoration(
            hintText: '可选：给用户回复...',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () async {
              Navigator.pop(ctx);
              final ok = await context.read<FeedbackProvider>().resolveFeedback(
                  fb.feedbackId, 'resolved',
                  reply: replyCtrl.text);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '操作成功' : '操作失败')),
                );
              }
            },
            child: const Text('确认解决'),
          ),
        ],
      ),
    );
  }

  Future<void> _doResolve(
      BuildContext context, FeedbackEntry fb, String status) async {
    final ok = await context
        .read<FeedbackProvider>()
        .resolveFeedback(fb.feedbackId, status);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '操作成功' : '操作失败')),
      );
    }
  }

  /// 复制完整反馈（含最新处理记录）到剪贴板
  Future<void> _copyFullFeedback(BuildContext context) async {
    final fb = context.read<FeedbackProvider>().currentFeedback;
    if (fb == null) return;
    final logs = context.read<FeedbackProvider>().logs;
    final sb = StringBuffer()
      ..writeln('反馈详情')
      ..writeln('ID: ${fb.feedbackId}')
      ..writeln('提交用户: ${fb.username}（${fb.createdAt}）')
      ..writeln(
          '类型: ${fb.categoryLabel} | 模块: ${fb.module.isNotEmpty ? fb.module : '未指定'}')
      ..writeln(
          '状态: ${fb.statusLabel}${fb.resolvedBy.isNotEmpty ? ' | 处理人: ${fb.resolvedBy}' : ''}')
      ..writeln('消息ID: ${fb.messageId}')
      ..writeln('关联资源: ${fb.resourceId.isEmpty ? '无' : fb.resourceId}')
      ..writeln('--')
      ..writeln('反馈内容:')
      ..writeln(fb.content);
    if (fb.reply.isNotEmpty) {
      sb
        ..writeln('--')
        ..writeln('处理回复:')
        ..writeln(fb.reply);
    }
    if (fb.rating > 0) {
      sb
        ..writeln('--')
        ..writeln(
            '满意度: ${fb.rating} 星${fb.ratingComment.isNotEmpty ? ' | ${fb.ratingComment}' : ''}');
    }
    if (logs.isNotEmpty) {
      sb
        ..writeln('--')
        ..writeln('处理记录:');
      for (final log in logs) {
        sb.writeln(
            '- [${log.createdAt}] ${log.actionLabel}${log.operator.isNotEmpty ? ' by ${log.operator}' : ''}${log.detail.isNotEmpty ? '（${log.detail}）' : ''}');
      }
    }
    await Clipboard.setData(ClipboardData(text: sb.toString()));
    if (context.mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已复制完整反馈')));
    }
  }
}

class _TimelineItem extends StatelessWidget {
  final FeedbackLog log;
  final bool isLast;

  const _TimelineItem({required this.log, required this.isLast});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Column(
            children: [
              Container(
                width: 12,
                height: 12,
                decoration: BoxDecoration(
                  color: theme.colorScheme.primary,
                  shape: BoxShape.circle,
                ),
              ),
              if (!isLast)
                Expanded(
                  child: Container(
                    width: 2,
                    color: theme.colorScheme.surfaceVariant,
                  ),
                ),
            ],
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(bottom: 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text(log.actionLabel,
                          style: theme.textTheme.bodyMedium
                              ?.copyWith(fontWeight: FontWeight.w600)),
                      const Spacer(),
                      Text(log.createdAt, style: theme.textTheme.bodySmall),
                    ],
                  ),
                  if (log.operator.isNotEmpty) ...[
                    const SizedBox(height: 2),
                    Text('操作人：${log.operator}',
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant)),
                  ],
                  if (log.detail.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(log.detail, style: theme.textTheme.bodySmall),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
