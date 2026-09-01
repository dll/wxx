import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import '../../models/models.dart';

/// AnswerCard 卡片组件
/// 渲染 AI 回答的结构化内容：结论 + 步骤 + 来源 + 风险 + 追问 + 导出/反馈
class AnswerCardWidget extends StatefulWidget {
  final AnswerCard card;
  final void Function(String question)? onFollowUp;
  final void Function()? onExport;
  final void Function()? onFeedback;
  final void Function(String resourceId)? onViewSourceDetail;

  const AnswerCardWidget({
    super.key,
    required this.card,
    this.onFollowUp,
    this.onExport,
    this.onFeedback,
    this.onViewSourceDetail,
  });

  @override
  State<AnswerCardWidget> createState() => _AnswerCardWidgetState();
}

class _AnswerCardWidgetState extends State<AnswerCardWidget>
    with SingleTickerProviderStateMixin {
  bool _sourcesExpanded = false;
  final Set<int> _expandedSourceIndices = {};
  int? _highlightedSourceIndex;

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
              if (widget.card.conclusion.isNotEmpty)
                _buildConclusionWithCitations(theme),

              // 多模态内容：后端流程步骤可携带图片/视频地址，回答中直接以媒体卡片呈现。
              if (widget.card.stepDetails
                  .any((s) => s.mediaUrls.isNotEmpty)) ...[
                const SizedBox(height: 12),
                _buildMediaSection(theme),
              ],

              // 参与智能体（D4-3 透明分层展示）：仅当后端下发了 agents 列表时显示，
              // 缺失/为空则整行隐藏，不影响既有布局与交互。
              if (widget.card.agents.isNotEmpty) ...[
                const SizedBox(height: 10),
                _buildAgentsSection(theme),
              ],

              // 步骤清单
              if (widget.card.steps.isNotEmpty) ...[
                const SizedBox(height: 12),
                _buildSection(
                  context,
                  icon: Icons.checklist,
                  title: '操作步骤',
                  child: Column(
                    children: widget.card.steps.asMap().entries.map((entry) {
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
                              child: Text(entry.value,
                                  style: theme.textTheme.bodyMedium),
                            ),
                          ],
                        ),
                      );
                    }).toList(),
                  ),
                ),
              ],

              // 来源引用（优化版）
              if (widget.card.sources.isNotEmpty) ...[
                const SizedBox(height: 12),
                _buildSourcesSection(theme),
              ],

              // 风险提示
              if (widget.card.risks.isNotEmpty) ...[
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.errorContainer.withOpacity(0.3),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Icon(Icons.warning_amber,
                              size: 18, color: theme.colorScheme.error),
                          const SizedBox(width: 6),
                          Text('注意事项',
                              style: TextStyle(
                                fontWeight: FontWeight.w600,
                                color: theme.colorScheme.error,
                              )),
                        ],
                      ),
                      const SizedBox(height: 6),
                      ...widget.card.risks.map((r) => Padding(
                            padding: const EdgeInsets.only(bottom: 4),
                            child:
                                Text('- $r', style: theme.textTheme.bodySmall),
                          )),
                    ],
                  ),
                ),
              ],

              // 追问建议
              if (widget.card.followUps.isNotEmpty) ...[
                const SizedBox(height: 12),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: widget.card.followUps.map((q) {
                    return ActionChip(
                      avatar: const Icon(Icons.chat_bubble_outline, size: 16),
                      label: Text(q, style: const TextStyle(fontSize: 12)),
                      onPressed: () => widget.onFollowUp?.call(q),
                    );
                  }).toList(),
                ),
              ],

              // 兜底标记
              if (widget.card.fallback) ...[
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
              if (widget.onExport != null || widget.onFeedback != null) ...[
                const Divider(height: 24),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    if (widget.onFeedback != null)
                      TextButton.icon(
                        onPressed: widget.onFeedback,
                        icon: const Icon(Icons.feedback_outlined, size: 16),
                        label: const Text('反馈纠错'),
                      ),
                    if (widget.onExport != null)
                      TextButton.icon(
                        onPressed: widget.onExport,
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

  /// 构建结论，支持引用标注 + Markdown 渲染
  /// 引用标注 [1] 转为 Markdown 链接 citation://N，由 MarkdownBody 渲染并可点击
  Widget _buildConclusionWithCitations(ThemeData theme) {
    final text = widget.card.conclusion;
    final pattern = RegExp(r'\[(?:资料)?(\d+)\]');

    // 将引用标注转为 Markdown 链接（保持 Markdown 解析，不裸露语法符号）
    String toMarkdown(String t) {
      return t.replaceAllMapped(pattern, (m) {
        final numStr = m.group(1)!;
        final idx = int.tryParse(numStr);
        if (idx != null && idx >= 1 && idx <= widget.card.sources.length) {
          return '[$numStr](citation://$idx)';
        }
        return m.group(0)!;
      });
    }

    return MarkdownBody(
      data: toMarkdown(text),
      selectable: true,
      onTapLink: (text, href, title) {
        if (href != null && href.startsWith('citation://')) {
          final idx = int.tryParse(href.substring('citation://'.length));
          if (idx != null && idx >= 1 && idx <= widget.card.sources.length) {
            _handleCitationTap(idx - 1);
          }
        }
      },
      styleSheet: MarkdownStyleSheet(
        p: theme.textTheme.bodyLarge?.copyWith(height: 1.6),
        h1: theme.textTheme.titleLarge,
        h2: theme.textTheme.titleMedium,
        h3: theme.textTheme.titleSmall,
        strong: const TextStyle(fontWeight: FontWeight.bold),
        listBullet: theme.textTheme.bodyLarge,
        blockquoteDecoration: BoxDecoration(
          color: theme.colorScheme.primaryContainer.withOpacity(0.4),
          border: Border(
            left: BorderSide(
              color: theme.colorScheme.primary,
              width: 3,
            ),
          ),
        ),
        code: TextStyle(
          fontFamily: 'monospace',
          fontSize: 13,
          backgroundColor: theme.colorScheme.surfaceContainerHighest,
          color: theme.colorScheme.onSurface,
        ),
        tableBorder: TableBorder.all(
          color: theme.colorScheme.outlineVariant,
        ),
        a: TextStyle(
          color: theme.colorScheme.primary,
          decoration: TextDecoration.none,
        ),
      ),
    );
  }

  Widget _buildMediaSection(ThemeData theme) {
    final urls = widget.card.stepDetails
        .expand((s) => s.mediaUrls)
        .where((url) => url.trim().isNotEmpty)
        .toSet()
        .toList();
    if (urls.isEmpty) return const SizedBox.shrink();
    return _buildSection(
      context,
      icon: Icons.perm_media_outlined,
      title: '相关图文资料（${urls.length}）',
      child: SizedBox(
        height: 128,
        child: ListView.separated(
          scrollDirection: Axis.horizontal,
          itemCount: urls.length,
          separatorBuilder: (_, __) => const SizedBox(width: 10),
          itemBuilder: (_, index) {
            final url = urls[index];
            final isVideo =
                RegExp(r'\.(mp4|webm|mov)(\?|$)', caseSensitive: false)
                    .hasMatch(url);
            return Container(
              width: 180,
              clipBehavior: Clip.antiAlias,
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Stack(fit: StackFit.expand, children: [
                if (!isVideo)
                  Image.network(url,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => Center(
                          child: Icon(Icons.broken_image_outlined,
                              color: theme.colorScheme.outline)))
                else
                  Center(
                      child: Icon(Icons.play_circle_fill,
                          size: 46, color: theme.colorScheme.primary)),
                Positioned(
                    left: 8,
                    bottom: 7,
                    child: Chip(
                      visualDensity: VisualDensity.compact,
                      avatar: Icon(
                          isVideo
                              ? Icons.videocam_outlined
                              : Icons.image_outlined,
                          size: 14),
                      label: Text(isVideo ? '视频资料' : '图片资料',
                          style: const TextStyle(fontSize: 11)),
                    )),
              ]),
            );
          },
        ),
      ),
    );
  }

  /// 处理引用点击
  void _handleCitationTap(int sourceIndex) {
    setState(() {
      _sourcesExpanded = true;
      _expandedSourceIndices.add(sourceIndex);
      _highlightedSourceIndex = sourceIndex;
    });

    Future.delayed(const Duration(milliseconds: 300), () {
      if (mounted) {
        setState(() {
          _highlightedSourceIndex = null;
        });
      }
    });
  }

  /// 构建来源区域
  Widget _buildSourcesSection(ThemeData theme) {
    final sources = widget.card.sources;
    final displaySources =
        _sourcesExpanded ? sources : sources.take(3).toList();

    return _buildSection(
      context,
      icon: Icons.menu_book,
      title: '参考来源（${sources.length}）',
      child: Column(
        children: [
          ...displaySources.asMap().entries.map((entry) {
            final index = entry.key;
            final source = entry.value;
            final isExpanded = _expandedSourceIndices.contains(index);
            final isHighlighted = _highlightedSourceIndex == index;
            return Padding(
              padding: EdgeInsets.only(
                bottom: index == displaySources.length - 1 ? 0 : 8,
              ),
              child: _SourceCard(
                source: source,
                index: index,
                isExpanded: isExpanded,
                isHighlighted: isHighlighted,
                onTap: () {
                  setState(() {
                    if (isExpanded) {
                      _expandedSourceIndices.remove(index);
                    } else {
                      _expandedSourceIndices.add(index);
                    }
                  });
                },
                onViewDetail: widget.onViewSourceDetail,
              ),
            );
          }),
          if (sources.length > 3) ...[
            const SizedBox(height: 8),
            TextButton.icon(
              onPressed: () {
                setState(() {
                  _sourcesExpanded = !_sourcesExpanded;
                });
              },
              icon: Icon(
                _sourcesExpanded ? Icons.expand_less : Icons.expand_more,
                size: 18,
              ),
              label: Text(
                _sourcesExpanded ? '收起来源' : '展开全部 ${sources.length} 个来源',
                style: const TextStyle(fontSize: 13),
              ),
            ),
          ],
        ],
      ),
    );
  }

  /// 参与智能体标签行（D4-3 透明分层展示）
  Widget _buildAgentsSection(ThemeData theme) {
    // 去重 + 去除空项，避免重复/空标签
    final agents = <String>[];
    for (final a in widget.card.agents) {
      final name = a.trim();
      if (name.isEmpty || agents.contains(name)) continue;
      agents.add(name);
    }
    if (agents.isEmpty) {
      return const SizedBox.shrink();
    }
    return _buildSection(
      context,
      icon: Icons.groups_outlined,
      title: '参与智能体（${agents.length}）',
      child: Wrap(
        spacing: 8,
        runSpacing: 8,
        children: agents.map((name) {
          return Chip(
            avatar: const Icon(Icons.smart_toy_outlined, size: 16),
            label: Text(name, style: const TextStyle(fontSize: 12)),
            visualDensity: VisualDensity.compact,
          );
        }).toList(),
      ),
    );
  }

  Widget _buildSection(
    BuildContext context, {
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
            Text(title,
                style: TextStyle(
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

/// 单个来源卡片组件
class _SourceCard extends StatelessWidget {
  final Source source;
  final int index;
  final bool isExpanded;
  final bool isHighlighted;
  final VoidCallback onTap;
  final void Function(String resourceId)? onViewDetail;

  const _SourceCard({
    required this.source,
    required this.index,
    required this.isExpanded,
    required this.isHighlighted,
    required this.onTap,
    this.onViewDetail,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final typeColor = source.typeColor;
    final hasSnippet = source.snippet.isNotEmpty || source.summary.isNotEmpty;
    final hasMeta =
        source.version.isNotEmpty || source.effectiveDate.isNotEmpty;
    final hasRelevance = source.relevanceScore > 0;
    final displayText =
        source.snippet.isNotEmpty ? source.snippet : source.summary;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeInOut,
      decoration: BoxDecoration(
        color: isHighlighted
            ? typeColor.withOpacity(0.12)
            : theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: isHighlighted
              ? typeColor.withOpacity(0.5)
              : theme.colorScheme.outlineVariant.withOpacity(0.5),
          width: isHighlighted ? 1.5 : 1,
        ),
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(10),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // 类型图标 + 编号
                    Container(
                      width: 32,
                      height: 32,
                      decoration: BoxDecoration(
                        color: typeColor.withOpacity(0.15),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Center(
                        child: Text(
                          '${index + 1}',
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.bold,
                            color: typeColor,
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(width: 10),
                    // 标题 + 类型标签
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Container(
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 6, vertical: 2),
                                decoration: BoxDecoration(
                                  color: typeColor.withOpacity(0.12),
                                  borderRadius: BorderRadius.circular(4),
                                ),
                                child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Icon(source.typeIcon,
                                        size: 12, color: typeColor),
                                    const SizedBox(width: 3),
                                    Text(
                                      source.typeLabel,
                                      style: TextStyle(
                                        fontSize: 10,
                                        fontWeight: FontWeight.w600,
                                        color: typeColor,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                              const SizedBox(width: 8),
                              if (hasRelevance) _buildRelevanceLabel(theme),
                            ],
                          ),
                          const SizedBox(height: 4),
                          Text(
                            source.title,
                            style: const TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w600,
                              height: 1.4,
                            ),
                            maxLines: isExpanded ? null : 2,
                            overflow: isExpanded
                                ? TextOverflow.visible
                                : TextOverflow.ellipsis,
                          ),
                        ],
                      ),
                    ),
                    // 展开箭头
                    Icon(
                      isExpanded ? Icons.expand_less : Icons.expand_more,
                      size: 20,
                      color: theme.colorScheme.outline,
                    ),
                  ],
                ),
                // 摘要
                if (hasSnippet) ...[
                  const SizedBox(height: 8),
                  AnimatedCrossFade(
                    duration: const Duration(milliseconds: 200),
                    firstChild: Text(
                      displayText,
                      style: TextStyle(
                        fontSize: 13,
                        color: theme.colorScheme.onSurfaceVariant,
                        height: 1.5,
                      ),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    secondChild: Text(
                      displayText,
                      style: TextStyle(
                        fontSize: 13,
                        color: theme.colorScheme.onSurfaceVariant,
                        height: 1.5,
                      ),
                    ),
                    crossFadeState: isExpanded
                        ? CrossFadeState.showSecond
                        : CrossFadeState.showFirst,
                  ),
                ],
                // 元信息（版本号、生效日期）
                if (hasMeta) ...[
                  const SizedBox(height: 6),
                  Row(
                    children: [
                      if (source.version.isNotEmpty) ...[
                        Icon(Icons.label_outline,
                            size: 12, color: theme.colorScheme.outline),
                        const SizedBox(width: 3),
                        Text(
                          source.version,
                          style: TextStyle(
                            fontSize: 11,
                            color: theme.colorScheme.outline,
                          ),
                        ),
                      ],
                      if (source.version.isNotEmpty &&
                          source.effectiveDate.isNotEmpty)
                        const SizedBox(width: 12),
                      if (source.effectiveDate.isNotEmpty) ...[
                        Icon(Icons.event_outlined,
                            size: 12, color: theme.colorScheme.outline),
                        const SizedBox(width: 3),
                        Text(
                          source.effectiveDate,
                          style: TextStyle(
                            fontSize: 11,
                            color: theme.colorScheme.outline,
                          ),
                        ),
                      ],
                    ],
                  ),
                ],
                // 相关度进度条（展开时显示）
                if (hasRelevance && isExpanded) ...[
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Text(
                        '相关度',
                        style: TextStyle(
                          fontSize: 11,
                          color: theme.colorScheme.outline,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(2),
                          child: LinearProgressIndicator(
                            value: source.relevanceScore.clamp(0.0, 1.0),
                            minHeight: 4,
                            backgroundColor: theme.colorScheme.outlineVariant,
                            valueColor:
                                AlwaysStoppedAnimation<Color>(typeColor),
                          ),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        '${(source.relevanceScore * 100).toInt()}%',
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          color: typeColor,
                        ),
                      ),
                    ],
                  ),
                ],
                // 查看详情按钮
                if (isExpanded && source.resourceId.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      TextButton.icon(
                        onPressed: () => onViewDetail?.call(source.resourceId),
                        icon: const Icon(Icons.open_in_new, size: 14),
                        label: const Text('在知识大厅查看',
                            style: TextStyle(fontSize: 12)),
                        style: TextButton.styleFrom(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 4),
                          minimumSize: Size.zero,
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// 相关度使用文本语义，避免与用户评分混淆。
  Widget _buildRelevanceLabel(ThemeData theme) {
    final label = source.relevanceScore >= 0.75
        ? '高度相关'
        : source.relevanceScore >= 0.45
            ? '相关依据'
            : '补充依据';
    return Text(
      label,
      style: theme.textTheme.labelSmall?.copyWith(
        color: theme.colorScheme.onSurfaceVariant,
      ),
    );
  }
}
