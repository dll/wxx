import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

/// 统一 Markdown 文本组件 — 正确解析渲染系统/AI 生成的 Markdown 内容。
///
/// 用法：把 `Text(x)` / `SelectableText(x)` 替换为 `MdText(x)`，
/// 即可自动解析标题/列表/加粗/表格/引用等语法，不再裸露 md 符号。
/// 支持主题亮暗、可选中复制，风格与 MarkdownBody 默认保持一致。
class MdText extends StatelessWidget {
  /// 要渲染的 Markdown 文本
  final String data;

  /// 正文样式（默认 bodyMedium，行高 1.6 便于阅读）
  final TextStyle? style;

  /// 是否可选复制（默认开启）
  final bool selectable;

  /// 内联模式：不额外加外边距/块间距（用于嵌在已有卡片/列表内）
  final bool inline;

  const MdText(
    this.data, {
    super.key,
    this.style,
    this.selectable = true,
    this.inline = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final base = style ?? theme.textTheme.bodyMedium;
    final sheet = MarkdownStyleSheet(
      p: base?.copyWith(height: 1.6),
      strong: base?.copyWith(fontWeight: FontWeight.bold, height: 1.6),
      em: base?.copyWith(fontStyle: FontStyle.italic, height: 1.6),
      h1: theme.textTheme.titleLarge,
      h1Padding: const EdgeInsets.only(top: 12, bottom: 6),
      h2: theme.textTheme.titleMedium,
      h2Padding: const EdgeInsets.only(top: 10, bottom: 5),
      h3: theme.textTheme.titleSmall,
      h3Padding: const EdgeInsets.only(top: 8, bottom: 4),
      h4: theme.textTheme.titleSmall,
      h5: theme.textTheme.titleSmall,
      h6: theme.textTheme.titleSmall,
      listBullet: base?.copyWith(height: 1.6),
      listBulletPadding: const EdgeInsets.only(right: 8),
      blockquoteDecoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withOpacity(0.4),
        border: Border(
          left: BorderSide(color: theme.colorScheme.primary, width: 3),
        ),
      ),
      code: TextStyle(
        fontFamily: 'monospace',
        fontSize: (base?.fontSize ?? 14) - 1,
        backgroundColor: theme.colorScheme.surfaceContainerHighest,
        color: theme.colorScheme.onSurface,
      ),
      codeblockDecoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(6),
      ),
      codeblockPadding: const EdgeInsets.all(10),
      tableBorder: TableBorder.all(color: theme.colorScheme.outlineVariant),
      tableHead: base?.copyWith(fontWeight: FontWeight.bold),
      tableBody: base?.copyWith(height: 1.4),
      a: TextStyle(
        color: theme.colorScheme.primary,
        decoration: TextDecoration.none,
        fontWeight: FontWeight.w500,
      ),
    );

    return MarkdownBody(
      data: data,
      selectable: selectable,
      styleSheet: sheet,
    );
  }
}
