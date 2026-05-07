import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:provider/provider.dart';
import '../../providers/enrollment_provider.dart';
import '../../widgets/error_view.dart';
import '../../widgets/flow_progress.dart';

/// 办事流程引导页（入学 / 离校）
class EnrollmentPage extends StatefulWidget {
  const EnrollmentPage({super.key});

  @override
  State<EnrollmentPage> createState() => _EnrollmentPageState();
}

class _EnrollmentPageState extends State<EnrollmentPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final prov = context.read<EnrollmentProvider>();
      if (prov.answerCard == null && !prov.loading) {
        prov.loadFlow();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final prov = context.watch<EnrollmentProvider>();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('办事流程')),
      body: Column(
        children: [
          // 流程类型切换
          _buildFlowSwitch(prov, theme),
          // 加载/错误/内容
          Expanded(child: _buildBody(prov, theme)),
        ],
      ),
    );
  }

  Widget _buildFlowSwitch(EnrollmentProvider prov, ThemeData theme) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      color: theme.colorScheme.surface,
      child: Row(
        children: [
          Expanded(
            child: _buildFlowChip(
              label: '入学流程',
              icon: Icons.school,
              active: prov.flowType == 'enrollment',
              onTap: () => prov.setFlowType('enrollment'),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _buildFlowChip(
              label: '离校流程',
              icon: Icons.celebration,
              active: prov.flowType == 'graduation',
              onTap: () => prov.setFlowType('graduation'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildFlowChip({
    required String label,
    required IconData icon,
    required bool active,
    required VoidCallback onTap,
  }) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          color: active ? theme.colorScheme.primaryContainer : theme.colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
          border: active
              ? Border.all(color: theme.colorScheme.primary, width: 1.5)
              : null,
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 18, color: active ? theme.colorScheme.onPrimaryContainer : theme.colorScheme.outline),
            const SizedBox(width: 8),
            Text(
              label,
              style: TextStyle(
                fontWeight: active ? FontWeight.w600 : FontWeight.normal,
                color: active ? theme.colorScheme.onPrimaryContainer : theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildBody(EnrollmentProvider prov, ThemeData theme) {
    if (prov.loading) {
      final flowName = prov.flowType == 'enrollment' ? '入学流程' : '离校流程';
      return FlowProgressIndicator(flowName: flowName);
    }

    if (prov.error != null) {
      return ErrorView.error(
        message: prov.error!,
        onRetry: () => prov.loadFlow(),
      );
    }

    final card = prov.answerCard;
    if (card == null) {
      return ErrorView.empty(
        message: '选择流程类型开始查看',
        icon: Icons.description_outlined,
      );
    }

    return RefreshIndicator(
      onRefresh: () => prov.loadFlow(),
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // 流程结论
          if (card.conclusion.isNotEmpty) ...[
            Card(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
                side: BorderSide(color: theme.colorScheme.outlineVariant),
              ),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(Icons.info_outline, size: 20, color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Text('流程概览', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                      ],
                    ),
                    const SizedBox(height: 8),
                    MarkdownBody(
                      data: card.conclusion,
                      selectable: true,
                      styleSheet: MarkdownStyleSheet(
                        p: theme.textTheme.bodyMedium?.copyWith(height: 1.6),
                        strong: const TextStyle(fontWeight: FontWeight.bold),
                        listBullet: theme.textTheme.bodyMedium,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
          ],

          // 进度条
          if (prov.totalSteps > 0) ...[
            _buildProgressSection(prov, theme),
            const SizedBox(height: 16),
          ],

          // 步骤列表
          if (prov.steps.isNotEmpty) ...[
            Text('办理步骤', style: theme.textTheme.titleSmall?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w600,
            )),
            const SizedBox(height: 8),
            ...List.generate(prov.steps.length, (i) => _buildStepCard(prov, theme, i)),
            const SizedBox(height: 16),
          ],

          // 操作按钮
          if (prov.totalSteps > 0) ...[
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: prov.completedCount == prov.totalSteps ? null : prov.completeAll,
                    icon: const Icon(Icons.done_all, size: 18),
                    label: const Text('全部完成'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: prov.completedSteps.isEmpty ? null : prov.resetProgress,
                    icon: const Icon(Icons.restart_alt, size: 18),
                    label: const Text('重置进度'),
                  ),
                ),
              ],
            ),
          ],

          // 来源引用
          if (card.sources.isNotEmpty) ...[
            const SizedBox(height: 16),
            Text('参考来源', style: theme.textTheme.titleSmall?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w600,
            )),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: card.sources.map((s) {
                return Chip(
                  avatar: const Icon(Icons.description_outlined, size: 16),
                  label: Text(s.title, style: const TextStyle(fontSize: 12)),
                  materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  visualDensity: VisualDensity.compact,
                );
              }).toList(),
            ),
          ],

          // 风险提示
          if (card.risks.isNotEmpty) ...[
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.errorContainer.withValues(alpha: 0.3),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(Icons.warning_amber, size: 18, color: theme.colorScheme.error),
                      const SizedBox(width: 6),
                      Text('注意事项', style: TextStyle(
                        fontWeight: FontWeight.w600,
                        color: theme.colorScheme.error,
                      )),
                    ],
                  ),
                  const SizedBox(height: 6),
                  ...card.risks.map((r) => Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Text('- $r', style: theme.textTheme.bodySmall),
                  )),
                ],
              ),
            ),
          ],

          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _buildProgressSection(EnrollmentProvider prov, ThemeData theme) {
    final progress = prov.progress;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Stack(
              alignment: Alignment.center,
              children: [
                SizedBox(
                  width: 48,
                  height: 48,
                  child: CircularProgressIndicator(
                    value: progress,
                    strokeWidth: 4,
                    backgroundColor: theme.colorScheme.surfaceContainerHighest,
                    color: progress == 1.0
                        ? Colors.green
                        : theme.colorScheme.primary,
                  ),
                ),
                Text(
                  '${(progress * 100).round()}%',
                  style: theme.textTheme.labelSmall?.copyWith(fontWeight: FontWeight.bold),
                ),
              ],
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '已完成 ${prov.completedCount} / ${prov.totalSteps} 步',
                    style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(height: 4),
                  LinearProgressIndicator(
                    value: progress,
                    backgroundColor: theme.colorScheme.surfaceContainerHighest,
                    color: progress == 1.0 ? Colors.green : theme.colorScheme.primary,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStepCard(EnrollmentProvider prov, ThemeData theme, int index) {
    final step = prov.steps[index];
    final isCompleted = prov.completedSteps.contains(index);
    final isLast = index == prov.steps.length - 1;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // 步骤指示器
        SizedBox(
          width: 36,
          child: Column(
            children: [
              GestureDetector(
                onTap: () => prov.toggleStep(index),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  width: 28,
                  height: 28,
                  decoration: BoxDecoration(
                    color: isCompleted ? Colors.green : theme.colorScheme.surfaceContainerHighest,
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: isCompleted ? Colors.green : theme.colorScheme.outlineVariant,
                      width: 2,
                    ),
                  ),
                  child: isCompleted
                      ? const Icon(Icons.check, size: 16, color: Colors.white)
                      : Center(
                          child: Text(
                            '${index + 1}',
                            style: TextStyle(
                              fontSize: 13,
                              fontWeight: FontWeight.w600,
                              color: theme.colorScheme.outline,
                            ),
                          ),
                        ),
                ),
              ),
              if (!isLast)
                Container(
                  width: 2,
                  height: 48,
                  color: isCompleted
                      ? Colors.green.withValues(alpha: 0.5)
                      : theme.colorScheme.outlineVariant,
                ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        // 步骤内容
        Expanded(
          child: GestureDetector(
            onTap: () => prov.toggleStep(index),
            child: Container(
              margin: EdgeInsets.only(bottom: isLast ? 0 : 16),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: isCompleted
                    ? Colors.green.withValues(alpha: 0.05)
                    : theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(
                  color: isCompleted
                      ? Colors.green.withValues(alpha: 0.3)
                      : Colors.transparent,
                ),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      step,
                      style: TextStyle(
                        fontSize: 14,
                        color: isCompleted ? theme.colorScheme.outline : null,
                        decoration: isCompleted ? TextDecoration.lineThrough : null,
                      ),
                    ),
                  ),
                  Icon(
                    isCompleted ? Icons.check_circle : Icons.circle_outlined,
                    size: 20,
                    color: isCompleted ? Colors.green : theme.colorScheme.outlineVariant,
                  ),
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }
}
