import 'dart:convert';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../config/api_config.dart';
import '../../models/models.dart';
import '../../providers/knowledge_provider.dart';
import '../../services/api_service.dart';
import '../../utils/capability_utils.dart';
import '../../utils/web_export.dart';
import '../../widgets/error_view.dart';
import '../../widgets/md_text.dart';
import 'resource_stat_chip.dart';
import 'resource_type_stats.dart';
import 'resource_tile_helpers.dart';

/// 知识治理页面（增强版）
/// 功能：搜索、多条件筛选、批量选择、批量操作、统计、预览、编辑
class KnowledgeGovernancePage extends StatefulWidget {
  const KnowledgeGovernancePage({super.key});

  @override
  State<KnowledgeGovernancePage> createState() =>
      _KnowledgeGovernancePageState();
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
          if (canReview)
            IconButton(
              icon: const Icon(Icons.fact_check_outlined),
              tooltip: '智能体审计',
              onPressed: () => _showGovernanceAuditDialog(),
            ),
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
            color:
                Theme.of(context).colorScheme.outlineVariant.withOpacity(0.3),
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
                  DropdownMenuItem(value: 'resource_type', child: Text('类型')),
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
                    ResourceStatChip(
                        label: '草稿',
                        count: stats['draft'] ?? 0,
                        color: Colors.grey),
                    ResourceStatChip(
                        label: '待审',
                        count: stats['pending'] ?? 0,
                        color: Colors.orange),
                    ResourceStatChip(
                        label: '已发',
                        count: stats['published'] ?? 0,
                        color: Colors.green),
                    ResourceStatChip(
                        label: '下架',
                        count: stats['retired'] ?? 0,
                        color: Colors.red),
                  ],
                  const Spacer(),
                  if (provider.selectedCount > 0) ...[
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: Theme.of(context).colorScheme.primaryContainer,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        '已选 ${provider.selectedCount} 条',
                        style: TextStyle(
                          color:
                              Theme.of(context).colorScheme.onPrimaryContainer,
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
                    Icon(Icons.category_outlined,
                        size: 14,
                        color: Theme.of(context).colorScheme.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text('类型分布：',
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: Theme.of(context)
                                  .colorScheme
                                  .onSurfaceVariant,
                            )),
                    const SizedBox(width: 4),
                    ResourceTypeStats(byType: stats['by_type']),
                  ],
                ),
              ],
            ],
          ),
        );
      },
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
                  icon: Icons.auto_awesome_outlined,
                  label: 'AI 精修',
                  color: Theme.of(context).colorScheme.primary,
                  onPressed: () => _confirmBatchRefine(),
                ),
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

  /// 批量 AI 精修：确认后调用接口，并展示成功/失败统计与失败原因
  Future<void> _confirmBatchRefine() async {
    final provider = context.read<KnowledgeProvider>();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('批量 AI 精修'),
        content: Text(
          '将对选中的 ${provider.selectedCount} 条资源执行 AI 精修'
          '（重新生成标题、摘要与标签，精修结果将直接写入）。\n'
          '注意：精修依赖大模型，请核对精修结果是否准确。',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('确认精修'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    final data = await provider.batchRefine();
    if (!mounted) return;
    if (data == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(provider.resourceError.isEmpty
              ? '批量精修失败，请稍后重试'
              : provider.resourceError),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
      return;
    }

    final success = (data['success'] ?? 0) as int;
    final failed = (data['failed'] ?? 0) as int;
    final results = (data['results'] as List<dynamic>?) ?? [];
    final failedReasons = results
        .where((e) =>
            e is Map && e['ok'] != true && (e['message'] ?? '').isNotEmpty)
        .map((e) => '${(e as Map)['resource_id']}: ${e['message']}')
        .take(5)
        .join('\n');

    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('批量精修完成'),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('成功 $success 条，失败 $failed 条。'),
              if (failedReasons.isNotEmpty) ...[
                const SizedBox(height: 12),
                const Text('失败原因：'),
                const SizedBox(height: 4),
                Text(
                  failedReasons,
                  style: TextStyle(
                    fontSize: 13,
                    color: Theme.of(ctx).colorScheme.error,
                  ),
                ),
              ],
              if (failed == 0)
                const Padding(
                  padding: EdgeInsets.only(top: 12),
                  child: Text('建议抽查精修结果，确认无误后发布。'),
                ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('知道了'),
          ),
        ],
      ),
    );
  }

  // ── 智能体审计 ──

  /// 知识治理智能体审计：调用 /kb/governance，展示确定性检查 + LLM 准确性审计报告
  Future<void> _showGovernanceAuditDialog() async {
    showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => const _GovernanceAuditDialog(),
    );
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
                onSubmit: () => _handleSubmit(provider, resource.resourceId),
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
            resourceTypeIcon(r.resourceType),
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
                MdText(full.summary,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        )),
                const SizedBox(height: 12),
              ],
              Expanded(
                child: SingleChildScrollView(
                  child: MdText(
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
              onPressed: () => Navigator.pop(context), child: const Text('关闭'))
        ],
      ),
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
              resourceTypeIcon(resource.resourceType),
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
                        resourceStatusBadge(resource.status, theme),
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
                      children: resource.tags
                          .take(3)
                          .map((tag) => Container(
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 6, vertical: 1),
                                decoration: BoxDecoration(
                                  color:
                                      theme.colorScheme.surfaceContainerHighest,
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: Text(
                                  tag,
                                  style: TextStyle(
                                    fontSize: 10,
                                    color: theme.colorScheme.onSurfaceVariant,
                                  ),
                                ),
                              ))
                          .toList(),
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 6,
                      runSpacing: 6,
                      children: [
                        OutlinedButton.icon(
                            onPressed: onPreview,
                            icon:
                                const Icon(Icons.visibility_outlined, size: 16),
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
                                size: 16, color: theme.colorScheme.error),
                            label: Text('删除',
                                style:
                                    TextStyle(color: theme.colorScheme.error))),
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
  bool _refining = false;
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
      if (['Policy', 'Process', 'FAQ', 'Activity'].contains(r.resourceType)) {
        _type = r.resourceType;
      }
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
                const SizedBox(height: 8),
                FilledButton.tonalIcon(
                  onPressed: (_refining || _uploading) ? null : _handleAiRefine,
                  icon: _refining
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(strokeWidth: 2))
                      : const Icon(Icons.auto_awesome_outlined),
                  label: Text(_refining ? 'AI 精修中...' : 'AI 一键精修标题/摘要/关键词'),
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

    if (file.size > 100 * 1024 * 1024) {
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
    final result =
        await provider.parseDocument(bytes: bytes, filename: file.name);
    if (!mounted) return;
    setState(() => _uploading = false);
    if (result == null) {
      final errMsg = provider.resourceError;
      messenger.showSnackBar(
        SnackBar(content: Text(errMsg.isEmpty ? '文档解析失败' : errMsg)),
      );
      return;
    }

    // 解析质量门槛：正文过短/无中文/疑似乱码时强制用户预览确认，避免低质量内容入库。
    final quality = result['quality'] as Map<String, dynamic>?;
    if (quality != null && quality['ok'] != true) {
      final reasons = (quality['reasons'] as List<dynamic>?)
              ?.map((e) => e.toString())
              .toList() ??
          const <String>['解析内容质量异常'];
      final proceed = await _confirmQualityWarning(reasons);
      if (!mounted) return;
      if (!proceed) {
        messenger.showSnackBar(
          const SnackBar(content: Text('已取消导入，请检查文件内容或手动编辑')),
        );
        return;
      }
    }

    setState(() {
      _parseResult = result;
    });

    _titleCtrl.text = (result['title'] ?? _titleCtrl.text).toString();
    _summaryCtrl.text = (result['summary'] ?? _summaryCtrl.text).toString();
    _contentCtrl.text = (result['content'] ?? '').toString();

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
      SnackBar(
        content: Text(_parseWarning(result).isNotEmpty
            ? '${_parseWarning(result)}；已回填可编辑内容，请核对后提交'
            : '文档已解析并回填，可继续编辑后提交'),
      ),
    );
  }

  /// 解析质量不达标时的强制预览确认对话框。
  /// 返回 true 表示用户已知晓风险并同意继续编辑，false 表示取消导入。
  Future<bool> _confirmQualityWarning(List<String> reasons) async {
    final theme = Theme.of(context);
    return await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            icon: Icon(
              Icons.warning_amber_rounded,
              color: theme.colorScheme.error,
              size: 32,
            ),
            title: const Text('解析内容可能存在质量问题'),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('检测到以下问题，建议先预览或手动编辑后再提交：'),
                const SizedBox(height: 12),
                ...reasons.map(
                  (r) => Padding(
                    padding: const EdgeInsets.only(bottom: 6),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(Icons.error_outline,
                            size: 16, color: theme.colorScheme.error),
                        const SizedBox(width: 6),
                        Expanded(child: Text(r)),
                      ],
                    ),
                  ),
                ),
              ],
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(false),
                child: const Text('取消导入'),
              ),
              FilledButton(
                onPressed: () => Navigator.of(ctx).pop(true),
                child: const Text('我已知晓，继续编辑'),
              ),
            ],
          ),
        ) ??
        false;
  }

  /// 调用后端 LLM 精修标题/摘要/关键词并回填编辑表单。
  /// 精修失败（未配置模型/超时/非法输出）时后端自动回退原值，此处仅提示。
  Future<void> _handleAiRefine() async {
    final content = _contentCtrl.text.trim();
    if (content.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('正文为空，请先填写或导入正文')),
      );
      return;
    }

    final keywords = _tagsCtrl.text
        .split(',')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    setState(() => _refining = true);
    final result = await context.read<KnowledgeProvider>().refineDocument(
          content: content,
          title: _titleCtrl.text.trim(),
          summary: _summaryCtrl.text.trim(),
          keywords: keywords,
        );
    if (!mounted) return;
    setState(() => _refining = false);

    if (result == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
            content: Text(
                context.read<KnowledgeProvider>().resourceError.isEmpty
                    ? 'AI 精修失败，请稍后重试'
                    : context.read<KnowledgeProvider>().resourceError)),
      );
      return;
    }

    final title = (result['title'] ?? '').toString().trim();
    final summary = (result['summary'] ?? '').toString().trim();
    final refinedKeywords = (result['keywords'] as List<dynamic>?)
            ?.map((e) => e.toString().trim())
            .where((e) => e.isNotEmpty)
            .toList() ??
        <String>[];

    if (title.isNotEmpty) _titleCtrl.text = title;
    if (summary.isNotEmpty) _summaryCtrl.text = summary;
    if (refinedKeywords.isNotEmpty) _tagsCtrl.text = refinedKeywords.join(',');

    final fallback = result['fallback'] == true;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
            fallback ? '暂无法 AI 精修（已保留当前内容），可手动编辑后提交' : 'AI 精修完成，已回填，请核对后提交'),
      ),
    );
  }

  bool _parseQualityNotOk(Map<String, dynamic> result) {
    final quality = result['quality'] as Map<String, dynamic>?;
    return quality != null && quality['ok'] != true;
  }

  List<String> _parseQualityReasons(Map<String, dynamic> result) {
    final quality = result['quality'] as Map<String, dynamic>?;
    final reasons = quality?['reasons'] as List<dynamic>?;
    if (reasons == null || reasons.isEmpty) return const ['解析内容质量异常'];
    return reasons.map((e) => e.toString()).toList();
  }

  /// 提取后端返回的非致命解析警告（如 PDF 部分页为图片/扫描页）。
  String _parseWarning(Map<String, dynamic> result) {
    return (result['parse_warning'] ?? '').toString().trim();
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
          if (_parseQualityNotOk(result)) ...[
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              decoration: BoxDecoration(
                color: theme.colorScheme.errorContainer.withOpacity(0.4),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: theme.colorScheme.error.withOpacity(0.4),
                ),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '存在质量问题，提交前请预览并修正',
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontWeight: FontWeight.w600,
                      color: theme.colorScheme.onErrorContainer,
                    ),
                  ),
                  const SizedBox(height: 4),
                  ...(_parseQualityReasons(result)).map(
                    (r) => Padding(
                      padding: const EdgeInsets.only(bottom: 2),
                      child: Text(
                        '· $r',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onErrorContainer,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
          if (_parseWarning(result).isNotEmpty) ...[
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              decoration: BoxDecoration(
                color: theme.colorScheme.tertiaryContainer.withOpacity(0.4),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: theme.colorScheme.tertiary.withOpacity(0.4),
                ),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    Icons.info_outline,
                    size: 16,
                    color: theme.colorScheme.onTertiaryContainer,
                  ),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      _parseWarning(result),
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onTertiaryContainer,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
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

/// 知识治理智能体审计弹窗
/// 调用 /kb/governance?with_llm=1，展示确定性检查发现 + LLM 准确性审计发现。
class _GovernanceAuditDialog extends StatefulWidget {
  const _GovernanceAuditDialog();

  @override
  State<_GovernanceAuditDialog> createState() => _GovernanceAuditDialogState();
}

class _GovernanceAuditDialogState extends State<_GovernanceAuditDialog> {
  bool _loading = true;
  String? _error;
  Map<String, dynamic>? _data;

  @override
  void initState() {
    super.initState();
    _run();
  }

  Future<void> _run() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final resp = await ApiService().get(
        '${ApiConfig.kbGovernance}?with_llm=1',
      );
      setState(() {
        _data = (resp.data as Map?)?['data'] as Map<String, dynamic>?;
        _loading = false;
      });
    } catch (e) {
      setState(() {
        _error = '审计失败：$e';
        _loading = false;
      });
    }
  }

  Color _levelColor(String level) {
    switch (level) {
      case 'critical':
        return const Color(0xFFC62828);
      case 'warning':
        return const Color(0xFFE65100);
      default:
        return const Color(0xFF2E7D32);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Dialog(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560, maxHeight: 640),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 18, 8, 8),
              child: Row(
                children: [
                  Icon(Icons.fact_check_outlined,
                      color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text('知识治理 · 智能体审计',
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w700)),
                  const Spacer(),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.of(context).pop(),
                  ),
                ],
              ),
            ),
            if (_loading)
              const Expanded(
                child: Center(
                    child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(),
                    SizedBox(height: 12),
                    Text('智能体正在审计知识库，请稍候…'),
                  ],
                )),
              )
            else if (_error != null)
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Center(
                    child: Text(_error!,
                        style: TextStyle(color: theme.colorScheme.error)),
                  ),
                ),
              )
            else
              Expanded(
                child: _buildReport(theme),
              ),
            const Padding(
              padding: EdgeInsets.fromLTRB(20, 8, 20, 16),
              child: Row(
                children: [
                  Icon(Icons.info_outline, size: 16),
                  SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      '智能体只给建议与报告，不自动改写知识内容；确需修改请在列表中人工编辑。',
                      style: TextStyle(fontSize: 12),
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

  Widget _buildReport(ThemeData theme) {
    final summary = (_data?['summary'] as Map?) ?? const <String, dynamic>{};
    final issues = ((_data?['issues'] as List?) ?? const []);
    final ds = (_data?['data_source'] as String?) ?? 'real';

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 8),
      children: [
        // 统计概览
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: theme.colorScheme.primaryContainer.withOpacity(0.25),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Wrap(
            spacing: 16,
            runSpacing: 6,
            children: [
              _stat('扫描资源', '${summary['scanned'] ?? 0}'),
              _stat('确定性发现', '${summary['determined'] ?? 0}'),
              _stat('LLM 审计', '${summary['llm_checked'] ?? 0} 条'),
              _stat('LLM 风险', '${summary['llm_findings'] ?? 0}'),
              _stat('数据源', ds == 'real' ? '真实' : ds),
            ],
          ),
        ),
        const SizedBox(height: 12),
        if (issues.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 16),
            child:
                Text('未发现问题，知识库质量良好。', style: TextStyle(color: Colors.green)),
          )
        else
          ...issues.map((it) => _issueTile(theme, it as Map)),
      ],
    );
  }

  Widget _stat(String label, String value) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(value,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w800)),
        Text(label, style: const TextStyle(fontSize: 12)),
      ],
    );
  }

  Widget _issueTile(ThemeData theme, Map it) {
    final level = (it['level'] as String?) ?? 'info';
    final category = (it['category'] as String?) ?? '';
    final title = (it['title'] as String?) ?? '';
    final message = (it['message'] as String?) ?? '';
    final rid = (it['resource_id'] as String?) ?? '';
    final color = _levelColor(level);

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: color.withOpacity(0.07),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            level == 'critical'
                ? Icons.error_outline
                : (level == 'warning'
                    ? Icons.warning_amber_rounded
                    : Icons.info_outline),
            size: 18,
            color: color,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${_categoryLabel(category)}${title.isNotEmpty ? ' · $title' : ''}${rid.isNotEmpty ? ' ($rid)' : ''}',
                  style: TextStyle(
                      fontWeight: FontWeight.w700, fontSize: 13, color: color),
                ),
                const SizedBox(height: 3),
                Text(message, style: const TextStyle(fontSize: 13)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _categoryLabel(String cat) {
    const map = {
      'missing_field': '字段缺失',
      'duplicate': '疑似重复',
      'short_content': '正文过短',
      'no_tags': '缺标签',
      'expired': '已失效',
      'accuracy_risk': '准确性风险',
      'audit': '提示',
    };
    return map[cat] ?? cat;
  }
}
