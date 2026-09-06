import 'package:flutter/material.dart';

/// 报到流程步骤卡，仅负责展示步骤状态并回传用户选择。
class CampusStepCard extends StatelessWidget {
  const CampusStepCard(
      {super.key,
      required this.index,
      required this.total,
      required this.icon,
      required this.title,
      required this.location,
      required this.duration,
      required this.task,
      required this.materials,
      required this.contact,
      required this.note,
      required this.done,
      required this.active,
      required this.onTap});
  final int index, total;
  final IconData icon;
  final String title, location, duration, task, materials, contact, note;
  final bool done, active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = done
        ? const Color(0xFF2E7D32)
        : active
            ? theme.colorScheme.primary
            : theme.colorScheme.outline;
    return Card(
        elevation: active ? 2 : 0,
        color: active
            ? theme.colorScheme.primaryContainer.withOpacity(0.38)
            : null,
        shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(14),
            side: BorderSide(
                color: active
                    ? theme.colorScheme.primary
                    : theme.colorScheme.outlineVariant)),
        child: InkWell(
            borderRadius: BorderRadius.circular(14),
            onTap: onTap,
            child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Column(children: [
                        CircleAvatar(
                            radius: 18,
                            backgroundColor: color.withOpacity(0.12),
                            child: Icon(done ? Icons.check : icon,
                                size: 18, color: color)),
                        if (index < total - 1)
                          Container(
                              width: 2,
                              height: 92,
                              color: color.withOpacity(0.25))
                      ]),
                      const SizedBox(width: 12),
                      Expanded(
                          child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                            Row(children: [
                              Expanded(
                                  child: Text('${index + 1}. $title',
                                      style: theme.textTheme.titleSmall
                                          ?.copyWith(
                                              fontWeight: FontWeight.bold,
                                              color: done ? color : null))),
                              Text(
                                  done
                                      ? '已完成'
                                      : active
                                          ? '进行中'
                                          : '待办理',
                                  style: theme.textTheme.labelSmall
                                      ?.copyWith(color: color))
                            ]),
                            const SizedBox(height: 6),
                            _meta(Icons.place_outlined, location),
                            _meta(Icons.schedule, duration),
                            _meta(Icons.assignment_outlined, task),
                            _meta(Icons.inventory_2_outlined, '材料：$materials'),
                            _meta(Icons.phone_outlined, contact),
                            const SizedBox(height: 8),
                            Text(note,
                                style: theme.textTheme.bodySmall?.copyWith(
                                    color: theme.colorScheme.onSurfaceVariant))
                          ]))
                    ]))));
  }

  static Widget _meta(IconData icon, String text) => Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Icon(icon, size: 14),
        const SizedBox(width: 5),
        Expanded(child: Text(text, style: const TextStyle(fontSize: 12.5)))
      ]));
}
