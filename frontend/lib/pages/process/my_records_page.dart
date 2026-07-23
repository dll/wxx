import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/enrollment_provider.dart';
import '../../providers/process_record_provider.dart';
import '../../utils/date_utils.dart';
import '../../widgets/error_view.dart';

/// 我的办事记录（所有用户可见，列出后端持久化的办事流程进度）
class MyRecordsPage extends StatefulWidget {
  const MyRecordsPage({super.key});

  @override
  State<MyRecordsPage> createState() => _MyRecordsPageState();
}

class _MyRecordsPageState extends State<MyRecordsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ProcessRecordProvider>().fetchAll();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的办事记录'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => context.read<ProcessRecordProvider>().fetchAll(),
          ),
        ],
      ),
      body: Consumer<ProcessRecordProvider>(
        builder: (_, prov, __) {
          if (prov.loading && prov.records.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (prov.error != null && prov.records.isEmpty) {
            return ErrorView.error(
              message: prov.error!,
              onRetry: () => prov.fetchAll(),
            );
          }
          if (prov.records.isEmpty) {
            return ErrorView.empty(
              message: '还没有办事记录',
              icon: Icons.assignment_outlined,
            );
          }
          return RefreshIndicator(
            onRefresh: () => prov.fetchAll(),
            child: ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: prov.records.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (_, i) => _RecordCard(record: prov.records[i]),
            ),
          );
        },
      ),
    );
  }
}

class _RecordCard extends StatelessWidget {
  final ProcessRecord record;
  const _RecordCard({required this.record});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    Color statusColor;
    switch (record.status) {
      case 'completed':
        statusColor = Colors.green;
        break;
      case 'abandoned':
        statusColor = theme.colorScheme.error;
        break;
      default:
        statusColor = theme.colorScheme.primary;
    }

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () {
          // 跳到对应办事页面（按 flow_type 切换）
          final enroll = context.read<EnrollmentProvider>();
          if (enroll.flowType != record.flowType) {
            enroll.setFlowType(record.flowType);
          }
          context.go('/enrollment');
        },
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      record.flowLabel.isEmpty ? record.flowType : record.flowLabel,
                      style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: statusColor.withOpacity( 0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      record.statusLabel,
                      style: TextStyle(
                        fontSize: 11,
                        color: statusColor,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              if (record.totalSteps > 0) ...[
                LinearProgressIndicator(
                  value: record.progressRatio,
                  minHeight: 6,
                  borderRadius: BorderRadius.circular(3),
                ),
                const SizedBox(height: 6),
                Text(
                  '已完成 ${record.completedSteps.length} / ${record.totalSteps} 步',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
              if (record.updatedAt.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(
                  '最后更新：${TimeFormatter.dateTime(record.updatedAt)}',
                  style: theme.textTheme.labelSmall,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
