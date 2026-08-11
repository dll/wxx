import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/admin_provider.dart';
import '../../widgets/error_view.dart';

/// 审计日志页面（college_admin 及以上可访问）
/// 含两个 Tab：审计日志 + 恢复操作
class AdminAuditPage extends StatefulWidget {
  const AdminAuditPage({super.key});

  @override
  State<AdminAuditPage> createState() => _AdminAuditPageState();
}

class _AdminAuditPageState extends State<AdminAuditPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tab;
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 2, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchAuditLogs();
      context.read<AdminProvider>().fetchSnapshots();
    });
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _tab.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >= _scrollController.position.maxScrollExtent - 200) {
      final provider = context.read<AdminProvider>();
      if (!provider.auditLoading && provider.auditLogs.length < provider.auditTotal) {
        provider.fetchAuditLogs();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('审计日志'),
        bottom: TabBar(
          controller: _tab,
          tabs: const [
            Tab(text: '审计日志'),
            Tab(text: '恢复操作'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tab,
        children: [
          _buildAuditTab(context),
          const _RestoreTab(),
        ],
      ),
    );
  }

  Widget _buildAuditTab(BuildContext context) {
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        if (provider.auditLoading && provider.auditLogs.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.error.isNotEmpty && provider.auditLogs.isEmpty) {
          return ErrorView.error(
            message: provider.error,
            onRetry: () => provider.fetchAuditLogs(),
          );
        }
        if (provider.auditLogs.isEmpty) {
          return ErrorView.empty(
              message: '暂无审计日志', icon: Icons.history);
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchAuditLogs(refresh: true),
          child: ListView.builder(
            controller: _scrollController,
            padding: const EdgeInsets.all(12),
            itemCount: provider.auditLogs.length + (provider.auditLogs.length < provider.auditTotal ? 1 : 0),
            itemBuilder: (context, index) {
              if (index == provider.auditLogs.length) {
                return const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                );
              }
              return _AuditTile(log: provider.auditLogs[index]);
            },
          ),
        );
      },
    );
  }

  /// 确认并清空全部审计日志
  Future<void> _confirmClear(BuildContext context) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('清空审计日志'),
        content: const Text('确定清空全部审计日志吗？此操作不可恢复。'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('清空')),
        ],
      ),
    );
    if (ok != true) return;
    if (!mounted) return;
    final success =
        await context.read<AdminProvider>().clearAuditLogs();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(success ? '审计日志已清空' : '清空失败，请稍后重试')),
    );
  }
}

/// 恢复操作 Tab — 展示可恢复的操作快照，供管理员回滚
class _RestoreTab extends StatelessWidget {
  const _RestoreTab();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Consumer<AdminProvider>(
      builder: (_, provider, __) {
        if (provider.snapshotsLoading && provider.snapshots.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }
        if (provider.snapshots.isEmpty) {
          return RefreshIndicator(
            onRefresh: () => provider.fetchSnapshots(),
            child: ListView(
              children: [
                const SizedBox(height: 160),
                ErrorView.empty(
                    message: '暂无可恢复的操作',
                    subtitle: '用户状态变更等可恢复操作会自动生成快照',
                    icon: Icons.restore),
              ],
            ),
          );
        }
        return RefreshIndicator(
          onRefresh: () => provider.fetchSnapshots(),
          child: ListView.separated(
            padding: const EdgeInsets.all(12),
            itemCount: provider.snapshots.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (context, index) {
              final s = provider.snapshots[index];
              return Card(
                elevation: 0,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(10),
                  side: BorderSide(color: theme.colorScheme.outlineVariant),
                ),
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Row(
                    children: [
                      const Icon(Icons.restore, color: Colors.orange),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(_opLabel(s.operation),
                                style: theme.textTheme.bodyMedium
                                    ?.copyWith(fontWeight: FontWeight.w600)),
                            const SizedBox(height: 2),
                            Text(
                              '记录 #${s.recordId} · ${s.beforeJson} → ${s.afterJson} · ${s.createdAt}',
                              style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant),
                            ),
                          ],
                        ),
                      ),
                      FilledButton.tonal(
                        onPressed: () => _confirmRestore(context, s),
                        child: const Text('恢复'),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        );
      },
    );
  }

  String _opLabel(String op) {
    switch (op) {
      case 'user.status':
        return '用户状态变更';
      case 'kb.status':
        return '知识资源状态';
      case 'feedback.status':
        return '反馈状态';
      default:
        return op;
    }
  }

  Future<void> _confirmRestore(BuildContext context, AuditSnapshot s) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('确认恢复'),
        content: Text('将 ${s.afterJson} 回滚为 ${s.beforeJson}（记录 #${s.recordId}），确认执行？'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('确认恢复')),
        ],
      ),
    );
    if (ok != true) return;
    final messenger = ScaffoldMessenger.of(context);
    final ok2 = await context.read<AdminProvider>().restoreSnapshot(s.id);
    messenger.showSnackBar(
        SnackBar(content: Text(ok2 ? '恢复成功' : '恢复失败')));
  }
}

class _AuditTile extends StatelessWidget {
  final dynamic log;
  const _AuditTile({required this.log});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final code = log.resultCode;
    final isOk = code >= 200 && code < 300;

    return Card(
      margin: const EdgeInsets.only(bottom: 6),
      child: ListTile(
        dense: true,
        leading: CircleAvatar(
          radius: 16,
          backgroundColor: isOk
              ? Colors.green.withOpacity( 0.1)
              : Colors.red.withOpacity( 0.1),
          child: Icon(
            isOk ? Icons.check : Icons.close,
            size: 16,
            color: isOk ? Colors.green : Colors.red,
          ),
        ),
        title: Text(
          '${log.username} · ${log.actionLabel}',
          style: theme.textTheme.bodyMedium,
        ),
        subtitle: Text(
          '${log.resource}  ·  ${log.ip}  ·  ${log.createdAt}',
          style: theme.textTheme.bodySmall,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('${log.durationMs}ms',
                style: theme.textTheme.labelSmall),
            Text('$code',
                style: TextStyle(
                    fontSize: 11,
                    color: isOk ? Colors.green : Colors.red)),
          ],
        ),
      ),
    );
  }
}
