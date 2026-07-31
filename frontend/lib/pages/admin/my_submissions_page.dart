import 'dart:convert';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/knowledge_provider.dart';
import '../../utils/capability_utils.dart';
import '../../utils/web_export.dart';
import '../../widgets/error_view.dart';

/// 知识治理页面（增强版）
/// 功能：搜索、多条件筛选、批量选择、批量操作、统计、预览、编辑
class KnowledgeGovernancePage extends StatefulWidget {
  const KnowledgeGovernancePage({super.key});

  @override
  State<KnowledgeGovernancePage> createState() => _KnowledgeGovernancePageState();
}

class _KnowledgeGovernancePageState extends State<KnowledgeGovernancePage> {
  final _scrollController = ScrollController();
  final _searchController = TextEditingController();
  bool _showAdvancedFilter = false;
  bool _dictLoaded = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (CapabilityUtils.has(Capability.counselorKbWrite)) {
        context.read<KnowledgeProvider>().searchResources(refresh: true);
        context.read<KnowledgeProvider>().fetchStats();
      }
    });
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      final provider = context.read<KnowledgeProvider>();
      if (!provider.resourcesLoading &&
          provider.resources.length < provider.resourceTotal) {
        provider.searchResources();
      }
    }
  }

  Future<void> _loadDicts() async {
    if (_dictLoaded) return;
    _dictLoaded = true;
    final provider = context.read<KnowledgeProvider>();
    await provider.fetchDictValues('resource_type');
    await provider.fetchDictValues('status');
    await provider.fetchDictValues('owner_scope');
  }

  @override
  Widget build(BuildContext context) {
    final canWrite = CapabilityUtils.has(Capability.counselorKbWrite);
    final canReview = CapabilityUtils.has(Capability.counselorKbReview);

    return Scaffold(
      appBar: AppBar(
        title: const Text('知识治理'),
        actions: [
          if (canWrite)
            IconButton(
              icon: const Icon(Icons.add),
              tooltip: '创建知识资源',
              onPressed: () => _showCreateDialog(),
            ),
        ],
      ),
      body: canWrite
          ? Column(
              children: [
                _buildSearchBar(),
                if (_showAdvancedFilter) _buildAdvancedFilter(),
                _buildStatsBar(),
                if (canReview || canWrite) _buildBatchActionBar(),
                Expanded(child: _buildResourceList()),
              ],
            )
          : _buildNoPermission(context),
    );
  }

  Widget _buildNoPermission(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.lock_outline,
                    size: 56,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(height: 16),
                  Text('无访问权限', style: theme.textTheme.titleLarge),
                  const SizedBox(height: 8),
                  const Text(
                    '当前角色无权访问知识治理功能。',
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  // ── 搜索栏 ──

  Widget _buildSearchBar() {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 6),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(
          bottom: BorderSide(
            color: Theme.of(context).colorScheme.outlineVariant.withOpacity(0.3),
          ),
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: '搜索标题、摘要、内容、标签...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear, size: 20),
                        onPressed: () {
                          _searchController.clear();
                          context.read<KnowledgeProvider>().setKeyword('');
                        },
                      )
                    : null,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                ),
                isDense: true,
                contentPadding: const EdgeInsets.symmetric(vertical: 10),
              ),
              onSubmitted: (value) {
                context.read<KnowledgeProvider>().setKeyword(value);
              },
              onChanged: (value) {
                if (value.isEmpty) {
                  context.read<KnowledgeProvider>().setKeyword('');
                }
              },
            ),
          ),
          const SizedBox(width: 8),
          IconButton.filledTonal(
            onPressed: () {
              setState(() {
                _showAdvancedFilter = !_showAdvancedFilter;
                if (_showAdvancedFilter) _loadDicts();
              });
            },
            icon: Icon(
              _showAdvancedFilter
                  ? Icons.filter_alt
                  : Icons.filter_alt_outlined,
            ),
            tooltip: '高级筛选',
          ),
        ],
      ),
    );
  }

  // ── 高级筛选面板 ──

  Widget _buildAdvancedFilter() {
    return Consumer<KnowledgeProvider>(
      builder: (_, provider, __) {
        return Container(
          padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
          decoration: BoxDecoration(
            color: Theme.of(context)
                .colorScheme
                .surfaceContainerHighest
                .withOpacity(0.3),
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context)
                    .colorScheme
                    .outlineVariant
                    .withOpacity(0.3),
              ),
            ),
          ),
          child: Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _buildFilterDropdown(
                label: '类型',
                value: provider.resourceTypeFilter.isEmpty
                    ? null
                    : provider.resourceTypeFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部类型')),
                  const DropdownMenuItem(value: 'Policy', child: Text('政策')),
                  const DropdownMenuItem(value: 'Process', child: Text('流程')),
                  const DropdownMenuItem(value: 'FAQ', child: Text('问答')),
                  const DropdownMenuItem(value: 'Activity', child: Text('活动')),
                ],
                onChanged: (v) =>
                    provider.setResourceFilter(resourceType: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '状态',
                value: provider.statusFilter.isEmpty
                    ? null
                    : provider.statusFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部状态')),
                  const DropdownMenuItem(value: 'draft', child: Text('草稿')),
                  const DropdownMenuItem(value: 'pending', child: Text('待审核')),
                  const DropdownMenuItem(
                      value: 'published', child: Text('已发布')),
                  const DropdownMenuItem(value: 'retired', child: Text('已下架')),
                ],
                onChanged: (v) => provider.setResourceFilter(status: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '范围',
                value: provider.ownerScopeFilter.isEmpty
                    ? null
                    : provider.ownerScopeFilter,
                items: [
                  const DropdownMenuItem(value: '', child: Text('全部范围')),
                  const DropdownMenuItem(value: 'school', child: Text('学校')),
                  const DropdownMenuItem(value: 'college', child: Text('学院')),
                  const DropdownMenuItem(value: 'class', child: Text('班级')),
                ],
                onChanged: (v) =>
                    provider.setResourceFilter(ownerScope: v ?? ''),
              ),
              _buildFilterDropdown(
                label: '排序',
                value: provider.selectedCount >= 0 ? 'updated_at' : null,
                items: const [
                  DropdownMenuItem(value: 'updated_at', child: Text('更新时间')),
                  DropdownMenuItem(value: 'created_at', child: Text('创建时间')),
                  DropdownMenuItem(value: 'title', child: Text('标题')),
                  DropdownMenuItem(
                      value: 'resource_type', child: Text('类型')),
                ],
                onChanged: (v) {
                  if (v != null) {
                    provider.setResourceFilter(sortBy: v);
                  }
                },
              ),
              TextButton.icon(
                onPressed: () {
                  _searchController.clear();
                  provider.resetResourceFilters();
                },
                icon: const Icon(Icons.refresh, size: 18),
                label: const Text('重置'),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildFilterDropdown({
    required String label,
    required String? value,
    required List<DropdownMenuItem<String>> items,
    required ValueChanged<String?> onChanged,
  }) {
    return SizedBox(
      width: 130,
      child: DropdownButtonFormField<String>(
        value: value,
        decoration: InputDecoration(
          labelText: label,
          border: const OutlineInputBorder(),
          isDense: true,
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        ),
        items: items,
        onChanged: onChanged,
      ),
    );
  }

  // ── 统计栏 ──

  Widget _buildStatsBar() {
    return Consumer<KnowledgeProvider>(
      builder: (_, provider, __) {
        final stats = provider.kbStats;
        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerLowest,
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context)
                    .colorScheme
                    .outlineVariant
                    .withOpacity(0.3),
              ),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text(
                    '共 ${provider.resourceTotal} 条',
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                  ),
                  const SizedBox(width: 16),
                  if (stats != null) ...[
                    _statChip('草稿', stats['draft'] ?? 0, Colors.grey),
                    _statChip('待审', stats['pending'] ?? 0, Colors.orange),
                    _statChip('已发', stats['published'] ?? 0, Colors.green),
                    _statChip('下架', stats['retired'] ?? 0, Colors.red),
                  ],
                  const Spacer(),
                  if (provider.selectedCount > 0) ...[
                    Container(
                      padding:
                          const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: Theme.of(context).colorScheme.primaryContainer,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        '已选 ${provider.selectedCount} 条',
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.onPrimaryContainer,
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    TextButton(
                      onPressed: provider.deselectAllResources,
                      child: const Text('取消选择', style: TextStyle(fontSize: 12)),
                    ),
                  ],
                ],
              ),
              if (stats != null && stats['by_type'] != null) ...[
                const SizedBox(height: 6),
                Row(
                  children: [
                    Icon(Icons.category_outlined, size: 14,
                        color: Theme.of(context).colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text('类型分布：',
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: Theme.of(context).colorScheme.onSurfaceVariant,
                            )),
                    const SizedBox(width: 4),
                    ..._buildTypeStats(stats['by_type']),
                  ],
                ),
              ],
            ],
          ),
        );
      },
    );
  }

  List<Widget> _buildTypeStats(dynamic byType) {
    if (byType is! Map) return [];
    const typeLabels = {
      'Policy': '政策',
      'Process': '流程',
      'FAQ': '问答',
      'Activity': '活动',
    };
    const typeColors = {
      'Policy': Colors.blue,
      'Process': Colors.purple,
      'FAQ': Colors.green,
      'Activity': Colors.orange,
    };
    final widgets = <Widget>[];
    for (final entry in (byType as Map).entries) {
      final label = typeLabels[entry.key] ?? entry.key.toString();
      final color = typeColors[entry.key] ?? Colors.grey;
      final count = entry.value is int ? entry.value : 0;
      widgets.add(Padding(
        padding: const EdgeInsets.only(right: 8),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                color: color,
                shape: BoxShape.circle,
              ),
            ),
            const SizedBox(width: 3),
            Text('$label $count',
                style: TextStyle(fontSize: 11, color: color)),
          ],
        ),
      ));
    }
    return widgets;
  }

  Widget _statChip(String label, dynamic count, Color color) {
    final num = count is int ? count : 0;
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
        decoration: BoxDecoration(
          color: color.withOpacity(0.1),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: color.withOpacity(0.3)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              '$num',
              style: TextStyle(
                color: color,
                fontWeight: FontWeight.w600,
                fontSize: 12,
              ),
            ),
            const SizedBox(width: 4),
            Text(
              label,
              style: TextStyle(
                color: color.withOpacity(0.8),
                fontSize: 11,
              ),
            ),
          ],
        ),
      ),
    );
  }

  // ── 批量操作栏 ──

  Widget _buildBatchActionBar() {
    return Consumer<KnowledgeProvider>(
      builder: (_, provider, __) {
        if (provider.selectedCount == 0) return const SizedBox.shrink();
        final canReview = CapabilityUtils.has(Capability.counselorKbReview);
        final canWrite = CapabilityUtils.has(Capability.counselorKbWrite);

        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color:
                Theme.of(context).colorScheme.primaryContainer.withOpacity(0.2),
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context)
                    .colorScheme
                    .outlineVariant
                    .withOpacity(0.3),
              ),
            ),
          ),
          child: Row(
            children: [
              Icon(
                Icons.check_circle,
                size: 20,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(width: 6),
              Text(
                '已选择 ${provider.selectedCount} 条资源',
                style: TextStyle(
                  fontWeight: FontWeight.w600,
                  color: Theme.of(context).colorScheme.primary,
                ),
              ),
              const Spacer(),
              if (canReview) ...[
                _batchActionButton(
                  icon: Icons.check_circle_outline,
                  label: '通过',
                  color: Colors.green,
                  onPressed: () => _confirmBatchAction(
                    title: '批量审核通过',
                    content: '确定通过选中的 ${provider.selectedCount} 条资源吗？',
                    onConfirm: () => provider.batchApprove(),
                    successMsg: '批量审核通过成功',
                  ),
                ),
                const SizedBox(width: 6),
                _batchActionButton(
                  icon: Icons.undo,
                  label: '驳回',
                  color: Colors.orange,
                  onPressed: () => _confirmBatchAction(
                    title: '批量驳回',
                    content: '确定驳回选中的 ${provider.selectedCount} 条资源吗？',
                    onConfirm: () => provider.batchReject(),
                    successMsg: '批量驳回成功',
                  ),
                ),
                const SizedBox(width: 6),
                _batchActionButton(
                  icon: Icons.download_done_outlined,
                  label: '下架',
                  color: Colors.red,
                  onPressed: () => _confirmBatchAction(
                    title: '批量下架',
                    content: '确定下架选中的 ${provider.selectedCount} 条资源吗？',
                    onConfirm: () => provider.batchRetire(),
                    successMsg: '批量下架成功',
                  ),
                ),
              ],
              if (canWrite) ...[
                const SizedBox(width: 6),
                _batchActionButton(
                  icon: Icons.delete_outline,
                  label: '删除',
                  color: Theme.of(context).colorScheme.error,
                  onPressed: () => _confirmBatchAction(
                    title: '批量删除',
                    content:
                        '确定删除选中的 ${provider.selectedCount} 条资源吗？\n该操作不可恢复！',
                    onConfirm: provider.batchDelete,
                    successMsg: '批量删除成功',
                    isDestructive: true,
                  ),
                ),
              ],
            ],
          ),
        );
      },
    );
  }

  Widget _batchActionButton({
    required IconData icon,
    required String label,
    required Color color,
    required VoidCallback onPressed,
  }) {
    return TextButton.icon(
      onPressed: onPressed,
      icon: Icon(icon, size: 18, color: color),
      label: Text(label, style: TextStyle(color: color, fontSize: 13)),
      style: TextButton.styleFrom(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        minimumSize: Size.zero,
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
    );
  }

  Future<void> _confirmBatchAction({
    required String title,
    required String content,
    required Future<bool> Function() onConfirm,
    required String successMsg,
    bool isDestructive = false,
  }) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            style: isDestructive
                ? FilledButton.styleFrom(
                    backgroundColor: Theme.of(ctx).colorScheme.error,
                  )
                : null,
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('确认'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    final ok = await onConfirm();
    if (!mounted) return;
    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(successMsg)),
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(context.read<KnowledgeProvider>().resourceError),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  // ── 资源列表 ──

  Widget _buildResourceList() {
    return Consumer<KnowledgeProvider>(
      builder: (_, provider, __) {
        if (provider.resourcesLoading && provider.resources.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.resourceError.isNotEmpty && provider.resources.isEmpty) {
          return ErrorView.error(
            message: provider.resourceError,
            onRetry: () => provider.searchResources(refresh: true),
          );
        }
        if (provider.resources.isEmpty) {
          return ErrorView.empty(
            message: '暂无知识资源',
            icon: Icons.menu_book_outlined,
          );
        }

        return RefreshIndicator(
          onRefresh: () => provider.searchResources(refresh: true),
          child: ListView.builder(
            controller: _scrollController,
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 24),
            itemCount: provider.resources.length +
                (provider.resources.length < provider.resourceTotal ? 1 : 0),
            itemBuilder: (context, index) {
              if (index == provider.resources.length) {
                return const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                );
              }
              final resource = provider.resources[index];
              final selected =
                  provider.selectedResourceIds.contains(resource.resourceId);
              return _ResourceTile(
                resource: resource,
                selected: selected,
                showCheckbox: true,
                onSelectChanged: (v) =>
                    provider.toggleSelectResource(resource.resourceId),
                onPreview: () => _previewResource(provider, resource),
                onEdit: () => _editResource(provider, resource),
                onPrint: () => _printResource(provider, resource),
                onSubmit: () =>
                    _handleSubmit(provider, resource.resourceId),
                onDelete: () => _handleDelete(provider, resource),
              );
            },
          ),
        );
      },
    );
  }

  Future<KnowledgeCard> _fullResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    if (r.content.isNotEmpty) return r;
    return await provider.getResource(r.resourceId) ?? r;
  }

  Future<void> _previewResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    final full = await _fullResource(provider, r);
    if (!mounted) return;
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: Row(
          children: [
            _typeIcon(r.resourceType),
            const SizedBox(width: 8),
            Expanded(child: Text(full.title)),
          ],
        ),
        content: SizedBox(
          width: 720,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (full.summary.isNotEmpty) ...[
                Text(full.summary,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        )),
                const SizedBox(height: 12),
              ],
              Expanded(
                child: SingleChildScrollView(
                  child: SelectableText(
                      full.content.isEmpty ? full.summary : full.content),
                ),
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: full.tags
                    .map((tag) => Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 2),
                          decoration: BoxDecoration(
                            color: Theme.of(context)
                                .colorScheme
                                .secondaryContainer
                                .withOpacity(0.5),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Text(
                            tag,
                            style: TextStyle(
                              fontSize: 11,
                              color: Theme.of(context)
                                  .colorScheme
                                  .onSecondaryContainer,
                            ),
                          ),
                        ))
                    .toList(),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('关闭'))
        ],
      ),
    );
  }

  Widget _typeIcon(String type) {
    IconData icon;
    Color color;
    switch (type) {
      case 'Policy':
        icon = Icons.policy_outlined;
        color = Colors.blue;
        break;
      case 'Process':
        icon = Icons.route_outlined;
        color = Colors.purple;
        break;
      case 'FAQ':
        icon = Icons.question_answer_outlined;
        color = Colors.green;
        break;
      case 'Activity':
        icon = Icons.event_outlined;
        color = Colors.orange;
        break;
      default:
        icon = Icons.article_outlined;
        color = Colors.grey;
    }
    return CircleAvatar(
      radius: 16,
      backgroundColor: color.withOpacity(0.1),
      child: Icon(icon, size: 18, color: color),
    );
  }

  Future<void> _editResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    final full = await _fullResource(provider, r);
    if (!mounted) return;
    showDialog(
        context: context,
        builder: (_) => _CreateResourceDialog(resource: full)).then((_) {
      if (mounted) provider.searchResources(refresh: true);
    });
  }

  Future<void> _printResource(
      KnowledgeProvider provider, KnowledgeCard r) async {
    final full = await _fullResource(provider, r);
    openHtmlInNewTab(
        '''<!doctype html><html><head><meta charset="utf-8"><title>${_escapeHtml(full.title)}</title><style>body{font-family:"Microsoft YaHei",sans-serif;line-height:1.8;padding:32px;max-width:860px;margin:auto}pre{white-space:pre-wrap}</style></head><body><h1>${_escapeHtml(full.title)}</h1><p>${_escapeHtml(full.summary)}</p><pre>${_escapeHtml(full.content)}</pre><script>window.print()</script></body></html>''');
  }

  Future<void> _handleSubmit(
      KnowledgeProvider provider, String resourceId) async {
    final ok = await provider.submitForReview(resourceId);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已提交审核' : '提交失败')),
      );
      if (ok) provider.searchResources(refresh: true);
    }
  }

  Future<void> _handleDelete(
      KnowledgeProvider provider, KnowledgeCard resource) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认删除'),
        content: Text('确定删除「${resource.title}」吗？\n该操作不可恢复！'),
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
    if (confirmed != true) return;

    final ok = await provider.deleteResource(resource.resourceId);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '删除成功' : '删除失败')),
      );
    }
  }

  void _showCreateDialog() {
    showDialog(context: context, builder: (_) => const _CreateResourceDialog())
        .then((_) {
      if (mounted) {
        context.read<KnowledgeProvider>().searchResources(refresh: true);
        context.read<KnowledgeProvider>().fetchStats();
      }
    });
  }

  static String _escapeHtml(String input) => input
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;');
}

