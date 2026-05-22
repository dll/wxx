import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';

/// 反馈截图渲染：兼容 data:image base64 URL（多实例持久方案）和 http URL（旧记录）。
///
/// 使用：替代 Image.network(url) 调用，自动按 url 协议选择 Image.memory 或 Image.network。
class FeedbackScreenshot extends StatelessWidget {
  final String url;
  final double? height;
  final double? width;
  final BoxFit fit;
  final int? cacheHeight;
  final WidgetBuilder? errorBuilder;

  const FeedbackScreenshot({
    super.key,
    required this.url,
    this.height,
    this.width,
    this.fit = BoxFit.cover,
    this.cacheHeight,
    this.errorBuilder,
  });

  @override
  Widget build(BuildContext context) {
    if (url.isEmpty) return const SizedBox.shrink();

    if (url.startsWith('data:')) {
      // 解析 data:image/xxx;base64,xxx
      final commaIdx = url.indexOf(',');
      if (commaIdx <= 0) return _buildError(context);
      try {
        final bytes = base64Decode(url.substring(commaIdx + 1));
        return Image.memory(
          Uint8List.fromList(bytes),
          height: height,
          width: width,
          fit: fit,
          cacheHeight: cacheHeight,
          errorBuilder: (_, __, ___) => _buildError(context),
        );
      } catch (_) {
        return _buildError(context);
      }
    }

    return Image.network(
      url,
      height: height,
      width: width,
      fit: fit,
      cacheHeight: cacheHeight,
      errorBuilder: (_, __, ___) => _buildError(context),
    );
  }

  Widget _buildError(BuildContext context) {
    if (errorBuilder != null) return errorBuilder!(context);
    final theme = Theme.of(context);
    return Container(
      height: height ?? 40,
      color: theme.colorScheme.surfaceContainerHighest,
      child: Center(
        child: Text('截图加载失败', style: theme.textTheme.labelSmall),
      ),
    );
  }
}
