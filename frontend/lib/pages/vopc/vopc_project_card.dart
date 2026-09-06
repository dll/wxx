import 'package:flutter/material.dart';

import '../../providers/vopc_provider.dart';
import 'vopc_meta_chip.dart';

class VopcProjectCard extends StatelessWidget {
  final VopcProject project;
  final VoidCallback onTap;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const VopcProjectCard({
    super.key,
    required this.project,
    required this.onTap,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
        margin: const EdgeInsets.only(bottom: 12),
        elevation: 0,
        shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: BorderSide(
                color: theme.colorScheme.outlineVariant.withOpacity(.55))),
        child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: onTap,
            child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(children: [
                  Container(
                      width: 46,
                      height: 46,
                      decoration: BoxDecoration(
                          color: theme.colorScheme.primary.withOpacity(.1),
                          borderRadius: BorderRadius.circular(14)),
                      child: Icon(Icons.rocket_launch_outlined,
                          color: theme.colorScheme.primary)),
                  const SizedBox(width: 14),
                  Expanded(
                      child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                        Text(project.name,
                            style: theme.textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.w700)),
                        const SizedBox(height: 6),
                        Text(
                            project.summary.isEmpty
                                ? '尚未填写项目摘要'
                                : project.summary,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant)),
                        const SizedBox(height: 10),
                        Wrap(spacing: 6, runSpacing: 6, children: [
                          VopcMetaChip(project.stage),
                          VopcMetaChip(project.status),
                          VopcMetaChip(project.riskLevel)
                        ])
                      ])),
                  const SizedBox(width: 4),
                  IconButton(
                      tooltip: '编辑',
                      visualDensity: VisualDensity.compact,
                      icon: const Icon(Icons.edit_outlined, size: 20),
                      onPressed: onEdit),
                  IconButton(
                      tooltip: '删除',
                      visualDensity: VisualDensity.compact,
                      color: theme.colorScheme.error,
                      icon: const Icon(Icons.delete_outline, size: 20),
                      onPressed: onDelete),
                  const SizedBox(width: 2),
                  Icon(Icons.chevron_right, color: theme.colorScheme.outline),
                ]))));
  }
}
