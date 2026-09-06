import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class HomeGradeGrowthCard extends StatelessWidget {
  final int grade;
  final Color accent;
  final String themeName;
  const HomeGradeGrowthCard(
      {super.key,
      required this.grade,
      required this.accent,
      required this.themeName});

  @override
  Widget build(BuildContext context) {
    if (grade < 2 || grade > 4) return const SizedBox.shrink();
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
          gradient: LinearGradient(
              colors: [accent.withOpacity(.16), accent.withOpacity(.06)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: accent.withOpacity(.25))),
      child: Row(children: [
        Icon(Icons.auto_awesome, color: accent, size: 28),
        const SizedBox(width: 12),
        Expanded(
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('本阶段成长计划',
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w800)),
          const SizedBox(height: 2),
          Text('$themeName阶段专属：现在该做什么，一看就懂',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
        ])),
        TextButton(
            onPressed: () => context.go('/student/grade-growth'),
            child: const Text('查看')),
      ]),
    );
  }
}
