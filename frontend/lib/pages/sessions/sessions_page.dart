import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../providers/session_provider.dart';
import '../../providers/chat_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/skeleton.dart';
import '../../utils/date_utils.dart';

/// 会话历史页
class SessionsPage extends StatefulWidget {
  const SessionsPage({super.key});

  @override
  State<SessionsPage> createState() => _SessionsPageState();
}

class _SessionsPageState extends State<SessionsPage> {
  @override
  void initState() {
    super.initState();
    // 页面打开时加载会话列表
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) context.read<SessionProvider>().fetchSessions();
    });
  }

  @override
  Widget build(BuildContext context) {
    final sessionProv = context.watch<SessionProvider>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('对话历史'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => sessionProv.fetchSessions(),
          ),
        ],
      ),
      body: _buildBody(sessionProv, theme),
    );
  }

  Widget _buildBody(SessionProvider prov, ThemeData theme) {
    if (prov.loading) {
      return const SessionsSkeleton();
    }

    if (prov.error != null) {
      return ErrorView.error(
        message: prov.error!,
        onRetry: () => prov.fetchSessions(),
      );
    }

    if (prov.sessions.isEmpty) {
      return RefreshIndicator(
        onRefresh: () => prov.fetchSessions(),
        child: ListView(
          children: [
            SizedBox(
              height: MediaQuery.of(context).size.height * 0.6,
              child: ErrorView.empty(
                message: '暂无对话记录',
                icon: Icons.chat_bubble_outline,
              ),
            ),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () => prov.fetchSessions(),
      child: ListView.separated(
        padding: const EdgeInsets.all(12),
        itemCount: prov.sessions.length,
        separatorBuilder: (_, __) => const SizedBox(height: 4),
        itemBuilder: (context, index) {
          final session = prov.sessions[index];
          return Dismissible(
            key: Key(session.id),
            direction: DismissDirection.endToStart,
            background: Container(
              alignment: Alignment.centerRight,
              padding: const EdgeInsets.only(right: 20),
              margin: const EdgeInsets.symmetric(vertical: 2),
              decoration: BoxDecoration(
                color: theme.colorScheme.errorContainer,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(Icons.delete, color: theme.colorScheme.onErrorContainer),
            ),
            confirmDismiss: (direction) async {
              return await showDialog<bool>(
                context: context,
                builder: (ctx) => AlertDialog(
                  title: const Text('删除会话'),
                  content: Text('确定要删除「${session.title}」吗？'),
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.pop(ctx, false),
                      child: const Text('取消'),
                    ),
                    TextButton(
                      onPressed: () => Navigator.pop(ctx, true),
                      child: Text('删除', style: TextStyle(color: theme.colorScheme.error)),
                    ),
                  ],
                ),
              );
            },
            onDismissed: (_) async {
              final ok = await prov.deleteSession(session.id);
              if (!context.mounted) return;
              if (!ok) {
                // 删除失败：数据已回滚，刷新列表恢复 Dismissible 项
                prov.fetchSessions();
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('删除失败，已恢复')),
                );
              }
            },
            child: Card(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
                side: BorderSide(color: theme.colorScheme.outlineVariant),
              ),
              child: ListTile(
                contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                leading: CircleAvatar(
                  backgroundColor: theme.colorScheme.primaryContainer,
                  child: Icon(Icons.chat, color: theme.colorScheme.onPrimaryContainer, size: 20),
                ),
                title: Text(
                  session.title.isEmpty ? '新对话' : session.title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: session.updatedAt.isNotEmpty
                    ? Text(
                        TimeFormatter.dateTime(session.updatedAt),
                        style: theme.textTheme.bodySmall,
                      )
                    : null,
                trailing: PopupMenuButton<String>(
                  icon: const Icon(Icons.more_vert),
                  tooltip: '更多',
                  onSelected: (action) async {
                    if (action == 'rename') {
                      await _showRenameDialog(context, prov, session.id, session.title);
                    } else if (action == 'open') {
                      context.read<ChatProvider>().loadSession(session.id);
                      if (context.mounted) context.go('/chat');
                    }
                  },
                  itemBuilder: (_) => const [
                    PopupMenuItem(value: 'open', child: Row(children: [Icon(Icons.chat_bubble_outline, size: 18), SizedBox(width: 8), Text('打开')])),
                    PopupMenuItem(value: 'rename', child: Row(children: [Icon(Icons.edit_outlined, size: 18), SizedBox(width: 8), Text('重命名')])),
                  ],
                ),
                onTap: () {
                  // 加载该会话的消息，然后跳到对话页
                  context.read<ChatProvider>().loadSession(session.id);
                  context.go('/chat');
                },
              ),
            ),
          );
        },
      ),
    );
  }

  /// 重命名会话对话框
  Future<void> _showRenameDialog(BuildContext context, SessionProvider prov, String id, String currentTitle) async {
    final ctrl = TextEditingController(text: currentTitle);
    final newTitle = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('重命名会话'),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          maxLength: 60,
          decoration: const InputDecoration(
            hintText: '请输入新的会话名称',
            border: OutlineInputBorder(),
          ),
          textInputAction: TextInputAction.done,
          onSubmitted: (v) => Navigator.pop(ctx, v.trim()),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
            child: const Text('保存'),
          ),
        ],
      ),
    );
    ctrl.dispose();
    if (newTitle == null || newTitle.isEmpty || newTitle == currentTitle) return;
    if (!context.mounted) return;
    final ok = await prov.renameSession(id, newTitle);
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '已重命名' : '重命名失败')),
      );
    }
  }
}
