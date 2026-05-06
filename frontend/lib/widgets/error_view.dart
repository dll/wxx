import 'package:flutter/material.dart';

/// 统一的错误/空状态组件
///
/// 通过 [isEmpty] 区分空状态和错误状态：
/// - 空状态：显示图标 + 主提示文本 + 副文本（可选），无重试按钮
/// - 错误状态：显示图标 + 错误消息 + 重试按钮
class ErrorView extends StatelessWidget {
  /// 错误或空状态描述文本
  final String message;

  /// 次要描述文本（仅空状态显示）
  final String? subtitle;

  /// 重试回调（传入则显示重试按钮）
  final VoidCallback? onRetry;

  /// 图标
  final IconData icon;

  /// 图标大小
  final double iconSize;

  /// 是否为无数据空状态（true 时不显示重试按钮，除非同时传了 onRetry）
  final bool isEmpty;

  const ErrorView({
    super.key,
    required this.message,
    this.subtitle,
    this.onRetry,
    this.icon = Icons.error_outline,
    this.iconSize = 48,
    this.isEmpty = false,
  });

  /// 工厂：错误状态
  factory ErrorView.error({
    Key? key,
    required String message,
    VoidCallback? onRetry,
  }) {
    return ErrorView(
      key: key,
      message: message,
      icon: Icons.error_outline,
      onRetry: onRetry,
    );
  }

  /// 工厂：空数据状态
  factory ErrorView.empty({
    Key? key,
    required String message,
    String? subtitle,
    IconData icon = Icons.inbox_outlined,
  }) {
    return ErrorView(
      key: key,
      message: message,
      subtitle: subtitle,
      icon: icon,
      isEmpty: true,
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = isEmpty
        ? theme.colorScheme.outline
        : theme.colorScheme.onSurfaceVariant;

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: iconSize, color: color.withValues(alpha: 0.6)),
            const SizedBox(height: 12),
            Text(
              message,
              style: theme.textTheme.bodyLarge?.copyWith(color: color),
              textAlign: TextAlign.center,
            ),
            if (subtitle != null) ...[
              const SizedBox(height: 4),
              Text(
                subtitle!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: color.withValues(alpha: 0.6),
                ),
                textAlign: TextAlign.center,
              ),
            ],
            if (onRetry != null) ...[
              const SizedBox(height: 16),
              FilledButton.tonalIcon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh, size: 18),
                label: const Text('重试'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
