import 'package:flutter/material.dart';

import 'vopc_section_widgets.dart';

class VopcHero extends StatelessWidget {
  final String title;
  final String subtitle;
  final int projectCount;
  final int pendingCount;

  const VopcHero({
    super.key,
    required this.title,
    required this.subtitle,
    required this.projectCount,
    required this.pendingCount,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            colors.primary,
            Color.alphaBlend(colors.tertiary.withOpacity(.35), colors.primary)
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(22),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
                color: Colors.white.withOpacity(.16),
                borderRadius: BorderRadius.circular(15)),
            child: const Icon(Icons.rocket_launch_rounded, color: Colors.white),
          ),
          const SizedBox(width: 14),
          Expanded(
              child: Text(title,
                  style: theme.textTheme.headlineSmall?.copyWith(
                      color: Colors.white, fontWeight: FontWeight.w800))),
        ]),
        const SizedBox(height: 12),
        Text(subtitle,
            style: theme.textTheme.bodyMedium
                ?.copyWith(color: Colors.white.withOpacity(.88))),
        const SizedBox(height: 20),
        Row(children: [
          VopcHeroMetric(label: '我的虚拟项目', value: '$projectCount'),
          const SizedBox(width: 28),
          VopcHeroMetric(label: '待处理邀请', value: '$pendingCount'),
        ]),
      ]),
    );
  }
}
