import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../providers/chat_provider.dart';

class ChatRoleSuggestions extends StatelessWidget {
  final String role;
  final List<String> questions;
  const ChatRoleSuggestions(
      {super.key, required this.role, required this.questions});

  String get label => switch (role) {
        'counselor' => '辅导员 · 为你推荐',
        'teacher' => '教师 · 为你推荐',
        'assistant' => '教辅 · 为你推荐',
        'college_admin' => '学院管理员 · 为你推荐',
        'school_admin' => '学校管理员 · 为你推荐',
        'sys_admin' => '系统管理员 · 为你推荐',
        'student_union' => '学生会 · 为你推荐',
        _ => '为你推荐',
      };

  @override
  Widget build(BuildContext context) {
    if (questions.isEmpty) return const SizedBox.shrink();
    final theme = Theme.of(context);
    final chat = context.read<ChatProvider>();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Icon(Icons.auto_awesome, size: 15, color: theme.colorScheme.primary),
        const SizedBox(width: 6),
        Text(label,
            style: theme.textTheme.labelLarge?.copyWith(
                color: theme.colorScheme.primary, fontWeight: FontWeight.w700))
      ]),
      const SizedBox(height: 8),
      Wrap(
          spacing: 8,
          runSpacing: 8,
          children: questions
              .map((q) => ActionChip(
                  label: Text(q, style: const TextStyle(fontSize: 12)),
                  visualDensity: VisualDensity.compact,
                  onPressed: () => chat.ask(q)))
              .toList()),
    ]);
  }
}
