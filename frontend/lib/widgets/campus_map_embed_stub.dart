import 'package:flutter/material.dart';

class CampusMapEmbed extends StatelessWidget {
  final String url;
  final String title;

  const CampusMapEmbed({super.key, required this.url, required this.title});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.45),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: theme.colorScheme.outlineVariant),
      ),
      child: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.map_outlined,
                  size: 56, color: theme.colorScheme.primary),
              const SizedBox(height: 12),
              Text(title,
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 6),
              Text('Android 端使用下方按钮拉起系统地图导航',
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ],
          ),
        ),
      ),
    );
  }
}
