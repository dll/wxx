import 'package:flutter/material.dart';

class HomeCalendarBar extends StatelessWidget {
  const HomeCalendarBar(
      {super.key,
      required this.weekNo,
      required this.weekday,
      required this.semesterName});
  final dynamic weekNo;
  final String weekday;
  final String semesterName;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final foreground = theme.colorScheme.onSecondaryContainer;
    return Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
            gradient: LinearGradient(colors: [
              theme.colorScheme.secondaryContainer,
              theme.colorScheme.tertiaryContainer.withOpacity(0.7)
            ], begin: Alignment.topLeft, end: Alignment.bottomRight),
            borderRadius: BorderRadius.circular(16)),
        child: Row(children: [
          Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                  color: foreground.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(12)),
              child: Icon(Icons.calendar_today, color: foreground, size: 24)),
          const SizedBox(width: 12),
          Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                Text('第 $weekNo 周 · $weekday',
                    style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        color: foreground)),
                const SizedBox(height: 2),
                Text(semesterName,
                    style: TextStyle(
                        fontSize: 12, color: foreground.withOpacity(0.8)))
              ]))
        ]));
  }
}
