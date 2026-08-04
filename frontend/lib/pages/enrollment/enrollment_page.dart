import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/enrollment_provider.dart';
import '../../utils/capability_utils.dart';
import '../../widgets/error_view.dart';
import '../../widgets/flow_progress.dart';

/// 办事服务页：动态流程列表 + 办理步骤 + 提醒节点
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
      prov.loadCatalog();
      if (prov.answerCard == null) prov.loadFlow();
    });
  }

  @override
  Widget build(BuildContext context) {
    final prov = context.watch<EnrollmentProvider>();
    final theme = Theme.of(context);
    final canManage = CapabilityUtils.has(Capability.counselorKbWrite);
    final canReview = CapabilityUtils.has(Capability.counselorKbReview);

    return Scaffold(
      appBar: AppBar(
        title: const Text('办事服务'),
        actions: [
          if (canReview)
            IconButton(
              tooltip: '流程审核',
              onPressed: () => context.go('/process-review'),
              icon: const Icon(Icons.rate_review_outlined),
            ),
          if (canManage)
            IconButton(
              tooltip: '流程管理',
              onPressed: () => context.go('/process-manage'),
              icon: const Icon(Icons.edit_note),
            ),
        ],
      ),
      body: Column(
        children: [
          _buildCatalog(prov, theme),
          const Divider(height: 1),
          Expanded(child: _buildBody(prov, theme)),
        ],
      ),
    );
  }

  Widget _buildCatalog(EnrollmentProvider prov, ThemeData theme) {
    if (prov.definitionsLoading && prov.definitions.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(12),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (prov.definitions.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(12),
        child: Text('暂无可办理流程'),
      );
    }

    final freshmen =
        prov.definitions.where((d) => d.isFreshmenRelated).toList();
    final others = prov.definitions.where((d) => !d.isFreshmenRelated).toList();

    return Container(
      constraints: const BoxConstraints(maxHeight: 230),
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
      color: theme.colorScheme.surface,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.route_outlined,
                  size: 18, color: theme.colorScheme.primary),
              const SizedBox(width: 6),
              Text('办事流程',
                  style: theme.textTheme.titleSmall
                      ?.copyWith(fontWeight: FontWeight.w600)),
              const Spacer(),
              FilterChip(
                label: const Text('只看新生相关'),
                selected: prov.freshmenOnly,
                showCheckmark: false,
                visualDensity: VisualDensity.compact,
                onSelected: (_) => prov.toggleFreshmenOnly(),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Expanded(
            child: ListView(
              shrinkWrap: true,
              children: [
                if (freshmen.isNotEmpty)
                  _ProcessSection(
                    title: '新生相关',
                    expanded: true,
                    definitions: freshmen,
                    activeId: prov.flowType,
                    onTap: prov.setFlowType,
                  ),
                if (!prov.freshmenOnly && others.isNotEmpty)
                  _ProcessSection(
                    title: '其他流程',
                    expanded: false,
                    definitions: others,
                    activeId: prov.flowType,
                    onTap: prov.setFlowType,
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBody(EnrollmentProvider prov, ThemeData theme) {
    if (prov.loading && prov.answerCard == null) {
      final current =
          prov.definitions.where((d) => d.resourceId == prov.flowType).toList();
      final flowName = current.isNotEmpty ? current.first.title : '办事流程';
      return FlowProgressIndicator(flowName: flowName);
    }

    if (prov.error != null && prov.answerCard == null) {
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
                        Icon(Icons.info_outline,
                            size: 20, color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Text('流程概览',
                            style: theme.textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.w600)),
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
          if (prov.totalSteps > 0) ...[
            _buildProgressSection(prov, theme),
            const SizedBox(height: 16),
          ],
          if (prov.totalSteps > 0) ...[
            _buildFlowDiagram(prov, theme),
            const SizedBox(height: 16),
          ],
          if (prov.reminders.isNotEmpty) ...[
            _buildReminderSection(prov, theme),
            const SizedBox(height: 16),
          ],
          if (prov.steps.isNotEmpty) ...[
            Text('办理步骤',
                style: theme.textTheme.titleSmall?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w600,
                )),
            const SizedBox(height: 8),
            ...List.generate(
                prov.steps.length, (i) => _buildStepCard(prov, theme, i)),
            const SizedBox(height: 16),
          ],
          if (prov.totalSteps > 0) ...[
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: prov.completedCount == prov.totalSteps
                        ? null
                        : prov.completeAll,
                    icon: const Icon(Icons.done_all, size: 18),
                    label: const Text('全部完成'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed:
                        prov.completedSteps.isEmpty ? null : prov.resetProgress,
                    icon: const Icon(Icons.restart_alt, size: 18),
                    label: const Text('重置进度'),
                  ),
                ),
              ],
            ),
          ],
          if (card.sources.isNotEmpty) ...[
            const SizedBox(height: 16),
            Text('参考来源',
                style: theme.textTheme.titleSmall?.copyWith(
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
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _buildReminderSection(EnrollmentProvider prov, ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('提醒节点',
            style: theme.textTheme.titleSmall?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w600,
            )),
        const SizedBox(height: 8),
        ...prov.reminders.map((r) {
          final state = _reminderState(prov, r);
          final color = state == 'done'
              ? Colors.green
              : state == 'overdue'
                  ? theme.colorScheme.error
                  : theme.colorScheme.primary;
          final label = state == 'done'
              ? '已完成'
              : state == 'overdue'
                  ? '已到期'
                  : '待办';
          return Card(
            margin: const EdgeInsets.only(bottom: 8),
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(10),
              side: BorderSide(color: theme.colorScheme.outlineVariant),
            ),
            child: ListTile(
              leading: Icon(Icons.alarm, color: color),
              title: Text(
                '${r.remindAt} · ${r.title}',
                style: theme.textTheme.bodyMedium
                    ?.copyWith(fontWeight: FontWeight.w600),
              ),
              subtitle: r.content.isEmpty ? null : Text(r.content),
              trailing: Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: color.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(10),
                ),
                child:
                    Text(label, style: TextStyle(fontSize: 11, color: color)),
              ),
            ),
          );
        }),
      ],
    );
  }

  String _reminderState(EnrollmentProvider prov, ProcessReminder r) {
    if (prov.recordStatus == 'completed') return 'done';
    if (r.stepOrder > 0 && prov.completedSteps.contains(r.stepOrder - 1)) {
      return 'done';
    }
    final datePart = r.remindAt.trim().split(' ').first;
    if (datePart.length == 10 && datePart[4] == '-' && datePart[7] == '-') {
      final today = DateTime.now().toIso8601String().substring(0, 10);
      if (datePart.compareTo(today) <= 0) return 'overdue';
    }
    return 'upcoming';
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
                  style: theme.textTheme.labelSmall
                      ?.copyWith(fontWeight: FontWeight.bold),
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
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(height: 4),
                  LinearProgressIndicator(
                    value: progress,
                    backgroundColor: theme.colorScheme.surfaceContainerHighest,
                    color: progress == 1.0
                        ? Colors.green
                        : theme.colorScheme.primary,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

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
                  Icon(Icons.account_tree_outlined,
                      size: 18, color: theme.colorScheme.primary),
                  const SizedBox(width: 6),
                  Text('流程全景',
                      style: theme.textTheme.titleSmall?.copyWith(
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
                                    : theme.colorScheme.primary
                                        .withOpacity(0.4),
                                width: 2,
                              ),
                            ),
                            child: Center(
                              child: isCompleted
                                  ? const Icon(Icons.check,
                                      size: 20, color: Colors.white)
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
                                fontWeight: isCompleted
                                    ? FontWeight.normal
                                    : FontWeight.w500,
                              ),
                            ),
                          ),
                        ],
                      ),
                      if (!isLast)
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 4),
                          child: Icon(
                            Icons.arrow_forward_ios,
                            size: 12,
                            color: isCompleted
                                ? Colors.green.withOpacity(0.5)
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
    final hasRichSteps =
        prov.stepDetails.isNotEmpty && index < prov.stepDetails.length;
    final detail = hasRichSteps ? prov.stepDetails[index] : null;
    final stepText =
        index < prov.steps.length ? prov.steps[index] : (detail?.title ?? '');

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
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
                    color: isCompleted
                        ? Colors.green
                        : theme.colorScheme.surfaceContainerHighest,
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: isCompleted
                          ? Colors.green
                          : theme.colorScheme.outlineVariant,
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
                      ? Colors.green.withOpacity(0.5)
                      : theme.colorScheme.outlineVariant,
                ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: GestureDetector(
            onTap: () => prov.toggleStep(index),
            child: Container(
              margin: EdgeInsets.only(bottom: isLast ? 0 : 12),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: isCompleted
                    ? Colors.green.withOpacity(0.05)
                    : theme.colorScheme.surfaceContainerHighest
                        .withOpacity(0.5),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(
                  color: isCompleted
                      ? Colors.green.withOpacity(0.3)
                      : Colors.transparent,
                ),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          stepText,
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                            color:
                                isCompleted ? theme.colorScheme.outline : null,
                            decoration:
                                isCompleted ? TextDecoration.lineThrough : null,
                          ),
                        ),
                      ),
                      Icon(
                        isCompleted
                            ? Icons.check_circle
                            : Icons.circle_outlined,
                        size: 20,
                        color: isCompleted
                            ? Colors.green
                            : theme.colorScheme.outlineVariant,
                      ),
                    ],
                  ),
                  if (detail != null) ...[
                    const SizedBox(height: 8),
                    const Divider(height: 1),
                    const SizedBox(height: 8),
                    if (detail.contact.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.person_outline,
                        detail.contact,
                        detail.phone,
                      ),
                      const SizedBox(height: 4),
                    ],
                    if (detail.location.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.location_on_outlined,
                        detail.location,
                        detail.officeHours,
                      ),
                      const SizedBox(height: 4),
                    ],
                    if (detail.materials.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.description_outlined,
                        _displayMaterials(detail.materials),
                        '',
                      ),
                      const SizedBox(height: 4),
                    ],
                    if (detail.contactWechat.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.chat_outlined,
                        '微信/企业微信：${detail.contactWechat}',
                        '',
                      ),
                      const SizedBox(height: 4),
                    ],
                    if (detail.entryUrl.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.open_in_new,
                        detail.entryUrl,
                        '',
                      ),
                      const SizedBox(height: 4),
                    ],
                    if (detail.deadline.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.schedule,
                        '截止时间：${detail.deadline}',
                        '',
                      ),
                      const SizedBox(height: 4),
                    ],
                    if (detail.notes.isNotEmpty) ...[
                      _buildDetailRow(
                        theme,
                        Icons.notes,
                        detail.notes,
                        '',
                      ),
                      const SizedBox(height: 4),
                    ],
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
                              childrenPadding:
                                  const EdgeInsets.only(left: 12, bottom: 8),
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

  Widget _buildDetailRow(
      ThemeData theme, IconData icon, String text1, String text2) {
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

  String _displayMaterials(String raw) {
    final trimmed = raw.trim();
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      try {
        final decoded = jsonDecode(trimmed);
        if (decoded is List) {
          return decoded.map((e) => e.toString()).join('、');
        }
      } catch (_) {}
    }
    return raw;
  }
}

