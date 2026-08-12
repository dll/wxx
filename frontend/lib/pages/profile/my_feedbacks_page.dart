import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/feedback_provider.dart';
import '../../utils/feedback_report.dart';
import '../../widgets/error_view.dart';
import '../../widgets/feedback_screenshot.dart';

/// 我的反馈（所有登录用户可见，仅展示自己提交过的反馈与处理状态/回复）
class MyFeedbacksPage extends StatefulWidget {
  const MyFeedbacksPage({super.key});

  @override
  State<MyFeedbacksPage> createState() => _MyFeedbacksPageState();
}

class _MyFeedbacksPageState extends State<MyFeedbacksPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<FeedbackProvider>().fetchMyFeedbacks(refresh: true);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('我的反馈')),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(child: _buildList()),
        ],
      ),
    );
  }

  Widget _buildFilterBar() {
    return Consumer<FeedbackProvider>(
      builder: (_, provider, __) {
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: SegmentedButton<String>(
            selected: {provider.myStatusFilter},
            onSelectionChanged: (v) => provider.setMyStatusFilter(v.first),
            segments: const [
              ButtonSegment(value: '', label: Text('全部')),
              ButtonSegment(value: 'pending', label: Text('待处理')),
              ButtonSegment(value: 'processing', label: Text('处理中')),
              ButtonSegment(value: 'resolved', label: Text('已解决')),
              ButtonSegment(value: 'dismissed', label: Text('已驳回')),
            ],
          ),
        );
      },
    );
  }

  Widget _buildList() {
    return Consumer<FeedbackProvider>(
      builder: (_, provider, __) {
        if (provider.myLoading && provider.myFeedbacks.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.myFeedbacks.isEmpty) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.fetchMyFeedbacks(refresh: true),
          );
        }
        if (provider.myFeedbacks.isEmpty) {
          return ErrorView.empty(
            message: '你还没有提交过反馈',
            icon: Icons.feedback_outlined,
          );
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchMyFeedbacks(refresh: true),
          child: ListView.builder(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            itemCount: provider.myFeedbacks.length + 1,
            itemBuilder: (context, index) {
              if (index == provider.myFeedbacks.length) {
                if (provider.myFeedbacks.length < provider.myTotal) {
                  provider.fetchMyFeedbacks();
                  return const Padding(
                    padding: EdgeInsets.all(16),
                    child: Center(child: CircularProgressIndicator()),
                  );
                }
                return const SizedBox.shrink();
              }
              return _MyFeedbackCard(feedback: provider.myFeedbacks[index]);
            },
          ),
        );
      },
    );
  }
}

class _MyFeedbackCard extends StatelessWidget {
  final FeedbackEntry feedback;
  const _MyFeedbackCard({required this.feedback});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

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
                        fontWeight: FontWeight.w600,
                      ),
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
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ],
                  const Spacer(),
                  Text(feedback.createdAt, style: theme.textTheme.labelSmall),
                  const SizedBox(width: 4),
                  IconButton(
                    tooltip: '复制反馈数据',
                    visualDensity: VisualDensity.compact,
                    iconSize: 18,
                    onPressed: () => _copyFeedback(context),
                    icon: const Icon(Icons.copy_outlined),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Text(feedback.content,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodyMedium),
              // 截图预览
              if (feedback.screenshotUrl.isNotEmpty) ...[
                const SizedBox(height: 8),
                ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: FeedbackScreenshot(
                    url: feedback.screenshotUrl,
                    height: 100,
                    cacheHeight: 200,
                    width: double.infinity,
                    fit: BoxFit.cover,
                  ),
                ),
              ],
              // 满意度评分显示
              if (feedback.rating > 0) ...[
                const SizedBox(height: 6),
                Row(
                  children: [
                    const Icon(Icons.star, size: 14, color: Colors.amber),
                    const SizedBox(width: 2),
                    Text(
                      '已评价：${feedback.rating} 星',
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: Colors.amber),
                    ),
                  ],
                ),
              ],
              // 处理人/时间
              if (feedback.resolvedBy.isNotEmpty ||
                  (feedback.resolvedAt ?? '').isNotEmpty) ...[
                const SizedBox(height: 6),
                Text(
                  '处理人：${feedback.resolvedBy.isEmpty ? "—" : feedback.resolvedBy}'
                  '${(feedback.resolvedAt ?? '').isNotEmpty ? "  ·  ${feedback.resolvedAt}" : ""}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
              // 管理员回复
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
              // 评价按钮
              if (feedback.status == 'resolved' && feedback.rating == 0) ...[
                const SizedBox(height: 8),
                Align(
                  alignment: Alignment.centerRight,
                  child: FilledButton.tonalIcon(
                    onPressed: () => _showRateDialog(context),
                    icon: const Icon(Icons.star_border, size: 18),
                    label: const Text('评价'),
                    style: FilledButton.styleFrom(
                      visualDensity: VisualDensity.compact,
                    ),
                  ),
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
        builder: (ctx) => MyFeedbackDetailPage(feedbackId: feedback.feedbackId),
      ),
    );
  }

  /// 复制反馈结构化数据（JSON + Markdown），供粘贴给 AI 工具修复
  void _copyFeedback(BuildContext context) {
    Clipboard.setData(ClipboardData(text: FeedbackReport.buildJson(feedback)));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('已复制反馈数据，可粘贴到 AI 工具')),
    );
  }

  void _showRateDialog(BuildContext context) {
    final ratingNotifier = ValueNotifier<int>(5);
    final commentCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('评价处理结果'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ValueListenableBuilder<int>(
              valueListenable: ratingNotifier,
              builder: (_, rating, __) {
                return Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: List.generate(5, (i) {
                    return IconButton(
                      onPressed: () => ratingNotifier.value = i + 1,
                      icon: Icon(
                        i < rating ? Icons.star : Icons.star_border,
                        color: Colors.amber,
                        size: 32,
                      ),
                    );
                  }),
                );
              },
            ),
            const SizedBox(height: 8),
            TextField(
              controller: commentCtrl,
              maxLines: 2,
              decoration: const InputDecoration(
                labelText: '评价内容（可选）',
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
              final ok = await context.read<FeedbackProvider>().rateFeedback(
                  feedback.feedbackId, ratingNotifier.value,
                  comment: commentCtrl.text);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '评价成功' : '评价失败')),
                );
                if (ok) {
                  context
                      .read<FeedbackProvider>()
                      .fetchMyFeedbacks(refresh: true);
                }
              }
            },
            child: const Text('提交评价'),
          ),
        ],
      ),
    );
  }
}

