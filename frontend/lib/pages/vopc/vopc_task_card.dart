import 'package:flutter/material.dart';
import '../../providers/vopc_provider.dart';
import 'vopc_meta_chip.dart';

class VopcTaskCard extends StatelessWidget {
  const VopcTaskCard(
      {super.key,
      required this.task,
      required this.enabled,
      required this.onStatus});
  final VopcTask task;
  final bool enabled;
  final ValueChanged<String> onStatus;
  @override
  Widget build(BuildContext context) {
    final next = _nextStatuses[task.status] ?? const <String>[];
    return Card(
        margin: EdgeInsets.zero,
        child: Padding(
            padding: const EdgeInsets.all(14),
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Expanded(
                    child: Text(task.title,
                        style: Theme.of(context).textTheme.titleMedium)),
                VopcMetaChip(_priorityLabels[task.priority] ?? task.priority),
                const SizedBox(width: 6),
                VopcMetaChip(_statusLabels[task.status] ?? task.status)
              ]),
              const SizedBox(height: 10),
              Text('验收：${task.acceptanceCriteria}'),
              if (task.assigneeUserId != null ||
                  task.assigneeAiRole != null) ...[
                const SizedBox(height: 6),
                Text(task.assigneeUserId != null
                    ? '负责人：用户 #${task.assigneeUserId}'
                    : '负责人：AI ${task.assigneeAiRole}')
              ],
              if (next.isNotEmpty) ...[
                const SizedBox(height: 10),
                Wrap(
                    spacing: 8,
                    runSpacing: 6,
                    children: next
                        .map((status) => OutlinedButton(
                            onPressed: enabled ? () => onStatus(status) : null,
                            child: Text(_actionLabels[status] ?? status)))
                        .toList())
              ]
            ])));
  }

  static const _nextStatuses = <String, List<String>>{
    'todo': ['in_progress', 'cancelled'],
    'in_progress': ['todo', 'review', 'cancelled'],
    'review': ['in_progress', 'done']
  };
  static const _statusLabels = {
    'todo': '待开始',
    'in_progress': '进行中',
    'review': '待验收',
    'done': '已完成',
    'cancelled': '已取消'
  };
  static const _priorityLabels = {
    'low': '低',
    'normal': '普通',
    'high': '高',
    'urgent': '紧急'
  };
  static const _actionLabels = {
    'todo': '退回待开始',
    'in_progress': '开始 / 退回',
    'review': '提交验收',
    'done': '验收通过',
    'cancelled': '取消任务'
  };
}
