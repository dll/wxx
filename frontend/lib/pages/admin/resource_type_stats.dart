import 'package:flutter/material.dart';

class ResourceTypeStats extends StatelessWidget {
  const ResourceTypeStats({super.key, required this.byType});
  final dynamic byType;
  @override
  Widget build(BuildContext context) {
    if (byType is! Map) return const SizedBox.shrink();
    const labels = {
      'Policy': '政策',
      'Process': '流程',
      'FAQ': '问答',
      'Activity': '活动'
    };
    const colors = {
      'Policy': Colors.blue,
      'Process': Colors.purple,
      'FAQ': Colors.green,
      'Activity': Colors.orange
    };
    return Row(children: [
      for (final entry in byType.entries)
        Padding(
            padding: const EdgeInsets.only(right: 8),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                      color: colors[entry.key] ?? Colors.grey,
                      shape: BoxShape.circle)),
              const SizedBox(width: 3),
              Text(
                  '${labels[entry.key] ?? entry.key} ${entry.value is int ? entry.value : 0}',
                  style: TextStyle(
                      fontSize: 11, color: colors[entry.key] ?? Colors.grey))
            ]))
    ]);
  }
}
