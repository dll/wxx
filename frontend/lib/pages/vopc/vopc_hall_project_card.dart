import 'package:flutter/material.dart';
import '../../providers/vopc_provider.dart';
import 'vopc_meta_chip.dart';

class VopcHallProjectCard extends StatelessWidget {
  const VopcHallProjectCard(
      {super.key, required this.project, required this.onTap});
  final VopcProject project;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
        margin: const EdgeInsets.only(bottom: 10),
        elevation: 0,
        shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(14),
            side: BorderSide(
                color: theme.colorScheme.outlineVariant.withOpacity(.5))),
        child: InkWell(
            borderRadius: BorderRadius.circular(14),
            onTap: onTap,
            child: Padding(
                padding: const EdgeInsets.all(14),
                child: Row(children: [
                  Container(
                      width: 40,
                      height: 40,
                      decoration: BoxDecoration(
                          color: theme.colorScheme.tertiaryContainer
                              .withOpacity(.5),
                          borderRadius: BorderRadius.circular(12)),
                      child: Icon(Icons.storefront_outlined,
                          color: theme.colorScheme.tertiary, size: 20)),
                  const SizedBox(width: 12),
                  Expanded(
                      child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                        Text(project.name,
                            style: theme.textTheme.titleSmall
                                ?.copyWith(fontWeight: FontWeight.w700)),
                        const SizedBox(height: 5),
                        Wrap(spacing: 6, runSpacing: 6, children: [
                          VopcMetaChip(project.projectType),
                          VopcMetaChip(project.stage),
                          VopcMetaChip(project.riskLevel),
                          VopcMetaChip(project.visibility == 'private'
                              ? '私有'
                              : project.visibility)
                        ])
                      ])),
                  const SizedBox(width: 4),
                  Text('看看',
                      style: theme.textTheme.labelMedium
                          ?.copyWith(color: theme.colorScheme.primary)),
                  Icon(Icons.chevron_right, color: theme.colorScheme.outline)
                ]))));
  }
}
