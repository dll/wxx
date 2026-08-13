import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../config/api_config.dart';
import '../../services/api_service.dart';

/// 个性化推荐区块：根据用户历史提问与角色偏好，展示推荐的知识条目
class RecommendationSection extends StatefulWidget {
  const RecommendationSection({super.key});

  @override
  State<RecommendationSection> createState() => _RecommendationSectionState();
}

class _RecommendationSectionState extends State<RecommendationSection> {
  final ApiService _api = ApiService();
  List<Map<String, dynamic>>? _items;
  bool _loading = false;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    Future.microtask(_load);
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _failed = false;
    });
    try {
      final res = await _api.get(ApiConfig.recommendations, params: {'limit': '6'});
      if (res.statusCode == 200 && res.data != null) {
        final list = res.data is List ? res.data : res.data['data'] ?? [];
        setState(() {
          _items = list
              .whereType<Map>()
              .map((e) => Map<String, dynamic>.from(e))
              .toList();
        });
      }
    } catch (_) {
      // 推荐失败不阻塞知识大厅，静默隐藏
      setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_loading || _failed || _items == null || _items!.isEmpty) {
      return const SizedBox.shrink();
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 8, bottom: 8),
          child: Row(
            children: [
              Icon(Icons.auto_awesome, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 6),
              Text('为你推荐',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w600)),
              const Spacer(),
              if (_items!.length > 3)
                TextButton(
                  style: TextButton.styleFrom(
                    visualDensity: VisualDensity.compact,
                    padding: const EdgeInsets.symmetric(horizontal: 4),
                  ),
                  onPressed: _load,
                  child: const Text('换一批'),
                ),
            ],
          ),
        ),
        SizedBox(
          height: 112,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            itemCount: _items!.length.clamp(0, 6),
            separatorBuilder: (_, __) => const SizedBox(width: 10),
            itemBuilder: (context, i) {
              final item = _items![i];
              return _RecommendationCard(item: item);
            },
          ),
        ),
      ],
    );
  }
}

class _RecommendationCard extends StatelessWidget {
  final Map<String, dynamic> item;
  const _RecommendationCard({required this.item});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final type = (item['resource_type'] ?? '').toString();
    final (color, icon) = switch (type) {
      'Policy' => (Colors.blue, Icons.gavel_outlined),
      'Process' => (Colors.indigo, Icons.account_tree_outlined),
      'FAQ' => (Colors.teal, Icons.help_outline),
      'Activity' => (Colors.deepOrange, Icons.event_outlined),
      _ => (theme.colorScheme.primary, Icons.description_outlined),
    };
    final title = (item['title'] ?? '推荐内容').toString();
    final reason = (item['reason'] ?? '').toString();

    return GestureDetector(
      onTap: () => _open(context),
      child: Container(
        width: 200,
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerLow,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: theme.colorScheme.outlineVariant.withOpacity( 0.3)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, size: 16, color: color),
                const SizedBox(width: 4),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                  decoration: BoxDecoration(
                    color: color.withOpacity( 0.1),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(type,
                      style: TextStyle(fontSize: 10, color: color, fontWeight: FontWeight.w600)),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Expanded(
              child: Text(title,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600)),
            ),
            if (reason.isNotEmpty)
              Text(reason,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.primary)),
          ],
        ),
      ),
    );
  }

  void _open(BuildContext context) {
    final title = (item['title'] ?? '').toString();
    if (title.isEmpty) return;
    context.go('/chat?ask=${Uri.encodeComponent(title)}');
  }
}