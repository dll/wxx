import 'package:flutter/material.dart';

class VopcHeroMetric extends StatelessWidget {
  const VopcHeroMetric({super.key, required this.label, required this.value});
  final String label, value;
  @override
  Widget build(BuildContext context) => Row(children: [
        Text(value,
            style: const TextStyle(
                color: Colors.white,
                fontSize: 24,
                fontWeight: FontWeight.w800)),
        const SizedBox(width: 7),
        Text(label,
            style:
                TextStyle(color: Colors.white.withOpacity(.82), fontSize: 12))
      ]);
}

class VopcSectionHeader extends StatelessWidget {
  const VopcSectionHeader(
      {super.key,
      required this.title,
      required this.subtitle,
      required this.icon});
  final String title, subtitle;
  final IconData icon;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(children: [
        Container(
            width: 34,
            height: 34,
            decoration: BoxDecoration(
                color: theme.colorScheme.primary.withOpacity(.1),
                borderRadius: BorderRadius.circular(10)),
            child: Icon(icon, size: 19, color: theme.colorScheme.primary)),
        const SizedBox(width: 10),
        Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700)),
          Text(subtitle,
              style: theme.textTheme.labelSmall
                  ?.copyWith(color: theme.colorScheme.outline)),
        ]),
      ]),
    );
  }
}
