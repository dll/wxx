import 'package:flutter/material.dart';

class VopcStageProgress extends StatelessWidget {
  final String currentStage;
  final String status;
  const VopcStageProgress(
      {super.key, required this.currentStage, required this.status});
  static const stages = ['G0', 'G1', 'G2', 'G3', 'G4'];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cur = currentStage.toUpperCase();
    final idx = stages.indexOf(cur);
    final progress = idx < 0 ? 0.0 : (idx + 1) / stages.length;
    final blocked =
        {'paused', 'risk_frozen', 'terminated', 'archived'}.contains(status);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text('OPC 虚拟主线阶段', style: theme.textTheme.titleSmall),
        Text('$cur / 共 ${stages.length} 个阶段',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
      ]),
      const SizedBox(height: 8),
      ClipRRect(
          borderRadius: BorderRadius.circular(6),
          child: LinearProgressIndicator(
              value: progress,
              minHeight: 10,
              backgroundColor: theme.colorScheme.surfaceContainerHighest,
              color: blocked
                  ? theme.colorScheme.error
                  : theme.colorScheme.primary)),
      const SizedBox(height: 8),
      Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        for (final stage in stages)
          Expanded(
              child: Container(
                  alignment: Alignment.center,
                  child: Text(stage,
                      style: theme.textTheme.labelSmall?.copyWith(
                          fontWeight:
                              cur == stage ? FontWeight.w800 : FontWeight.w400,
                          color: cur == stage
                              ? theme.colorScheme.primary
                              : theme.colorScheme.outline)))),
      ]),
      if (blocked) ...[
        const SizedBox(height: 6),
        Text('项目已暂停/冻结/终止/归档',
            style: TextStyle(color: theme.colorScheme.error, fontSize: 12)),
      ],
    ]);
  }
}
