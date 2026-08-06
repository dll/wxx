import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/knowledge_provider.dart';
import '../../models/models.dart';
import '../../utils/web_export.dart';
import '../../widgets/error_view.dart';
import '../../widgets/md_text.dart';

/// 知识审核页面（counselor 及以上可访问）
class ReviewPage extends StatefulWidget {
  const ReviewPage({super.key});

  @override
  State<ReviewPage> createState() => _ReviewPageState();
}

class _ReviewPageState extends State<ReviewPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<KnowledgeProvider>().listPendingReviews();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('知识审核')),
      body: Consumer<KnowledgeProvider>(
        builder: (_, provider, __) {
          if (provider.reviewsLoading && provider.pendingReviews.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.pendingReviews.isEmpty) {
            return ErrorView.empty(
                message: '暂无待审核资源', icon: Icons.rate_review_outlined);
          }
          return RefreshIndicator(
            onRefresh: () => provider.listPendingReviews(refresh: true),
            child: ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: provider.pendingReviews.length + 1,
              itemBuilder: (context, index) {
                if (index == provider.pendingReviews.length) {
                  if (provider.pendingReviews.length < provider.reviewTotal) {
                    provider.listPendingReviews();
                    return const Padding(
                      padding: EdgeInsets.all(16),
                      child: Center(child: CircularProgressIndicator()),
                    );
                  }
                  return const SizedBox.shrink();
                }
                return _ReviewCard(resource: provider.pendingReviews[index]);
              },
            ),
          );
        },
      ),
    );
  }
}

class _ReviewCard extends StatelessWidget {
  final KnowledgeCard resource;
  const _ReviewCard({required this.resource});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
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
                    color: theme.colorScheme.tertiaryContainer,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(resource.typeLabel,
                      style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onTertiaryContainer)),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child:
                      Text(resource.title, style: theme.textTheme.titleSmall),
                ),
              ],
            ),
            if (resource.summary.isNotEmpty) ...[
              const SizedBox(height: 6),
              MdText(resource.summary,
                  style: theme.textTheme.bodySmall),            ],
            if (resource.tags.isNotEmpty) ...[
              const SizedBox(height: 6),
              Wrap(
                spacing: 4,
                children: resource.tags.map((t) {
                  return Chip(
                    label: Text(t, style: const TextStyle(fontSize: 10)),
                    padding: EdgeInsets.zero,
                    visualDensity: VisualDensity.compact,
                    materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  );
                }).toList(),
              ),
            ],
            const SizedBox(height: 10),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                OutlinedButton.icon(
                  onPressed: () => _handleReject(context),
                  icon: const Icon(Icons.close, size: 16),
                  label: const Text('驳回'),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: theme.colorScheme.error,
                    visualDensity: VisualDensity.compact,
                  ),
                ),
                const SizedBox(width: 8),
                OutlinedButton.icon(
                  onPressed: () => _handlePreview(context),
                  icon: const Icon(Icons.visibility_outlined, size: 16),
                  label: const Text('预览'),
                  style: OutlinedButton.styleFrom(
                      visualDensity: VisualDensity.compact),
                ),
                const SizedBox(width: 8),
                OutlinedButton.icon(
                  onPressed: () => _handlePrint(context),
                  icon: const Icon(Icons.print_outlined, size: 16),
                  label: const Text('打印'),
                  style: OutlinedButton.styleFrom(
                      visualDensity: VisualDensity.compact),
                ),
                const SizedBox(width: 8),
                FilledButton.icon(
                  onPressed: () => _handleApprove(context),
                  icon: const Icon(Icons.check, size: 16),
                  label: const Text('通过'),
                  style: FilledButton.styleFrom(
                    visualDensity: VisualDensity.compact,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _handleApprove(BuildContext context) async {
    final provider = context.read<KnowledgeProvider>();
    final ok = await provider.approveResource(resource.resourceId);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '审核通过' : '操作失败')),
      );
      if (ok) provider.listPendingReviews(refresh: true);
    }
  }

  Future<KnowledgeCard> _fullResource(BuildContext context) async {
    if (resource.content.isNotEmpty) return resource;
    return await context
            .read<KnowledgeProvider>()
            .getResource(resource.resourceId) ??
        resource;
  }

  Future<void> _handlePreview(BuildContext context) async {
    final full = await _fullResource(context);
    if (!context.mounted) return;
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(full.title),
        content: SizedBox(
          width: 720,
          child: SingleChildScrollView(
            child: MdText(
                full.content.isEmpty ? full.summary : full.content),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context), child: const Text('关闭'))
        ],
      ),
    );
  }

  Future<void> _handlePrint(BuildContext context) async {
    final full = await _fullResource(context);
    openHtmlInNewTab(
        '''<!doctype html><html><head><meta charset="utf-8"><title>${_escapeHtml(full.title)}</title><style>body{font-family:"Microsoft YaHei",sans-serif;line-height:1.8;padding:32px;max-width:860px;margin:auto}pre{white-space:pre-wrap}</style></head><body><h1>${_escapeHtml(full.title)}</h1><p>${_escapeHtml(full.summary)}</p><pre>${_escapeHtml(full.content)}</pre><script>window.print()</script></body></html>''');
  }

  static String _escapeHtml(String input) => input
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;');

  Future<void> _handleReject(BuildContext context) async {
    final provider = context.read<KnowledgeProvider>();
    final ok = await provider.rejectResource(resource.resourceId);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已驳回' : '操作失败')),
      );
      if (ok) provider.listPendingReviews(refresh: true);
    }
  }
}
