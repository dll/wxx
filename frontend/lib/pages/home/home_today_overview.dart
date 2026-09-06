import 'package:flutter/material.dart';

import 'home_overview_item.dart';

class HomeTodayOverview extends StatelessWidget {
  final int courses;
  final int tasks;
  final int unread;
  final int plans;
  const HomeTodayOverview(
      {super.key,
      required this.courses,
      required this.tasks,
      required this.unread,
      required this.plans});

  Widget _item(
          {required IconData icon,
          required String label,
          required int count,
          required String unit,
          required Color color}) =>
      Expanded(
          child: HomeOverviewItem(
              icon: icon,
              label: label,
              count: count,
              unit: unit,
              color: color));

  @override
  Widget build(BuildContext context) =>
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('今日概览', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(
            children: [
          _item(
              icon: Icons.menu_book_outlined,
              label: '课程',
              count: courses,
              unit: '节',
              color: const Color(0xFF1565C0)),
          const SizedBox(width: 8),
          _item(
              icon: Icons.task_alt_outlined,
              label: '任务',
              count: tasks,
              unit: '个',
              color: const Color(0xFF2E7D32)),
          const SizedBox(width: 8),
          _item(
              icon: Icons.notifications_outlined,
              label: '通知',
              count: unread,
              unit: '条',
              color: const Color(0xFFE65100)),
          const SizedBox(width: 8),
          _item(
              icon: Icons.fact_check_outlined,
              label: '计划',
              count: plans,
              unit: '个',
              color: const Color(0xFF7B1FA2)),
        ]
                .indexed
                .map((entry) {
                  final item = entry.$2;
                  final count = switch (entry.$1) {
                    0 => courses,
                    1 => tasks,
                    2 => unread,
                    _ => plans
                  };
                  return entry.$1 == 0
                      ? item.copyWith(count: count)
                      : Row(children: [item.copyWith(count: count)]);
                })
                .expand((e) => e is Row ? e.children : [e])
                .toList()),
      ]);
}
