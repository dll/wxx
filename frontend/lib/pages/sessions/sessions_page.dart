import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import '../../providers/session_provider.dart';
import '../../providers/chat_provider.dart';

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
      return const Center(child: CircularProgressIndicator());
    }

    if (prov.error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.error_outline, size: 48, color: theme.colorScheme.error),
            const SizedBox(height: 12),
            Text(prov.error!, style: theme.textTheme.bodyLarge),
            const SizedBox(height: 12),
            FilledButton.tonal(
              onPressed: () => prov.fetchSessions(),
              child: const Text('重试'),
            ),
          ],
        ),
      );
    }

    if (prov.sessions.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.chat_bubble_outline, size: 48, color: theme.colorScheme.outline),
            const SizedBox(height: 12),
            Text('暂无对话记录', style: TextStyle(color: theme.colorScheme.outline)),
          ],
        ),
      );
    }

    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: prov.sessions.length,
      separatorBuilder: (_, __) => const SizedBox(height: 4),
      itemBuilder: (context, index) {
        final session = prov.sessions[index];
        return Card(
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
              session.title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            subtitle: session.updatedAt.isNotEmpty
                ? Text(
                    _formatTime(session.updatedAt),
                    style: theme.textTheme.bodySmall,
                  )
                : null,
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              // 加载该会话的消息，然后跳到对话页
              context.read<ChatProvider>().loadSession(session.id);
              context.go('/chat');
            },
          ),
        );
      },
    );
  }

  String _formatTime(String isoTime) {
    try {
      final dt = DateTime.parse(isoTime);
      final now = DateTime.now();
      if (dt.year == now.year && dt.month == now.month && dt.day == now.day) {
        return '今天 ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
      }
      return '${dt.month}/${dt.day} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    } catch (_) {
      return isoTime;
    }
  }
}
