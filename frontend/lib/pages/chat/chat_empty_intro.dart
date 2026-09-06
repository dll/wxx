import 'package:flutter/material.dart';

/// 问芯空会话欢迎头部，独立于推荐问题和消息状态。
class ChatEmptyIntro extends StatelessWidget {
  const ChatEmptyIntro({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final accent = theme.colorScheme.tertiary;
    return Column(children: [
      Container(
          width: 88,
          height: 88,
          decoration: BoxDecoration(
              gradient: LinearGradient(colors: [
                theme.colorScheme.primary,
                Color.alphaBlend(
                    theme.colorScheme.primary, accent.withOpacity(0.4))
              ], begin: Alignment.topLeft, end: Alignment.bottomRight),
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                    color: theme.colorScheme.primary.withOpacity(0.22),
                    blurRadius: 22,
                    offset: const Offset(0, 8))
              ]),
          child: const Icon(Icons.auto_awesome, size: 40, color: Colors.white)),
      const SizedBox(height: 18),
      Text('你好！我是蔚小芯',
          style: theme.textTheme.headlineSmall
              ?.copyWith(fontWeight: FontWeight.w800)),
      const SizedBox(height: 6),
      Text('内置 5 个专用智能体，有任何学工问题都可以问我',
          style: theme.textTheme.bodyMedium
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
      const SizedBox(height: 8),
      Text('选择上方智能体，或直接点击下方问题开始',
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.outline)),
    ]);
  }
}