// ── 资源列表项 ──

class _ResourceTile extends StatelessWidget {
  final KnowledgeCard resource;
  final bool selected;
  final bool showCheckbox;
  final ValueChanged<bool?>? onSelectChanged;
  final VoidCallback onPreview;
  final VoidCallback onEdit;
  final VoidCallback onPrint;
  final VoidCallback onSubmit;
  final VoidCallback onDelete;

  const _ResourceTile({
    required this.resource,
    required this.selected,
    required this.showCheckbox,
    this.onSelectChanged,
    required this.onPreview,
    required this.onEdit,
    required this.onPrint,
    required this.onSubmit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final submitted =
        resource.status == 'pending' || resource.status == 'published';

    return Card(
      margin: const EdgeInsets.only(bottom: 6),
      elevation: selected ? 2 : 0,
      color: selected
          ? theme.colorScheme.primaryContainer.withOpacity(0.15)
          : null,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: selected
              ? theme.colorScheme.primary.withOpacity(0.5)
              : theme.colorScheme.outlineVariant.withOpacity(0.3),
          width: selected ? 1.5 : 1,
        ),
      ),
      child: InkWell(
        onTap: onPreview,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (showCheckbox)
                Checkbox(
                  value: selected,
                  onChanged: onSelectChanged,
                  visualDensity: VisualDensity.compact,
                ),
              _typeIconWidget(resource.resourceType),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Flexible(
                          child: Text(
                            resource.title,
                            style: theme.textTheme.titleSmall
                                ?.copyWith(fontWeight: FontWeight.w600),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 8),
                        _statusBadge(resource.status, theme),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '${resource.typeLabel} · ${resource.resourceId}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    if (resource.updatedBy.isNotEmpty) ...[
                      const SizedBox(height: 2),
                      Text(
                        '上传者：${resource.updatedBy}',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                          fontSize: 11,
                        ),
                      ),
                    ],
                    if (resource.remark.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        '备注：${resource.remark}',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                          fontSize: 11,
                          fontStyle: FontStyle.italic,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                    if (resource.summary.isNotEmpty) ...[
                      const SizedBox(height: 6),
                      Text(
                        resource.summary,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 6,
                      runSpacing: 6,
                      children: resource.tags.take(3).map((tag) =>
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 1),
                          decoration: BoxDecoration(
                            color: theme.colorScheme.surfaceContainerHighest,
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Text(
                            tag,
                            style: TextStyle(
                              fontSize: 10,
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        )
                      ).toList(),
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 6,
                      runSpacing: 6,
                      children: [
                        OutlinedButton.icon(
                            onPressed: onPreview,
                            icon: const Icon(Icons.visibility_outlined, size: 16),
                            label: const Text('预览')),
                        OutlinedButton.icon(
                            onPressed: onEdit,
                            icon: const Icon(Icons.edit_outlined, size: 16),
                            label: const Text('编辑')),
                        OutlinedButton.icon(
                            onPressed: onPrint,
                            icon: const Icon(Icons.print_outlined, size: 16),
                            label: const Text('打印')),
                        OutlinedButton.icon(
                            onPressed: onDelete,
                            icon: Icon(Icons.delete_outline,
                                size: 16,
                                color: theme.colorScheme.error),
                            label: Text('删除',
                                style: TextStyle(
                                    color: theme.colorScheme.error))),
                        FilledButton.tonal(
                            onPressed: submitted ? null : onSubmit,
                            child: Text(submitted ? '已提交' : '提交审核')),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _typeIconWidget(String type) {
    IconData icon;
    Color color;
    switch (type) {
      case 'Policy':
        icon = Icons.policy_outlined;
        color = Colors.blue;
        break;
      case 'Process':
        icon = Icons.route_outlined;
        color = Colors.purple;
        break;
      case 'FAQ':
        icon = Icons.question_answer_outlined;
        color = Colors.green;
        break;
      case 'Activity':
        icon = Icons.event_outlined;
        color = Colors.orange;
        break;
      default:
        icon = Icons.article_outlined;
        color = Colors.grey;
    }
    return CircleAvatar(
      radius: 20,
      backgroundColor: color.withOpacity(0.1),
      child: Icon(icon, size: 20, color: color),
    );
  }

  Widget _statusBadge(String status, ThemeData theme) {
    Color bgColor;
    Color fgColor;
    String label;
    switch (status) {
      case 'draft':
        bgColor = Colors.grey.withOpacity(0.1);
        fgColor = Colors.grey;
        label = '草稿';
        break;
      case 'pending':
        bgColor = Colors.orange.withOpacity(0.1);
        fgColor = Colors.orange;
        label = '待审核';
        break;
      case 'published':
        bgColor = Colors.green.withOpacity(0.1);
        fgColor = Colors.green;
        label = '已发布';
        break;
      case 'retired':
        bgColor = Colors.red.withOpacity(0.1);
        fgColor = Colors.red;
        label = '已下架';
        break;
      default:
        bgColor = theme.colorScheme.surfaceContainerHighest;
        fgColor = theme.colorScheme.onSurfaceVariant;
        label = status;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 10,
          color: fgColor,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}

// ── 创建/编辑资源弹窗 ──

class _CreateResourceDialog extends StatefulWidget {
  final KnowledgeCard? resource;

  const _CreateResourceDialog({this.resource});

  @override
  State<_CreateResourceDialog> createState() => _CreateResourceDialogState();
}

class _CreateResourceDialogState extends State<_CreateResourceDialog> {
  final _formKey = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _summaryCtrl = TextEditingController();
  final _contentCtrl = TextEditingController();
  final _tagsCtrl = TextEditingController();
  final _remarkCtrl = TextEditingController();
  String _type = 'FAQ';
  String _scope = 'school';
  final String _roleScope = 'student';
  bool _saving = false;
  bool _uploading = false;
  Map<String, dynamic>? _parseResult;

  bool get _isEdit => widget.resource != null;

  @override
  void initState() {
    super.initState();
    final r = widget.resource;
    if (r != null) {
      _titleCtrl.text = r.title;
      _summaryCtrl.text = r.summary;
      _contentCtrl.text = r.content;
      _tagsCtrl.text = r.tags.join(',');
      _remarkCtrl.text = r.remark;
      if (['Policy', 'Process', 'FAQ', 'Activity'].contains(r.resourceType))
        _type = r.resourceType;
    }
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _summaryCtrl.dispose();
    _contentCtrl.dispose();
    _tagsCtrl.dispose();
    _remarkCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(_isEdit ? '编辑知识资源' : '创建知识资源'),
      content: SizedBox(
        width: 560,
        child: Form(
          key: _formKey,
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.tonalIcon(
                    onPressed: _uploading ? null : _handleUpload,
                    icon: _uploading
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.description_outlined),
                    label: Text(_uploading ? '正在解析文档...' : '从文档导入'),
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '支持 PDF、Word、TXT、Markdown 等格式，最大 100MB',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.onSurfaceVariant,
                      ),
                ),
                if (_parseResult != null) ...[
                  const SizedBox(height: 12),
                  _buildParseResultCard(),
                ],
                const SizedBox(height: 10),
                TextFormField(
                    controller: _titleCtrl,
                    decoration: const InputDecoration(
                        labelText: '标题 *',
                        border: OutlineInputBorder(),
                        isDense: true),
                    validator: (v) =>
                        (v == null || v.trim().isEmpty) ? '必填' : null),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _summaryCtrl,
                    decoration: const InputDecoration(
                        labelText: '摘要',
                        border: OutlineInputBorder(),
                        isDense: true),
                    maxLines: 2),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _contentCtrl,
                    decoration: const InputDecoration(
                        labelText: '正文 *',
                        border: OutlineInputBorder(),
                        isDense: true),
                    maxLines: 8,
                    validator: (v) =>
                        (v == null || v.trim().isEmpty) ? '必填' : null),
                const SizedBox(height: 10),
                Row(
                  children: [
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        value: _type,
                        decoration: const InputDecoration(
                            labelText: '类型',
                            border: OutlineInputBorder(),
                            isDense: true),
                        items: const [
                          DropdownMenuItem(value: 'Policy', child: Text('政策')),
                          DropdownMenuItem(value: 'Process', child: Text('流程')),
                          DropdownMenuItem(value: 'FAQ', child: Text('问答')),
                          DropdownMenuItem(
                              value: 'Activity', child: Text('活动')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _type = v);
                        },
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        value: _scope,
                        decoration: const InputDecoration(
                            labelText: '范围',
                            border: OutlineInputBorder(),
                            isDense: true),
                        items: const [
                          DropdownMenuItem(value: 'school', child: Text('学校')),
                          DropdownMenuItem(value: 'college', child: Text('学院')),
                          DropdownMenuItem(value: 'class', child: Text('班级')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _scope = v);
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _tagsCtrl,
                    decoration: const InputDecoration(
                        labelText: '标签（逗号分隔）',
                        border: OutlineInputBorder(),
                        isDense: true)),
                const SizedBox(height: 10),
                TextFormField(
                    controller: _remarkCtrl,
                    decoration: const InputDecoration(
                        labelText: '备注（上传者说明）',
                        border: OutlineInputBorder(),
                        isDense: true)),
              ],
            ),
          ),
        ),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(
            onPressed: _saving ? null : _handleSave,
            child: _saving
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2))
                : Text(_isEdit ? '保存' : '创建')),
      ],
    );
  }

