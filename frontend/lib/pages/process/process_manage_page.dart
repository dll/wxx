import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/process_provider.dart';
import '../../utils/capability_utils.dart';
import '../../widgets/error_view.dart';
import 'process_document.dart';
import 'process_editor_page.dart';

class ProcessManagePage extends StatefulWidget {
  const ProcessManagePage({super.key});

  @override
  State<ProcessManagePage> createState() => _ProcessManagePageState();
}

class _ProcessManagePageState extends State<ProcessManagePage> {
  final _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ProcessProvider>().loadAdmin(refresh: true);
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<ProcessProvider>();
    final canWrite = CapabilityUtils.has(Capability.counselorKbWrite);
    return Scaffold(
      appBar: AppBar(
        title: const Text('办事流程管理'),
        actions: [
          if (canWrite)
            IconButton(
              tooltip: '新建流程',
              onPressed: () => _openEditor(),
              icon: const Icon(Icons.add),
            ),
          IconButton(
            tooltip: '刷新',
            onPressed: () => provider.loadAdmin(refresh: true),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: '搜索流程标题、资源ID、标签...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear, size: 20),
                        onPressed: () {
                          _searchController.clear();
                          provider.setAdminKeyword('');
                        },
                      )
                    : null,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                ),
                isDense: true,
              ),
              onSubmitted: (v) => provider.setAdminKeyword(v),
            ),
          ),
          SizedBox(
            height: 52,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.symmetric(horizontal: 12),
              children: [
                _statusChip(context, '', '全部'),
                _statusChip(context, 'draft', '草稿'),
                _statusChip(context, 'pending', '待审核'),
                _statusChip(context, 'published', '已发布'),
                _statusChip(context, 'retired', '已下架'),
              ],
            ),
          ),
          Expanded(child: _buildBody(theme, provider)),
        ],
      ),
    );
  }

  Widget _statusChip(BuildContext context, String status, String label) {
    final selected = context.watch<ProcessProvider>().adminStatus == status;
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: FilterChip(
        label: Text(label),
        selected: selected,
        showCheckmark: false,
        onSelected: (_) =>
            context.read<ProcessProvider>().setAdminStatus(status),
      ),
    );
  }

  Widget _buildBody(ThemeData theme, ProcessProvider provider) {
    if (provider.adminLoading && provider.adminResources.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (provider.error != null && provider.adminResources.isEmpty) {
      return ErrorView.error(
        message: provider.error!,
        onRetry: () => provider.loadAdmin(refresh: true),
      );
    }
    if (provider.adminResources.isEmpty) {
      return ErrorView.empty(
        message: '暂无办事流程',
        icon: Icons.account_tree_outlined,
      );
    }
    return RefreshIndicator(
      onRefresh: () => provider.loadAdmin(refresh: true),
      child: ListView.builder(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 24),
        itemCount: provider.adminResources.length,
        itemBuilder: (context, index) {
          final def = provider.adminResources[index];
          return _ProcessAdminCard(
            definition: def,
            onPreview: () => _preview(def),
            onEdit: () => _openEditor(def),
            onPrint: () => openProcessPrint(def),
            onExport: () => showProcessExportDialog(context, def),
            onSubmit: () => _submit(def),
            onRetire: () => _retire(def),
            onDelete: () => _delete(def),
          );
        },
      ),
    );
  }

  Future<void> _openEditor([ProcessDefinition? def]) async {
    await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => ProcessEditorPage(definition: def)),
    );
    if (mounted) context.read<ProcessProvider>().loadAdmin(refresh: true);
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
                  SelectableText(def.content),
                ],
                if (def.steps.isNotEmpty) ...[
                  const SizedBox(height: 12),
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
                  const SizedBox(height: 8),
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

  Future<void> _submit(ProcessDefinition def) async {
    final messenger = ScaffoldMessenger.of(context);
    final ok =
        await context.read<ProcessProvider>().submitForReview(def.resourceId);
    if (!mounted) return;
    messenger.showSnackBar(SnackBar(content: Text(ok ? '已提交审核' : '提交失败')));
  }

  Future<void> _retire(ProcessDefinition def) async {
    final messenger = ScaffoldMessenger.of(context);
    final ok =
        await context.read<ProcessProvider>().retireProcess(def.resourceId);
    if (!mounted) return;
    messenger.showSnackBar(SnackBar(content: Text(ok ? '已下架' : '下架失败')));
  }

  Future<void> _delete(ProcessDefinition def) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认删除'),
        content: Text('确定删除「${def.title}」吗？步骤和提醒会一并删除，不可恢复。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final ok =
        await context.read<ProcessProvider>().deleteProcess(def.resourceId);
    if (!mounted) return;
    messenger.showSnackBar(SnackBar(content: Text(ok ? '删除成功' : '删除失败')));
  }
}

class _ProcessAdminCard extends StatelessWidget {
  final ProcessDefinition definition;
  final VoidCallback onPreview;
  final VoidCallback onEdit;
  final VoidCallback onPrint;
  final VoidCallback onExport;
  final VoidCallback onSubmit;
  final VoidCallback onRetire;
  final VoidCallback onDelete;

  const _ProcessAdminCard({
    required this.definition,
    required this.onPreview,
    required this.onEdit,
    required this.onPrint,
    required this.onExport,
    required this.onSubmit,
    required this.onRetire,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final def = definition;
    final submitted = def.status == 'pending' || def.status == 'published';
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.account_tree_outlined,
                    color: theme.colorScheme.primary),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(def.title,
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w600)),
                ),
                _statusBadge(def.status, theme),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              '${def.resourceId} · ${def.ownerScope} · ${def.steps.length} 个步骤 · ${def.reminders.where((r) => r.isEnabled).length} 条提醒 · ${def.updatedBy}',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            ),
            if (def.summary.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(def.summary,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall),
            ],
            const SizedBox(height: 10),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                OutlinedButton.icon(
                  onPressed: onPreview,
                  icon: const Icon(Icons.visibility_outlined, size: 16),
                  label: const Text('预览'),
                ),
                OutlinedButton.icon(
                  onPressed: onEdit,
                  icon: const Icon(Icons.edit_outlined, size: 16),
                  label: const Text('编辑'),
                ),
                OutlinedButton.icon(
                  onPressed: onPrint,
                  icon: const Icon(Icons.print_outlined, size: 16),
                  label: const Text('打印'),
                ),
                OutlinedButton.icon(
                  onPressed: onExport,
                  icon: const Icon(Icons.download_outlined, size: 16),
                  label: const Text('导出'),
                ),
                FilledButton.tonal(
                  onPressed: submitted ? null : onSubmit,
                  child: Text(submitted ? '已提交' : '提交审核'),
                ),
                if (def.status == 'published')
                  OutlinedButton.icon(
                    onPressed: onRetire,
                    icon: const Icon(Icons.visibility_off_outlined, size: 16),
                    label: const Text('下架'),
                  ),
                OutlinedButton.icon(
                  onPressed: onDelete,
                  icon: Icon(Icons.delete_outline,
                      size: 16, color: theme.colorScheme.error),
                  label: Text('删除',
                      style: TextStyle(color: theme.colorScheme.error)),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusBadge(String status, ThemeData theme) {
    final map = {
      'draft': ('草稿', Colors.grey),
      'pending': ('待审核', Colors.orange),
      'published': ('已发布', Colors.green),
      'retired': ('已下架', theme.colorScheme.error),
    };
    final entry = map[status] ?? (status, theme.colorScheme.outline);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: entry.$2.withOpacity(0.1),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(
        entry.$1,
        style: TextStyle(
          fontSize: 11,
          color: entry.$2,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}
