import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/knowledge_provider.dart';
import '../../widgets/error_view.dart';

/// 知识大厅页面 — 卡片式分类浏览
class BrowsePage extends StatefulWidget {
  const BrowsePage({super.key});

  @override
  State<BrowsePage> createState() => _BrowsePageState();
}

class _BrowsePageState extends State<BrowsePage> {
  @override
  void initState() {
    super.initState();
    // 首次加载
    Future.microtask(() {
      if (mounted) context.read<KnowledgeProvider>().load();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<KnowledgeProvider>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('知识大厅'),
        centerTitle: false,
      ),
      body: _buildBody(context, provider, theme),
    );
  }

  Widget _buildBody(BuildContext context, KnowledgeProvider provider, ThemeData theme) {
    if (provider.loading && provider.isEmpty) {
      return _buildSkeleton(theme);
    }

    if (provider.error != null) {
      return _buildError(context, provider, theme);
    }

    return Column(
      children: [
        // 类型过滤条
        _buildFilterBar(context, provider, theme),
        // 知识卡片列表
        Expanded(child: _buildContent(context, provider, theme)),
      ],
    );
  }

  /// 类型过滤 chip 行
  Widget _buildFilterBar(BuildContext context, KnowledgeProvider provider, ThemeData theme) {
    const filters = [
      {'label': '全部', 'value': ''},
      {'label': '政策', 'value': 'Policy'},
      {'label': '流程', 'value': 'Process'},
      {'label': '问答', 'value': 'FAQ'},
      {'label': '活动', 'value': 'Activity'},
    ];

    return SizedBox(
      height: 48,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: filters.length,
        separatorBuilder: (_, __) => const SizedBox(width: 8),
        itemBuilder: (context, index) {
          final f = filters[index];
          final selected = provider.selectedType == f['value'];
          return FilterChip(
            label: Text(f['label']!),
            selected: selected,
            onSelected: (_) => provider.filterByType(
              f['value']!.isEmpty ? null : f['value'],
            ),
            showCheckmark: false,
          );
        },
      ),
    );
  }

  /// 知识卡片内容区
  Widget _buildContent(BuildContext context, KnowledgeProvider provider, ThemeData theme) {
    if (provider.isEmpty) {
      return _buildEmpty(theme);
    }

    final categories = provider.orderedCategories;

    return RefreshIndicator(
      onRefresh: () => provider.load(type: provider.selectedType.isEmpty ? null : provider.selectedType),
      child: ListView.builder(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        itemCount: categories.length,
        itemBuilder: (context, index) {
          final entry = categories[index];
          return _buildSection(context, entry.key, entry.value, theme);
        },
      ),
    );
  }

  /// 分类区块
  Widget _buildSection(BuildContext context, String type, List<KnowledgeCard> cards, ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // 分类标题行
        Padding(
          padding: const EdgeInsets.only(top: 16, bottom: 8),
          child: Row(
            children: [
              _typeIcon(type, size: 24),
              const SizedBox(width: 8),
              Text(
                _typeLabel(type),
                style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: theme.colorScheme.secondaryContainer,
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Text(
                  '${cards.length}',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSecondaryContainer,
                  ),
                ),
              ),
            ],
          ),
        ),
        // 卡片列表
        ...cards.map((card) => Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: _buildCard(context, card, theme),
            )),
      ],
    );
  }

  /// 单张知识卡片
  Widget _buildCard(BuildContext context, KnowledgeCard card, ThemeData theme) {
    final typeColor = _typeColor(card.resourceType, theme);

    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerLow,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant.withValues(alpha: 0.3)),
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => _onCardTap(context, card),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // 标题行：类型图标 + 标题
              Row(
                children: [
                  Container(
                    width: 36,
                    height: 36,
                    decoration: BoxDecoration(
                      color: typeColor.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Icon(_typeIconData(card.resourceType), size: 20, color: typeColor),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      card.title,
                      style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  Icon(Icons.chevron_right, size: 20, color: theme.colorScheme.outline),
                ],
              ),
              // 摘要
              if (card.summary.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  card.summary,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                    height: 1.4,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              // 标签
              if (card.tags.isNotEmpty) ...[
                const SizedBox(height: 10),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: card.tags.take(3).map((tag) => Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.tertiaryContainer.withValues(alpha: 0.5),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(
                          tag,
                          style: theme.textTheme.labelSmall?.copyWith(
                            color: theme.colorScheme.onTertiaryContainer,
                          ),
                        ),
                      )).toList(),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  /// 点击卡片 → 跳转到对话页发起提问
  void _onCardTap(BuildContext context, KnowledgeCard card) {
    context.go('/chat?ask=${Uri.encodeComponent(card.title)}');
  }

  // ── 工具方法 ──

  /// 类型对应图标
  static Widget _typeIcon(String type, {double size = 20}) {
    return Icon(_typeIconData(type), size: size);
  }

  static IconData _typeIconData(String type) {
    switch (type) {
      case 'Policy':
        return Icons.gavel;
      case 'Process':
        return Icons.account_tree_outlined;
      case 'FAQ':
        return Icons.help_outline;
      case 'Activity':
        return Icons.celebration_outlined;
      default:
        return Icons.article_outlined;
    }
  }

  /// 类型对应颜色
  static Color _typeColor(String type, ThemeData theme) {
    switch (type) {
      case 'Policy':
        return const Color(0xFFE53935); // 红色 — 政策
      case 'Process':
        return const Color(0xFF1E88E5); // 蓝色 — 流程
      case 'FAQ':
        return const Color(0xFF43A047); // 绿色 — 问答
      case 'Activity':
        return const Color(0xFFFB8C00); // 橙色 — 活动
      default:
        return theme.colorScheme.primary;
    }
  }

  /// 类型中文标签
  static String _typeLabel(String type) {
    const map = {
      'Policy': '政策文件',
      'Process': '办事流程',
      'FAQ': '常见问答',
      'Activity': '活动通知',
    };
    return map[type] ?? type;
  }

  // ── 状态占位 ──

  Widget _buildSkeleton(ThemeData theme) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: 6,
      itemBuilder: (context, index) => Card(
        elevation: 0,
        color: theme.colorScheme.surfaceContainerLow,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 36, height: 36,
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(10),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Container(
                    width: 160, height: 14,
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Container(
                width: double.infinity, height: 12,
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 6),
              Container(
                width: 200, height: 12,
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildEmpty(ThemeData theme) {
    return ErrorView.empty(
      message: '暂无知识内容',
      subtitle: '知识库正在建设中，敬请期待',
      icon: Icons.menu_book_outlined,
    );
  }

  Widget _buildError(BuildContext context, KnowledgeProvider provider, ThemeData theme) {
    return ErrorView.error(
      message: provider.error ?? '加载失败',
      onRetry: () => provider.load(),
    );
  }
}
