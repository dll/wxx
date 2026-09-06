import 'package:flutter/material.dart';

class HomeCourseItem extends StatelessWidget {
  final Map<String, dynamic> course;
  const HomeCourseItem({super.key, required this.course});

  Widget? _status(String time) {
    final parts = time.split('-');
    if (parts.length != 2) return null;
    final startParts = parts[0].trim().split(':'),
        endParts = parts[1].trim().split(':');
    if (startParts.length != 2 || endParts.length != 2) return null;
    final now = DateTime.now();
    final start = DateTime(now.year, now.month, now.day,
        int.tryParse(startParts[0]) ?? 0, int.tryParse(startParts[1]) ?? 0);
    final end = DateTime(now.year, now.month, now.day,
        int.tryParse(endParts[0]) ?? 0, int.tryParse(endParts[1]) ?? 0);
    late String label;
    late Color color;
    if (now.isBefore(start)) {
      final minutes = start.difference(now).inMinutes;
      if (minutes > 120) return null;
      label = minutes <= 0 ? '即将开始' : '$minutes 分钟后开始';
      color = Colors.orange;
    } else if (now.isAfter(end)) {
      return null;
    } else {
      label = '进行中';
      color = Colors.green;
    }
    return Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
        decoration: BoxDecoration(
            color: color.withOpacity(.12),
            borderRadius: BorderRadius.circular(10)),
        child: Text(label,
            style: TextStyle(
                fontSize: 11, color: color, fontWeight: FontWeight.w700)));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = Color(int.parse(
        (course['color'] as String? ?? '#1565C0').replaceAll('#', '0xFF')));
    final name = course['course_name'] ?? '未知课程';
    final time = course['time'] ?? '';
    final location = course['location'] ?? '';
    final teacher = course['teacher'] ?? '';
    final status = _status(time);
    return Padding(
        padding: const EdgeInsets.all(14),
        child: Row(children: [
          Container(
              width: 4,
              height: 48,
              decoration: BoxDecoration(
                  color: color, borderRadius: BorderRadius.circular(2))),
          const SizedBox(width: 12),
          Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                Text(name,
                    style: const TextStyle(
                        fontSize: 15, fontWeight: FontWeight.w600)),
                if (status != null) ...[const SizedBox(height: 4), status],
                const SizedBox(height: 4),
                Row(children: [
                  Icon(Icons.access_time,
                      size: 14, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Text(time,
                      style: TextStyle(
                          fontSize: 12,
                          color: theme.colorScheme.onSurfaceVariant)),
                  const SizedBox(width: 12),
                  Icon(Icons.location_on_outlined,
                      size: 14, color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 4),
                  Expanded(
                      child: Text(location,
                          style: TextStyle(
                              fontSize: 12,
                              color: theme.colorScheme.onSurfaceVariant),
                          overflow: TextOverflow.ellipsis)),
                ]),
                if (teacher.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text('教师：$teacher',
                      style: TextStyle(
                          fontSize: 11, color: theme.colorScheme.outline))
                ],
              ])),
        ]));
  }
}
