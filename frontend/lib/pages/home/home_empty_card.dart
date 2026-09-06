import 'package:flutter/material.dart';

class HomeEmptyCard extends StatelessWidget {
  const HomeEmptyCard({super.key, required this.message, required this.icon});
  final String message;
  final IconData icon;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
        width: double.infinity,
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.3),
            borderRadius: BorderRadius.circular(12)),
        child: Column(children: [
          Icon(icon, size: 32, color: theme.colorScheme.onSurfaceVariant),
          const SizedBox(height: 8),
          Text(message,
              style: TextStyle(
                  color: theme.colorScheme.onSurfaceVariant, fontSize: 14))
        ]));
  }
}
