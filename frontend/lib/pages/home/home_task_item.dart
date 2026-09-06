import 'package:flutter/material.dart';

class HomeTaskItem extends StatelessWidget {
  final Map<String, dynamic> task;
  final VoidCallback onTap;
  const HomeTaskItem({super.key, required this.task, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title = task['title'] ?? '未命名任务';
    final duration = task['duration'] ?? 0;
    final completed = task['status'] == 'completed';
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(children: [
          Container(
              width: 24,
              height: 24,
              decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: completed
                      ? theme.colorScheme.primary
                      : Colors.transparent,
                  border: Border.all(
                      color: completed
                          ? theme.colorScheme.primary
                          : theme.colorScheme.outline,
                      width: 2)),
              child: completed
                  ? const Icon(Icons.check, size: 16, color: Colors.white)
                  : null),
          const SizedBox(width: 12),
          Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                Text(title,
                    style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w500,
                        decoration:
                            completed ? TextDecoration.lineThrough : null,
                        color: completed
                            ? theme.colorScheme.onSurfaceVariant
                            : null)),
                if (duration > 0) ...[
                  const SizedBox(height: 2),
                  Text('预计 $duration 分钟',
                      style: TextStyle(
                          fontSize: 12,
                          color: theme.colorScheme.onSurfaceVariant))
                ],
              ])),
        ]),
      ),
    );
  }
}
