import 'package:flutter/material.dart';

/// 学生首页加载骨架，独立于首页数据请求与角色状态。
class StudentHomeSkeleton extends StatelessWidget {
  const StudentHomeSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final placeholder =
        theme.colorScheme.surfaceContainerHighest.withOpacity(0.5);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
            height: 72,
            decoration: BoxDecoration(
                color: placeholder, borderRadius: BorderRadius.circular(16))),
        const SizedBox(height: 16),
        Text('今日概览', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(children: [
          for (var i = 0; i < 4; i++) ...[
            Expanded(
                child: Container(
                    height: 88,
                    decoration: BoxDecoration(
                        color: placeholder,
                        borderRadius: BorderRadius.circular(12)))),
            if (i < 3) const SizedBox(width: 8)
          ]
        ]),
        const SizedBox(height: 16),
        Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
          Text('今日课表', style: theme.textTheme.titleMedium),
          _bar(80, 16, placeholder)
        ]),
        const SizedBox(height: 8),
        _bar(double.infinity, 80, placeholder),
        const SizedBox(height: 16),
        Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
          Text('今日任务', style: theme.textTheme.titleMedium),
          _bar(80, 16, placeholder)
        ]),
        const SizedBox(height: 8),
        _bar(double.infinity, 60, placeholder),
      ],
    );
  }

  static Widget _bar(double width, double height, Color color) => Container(
      width: width,
      height: height,
      decoration:
          BoxDecoration(color: color, borderRadius: BorderRadius.circular(12)));
}
