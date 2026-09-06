import 'package:flutter/material.dart';

class HomeErrorCard extends StatelessWidget {
  const HomeErrorCard(
      {super.key, required this.message, required this.onRetry});
  final String? message;
  final VoidCallback onRetry;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
            color: theme.colorScheme.errorContainer.withOpacity(0.3),
            borderRadius: BorderRadius.circular(12)),
        child: Column(children: [
          Icon(Icons.error_outline, color: theme.colorScheme.error, size: 32),
          const SizedBox(height: 8),
          Text('加载失败',
              style: TextStyle(
                  color: theme.colorScheme.error, fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          Text(message ?? '未知错误',
              style: TextStyle(
                  color: theme.colorScheme.onSurfaceVariant, fontSize: 12),
              textAlign: TextAlign.center),
          const SizedBox(height: 12),
          TextButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('重新加载'))
        ]));
  }
}
