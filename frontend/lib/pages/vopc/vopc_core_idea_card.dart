import 'package:flutter/material.dart';

class VopcCoreIdeaCard extends StatelessWidget {
  final String title;
  final String body;
  const VopcCoreIdeaCard({super.key, required this.title, required this.body});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withOpacity(.4),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title,
            style: theme.textTheme.titleSmall
                ?.copyWith(fontWeight: FontWeight.w700)),
        const SizedBox(height: 6),
        Text(body, style: theme.textTheme.bodyMedium),
      ]),
    );
  }
}