  Future<void> _handleSave() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _saving = true);
    final tags = _tagsCtrl.text
        .split(',')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    final data = {
      'title': _titleCtrl.text.trim(),
      'summary': _summaryCtrl.text.trim(),
      'content': _contentCtrl.text.trim(),
      'resource_type': _type,
      'owner_scope': _scope,
      'owner_id': '',
      'role_scope': '["$_roleScope"]',
      'tags': jsonEncode(tags),
      'remark': _remarkCtrl.text.trim(),
    };
    final provider = context.read<KnowledgeProvider>();
    final ok = _isEdit
        ? await provider.updateResource(widget.resource!.resourceId, data)
        : await provider.createResource(data);
    if (!mounted) return;
    if (ok) {
      Navigator.pop(context);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_isEdit ? '保存成功' : '创建成功')));
    } else {
      setState(() => _saving = false);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_isEdit ? '保存失败' : '创建失败')));
    }
  }

  Future<void> _handleUpload() async {
    // 选择器初始化失败时（如 Web 端插件未注册）会同步抛错，
    // 必须捕获后给出提示，否则用户只会看到「点击无反应」。
    FilePickerResult? picked;
    try {
      picked = await FilePicker.platform.pickFiles(
        withData: true,
        type: FileType.custom,
        allowedExtensions: const [
          'txt',
          'md',
          'pdf',
          'docx',
          'csv',
          'xlsx',
        ],
      );
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('无法打开文件选择器：$e')),
        );
      }
      return;
    }
    if (picked == null || picked.files.isEmpty) return;
    final file = picked.files.single;
    final bytes = file.bytes;
    if (bytes == null) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('无法读取文件内容')));
      }
      return;
    }

    if (file.size > 10 * 1024 * 1024) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('文件大小超过限制（最大 100MB）')),
        );
      }
      return;
    }

    final provider = context.read<KnowledgeProvider>();
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _uploading = true);
    final result = await provider.parseDocument(bytes: bytes, filename: file.name);
    if (!mounted) return;
    setState(() => _uploading = false);
    if (result == null) {
      final errMsg = provider.resourceError;
      messenger.showSnackBar(
        SnackBar(content: Text(errMsg.isEmpty ? '文档解析失败' : errMsg)),
      );
      return;
    }

    setState(() {
      _parseResult = result;
    });

    _titleCtrl.text = (result['title'] ?? _titleCtrl.text).toString();
    _summaryCtrl.text =
        (result['summary'] ?? _summaryCtrl.text).toString();
    _contentCtrl.text =
        (result['content'] ?? '').toString();

    final keywords = result['keywords'] as List<dynamic>?;
    if (keywords != null && keywords.isNotEmpty) {
      final keywordStr = keywords.map((e) => e.toString()).join(',');
      if (_tagsCtrl.text.trim().isEmpty) {
        _tagsCtrl.text = keywordStr;
      } else {
        _tagsCtrl.text = '${_tagsCtrl.text},$keywordStr';
      }
    } else if (_tagsCtrl.text.trim().isEmpty) {
      _tagsCtrl.text = '上传文档,${file.extension ?? ''}';
    }

    messenger.showSnackBar(
      const SnackBar(content: Text('文档已解析并回填，可继续编辑后提交')),
    );
  }

  Widget _buildParseResultCard() {
    final result = _parseResult!;
    final theme = Theme.of(context);
    final wordCount = result['word_count'] ?? 0;
    final paragraphs = result['paragraphs'] ?? 0;
    final pages = result['pages'] ?? 0;
    final keywords = (result['keywords'] as List<dynamic>?)
            ?.map((e) => e.toString())
            .toList() ??
        [];

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withOpacity(0.3),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: theme.colorScheme.primary.withOpacity(0.2),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.check_circle,
                size: 18,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 6),
              Text(
                '解析成功',
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                  color: theme.colorScheme.primary,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 12,
            runSpacing: 4,
            children: [
              _statItem('字数', '$wordCount'),
              _statItem('段落', '$paragraphs'),
              if (pages > 0) _statItem('页数', '$pages'),
            ],
          ),
          if (keywords.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              '提取关键词：',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 4),
            Wrap(
              spacing: 4,
              runSpacing: 4,
              children: keywords
                  .map((k) => Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.secondaryContainer
                              .withOpacity(0.5),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(
                          k,
                          style: TextStyle(
                            fontSize: 11,
                            color: theme.colorScheme.onSecondaryContainer,
                          ),
                        ),
                      ))
                  .toList(),
            ),
          ],
          const SizedBox(height: 8),
          Text(
            '标题、摘要、正文已自动回填，请核对后再提交；长文档建议裁剪正文，仅保留与知识主题相关的段落。',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }

  Widget _statItem(String label, String value) {
    final theme = Theme.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          value,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(width: 2),
        Text(
          label,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

extension on KnowledgeCard {
  String get statusLabel {
    const map = {
      'draft': '草稿',
      'pending': '待审核',
      'published': '已发布',
      'retired': '已下架'
    };
    return map[status] ?? (status.isEmpty ? '未知' : status);
  }
}
