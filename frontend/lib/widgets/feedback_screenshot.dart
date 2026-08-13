import 'dart:convert';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

/// 校验图片字节是否为可解码的有效图片（用于反馈截图防伪：避免"已截屏"但无图）。
/// 传入的 bytes 可以是 PNG/JPEG 等 Flutter 支持的格式。
Future<bool> isDecodableImage(List<int> bytes) async {
  if (bytes.isEmpty) return false;
  try {
    final codec = await ui.instantiateImageCodec(
      Uint8List.fromList(bytes),
      targetWidth: 10,
      targetHeight: 10,
    );
    codec.dispose();
    return true;
  } catch (_) {
    return false;
  }
}

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
