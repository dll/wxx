import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../models/models.dart';
import '../../providers/ai_briefing_provider.dart';
import '../../widgets/error_view.dart';

/// AI 简讯 — 用户端资讯列表（序号/来源/主题/摘要/详情链接）
class AIBriefingsPage extends StatefulWidget {
  const AIBriefingsPage({super.key});

  @override
  State<AIBriefingsPage> createState() => _AIBriefingsPageState();
}

class _AIBriefingsPageState extends State<AIBriefingsPage> {
  String _category = '';
  final TextEditingController _searchCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AIBriefingProvider>().fetchUserBriefings();
    });
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<AIBriefingProvider>();

    return Scaffold(
      appBar: AppBar(title: const Text('AI 简讯')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchUserBriefings(category: _category),
        child: provider.userLoading && provider.userBriefings.isEmpty
            ? const Center(child: CircularProgressIndicator())
            : provider.userError.isNotEmpty && provider.userBriefings.isEmpty
                ? ErrorView.error(
                    message: provider.userError,
                    onRetry: () => provider
                        .fetchUserBriefings(category: _category))
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, AIBriefingProvider provider) {
    return Column(
      children: [
        _buildFilterBar(theme),
        const Divider(height: 1),
        Expanded(
          child: provider.userBriefings.isEmpty
              ? const Center(child: Text('暂无资讯'))
              : ListView.separated(
                  padding: const EdgeInsets.all(16),
                  itemCount: provider.userBriefings.length,
                  separatorBuilder: (_, __) => const SizedBox(height: 12),
                  itemBuilder: (context, index) =>
                      _buildBriefingCard(theme, provider.userBriefings[index], index),
                ),
        ),
      ],
    );
  }

  Widget _buildFilterBar(ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _searchCtrl,
              decoration: const InputDecoration(
                hintText: '搜索主题 / 摘要 / 关键词',
                isDense: true,
                prefixIcon: Icon(Icons.search, size: 20),
                border: OutlineInputBorder(),
              ),
              onSubmitted: (v) => context
                  .read<AIBriefingProvider>()
                  .fetchUserBriefings(category: _category, q: v.trim()),
            ),
          ),
          const SizedBox(width: 8),
          PopupMenuButton<String>(
            initialValue: _category,
            tooltip: '分类筛选',
            onSelected: (v) {
              setState(() => _category = v);
              context
                  .read<AIBriefingProvider>()
                  .fetchUserBriefings(category: v);
            },
            itemBuilder: (_) => [
              const PopupMenuItem(value: '', child: Text('全部分类')),
              ...AIBriefingCategory.all.map((c) =>
                  PopupMenuItem(value: c.key, child: Text(c.label))),
            ],
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                border: Border.all(color: theme.colorScheme.outlineVariant),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  Text(_category.isEmpty
                      ? '全部分类'
                      : AIBriefingCategory.labelOf(_category)),
                  const Icon(Icons.arrow_drop_down),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBriefingCard(
      ThemeData theme, AIBriefing b, int index) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: b.link.isNotEmpty ? () => _openLink(b.link) : null,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text('#${index + 1}',
                      style: theme.textTheme.labelMedium
                          ?.copyWith(color: theme.colorScheme.primary)),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      AIBriefingCategory.labelOf(b.category),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  if (b.source.isNotEmpty)
                    Text('来源：${b.source}',
                        style: theme.textTheme.bodySmall),
                ],
              ),
              const SizedBox(height: 8),
              Text(b.topic,
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w600)),
              if (b.summary.isNotEmpty) ...[
                const SizedBox(height: 6),
                Text(b.summary,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant)),
              ],
              const SizedBox(height: 8),
              Row(
                children: [
                  Icon(Icons.schedule, size: 14,
                      color: theme.colorScheme.outline),
                  const SizedBox(width: 4),
                  Text(b.publishedAt, style: theme.textTheme.bodySmall),
                  const Spacer(),
                  if (b.link.isNotEmpty) ...[
                    Icon(Icons.open_in_new, size: 14,
                        color: theme.colorScheme.primary),
                    const SizedBox(width: 4),
                    Text('详情链接', style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.primary)),
                  ],
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openLink(String url) async {
    final uri = Uri.parse(url);
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('无法打开链接')));
      }
    }
  }
}
