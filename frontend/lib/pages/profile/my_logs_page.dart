import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../config/api_config.dart';
import '../../models/models.dart';
import '../../providers/admin_provider.dart';
import '../../services/api_service.dart';
import '../../widgets/error_view.dart';

/// 我的操作日志 — 所有角色查看自己的操作记录（可管理：删除/清空）
class MyLogsPage extends StatefulWidget {
  const MyLogsPage({super.key});

  @override
  State<MyLogsPage> createState() => _MyLogsPageState();
}

class _MyLogsPageState extends State<MyLogsPage> {
  String _view = 'user'; // user=仅操作 | all=全部

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchMyLogs();
    });
  }

  void _reload() {
    context.read<AdminProvider>().fetchMyLogs();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<AdminProvider>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的操作日志'),
        actions: [
          IconButton(
            tooltip: '清空我的日志',
            icon: const Icon(Icons.delete_sweep_outlined),
            onPressed: _confirmClear,
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
            child: Row(
              children: [
                SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(value: 'user', label: Text('我的操作')),
                    ButtonSegment(value: 'all', label: Text('全部记录')),
                  ],
                  selected: {_view},
                  onSelectionChanged: (s) {
                    setState(() => _view = s.first);
                    // 切换视图通过后端过滤
                    context
                        .read<AdminProvider>()
                        .fetchMyLogsAll(_view == 'all');
                  },
                ),
                const Spacer(),
                Text('共 ${provider.myLogsTotal} 条',
                    style: theme.textTheme.bodySmall),
              ],
            ),
          ),
          const SizedBox(height: 4),
          Expanded(
            child: RefreshIndicator(
              onRefresh: () => provider.fetchMyLogs(),
              child: provider.myLogsLoading && provider.myLogs.isEmpty
                  ? const Center(child: CircularProgressIndicator())
                  : provider.error.isNotEmpty && provider.myLogs.isEmpty
                      ? ErrorView.error(
                          message: provider.error, onRetry: _reload)
                      : provider.myLogs.isEmpty
                          ? ErrorView.empty(
                              message: '暂无操作记录', icon: Icons.history)
                          : _buildList(theme, provider),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildList(ThemeData theme, AdminProvider provider) {
    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: provider.myLogs.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        final log = provider.myLogs[index];
        return Dismissible(
          key: ValueKey(log.id),
          direction: DismissDirection.endToStart,
          onDismissed: (_) => _deleteOne(log.id),
          background: Container(
            alignment: Alignment.centerRight,
            padding: const EdgeInsets.only(right: 20),
            color: theme.colorScheme.errorContainer,
            child: Icon(Icons.delete_outline,
                color: theme.colorScheme.error),
          ),
          child: _buildLogCard(theme, log),
        );
      },
    );
  }

  Widget _buildLogCard(ThemeData theme, AuditLog log) {
    return Card(
      elevation: 0,
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
                _methodChip(theme, log.action),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(log.resource,
                      style: theme.textTheme.bodyMedium
                          ?.copyWith(fontWeight: FontWeight.w600),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis),
                ),
              ],
            ),
            if (log.detail.isNotEmpty && log.detail != log.resource) ...[
              const SizedBox(height: 4),
              Text(log.detail,
                  style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant)),
            ],
            const SizedBox(height: 6),
            Row(
              children: [
                Icon(Icons.schedule, size: 14,
                    color: theme.colorScheme.outline),
                const SizedBox(width: 4),
                Text(log.createdAt, style: theme.textTheme.bodySmall),
                const Spacer(),
                Text('结果 ${log.resultCode}',
                    style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant)),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _methodChip(ThemeData theme, String method) {
    final color = switch (method) {
      'POST' => const Color(0xFF2E7D32),
      'PUT' || 'PATCH' => const Color(0xFF1565C0),
      'DELETE' => const Color(0xFFC62828),
      _ => theme.colorScheme.outline,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(method,
          style: theme.textTheme.labelSmall?.copyWith(
              color: color, fontWeight: FontWeight.w700)),
    );
  }

  Future<void> _deleteOne(int id) async {
    try {
      await ApiService().delete(ApiConfig.myLog('$id'));
      _reload();
    } catch (_) {}
  }

  Future<void> _confirmClear() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('清空我的日志'),
        content: const Text('确定清空自己的操作日志吗？'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消')),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('清空')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ApiService().delete(ApiConfig.myLog('0'));
      _reload();
    } catch (_) {}
  }
}
