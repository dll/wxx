import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/feedback_provider.dart';
import '../../widgets/error_view.dart';

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
              ButtonSegment(value: 'resolved', label: Text('已处理')),
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
      default:
        statusColor = Colors.orange;
    }

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
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: statusColor.withValues(alpha: 0.1),
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
                const Spacer(),
                Text(feedback.createdAt, style: theme.textTheme.labelSmall),
              ],
            ),
            const SizedBox(height: 6),
            Text(feedback.content, style: theme.textTheme.bodyMedium),
            // 截图预览
            if (feedback.screenshotUrl.isNotEmpty) ...[
              const SizedBox(height: 8),
              GestureDetector(
                onTap: () => _showFullScreenshot(context, feedback.screenshotUrl),
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: Image.network(
                    feedback.screenshotUrl,
                    height: 120,
                    cacheHeight: 240,
                    width: double.infinity,
                    fit: BoxFit.cover,
                    errorBuilder: (_, __, ___) => Container(
                      height: 40,
                      color: theme.colorScheme.surfaceContainerHighest,
                      child: Center(
                        child: Text('截图加载失败', style: theme.textTheme.labelSmall),
                      ),
                    ),
                  ),
                ),
              ),
            ],
            // 处理人/时间
            if (feedback.resolvedBy.isNotEmpty || (feedback.resolvedAt ?? '').isNotEmpty) ...[
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
                  color: theme.colorScheme.primaryContainer.withValues(alpha: 0.3),
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
          child: Image.network(url, errorBuilder: (_, __, ___) => const SizedBox.shrink()),
        ),
      ),
    );
  }
}
