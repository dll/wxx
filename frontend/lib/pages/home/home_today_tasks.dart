import 'package:flutter/material.dart';

import 'home_empty_card.dart';
import 'home_task_item.dart';

class HomeTodayTasks extends StatelessWidget {
  final List<Map<String, dynamic>> tasks;
  final VoidCallback onViewAll;
  final ValueChanged<Map<String, dynamic>> onToggle;
  const HomeTodayTasks(
      {super.key,
      required this.tasks,
      required this.onViewAll,
      required this.onToggle});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text('今日任务', style: theme.textTheme.titleMedium),
        TextButton.icon(
            onPressed: onViewAll,
            icon: const Icon(Icons.arrow_forward, size: 16),
            label: const Text('查看全部', style: TextStyle(fontSize: 13))),
      ]),
      const SizedBox(height: 8),
      if (tasks.isEmpty)
        const HomeEmptyCard(
            message: '今日没有任务安排', icon: Icons.check_circle_outline)
      else
        Container(
          decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                  color: theme.colorScheme.outlineVariant.withOpacity(.5))),
          child: Column(children: [
            for (var i = 0; i < tasks.length; i++) ...[
              HomeTaskItem(task: tasks[i], onTap: () => onToggle(tasks[i])),
              if (i < tasks.length - 1)
                Divider(
                    height: 1,
                    indent: 16,
                    endIndent: 16,
                    color: theme.colorScheme.outlineVariant.withOpacity(.3)),
            ],
          ]),
        ),
    ]);
  }
}
