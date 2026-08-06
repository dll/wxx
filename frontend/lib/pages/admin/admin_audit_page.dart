import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/admin_provider.dart';
import '../../widgets/error_view.dart';

/// 审计日志页面（college_admin 及以上可访问）
class AdminAuditPage extends StatefulWidget {
  const AdminAuditPage({super.key});

  @override
  State<AdminAuditPage> createState() => _AdminAuditPageState();
}

class _AdminAuditPageState extends State<AdminAuditPage> {
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchAuditLogs();
    });
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
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
        actions: [
          IconButton(
            tooltip: '清空审计日志',
            icon: const Icon(Icons.delete_sweep_outlined),
            onPressed: () => _confirmClear(context),
          ),
        ],
      ),
      body: Consumer<AdminProvider>(
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
      ),
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
