import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/feedback_provider.dart';
import '../../models/models.dart';
import '../../widgets/error_view.dart';
import '../../widgets/feedback_screenshot.dart';

/// 反馈管理页面（student_union 及以上可访问）
class FeedbackPage extends StatefulWidget {
  const FeedbackPage({super.key});

  @override
  State<FeedbackPage> createState() => _FeedbackPageState();
}

class _FeedbackPageState extends State<FeedbackPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<FeedbackProvider>().fetchFeedbacks();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('反馈管理')),
      body: Column(
        children: [
          _buildFilterBar(),
          Expanded(child: _buildFeedbackList()),
        ],
      ),
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
                  selected: {provider.statusFilter.isEmpty ? '' : provider.statusFilter},
                  onSelectionChanged: (v) {
                    provider.setStatusFilter(v.first == '' ? '' : v.first);
                  },
                  segments: const [
                    ButtonSegment(value: '', label: Text('全部')),
                    ButtonSegment(value: 'pending', label: Text('待处理')),
                    ButtonSegment(value: 'resolved', label: Text('已处理')),
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

class _FeedbackCard extends StatelessWidget {
  final FeedbackEntry feedback;
  const _FeedbackCard({required this.feedback});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isPending = feedback.status == 'pending';

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
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
                    color: isPending
                        ? Colors.orange.withOpacity( 0.1)
                        : Colors.green.withOpacity( 0.1),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    '${feedback.categoryLabel} · ${feedback.statusLabel}',
                    style: TextStyle(
                        fontSize: 11,
                        color: isPending ? Colors.orange : Colors.green,
                        fontWeight: FontWeight.w600),
                  ),
                ),
                const Spacer(),
                Text(feedback.createdAt,
                    style: theme.textTheme.labelSmall),
              ],
            ),
            const SizedBox(height: 6),
            Text(feedback.content, style: theme.textTheme.bodyMedium),
            const SizedBox(height: 4),
            Text('— ${feedback.username}',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            // 截图预览
            if (feedback.screenshotUrl.isNotEmpty) ...[
              const SizedBox(height: 8),
              GestureDetector(
                onTap: () => _showFullScreenshot(context, feedback.screenshotUrl),
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: FeedbackScreenshot(
                    url: feedback.screenshotUrl,
                    height: 120,
                    cacheHeight: 240,
                    width: double.infinity,
                    fit: BoxFit.cover,
                  ),
                ),
              ),
            ],
            // 回复内容
            if (feedback.reply.isNotEmpty) ...[
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer.withOpacity( 0.3),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  '回复：${feedback.reply}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onPrimaryContainer,
                  ),
                ),
              ),
            ],
            if (isPending) ...[
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
                  FilledButton.icon(
                    onPressed: () => _showReplyDialog(context),
                    icon: const Icon(Icons.check, size: 16),
                    label: const Text('处理'),
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
              final ok = await context
                  .read<FeedbackProvider>()
                  .resolveFeedback(feedback.feedbackId, 'resolved', reply: replyCtrl.text);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(ok ? '操作成功' : '操作失败')),
                );
              }
            },
            child: const Text('确认处理'),
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