class _ProcessSection extends StatelessWidget {
  final String title;
  final bool expanded;
  final List<ProcessDefinition> definitions;
  final String activeId;
  final ValueChanged<String> onTap;

  const _ProcessSection({
    required this.title,
    required this.expanded,
    required this.definitions,
    required this.activeId,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ExpansionTile(
      initiallyExpanded: expanded,
      tilePadding: EdgeInsets.zero,
      childrenPadding: const EdgeInsets.only(left: 8, bottom: 8),
      title: Text(
        '$title（${definitions.length}）',
        style: theme.textTheme.labelLarge
            ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
      ),
      children: definitions.map((def) {
        final selected = def.resourceId == activeId;
        return Card(
          margin: const EdgeInsets.only(bottom: 4),
          elevation: 0,
          color: selected
              ? theme.colorScheme.primaryContainer.withOpacity(0.25)
              : theme.colorScheme.surfaceContainerLow,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
            side: BorderSide(
              color: selected
                  ? theme.colorScheme.primary.withOpacity(0.5)
                  : theme.colorScheme.outlineVariant.withOpacity(0.3),
            ),
          ),
          child: ListTile(
            dense: true,
            leading: Icon(
              Icons.account_tree_outlined,
              color: selected
                  ? theme.colorScheme.primary
                  : theme.colorScheme.outline,
            ),
            title: Text(def.title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
                )),
            subtitle: Text(
              '${def.steps.length} 个步骤 · ${def.reminders.length} 条提醒',
              style: theme.textTheme.labelSmall,
            ),
            trailing: const Icon(Icons.chevron_right, size: 18),
            onTap: () => onTap(def.resourceId),
          ),
        );
      }).toList(),
    );
  }
}
