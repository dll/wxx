import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../config/api_config.dart';
import '../../models/models.dart';
import '../../providers/ai_briefing_provider.dart';
import '../../widgets/error_view.dart';

/// AI 简讯管理 — sys_admin 专属
/// 资讯 CRUD + 筛选查找 + 汇总统计 + 来源设置 + 自动抓取 + 导出(md/pdf)
class AIBriefingAdminPage extends StatefulWidget {
  const AIBriefingAdminPage({super.key});

  @override
  State<AIBriefingAdminPage> createState() => _AIBriefingAdminPageState();
}

class _AIBriefingAdminPageState extends State<AIBriefingAdminPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tab;
  final TextEditingController _searchCtrl = TextEditingController();
  String _category = '';
  String _status = '';
  final Set<int> _selected = {};

  // 编辑弹窗表单
  final TextEditingController _fTopic = TextEditingController();
  final TextEditingController _fSource = TextEditingController();
  final TextEditingController _fSummary = TextEditingController();
  final TextEditingController _fContent = TextEditingController();
  final TextEditingController _fLink = TextEditingController();
  final TextEditingController _fKeyword = TextEditingController();
  final TextEditingController _fPublishedAt = TextEditingController();
  String _fCategory = 'ai_teaching';
  int _fStatus = 1;
  int? _editId;

  // 来源表单
  final TextEditingController _sName = TextEditingController();
  final TextEditingController _sUrl = TextEditingController();
  final TextEditingController _sTime = TextEditingController();
  String _sCategory = 'ai_teaching';
  int _sEnabled = 1;
  int _sFetchEnabled = 1;
  int? _editSourceId;

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 2, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<AIBriefingProvider>();
      p.fetchAdminBriefings();
      p.fetchStats();
      p.fetchSources();
    });
  }

  @override
  void dispose() {
    _tab.dispose();
    _searchCtrl.dispose();
    _fTopic.dispose();
    _fSource.dispose();
    _fSummary.dispose();
    _fContent.dispose();
    _fLink.dispose();
    _fKeyword.dispose();
    _fPublishedAt.dispose();
    _sName.dispose();
    _sUrl.dispose();
    _sTime.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<AIBriefingProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('AI 简讯管理'),
        bottom: TabBar(
          controller: _tab,
          tabs: const [
            Tab(text: '资讯管理'),
            Tab(text: '来源设置'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tab,
        children: [
          _buildBriefingsTab(theme, provider),
          _buildSourcesTab(theme, provider),
        ],
      ),
    );
  }

  // ── 资讯管理 Tab ──

  Widget _buildBriefingsTab(ThemeData theme, AIBriefingProvider provider) {
    return Column(
      children: [
        _buildAdminFilterBar(theme),
        if (provider.stats != null) _buildStatsCard(theme, provider.stats!),
        if (_selected.isNotEmpty) _buildBatchBar(theme),
        const Divider(height: 1),
        Expanded(
          child: provider.adminLoading && provider.adminBriefings.isEmpty
              ? const Center(child: CircularProgressIndicator())
              : provider.adminError.isNotEmpty &&
                      provider.adminBriefings.isEmpty
                  ? ErrorView.error(
                      message: provider.adminError,
                      onRetry: () => provider.fetchAdminBriefings())
                  : _buildAdminList(theme, provider),
        ),
      ],
    );
  }

  Widget _buildAdminFilterBar(ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.all(12),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _searchCtrl,
              decoration: const InputDecoration(
                hintText: '查找：主题 / 摘要 / 关键词 / 来源',
                isDense: true,
                prefixIcon: Icon(Icons.search, size: 20),
                border: OutlineInputBorder(),
              ),
              onSubmitted: (v) => _reload(v.trim()),
            ),
          ),
          const SizedBox(width: 8),
          _buildDropdown<String>(
            value: _category,
            items: {
              '': '全部分类',
              ...{for (final c in AIBriefingCategory.all) c.key: c.label},
            },
            onChanged: (v) {
              setState(() => _category = v);
              _reload();
            },
          ),
          const SizedBox(width: 8),
          _buildDropdown<String>(
            value: _status,
            items: const {
              '': '全部状态',
              '1': '已上架',
              '0': '已下架',
            },
            onChanged: (v) {
              setState(() => _status = v);
              _reload();
            },
          ),
          const SizedBox(width: 8),
          IconButton(
            tooltip: '立即抓取',
            onPressed: _fetchNow,
            icon: const Icon(Icons.sync),
          ),
          IconButton(
            tooltip: '新增资讯',
            onPressed: () => _showEditDialog(),
            icon: const Icon(Icons.add),
          ),
        ],
      ),
    );
  }

  Widget _buildStatsCard(ThemeData theme, Map<String, dynamic> stats) {
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 0, 12, 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: [
          _stat(theme, '总条数', '${stats['total'] ?? 0}'),
          _stat(theme, '已上架', '${stats['published'] ?? 0}'),
          _stat(theme, '已下架', '${stats['draft'] ?? 0}'),
          _stat(theme, '自动抓取', '${stats['auto_fetched'] ?? 0}'),
          _stat(theme, '手动录入', '${stats['manual'] ?? 0}'),
        ],
      ),
    );
  }

  Widget _stat(ThemeData theme, String label, String value) {
    return Expanded(
      child: Column(
        children: [
          Text(value,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 2),
          Text(label, style: theme.textTheme.bodySmall),
        ],
      ),
    );
  }

  Widget _buildBatchBar(ThemeData theme) {
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(12, 0, 12, 12),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withOpacity(0.5),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        children: [
          Text('已选 ${_selected.length} 条',
              style: theme.textTheme.bodyMedium),
          const Spacer(),
          TextButton.icon(
            onPressed: () => _exportSelected('md'),
            icon: const Icon(Icons.description_outlined, size: 18),
            label: const Text('导出 MD'),
          ),
          TextButton.icon(
            onPressed: () => _exportSelected('pdf'),
            icon: const Icon(Icons.picture_as_pdf, size: 18),
            label: const Text('导出 PDF'),
          ),
          TextButton.icon(
            onPressed: () => _confirmDeleteSelected(),
            icon: const Icon(Icons.delete_outline, size: 18),
            label: const Text('删除'),
          ),
          TextButton(
            onPressed: () => setState(() => _selected.clear()),
            child: const Text('取消'),
          ),
        ],
      ),
    );
  }

  Widget _buildAdminList(ThemeData theme, AIBriefingProvider provider) {
    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 16),
      itemCount: provider.adminBriefings.length + 1,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        if (index == 0) {
          return Row(
            children: [
              const SizedBox(width: 4),
              IconButton(
                tooltip: '导出全部(MD)',
                icon: const Icon(Icons.description_outlined),
                onPressed: () => _exportAll('md'),
              ),
              IconButton(
                tooltip: '导出全部(PDF)',
                icon: const Icon(Icons.picture_as_pdf),
                onPressed: () => _exportAll('pdf'),
              ),
              IconButton(
                tooltip: '清空历史',
                icon: const Icon(Icons.delete_sweep_outlined),
                onPressed: _confirmClearAll,
              ),
              const Spacer(),
              Text('共 ${provider.total} 条',
                  style: theme.textTheme.bodySmall),
              const SizedBox(width: 8),
            ],
          );
        }
        final b = provider.adminBriefings[index - 1];
        return _buildAdminRow(theme, b);
      },
    );
  }

  Widget _buildAdminRow(ThemeData theme, AIBriefing b) {
    final selected = _selected.contains(b.id);
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: selected
              ? theme.colorScheme.primary
              : theme.colorScheme.outlineVariant,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Row(
          children: [
            Checkbox(
              value: selected,
              onChanged: (v) => setState(() {
                if (v == true) {
                  _selected.add(b.id);
                } else {
                  _selected.remove(b.id);
                }
              }),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(b.topic,
                            style: theme.textTheme.bodyMedium?.copyWith(
                                fontWeight: FontWeight.w600),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis),
                      ),
                      _statusChip(theme, b.status),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${AIBriefingCategory.labelOf(b.category)} · 来源：${b.source} · ${b.publishedAt}',
                    style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  if (b.summary.isNotEmpty)
                    Text(b.summary,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodySmall),
                ],
              ),
            ),
            PopupMenuButton<String>(
              onSelected: (v) {
                switch (v) {
                  case 'edit':
                    _showEditDialog(b);
                    break;
                  case 'toggle':
                    _toggleStatus(b);
                    break;
                  case 'del':
                    _confirmDeleteOne(b);
                    break;
                }
              },
              itemBuilder: (_) => [
                const PopupMenuItem(value: 'edit', child: Text('编辑')),
                PopupMenuItem(
                    value: 'toggle',
                    child: Text(b.status == 1 ? '下架' : '上架')),
                const PopupMenuItem(value: 'del', child: Text('删除')),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusChip(ThemeData theme, int status) {
    final color = status == 1 ? const Color(0xFF2E7D32) : Colors.orange;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(status == 1 ? '上架' : '下架',
          style: theme.textTheme.labelSmall
              ?.copyWith(color: color, fontWeight: FontWeight.w600)),
    );
  }

  // ── 来源设置 Tab ──

  Widget _buildSourcesTab(ThemeData theme, AIBriefingProvider provider) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('自动获取资讯设置', style: theme.textTheme.titleMedium),
        const SizedBox(height: 4),
        Text(
          '配置 RSS/Atom 来源与抓取时间，服务将按设定时间自动拉取并入库存档；抓取内容归入对应分类。',
          style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant),
        ),
        const SizedBox(height: 12),
        FilledButton.icon(
          onPressed: _showSourceDialog,
          icon: const Icon(Icons.add),
          label: const Text('新增来源'),
        ),
        const SizedBox(height: 12),
        if (provider.sources.isEmpty)
          const Padding(
            padding: EdgeInsets.all(24),
            child: Center(child: Text('暂无来源配置')),
          )
        else
          ...provider.sources.map((s) => _buildSourceRow(theme, s)),
      ],
    );
  }

  Widget _buildSourceRow(ThemeData theme, AIBriefingSource s) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 10),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(s.name,
                      style: theme.textTheme.titleSmall
                          ?.copyWith(fontWeight: FontWeight.w600)),
                ),
                Switch(
                  value: s.enabled == 1,
                  onChanged: (v) {
                    final updated = AIBriefingSource(
                      id: s.id,
                      name: s.name,
                      url: s.url,
                      category: s.category,
                      enabled: v ? 1 : 0,
                      fetchEnabled: s.fetchEnabled,
                      fetchTime: s.fetchTime,
                    );
                    context.read<AIBriefingProvider>().updateSource(updated);
                    context.read<AIBriefingProvider>().fetchSources();
                  },
                ),
                IconButton(
                  tooltip: '编辑',
                  icon: const Icon(Icons.edit_outlined, size: 20),
                  onPressed: () => _showSourceDialog(s),
                ),
                IconButton(
                  tooltip: '删除',
                  icon: const Icon(Icons.delete_outline, size: 20),
                  onPressed: () => _confirmDeleteSource(s),
                ),
              ],
            ),
            Text(AIBriefingCategory.labelOf(s.category),
                style: theme.textTheme.labelSmall
                    ?.copyWith(color: theme.colorScheme.primary)),
            const SizedBox(height: 4),
            if (s.url.isNotEmpty)
              Text(s.url,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall),
            const SizedBox(height: 4),
            Row(
              children: [
                Icon(Icons.schedule, size: 14,
                    color: theme.colorScheme.outline),
                const SizedBox(width: 4),
                Text('每日抓取：${s.fetchTime}',
                    style: theme.textTheme.bodySmall),
                const SizedBox(width: 16),
                Icon(Icons.sync, size: 14, color: theme.colorScheme.outline),
                const SizedBox(width: 4),
                Text(
                  s.fetchEnabled == 1 ? '自动抓取开启' : '自动抓取关闭',
                  style: theme.textTheme.bodySmall,
                ),
                const SizedBox(width: 16),
                if (s.lastFetchAt.isNotEmpty)
                  Text('上次抓取：${s.lastFetchAt}',
                      style: theme.textTheme.bodySmall),
              ],
            ),
          ],
        ),
      ),
    );
  }

  // ── 弹窗 ──

  void _showEditDialog([AIBriefing? b]) {
    _editId = b?.id;
    _fTopic.text = b?.topic ?? '';
    _fSource.text = b?.source ?? '';
    _fCategory = b?.category ?? 'ai_teaching';
    _fSummary.text = b?.summary ?? '';
    _fContent.text = b?.content ?? '';
    _fLink.text = b?.link ?? '';
    _fKeyword.text = b?.keyword ?? '';
    _fPublishedAt.text = b?.publishedAt ?? '';
    _fStatus = b?.status ?? 1;

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(_editId == null ? '新增资讯' : '编辑资讯'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _textField(_fTopic, '主题（必填）'),
              _textField(_fSource, '来源'),
              DropdownButtonFormField<String>(
                value: _fCategory,
                decoration: const InputDecoration(labelText: '分类'),
                items: AIBriefingCategory.all
                    .map((c) => DropdownMenuItem(
                        value: c.key, child: Text(c.label)))
                    .toList(),
                onChanged: (v) => setState(() => _fCategory = v ?? 'ai_teaching'),
              ),
              _textField(_fSummary, '摘要'),
              _textField(_fContent, '正文'),
              _textField(_fLink, '详情链接'),
              _textField(_fKeyword, '关键词（逗号分隔）'),
              _textField(_fPublishedAt, '发布时间（YYYY-MM-DD HH:MM:SS）'),
              DropdownButtonFormField<int>(
                value: _fStatus,
                decoration: const InputDecoration(labelText: '状态'),
                items: const [
                  DropdownMenuItem(value: 1, child: Text('上架')),
                  DropdownMenuItem(value: 0, child: Text('下架')),
                ],
                onChanged: (v) => setState(() => _fStatus = v ?? 1),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(onPressed: () => _saveBriefing(ctx), child: const Text('保存')),
        ],
      ),
    );
  }

  void _showSourceDialog([AIBriefingSource? s]) {
    _editSourceId = s?.id;
    _sName.text = s?.name ?? '';
    _sUrl.text = s?.url ?? '';
    _sCategory = s?.category ?? 'ai_teaching';
    _sEnabled = s?.enabled ?? 1;
    _sFetchEnabled = s?.fetchEnabled ?? 1;
    _sTime.text = s?.fetchTime ?? '08:00';

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(_editSourceId == null ? '新增来源' : '编辑来源'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _textField(_sName, '来源名称（必填）'),
              _textField(_sUrl, 'RSS/Atom URL'),
              DropdownButtonFormField<String>(
                value: _sCategory,
                decoration: const InputDecoration(labelText: '抓取归入分类'),
                items: AIBriefingCategory.all
                    .map((c) => DropdownMenuItem(
                        value: c.key, child: Text(c.label)))
                    .toList(),
                onChanged: (v) => setState(() => _sCategory = v ?? 'ai_teaching'),
              ),
              _textField(_sTime, '每日抓取时间（HH:MM）'),
              SwitchListTile(
                title: const Text('启用来源'),
                value: _sEnabled == 1,
                onChanged: (v) => setState(() => _sEnabled = v ? 1 : 0),
              ),
              SwitchListTile(
                title: const Text('参与定时抓取'),
                value: _sFetchEnabled == 1,
                onChanged: (v) => setState(() => _sFetchEnabled = v ? 1 : 0),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(onPressed: () => _saveSource(ctx), child: const Text('保存')),
        ],
      ),
    );
  }

  Widget _textField(TextEditingController ctrl, String label) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: TextField(
        controller: ctrl,
        decoration: InputDecoration(labelText: label, isDense: true),
      ),
    );
  }

  Widget _buildDropdown<T>(
      {required T value,
      required Map<T, String> items,
      required ValueChanged<T> onChanged}) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8),
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(8),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<T>(
          value: value,
          isDense: true,
          items: items.entries
              .map((e) => DropdownMenuItem(value: e.key, child: Text(e.value)))
              .toList(),
          onChanged: (v) {
            if (v != null) onChanged(v);
          },
        ),
      ),
    );
  }

  // ── 操作 ──

  void _reload([String? q]) {
    context.read<AIBriefingProvider>().fetchAdminBriefings(
          status: _status,
          category: _category,
          q: q ?? _searchCtrl.text.trim(),
        );
  }

  void _saveBriefing(BuildContext ctx) {
    final topic = _fTopic.text.trim();
    if (topic.isEmpty) {
      ScaffoldMessenger.of(ctx).showSnackBar(
          const SnackBar(content: Text('主题不能为空')));
      return;
    }
    final b = AIBriefing(
      id: _editId ?? 0,
      source: _fSource.text.trim(),
      category: _fCategory,
      topic: topic,
      summary: _fSummary.text.trim(),
      content: _fContent.text.trim(),
      link: _fLink.text.trim(),
      keyword: _fKeyword.text.trim(),
      publishedAt: _fPublishedAt.text.trim(),
      status: _fStatus,
    );
    final p = context.read<AIBriefingProvider>();
    final ok = _editId == null ? p.createBriefing(b) : p.updateBriefing(b);
    ok.then((success) {
      if (success) {
        Navigator.pop(ctx);
        p.fetchAdminBriefings(status: _status, category: _category, q: _searchCtrl.text.trim());
        p.fetchStats();
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('保存成功')));
      } else {
        ScaffoldMessenger.of(ctx).showSnackBar(
            const SnackBar(content: Text('保存失败')));
      }
    });
  }

  void _saveSource(BuildContext ctx) {
    final name = _sName.text.trim();
    if (name.isEmpty) {
      ScaffoldMessenger.of(ctx).showSnackBar(
          const SnackBar(content: Text('来源名称不能为空')));
      return;
    }
    final s = AIBriefingSource(
      id: _editSourceId ?? 0,
      name: name,
      url: _sUrl.text.trim(),
      category: _sCategory,
      enabled: _sEnabled,
      fetchEnabled: _sFetchEnabled,
      fetchTime: _sTime.text.trim().isEmpty ? '08:00' : _sTime.text.trim(),
    );
    final p = context.read<AIBriefingProvider>();
    final ok = _editSourceId == null ? p.createSource(s) : p.updateSource(s);
    ok.then((success) {
      if (success) {
        Navigator.pop(ctx);
        p.fetchSources();
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('保存成功')));
      } else {
        ScaffoldMessenger.of(ctx).showSnackBar(
            const SnackBar(content: Text('保存失败')));
      }
    });
  }

  void _toggleStatus(AIBriefing b) {
    final p = context.read<AIBriefingProvider>();
    p.updateBriefingStatus(b.id, b.status == 1 ? 0 : 1).then((ok) {
      if (ok) p.fetchAdminBriefings(status: _status, category: _category, q: _searchCtrl.text.trim());
    });
  }

  Future<void> _fetchNow() async {
    final messenger = ScaffoldMessenger.of(context);
    final result = await context.read<AIBriefingProvider>().fetchNow();
    if (result != null) {
      final summary = result.entries.map((e) => '${e.key}+${e.value}').join(' ');
      messenger.showSnackBar(SnackBar(
          content: Text(result.isEmpty ? '抓取完成（无可用来源）' : '抓取完成：$summary')));
      _reload();
    } else {
      messenger.showSnackBar(const SnackBar(content: Text('抓取失败')));
    }
  }

  void _exportAll(String format) {
    _openExport(context.read<AIBriefingProvider>().exportUrl(format: format, all: true));
  }

  void _exportSelected(String format) {
    if (_selected.isEmpty) return;
    _openExport(context
        .read<AIBriefingProvider>()
        .exportUrl(format: format, ids: _selected.toList()));
  }

  void _openExport(String url) {
    launchUrl(Uri.parse('${ApiConfig.baseUrl}$url'),
        mode: LaunchMode.externalApplication);
  }

  void _confirmDeleteOne(AIBriefing b) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除资讯'),
        content: Text('确定删除「${b.topic}」？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              Navigator.pop(ctx);
              context.read<AIBriefingProvider>().deleteBriefing(b.id).then((ok) {
                if (ok) _reload();
              });
            },
            child: const Text('删除'),
          ),
        ],
      ),
    );
  }

  void _confirmDeleteSelected() {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('批量删除'),
        content: Text('确定删除选中的 ${_selected.length} 条资讯？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              Navigator.pop(ctx);
              final ids = _selected.toList();
              _selected.clear();
              context.read<AIBriefingProvider>().deleteManyBriefings(ids).then((ok) {
                if (ok) {
                  _reload();
                  setState(() {});
                }
              });
            },
            child: const Text('删除'),
          ),
        ],
      ),
    );
  }

  void _confirmClearAll() {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('清空历史'),
        content: const Text('确定清空全部资讯记录？此操作不可恢复。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              Navigator.pop(ctx);
              context.read<AIBriefingProvider>().clearAll().then((ok) {
                if (ok) _reload();
              });
            },
            child: const Text('清空'),
          ),
        ],
      ),
    );
  }

  void _confirmDeleteSource(AIBriefingSource s) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除来源'),
        content: Text('确定删除来源「${s.name}」？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              Navigator.pop(ctx);
              context.read<AIBriefingProvider>().deleteSource(s.id).then((ok) {
                if (ok) context.read<AIBriefingProvider>().fetchSources();
              });
            },
            child: const Text('删除'),
          ),
        ],
      ),
    );
  }
}
