import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import '../../models/models.dart';

/// AnswerCard 卡片组件
/// 渲染 AI 回答的结构化内容：结论 + 步骤 + 来源 + 风险 + 追问 + 导出/反馈
class AnswerCardWidget extends StatelessWidget {
  final AnswerCard card;
  final void Function(String question)? onFollowUp;
  final void Function()? onExport;
  final void Function()? onFeedback;

  const AnswerCardWidget({
    super.key,
    required this.card,
    this.onFollowUp,
    this.onExport,
    this.onFeedback,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Semantics(
      label: 'AI 回答卡片',
      child: Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
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
            // 结论（Markdown 渲染）
            if (card.conclusion.isNotEmpty)
              MarkdownBody(
                data: card.conclusion,
                selectable: true,
                styleSheet: MarkdownStyleSheet(
                  p: theme.textTheme.bodyLarge?.copyWith(height: 1.6),
                  h1: theme.textTheme.titleLarge,
                  h2: theme.textTheme.titleMedium,
                  h3: theme.textTheme.titleSmall,
                  strong: TextStyle(fontWeight: FontWeight.bold),
                  listBullet: theme.textTheme.bodyLarge,
                ),
              ),

            // 步骤清单
            if (card.steps.isNotEmpty) ...[
              const SizedBox(height: 12),
              _buildSection(
                context,
                icon: Icons.checklist,
                title: '操作步骤',
                child: Column(
                  children: card.steps.asMap().entries.map((entry) {
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Container(
                            width: 22,
                            height: 22,
                            margin: const EdgeInsets.only(right: 8, top: 1),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.primaryContainer,
                              shape: BoxShape.circle,
                            ),
                            child: Center(
                              child: Text(
                                '${entry.key + 1}',
                                style: TextStyle(
                                  fontSize: 12,
                                  fontWeight: FontWeight.bold,
                                  color: theme.colorScheme.onPrimaryContainer,
                                ),
                              ),
                            ),
                          ),
                          Expanded(
                            child: Text(entry.value, style: theme.textTheme.bodyMedium),
                          ),
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
            ],

            // 来源引用
            if (card.sources.isNotEmpty) ...[
              const SizedBox(height: 12),
              _buildSection(
                context,
                icon: Icons.menu_book,
                title: '参考来源',
                child: Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: card.sources.map((s) {
                    return Chip(
                      avatar: const Icon(Icons.description_outlined, size: 16),
                      label: Text(s.title, style: const TextStyle(fontSize: 12)),
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      visualDensity: VisualDensity.compact,
                    );
                  }).toList(),
                ),
              ),
            ],

            // 风险提示
            if (card.risks.isNotEmpty) ...[
              const SizedBox(height: 12),
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

            // 追问建议
            if (card.followUps.isNotEmpty) ...[
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: card.followUps.map((q) {
                  return ActionChip(
                    avatar: const Icon(Icons.chat_bubble_outline, size: 16),
                    label: Text(q, style: const TextStyle(fontSize: 12)),
                    onPressed: () => onFollowUp?.call(q),
                  );
                }).toList(),
              ),
            ],

            // 兜底标记
            if (card.fallback) ...[
              const SizedBox(height: 8),
              Text(
                '以上回答仅供参考，建议咨询辅导员获取更准确信息',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],

            // 操作按钮（导出 + 反馈）
            if (onExport != null || onFeedback != null) ...[
              const Divider(height: 24),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  if (onFeedback != null)
                    TextButton.icon(
                      onPressed: onFeedback,
                      icon: const Icon(Icons.feedback_outlined, size: 16),
                      label: const Text('反馈纠错'),
                    ),
                  if (onExport != null)
                    TextButton.icon(
                      onPressed: onExport,
                      icon: const Icon(Icons.download_outlined, size: 16),
                      label: const Text('导出'),
                    ),
                ],
              ),
            ],
          ],
        ),
      ),
    ),
    );
  }

  Widget _buildSection(BuildContext context, {
    required IconData icon,
    required String title,
    required Widget child,
  }) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(icon, size: 18, color: theme.colorScheme.primary),
            const SizedBox(width: 6),
            Text(title, style: TextStyle(
              fontWeight: FontWeight.w600,
              color: theme.colorScheme.primary,
            )),
          ],
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }
}
