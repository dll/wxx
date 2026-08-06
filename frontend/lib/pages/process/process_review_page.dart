import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/process_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/md_text.dart';
import 'process_document.dart';

class ProcessReviewPage extends StatefulWidget {
  const ProcessReviewPage({super.key});

  @override
  State<ProcessReviewPage> createState() => _ProcessReviewPageState();
}

class _ProcessReviewPageState extends State<ProcessReviewPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ProcessProvider>().loadPending(refresh: true);
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<ProcessProvider>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('办事流程审核'),
        actions: [
          IconButton(
            tooltip: '刷新',
            onPressed: () => provider.loadPending(refresh: true),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: provider.reviewLoading && provider.pendingReviews.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : provider.pendingReviews.isEmpty
              ? ErrorView.empty(
                  message: '暂无待审核流程',
                  icon: Icons.rate_review_outlined,
                )
              : RefreshIndicator(
                  onRefresh: () => provider.loadPending(refresh: true),
                  child: ListView.builder(
                    padding: const EdgeInsets.all(12),
                    itemCount: provider.pendingReviews.length,
                    itemBuilder: (context, index) {
                      final def = provider.pendingReviews[index];
                      return _ReviewCard(
                        definition: def,
                        onApprove: () => _approve(def),
                        onReject: () => _reject(def),
                        onPreview: () => _preview(def),
                        onPrint: () => openProcessPrint(def),
                      );
                    },
                  ),
                ),
    );
  }

  Future<void> _approve(ProcessDefinition def) async {
    final ok =
        await context.read<ProcessProvider>().approveProcess(def.resourceId);
    if (!mounted) return;
    _toast(ok ? '审核通过，已发布' : '操作失败');
  }

  Future<void> _reject(ProcessDefinition def) async {
    final reasonController = TextEditingController();
    final reason = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('驳回流程'),
        content: TextField(
          controller: reasonController,
          maxLines: 3,
          decoration: const InputDecoration(
            labelText: '驳回原因',
            border: OutlineInputBorder(),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, reasonController.text.trim()),
            child: const Text('确认驳回'),
          ),
        ],
      ),
    );
    reasonController.dispose();
    if (reason == null || !mounted) return;
    final ok = await context
        .read<ProcessProvider>()
        .rejectProcess(def.resourceId, reason);
    if (!mounted) return;
    _toast(ok ? '已驳回' : '驳回失败');
  }

  Future<void> _preview(ProcessDefinition def) async {
    showDialog<void>(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(def.title),
        content: SizedBox(
          width: 760,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('${def.statusLabel} · ${def.resourceId}'),
                if (def.summary.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(def.summary),
                ],
                if (def.content.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  const Divider(),
                  MdText(def.content),
                ],
                if (def.steps.isNotEmpty) ...[
                  const Divider(),
                  for (final s in def.steps)
                    ListTile(
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      leading: CircleAvatar(
                        radius: 14,
                        child: Text('${s.stepOrder}'),
                      ),
                      title: Text(s.title),
                      subtitle: Text([
                        if (s.deadline.isNotEmpty) '截止：${s.deadline}',
                        if (s.location.isNotEmpty) '地点：${s.location}',
                        if (s.contact.isNotEmpty) '联系：${s.contact}',
                      ].join(' · ')),
                    ),
                ],
                if (def.reminders.where((r) => r.isEnabled).isNotEmpty) ...[
                  const Divider(),
                  for (final r in def.reminders.where((r) => r.isEnabled))
                    ListTile(
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      leading: const Icon(Icons.alarm),
                      title: Text('${r.remindAt} · ${r.title}'),
                      subtitle: Text(r.content),
                    ),
                ],
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('关闭'),
          ),
          TextButton.icon(
            onPressed: () {
              Navigator.pop(context);
              openProcessPrint(def);
            },
            icon: const Icon(Icons.print),
            label: const Text('打印'),
          ),
        ],
      ),
    );
  }

  void _toast(String message) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(message)));
  }
}

class _ReviewCard extends StatelessWidget {
  final ProcessDefinition definition;
  final VoidCallback onApprove;
  final VoidCallback onReject;
  final VoidCallback onPreview;
  final VoidCallback onPrint;

  const _ReviewCard({
    required this.definition,
    required this.onApprove,
    required this.onReject,
    required this.onPreview,
    required this.onPrint,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final def = definition;
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
                    color: Colors.orange.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: const Text('待审核',
                      style: TextStyle(fontSize: 11, color: Colors.orange)),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(def.title,
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w600)),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              '${def.resourceId} · ${def.steps.length} 个步骤 · ${def.reminders.length} 条提醒 · ${def.updatedBy}',
              style: theme.textTheme.bodySmall,
            ),
            if (def.summary.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(def.summary,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall),
            ],
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                OutlinedButton.icon(
                  onPressed: onReject,
                  icon: const Icon(Icons.close, size: 16),
                  label: const Text('驳回'),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: theme.colorScheme.error,
                  ),
                ),
                OutlinedButton.icon(
                  onPressed: onPreview,
                  icon: const Icon(Icons.visibility_outlined, size: 16),
                  label: const Text('预览'),
                ),
                OutlinedButton.icon(
                  onPressed: onPrint,
                  icon: const Icon(Icons.print_outlined, size: 16),
                  label: const Text('打印'),
                ),
                FilledButton.icon(
                  onPressed: onApprove,
                  icon: const Icon(Icons.check, size: 16),
                  label: const Text('通过'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
