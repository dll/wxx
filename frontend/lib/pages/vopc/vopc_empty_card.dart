import 'package:flutter/material.dart';

class VopcEmptyCard extends StatelessWidget {
  const VopcEmptyCard({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(children: [
          Icon(Icons.lightbulb_outline_rounded,
              size: 46, color: theme.colorScheme.primary),
          const SizedBox(height: 12),
          Text('还没有虚拟项目',
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 5),
          Text('从一个想法开始，走通 OPC 主线：需求 → 成果 → 反馈 → 复盘。',
              style: theme.textTheme.bodySmall),
        ]),
      ),
    );
  }
}
