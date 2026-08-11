import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/models.dart';
import '../../providers/admin_provider.dart';
import '../../widgets/error_view.dart';

/// 我的操作日志 — 所有角色查看自己的操作记录
class MyLogsPage extends StatefulWidget {
  const MyLogsPage({super.key});

  @override
  State<MyLogsPage> createState() => _MyLogsPageState();
}

class _MyLogsPageState extends State<MyLogsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AdminProvider>().fetchMyLogs();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<AdminProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('我的操作日志')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchMyLogs(),
        child: provider.myLogsLoading && provider.myLogs.isEmpty
            ? const Center(child: CircularProgressIndicator())
            : provider.error.isNotEmpty && provider.myLogs.isEmpty
                ? ErrorView.error(
                    message: provider.error,
                    onRetry: () => provider.fetchMyLogs())
                : _buildContent(theme, provider),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, AdminProvider provider) {
    if (provider.myLogs.isEmpty) {
      return const Center(child: Text('暂无操作记录'));
    }
    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: provider.myLogs.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        final log = provider.myLogs[index];
        return _buildLogCard(theme, log);
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
                Text('结果 ${log.resultCode} · ${log.durationMs}ms',
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
}
