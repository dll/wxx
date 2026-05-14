import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../providers/bookmark_provider.dart';
import '../../providers/chat_provider.dart';
import '../../widgets/error_view.dart';

/// 收藏页面 — 展示所有已收藏的问答
class BookmarksPage extends StatefulWidget {
  const BookmarksPage({super.key});

  @override
  State<BookmarksPage> createState() => _BookmarksPageState();
}

class _BookmarksPageState extends State<BookmarksPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<BookmarkProvider>().load();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('我的收藏')),
      body: Consumer<BookmarkProvider>(
        builder: (_, provider, __) {
          if (!provider.loaded) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.bookmarks.isEmpty) {
            return ErrorView.empty(
              message: '暂无收藏',
              subtitle: '在对话页点击星标按钮收藏回答',
              icon: Icons.star_outline,
            );
          }
          return ListView.builder(
            padding: const EdgeInsets.all(12),
            itemCount: provider.bookmarks.length,
            itemBuilder: (context, index) {
              final entry = provider.bookmarks[index];
              return _BookmarkCard(
                entry: entry,
                onRemove: () => provider.remove(entry.id),
              );
            },
          );
        },
      ),
    );
  }
}

class _BookmarkCard extends StatelessWidget {
  final BookmarkEntry entry;
  final VoidCallback onRemove;

  const _BookmarkCard({required this.entry, required this.onRemove});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // 问题
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(Icons.help_outline, size: 18,
                    color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(entry.question,
                      style: theme.textTheme.titleSmall),
                ),
              ],
            ),
            const SizedBox(height: 8),
            // 回答摘要
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest
                    .withValues(alpha: 0.5),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                entry.conclusion,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
            const SizedBox(height: 8),
            // 操作栏
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  _formatTime(entry.createdAt),
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
                ),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    // 跳转对话（携带问题重新提问）
                    TextButton.icon(
                      onPressed: () {
                        context.read<ChatProvider>().ask(entry.question);
                        context.go('/chat');
                      },
                      icon: const Icon(Icons.chat, size: 16),
                      label: const Text('追问', style: TextStyle(fontSize: 12)),
                      style: TextButton.styleFrom(visualDensity: VisualDensity.compact),
                    ),
                    // 删除收藏
                    IconButton(
                      onPressed: onRemove,
                      icon: const Icon(Icons.delete_outline, size: 18),
                      tooltip: '取消收藏',
                      visualDensity: VisualDensity.compact,
                    ),
                  ],
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _formatTime(String iso) {
    if (iso.isEmpty) return '';
    try {
      final dt = DateTime.parse(iso);
      return '${dt.month}/${dt.day} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    } catch (_) {
      return '';
    }
  }
}