/// 我的反馈详情页
class MyFeedbackDetailPage extends StatefulWidget {
  final String feedbackId;
  const MyFeedbackDetailPage({super.key, required this.feedbackId});

  @override
  State<MyFeedbackDetailPage> createState() => _MyFeedbackDetailPageState();
}

class _MyFeedbackDetailPageState extends State<MyFeedbackDetailPage> {
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
          Consumer<FeedbackProvider>(
            builder: (_, provider, __) {
              final fb = provider.currentFeedback;
              if (fb == null) return const SizedBox.shrink();
              return IconButton(
                tooltip: '复制反馈数据',
                onPressed: () {
                  Clipboard.setData(ClipboardData(
                      text: FeedbackReport.buildJson(fb, logs: provider.logs)));
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('已复制反馈数据，可粘贴到 AI 工具')),
                  );
                },
                icon: const Icon(Icons.copy_all_outlined),
              );
            },
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
              if (fb.status == 'resolved' && fb.rating == 0)
                _buildRateSection(fb, context),
              if (fb.rating > 0) _buildRatingDisplay(fb, context),
              if (fb.rating > 0) const SizedBox(height: 12),
              _buildTimelineSection(context),
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
                const Spacer(),
                Text(fb.feedbackId, style: theme.textTheme.labelSmall),
              ],
            ),
            const SizedBox(height: 12),
            Text('提交于 ${fb.createdAt}',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            if (fb.resolvedBy.isNotEmpty) ...[
              const SizedBox(height: 4),
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
            Text(fb.content, style: theme.textTheme.bodyLarge),
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
    if (fb.resourceId.isEmpty) {
      return const SizedBox.shrink();
    }
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('关联知识资源', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
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
                child: Text(fb.reply, style: theme.textTheme.bodyMedium),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildRateSection(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('满意度评价', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text('请对我们的处理结果进行评价',
                style: theme.textTheme.bodyMedium
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () => _showRateDialog(context, fb),
                icon: const Icon(Icons.star_border),
                label: const Text('去评价'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildRatingDisplay(FeedbackEntry fb, BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('我的评价', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
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
              Text('评价：${fb.ratingComment}', style: theme.textTheme.bodyMedium),
            ],
            if (fb.ratedAt != null) ...[
              const SizedBox(height: 4),
              Text('评价时间：${fb.ratedAt}',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ],
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
                Text('处理进度', style: theme.textTheme.titleMedium),
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

  void _showRateDialog(BuildContext context, FeedbackEntry fb) {
    final ratingNotifier = ValueNotifier<int>(5);
    final commentCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('评价处理结果'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ValueListenableBuilder<int>(
              valueListenable: ratingNotifier,
              builder: (_, rating, __) {
                return Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: List.generate(5, (i) {
                    return IconButton(
                      onPressed: () => ratingNotifier.value = i + 1,
                      icon: Icon(
                        i < rating ? Icons.star : Icons.star_border,
                        color: Colors.amber,
                        size: 32,
                      ),
                    );
                  }),
                );
              },
            ),
            const SizedBox(height: 8),
            TextField(
              controller: commentCtrl,
              maxLines: 2,
              decoration: const InputDecoration(
                labelText: '评价内容（可选）',
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
              final ok = await context.read<FeedbackProvider>().rateFeedback(
                  fb.feedbackId, ratingNotifier.value,
                  comment: commentCtrl.text);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '评价成功' : '评价失败')),
                );
              }
            },
            child: const Text('提交评价'),
          ),
        ],
      ),
    );
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
                      Expanded(
                        child: Text(log.actionLabel,
                            style: theme.textTheme.bodyMedium
                                ?.copyWith(fontWeight: FontWeight.w600)),
                      ),
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
