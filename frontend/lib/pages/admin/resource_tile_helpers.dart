import 'package:flutter/material.dart';

Widget resourceTypeIcon(String type) {
  final (icon, color) = switch (type) {
    'Policy' => (Icons.policy_outlined, Colors.blue),
    'Process' => (Icons.route_outlined, Colors.purple),
    'FAQ' => (Icons.question_answer_outlined, Colors.green),
    'Activity' => (Icons.event_outlined, Colors.orange),
    _ => (Icons.article_outlined, Colors.grey),
  };
  return CircleAvatar(
    radius: 20,
    backgroundColor: color.withOpacity(0.1),
    child: Icon(icon, size: 20, color: color),
  );
}

Widget resourceStatusBadge(String status, ThemeData theme) {
  final (bgColor, fgColor, label) = switch (status) {
    'draft' => (Colors.grey.withOpacity(0.1), Colors.grey, '草稿'),
    'pending' => (Colors.orange.withOpacity(0.1), Colors.orange, '待审核'),
    'published' => (Colors.green.withOpacity(0.1), Colors.green, '已发布'),
    'retired' => (Colors.red.withOpacity(0.1), Colors.red, '已下架'),
    _ => (
        theme.colorScheme.surfaceContainerHighest,
        theme.colorScheme.onSurfaceVariant,
        status
      ),
  };
  return Container(
    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
    decoration:
        BoxDecoration(color: bgColor, borderRadius: BorderRadius.circular(8)),
    child: Text(label, style: TextStyle(fontSize: 10, color: fgColor)),
  );
}
