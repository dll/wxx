import 'package:flutter/material.dart';

import 'home_course_item.dart';
import 'home_empty_card.dart';

class HomeTodayCourses extends StatelessWidget {
  final List<Map<String, dynamic>> courses;
  final VoidCallback onViewAll;
  const HomeTodayCourses(
      {super.key, required this.courses, required this.onViewAll});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text('今日课表', style: theme.textTheme.titleMedium),
        TextButton.icon(
            onPressed: onViewAll,
            icon: const Icon(Icons.arrow_forward, size: 16),
            label: const Text('查看全部', style: TextStyle(fontSize: 13))),
      ]),
      const SizedBox(height: 8),
      if (courses.isEmpty)
        const HomeEmptyCard(message: '今日没有课程', icon: Icons.coffee_outlined)
      else
        Container(
          decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                  color: theme.colorScheme.outlineVariant.withOpacity(.5))),
          child: Column(children: [
            for (var i = 0; i < courses.length; i++) ...[
              HomeCourseItem(course: courses[i]),
              if (i < courses.length - 1)
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
