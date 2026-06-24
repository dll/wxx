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
      child: Column(
        children: [
          Row(
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
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(
                child: _buildFlowChip(
                  label: '转专业',
                  icon: Icons.swap_horiz,
                  active: prov.flowType == 'major_change',
                  onTap: () => prov.setFlowType('major_change'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: _buildFlowChip(
                  label: '助学贷款',
                  icon: Icons.account_balance,
                  active: prov.flowType == 'student_loan',
                  onTap: () => prov.setFlowType('student_loan'),
                ),
              ),
            ],
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

          // 流程节点图示
          if (prov.totalSteps > 0) ...[
            _buildFlowDiagram(prov, theme),
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

  /// 流程节点图示 — 所有步骤作为横向连接节点展示
  Widget _buildFlowDiagram(EnrollmentProvider prov, ThemeData theme) {
    final labels = prov.steps;
    if (labels.isEmpty) return const SizedBox.shrink();
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.only(left: 8, bottom: 12),
              child: Row(
                children: [
                  Icon(Icons.account_tree_outlined, size: 18, color: theme.colorScheme.primary),
                  const SizedBox(width: 6),
                  Text('流程全景', style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.primary,
                  )),
                ],
              ),
            ),
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: List.generate(labels.length, (i) {
                  final isCompleted = prov.completedSteps.contains(i);
                  final isLast = i == labels.length - 1;
                  return Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      // 节点
                      Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          AnimatedContainer(
                            duration: const Duration(milliseconds: 200),
                            width: 40,
                            height: 40,
                            decoration: BoxDecoration(
                              color: isCompleted
                                  ? Colors.green
                                  : theme.colorScheme.primaryContainer,
                              shape: BoxShape.circle,
                              border: Border.all(
                                color: isCompleted
                                    ? Colors.green
                                    : theme.colorScheme.primary.withValues(alpha: 0.4),
                                width: 2,
                              ),
                            ),
                            child: Center(
                              child: isCompleted
                                  ? const Icon(Icons.check, size: 20, color: Colors.white)
                                  : Text(
                                      '${i + 1}',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                        fontSize: 15,
                                        color: theme.colorScheme.primary,
                                      ),
                                    ),
                            ),
                          ),
                          const SizedBox(height: 4),
                          SizedBox(
                            width: 72,
                            child: Text(
                              labels[i],
                              textAlign: TextAlign.center,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 10,
                                color: isCompleted
                                    ? theme.colorScheme.outline
                                    : theme.colorScheme.onSurfaceVariant,
                                fontWeight: isCompleted ? FontWeight.normal : FontWeight.w500,
                              ),
                            ),
                          ),
                        ],
                      ),
                      // 连接箭头
                      if (!isLast)
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 4),
                          child: Icon(
                            Icons.arrow_forward_ios,
                            size: 12,
                            color: isCompleted
                                ? Colors.green.withValues(alpha: 0.5)
                                : theme.colorScheme.outlineVariant,
                          ),
                        ),
                    ],
                  );
                }),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStepCard(EnrollmentProvider prov, ThemeData theme, int index) {
    final isCompleted = prov.completedSteps.contains(index);
    final isLast = index == prov.totalSteps - 1;
    final hasRichSteps = prov.stepDetails.isNotEmpty && index < prov.stepDetails.length;

    // 富文本步骤详情
    final detail = hasRichSteps ? prov.stepDetails[index] : null;
    final stepText = index < prov.steps.length ? prov.steps[index] : (detail?.title ?? '');

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
                  height: detail != null ? 100 : 48,
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
              margin: EdgeInsets.only(bottom: isLast ? 0 : 12),
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
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // 步骤标题
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          stepText,
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
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

                  // 富文本详情（联系人/地点/电话/FAQ等）
                  if (detail != null) ...[
                    const SizedBox(height: 8),
                    const Divider(height: 1),
                    const SizedBox(height: 8),

                    // 联系人 + 电话
                    if (detail.contact.isNotEmpty) ...[
                      _buildDetailRow(
                        theme, Icons.person_outline, detail.contact, detail.phone,
                      ),
                      const SizedBox(height: 4),
                    ],

                    // 办理地点
                    if (detail.location.isNotEmpty) ...[
                      _buildDetailRow(
                        theme, Icons.location_on_outlined, detail.location, detail.officeHours,
                      ),
                      const SizedBox(height: 4),
                    ],

                    // 所需材料
                    if (detail.materials.isNotEmpty) ...[
                      _buildDetailRow(
                        theme, Icons.description_outlined, detail.materials, '',
                      ),
                      const SizedBox(height: 4),
                    ],

                    // 办理入口
                    if (detail.entryUrl.isNotEmpty) ...[
                      _buildDetailRow(
                        theme, Icons.open_in_new, detail.entryUrl, '',
                      ),
                      const SizedBox(height: 4),
                    ],

                    // 截止时间
                    if (detail.deadline.isNotEmpty) ...[
                      _buildDetailRow(
                        theme, Icons.schedule, '截止时间：${detail.deadline}', '',
                      ),
                      const SizedBox(height: 4),
                    ],

                    // FAQ
                    if (detail.faq.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      ...detail.faq.map((f) => Padding(
                        padding: const EdgeInsets.only(bottom: 4),
                        child: ExpansionTile(
                          tilePadding: EdgeInsets.zero,
                          dense: true,
                          title: Text(
                            f.q,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.primary,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          childrenPadding: const EdgeInsets.only(left: 12, bottom: 8),
                          children: [
                            Text(
                              f.a,
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      )),
                    ],
                  ],
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }

  /// 构建详情行（图标 + 文本1 + 文本2）
  Widget _buildDetailRow(ThemeData theme, IconData icon, String text1, String text2) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 14, color: theme.colorScheme.onSurfaceVariant),
        const SizedBox(width: 6),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                text1,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              if (text2.isNotEmpty)
                Text(
                  text2,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                    fontSize: 11,
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }
}
