import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../models/models.dart';
import '../../providers/ai_briefing_provider.dart';
import '../../theme/app_theme.dart';
import '../../widgets/error_view.dart';

/// AI 简讯 — 用户端资讯门户（参考 AIHOT 形态）
/// 三个 Tab：精选流（日期分组）/ 热度榜 / 我的收藏；支持分类筛选、关键词搜索、收藏。
class AIBriefingsPage extends StatefulWidget {
  const AIBriefingsPage({super.key});

  @override
  State<AIBriefingsPage> createState() => _AIBriefingsPageState();
}

class _AIBriefingsPageState extends State<AIBriefingsPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tab;
  String _category = '';
  final TextEditingController _searchCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 3, vsync: this);
    _tab.addListener(() {
      if (!_tab.indexIsChanging) return;
      final p = context.read<AIBriefingProvider>();
      if (_tab.index == 1 && p.hotBriefings.isEmpty) {
        p.fetchHotBriefings();
      } else if (_tab.index == 2 && p.favoriteBriefings.isEmpty) {
        p.fetchFavoriteBriefings();
      }
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AIBriefingProvider>().fetchUserBriefings();
    });
  }

  @override
  void dispose() {
    _tab.dispose();
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<AIBriefingProvider>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('AI 简讯'),
        bottom: TabBar(
          controller: _tab,
          tabs: const [
            Tab(text: '精选'),
            Tab(text: '热度榜'),
            Tab(text: '我的收藏'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tab,
        children: [
          _buildFeedTab(theme, provider),
          _buildHotTab(theme, provider),
          _buildFavoritesTab(theme, provider),
        ],
      ),
    );
  }

  // ── 精选流 ──

  Widget _buildFeedTab(ThemeData theme, AIBriefingProvider provider) {
    return Column(
      children: [
        _buildFilterBar(theme),
        const Divider(height: 1),
        Expanded(
          child: RefreshIndicator(
            onRefresh: () => provider.fetchUserBriefings(category: _category),
            child: provider.userLoading && provider.userBriefings.isEmpty
                ? const Center(child: CircularProgressIndicator())
                : provider.userError.isNotEmpty &&
                        provider.userBriefings.isEmpty
                    ? _scrollableError(provider.userError,
                        () => provider.fetchUserBriefings(category: _category))
                    : provider.userBriefings.isEmpty
                        ? _scrollableEmpty('暂无资讯')
                        : _buildGroupedFeed(theme, provider),
          ),
        ),
      ],
    );
  }

  /// 按日期分组的信息流（今天 / 更早日期）
  Widget _buildGroupedFeed(ThemeData theme, AIBriefingProvider provider) {
    final grouped = <String, List<AIBriefing>>{};
    for (final b in provider.userBriefings) {
      final day = _dateKey(b.publishedAt);
      grouped.putIfAbsent(day, () => []).add(b);
    }
    final dates = grouped.keys.toList()..sort((a, b) => b.compareTo(a));

    return ListView(
      padding: const EdgeInsets.symmetric(vertical: 8),
      children: [
        for (final date in dates) ...[
          _GroupHeader(label: _dateLabel(date), weekday: _weekdayLabel(date)),
          for (final b in grouped[date]!) _buildBriefingCard(theme, b),
        ],
        const SizedBox(height: 16),
      ],
    );
  }

  // ── 热度榜 ──

  Widget _buildHotTab(ThemeData theme, AIBriefingProvider provider) {
    return RefreshIndicator(
      onRefresh: provider.fetchHotBriefings,
      child: provider.hotLoading && provider.hotBriefings.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : provider.hotBriefings.isEmpty
              ? ListView(children: [
                  const SizedBox(height: 120),
                  _scrollableEmpty('暂无热度数据'),
                ])
              : ListView(
                  padding: const EdgeInsets.symmetric(vertical: 8),
                  children: [
                    _buildHotHeader(theme),
                    for (final b in provider.hotBriefings)
                      _buildBriefingCard(theme, b),
                  ],
                ),
    );
  }

  Widget _buildHotHeader(ThemeData theme) {
    return Container(
      margin: const EdgeInsets.fromLTRB(16, 8, 16, 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        gradient: LinearGradient(
        colors: [
          AppColors.brandPrimary.withOpacity(0.12),
          AppColors.aiAccent.withOpacity(0.08),
        ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: theme.colorScheme.outlineVariant),
      ),
      child: Row(
        children: [
          Icon(Icons.local_fire_department, color: AppColors.attention, size: 22),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('今日热度榜',
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.w700)),
                Text('按热度值排序，热度越高越受关注',
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // ── 我的收藏 ──

  Widget _buildFavoritesTab(ThemeData theme, AIBriefingProvider provider) {
    return RefreshIndicator(
      onRefresh: provider.fetchFavoriteBriefings,
      child: provider.favoritesLoading && provider.favoriteBriefings.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : provider.favoriteBriefings.isEmpty
              ? ListView(children: [
                  const SizedBox(height: 120),
                  _scrollableEmpty('还没有收藏的资讯'),
                ])
              : ListView(
                  padding: const EdgeInsets.symmetric(vertical: 8),
                  children: [
                    for (final b in provider.favoriteBriefings)
                      _buildBriefingCard(theme, b),
                  ],
                ),
    );
  }

  // ── 通用构建 ──

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
              ...AIBriefingCategory.all.map(
                  (c) => PopupMenuItem(value: c.key, child: Text(c.label))),
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

  /// 信息流卡片（AIHOT 风格：来源 · 时间 · 热度 · 分类 · 标题 · 摘要 · 推荐理由 · 收藏）
  Widget _buildBriefingCard(ThemeData theme, AIBriefing b) {
    return Card(
      margin: const EdgeInsets.fromLTRB(16, 4, 16, 10),
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
                  if (b.source.isNotEmpty) _sourceBadge(theme, b.source),
                  const SizedBox(width: 8),
                  if (b.heat > 0) ...[
                    Icon(Icons.local_fire_department,
                        size: 14, color: AppColors.attention),
                    const SizedBox(width: 2),
                    Text('${b.heat}',
                        style: theme.textTheme.labelSmall?.copyWith(
                            color: AppColors.attention,
                            fontWeight: FontWeight.w700)),
                    const SizedBox(width: 8),
                  ],
                  Expanded(
                    child: Text(
                      AIBriefingCategory.labelOf(b.category),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  _timeText(theme, b.publishedAt),
                  const SizedBox(width: 4),
                  IconButton(
                    tooltip: b.favorited ? '取消收藏' : '收藏',
                    visualDensity: VisualDensity.compact,
                    iconSize: 20,
                    onPressed: () async {
                      final p = context.read<AIBriefingProvider>();
                      final ok = await p.toggleFavorite(b);
                      if (!ok && mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('操作失败，请重试')));
                      }
                    },
                    icon: Icon(
                      b.favorited ? Icons.star : Icons.star_border,
                      color: b.favorited
                          ? AppColors.attention
                          : theme.colorScheme.outline,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Text(b.topic,
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w700)),
              if (b.summary.isNotEmpty) ...[
                const SizedBox(height: 6),
                Text(b.summary,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodyMedium
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              ],
              if (b.reason.isNotEmpty) ...[
                const SizedBox(height: 8),
                _reasonBox(theme, b.reason),
              ],
              if (b.link.isNotEmpty) ...[
                const SizedBox(height: 8),
                Row(
                  children: [
                    Icon(Icons.open_in_new,
                        size: 14, color: theme.colorScheme.primary),
                    const SizedBox(width: 4),
                    Text('阅读原文',
                        style: theme.textTheme.labelSmall
                            ?.copyWith(color: theme.colorScheme.primary)),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Widget _sourceBadge(ThemeData theme, String source) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: theme.colorScheme.secondaryContainer.withOpacity(0.6),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(source,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSecondaryContainer,
              fontWeight: FontWeight.w600)),
    );
  }

  Widget _reasonBox(ThemeData theme, String reason) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: AppColors.aiAccent.withOpacity(0.06),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.aiAccent.withOpacity(0.25)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.auto_awesome, size: 15, color: AppColors.aiAccent),
          const SizedBox(width: 6),
          Expanded(
            child: Text(reason,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
        ],
      ),
    );
  }

  Widget _timeText(ThemeData theme, String publishedAt) {
    return Text(_timeLabel(publishedAt),
        style: theme.textTheme.labelSmall
            ?.copyWith(color: theme.colorScheme.outline));
  }

  Widget _scrollableError(String message, VoidCallback onRetry) {
    return ListView(
      children: [
        const SizedBox(height: 120),
        ErrorView.error(message: message, onRetry: onRetry),
      ],
    );
  }

  Widget _scrollableEmpty(String text) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.inbox_outlined,
                size: 48, color: Theme.of(context).colorScheme.outline),
            const SizedBox(height: 12),
            Text(text,
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: Theme.of(context).colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }

  // ── 工具 ──

  /// 从 "2006-01-02 15:04:05" 提取日期键
  String _dateKey(String publishedAt) {
    if (publishedAt.length >= 10) return publishedAt.substring(0, 10);
    return publishedAt;
  }

  /// 今天 → "今天"，昨天 → "昨天"，否则 → "M月d日"
  String _dateLabel(String dateKey) {
    final now = DateTime.now();
    final today = _dateKey(
        '${now.year.toString().padLeft(4, '0')}-${now.month.toString().padLeft(2, '0')}-${now.day.toString().padLeft(2, '0')}');
    if (dateKey == today) return '今天';
    final yesterday = _dateKey(now
        .subtract(const Duration(days: 1))
        .toIso8601String()
        .substring(0, 10));
    if (dateKey == yesterday) return '昨天';
    final parts = dateKey.split('-');
    if (parts.length == 3)
      return '${int.parse(parts[1])}月${int.parse(parts[2])}日';
    return dateKey;
  }

  String _weekdayLabel(String dateKey) {
    try {
      final parts = dateKey.split('-');
      final d = DateTime(
          int.parse(parts[0]), int.parse(parts[1]), int.parse(parts[2]));
      const week = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
      return week[d.weekday - 1];
    } catch (_) {
      return '';
    }
  }

  /// 从 "2006-01-02 15:04:05" 提取 HH:MM
  String _timeLabel(String publishedAt) {
    if (publishedAt.length >= 16) return publishedAt.substring(11, 16);
    return publishedAt;
  }

  Future<void> _openLink(String url) async {
    final uri = Uri.parse(url);
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('无法打开链接')));
      }
    }
  }
}

/// 日期分组头（今天 / 昨天 / 8月11日 · 周二）
class _GroupHeader extends StatelessWidget {
  final String label;
  final String weekday;
  const _GroupHeader({required this.label, this.weekday = ''});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 6),
      child: Row(
        children: [
          Text(label,
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w700)),
          if (weekday.isNotEmpty) ...[
            const SizedBox(width: 8),
            Text(weekday,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
          const SizedBox(width: 8),
          Expanded(child: Divider(color: theme.colorScheme.outlineVariant)),
        ],
      ),
    );
  }
}
