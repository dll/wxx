import 'package:flutter/material.dart';

class HomeEventItem extends StatelessWidget {
  final Map<String, dynamic> event;
  const HomeEventItem({super.key, required this.event});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = event['event_name'] ?? '';
    final type = event['event_type'] ?? '';
    final date = event['start_date'] ?? '';
    final daysLeft = event['days_left'] ?? 0;
    final icon = <String, IconData>{
          'holiday': Icons.celebration_outlined,
          'exam': Icons.edit_note_outlined,
          'registration': Icons.how_to_reg_outlined,
          'vacation': Icons.beach_access_outlined
        }[type] ??
        Icons.event_outlined;
    final color = <String, Color>{
          'holiday': const Color(0xFFE65100),
          'exam': const Color(0xFFC62828),
          'registration': const Color(0xFF2E7D32),
          'vacation': const Color(0xFF1565C0)
        }[type] ??
        theme.colorScheme.primary;
    final daysText = daysLeft < 0
        ? '已过${-daysLeft}天'
        : daysLeft == 0
            ? '今天'
            : '还有$daysLeft天';
    return Padding(
        padding: const EdgeInsets.all(14),
        child: Row(children: [
          Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                  color: color.withOpacity(.12),
                  borderRadius: BorderRadius.circular(10)),
              child: Icon(icon, color: color, size: 20)),
          const SizedBox(width: 12),
          Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                Text(name,
                    style: const TextStyle(
                        fontSize: 14, fontWeight: FontWeight.w600)),
                const SizedBox(height: 2),
                Text(date,
                    style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurfaceVariant))
              ])),
          Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                  color: color.withOpacity(.1),
                  borderRadius: BorderRadius.circular(999)),
              child: Text(daysText,
                  style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      color: color))),
        ]));
  }
}
