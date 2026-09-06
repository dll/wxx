import 'package:flutter/material.dart';

class VopcFlowStrip extends StatelessWidget {
  final List<Map<String, dynamic>> steps;
  const VopcFlowStrip({super.key, required this.steps});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final items = steps
        .map((s) => _FlowNode(
              title: s['title']?.toString() ?? '',
              desc: s['desc']?.toString() ?? '',
            ))
        .toList();
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        for (var i = 0; i < items.length; i++) ...[
          items[i],
          if (i < items.length - 1)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Icon(Icons.arrow_forward_rounded,
                  size: 16, color: theme.colorScheme.outline),
            ),
        ],
      ],
    );
  }
}

class _FlowNode extends StatelessWidget {
  final String title;
  final String desc;
  const _FlowNode({required this.title, required this.desc});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title,
            style: theme.textTheme.labelMedium
                ?.copyWith(fontWeight: FontWeight.w700)),
        if (desc.isNotEmpty)
          Text(desc,
              style: theme.textTheme.labelSmall
                  ?.copyWith(color: theme.colorScheme.outline)),
      ]),
    );
  }
}
