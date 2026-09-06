import 'package:flutter/material.dart';

import 'home_overview_item.dart';

/// 首页「今日概览」统计卡：课程/任务/通知/计划四项横向排布。
/// 注意：HomeOverviewItem 自带 Expanded，直接作为 Row 的平铺子项即可，
/// 不可再包一层 Expanded（会触发 Flexible 祖先断言）。
class HomeTodayOverview extends StatelessWidget {
  final int courses;
  final int tasks;
  final int unread;
  final int plans;
  const HomeTodayOverview({
    super.key,
    required this.courses,
    required this.tasks,
    required this.unread,
    required this.plans,
  });

  @override
  Widget build(BuildContext context) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('今日概览', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 12),
          Row(
            children: [
              HomeOverviewItem(
                icon: Icons.menu_book_outlined,
                label: '课程',
                count: courses,
                unit: '节',
                color: const Color(0xFF1565C0),
              ),
              const SizedBox(width: 8),
              HomeOverviewItem(
                icon: Icons.task_alt_outlined,
                label: '任务',
                count: tasks,
                unit: '个',
                color: const Color(0xFF2E7D32),
              ),
              const SizedBox(width: 8),
              HomeOverviewItem(
                icon: Icons.notifications_outlined,
                label: '通知',
                count: unread,
                unit: '条',
                color: const Color(0xFFE65100),
              ),
              const SizedBox(width: 8),
              HomeOverviewItem(
                icon: Icons.fact_check_outlined,
                label: '计划',
                count: plans,
                unit: '个',
                color: const Color(0xFF7B1FA2),
              ),
            ],
          ),
        ],
      );
}
