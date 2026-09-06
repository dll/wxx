import 'package:flutter/material.dart';

/// 问芯发送中的状态气泡。
class ChatLoadingBubble extends StatelessWidget {
  const ChatLoadingBubble({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Align(
        alignment: Alignment.centerLeft,
        child: Container(
            margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
            decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(16)),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: theme.colorScheme.primary)),
              const SizedBox(width: 10),
              Text('思考中...', style: TextStyle(color: theme.colorScheme.outline))
            ])));
  }
}
