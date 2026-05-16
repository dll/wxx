// ignore_for_file: avoid_web_libraries_in_flutter
import 'dart:convert';
import 'dart:html' as html;

import 'screenshot_result.dart';

/// Web 平台：从 Flutter 渲染的 <canvas> 抓取当前页面 PNG
Future<ScreenshotResult> captureCurrentPage() async {
  try {
    final canvases = html.document.querySelectorAll('canvas');
    if (canvases.isEmpty) {
      return const ScreenshotResult(error: '当前页面未找到画布');
    }

    html.CanvasElement? canvas;
    for (var i = canvases.length - 1; i >= 0; i--) {
      final el = canvases[i];
      if (el is html.CanvasElement) {
        canvas = el;
        break;
      }
    }
    if (canvas == null) {
      return const ScreenshotResult(error: '当前渲染器不支持截屏（HTML 渲染器）');
    }

    final dataUrl = canvas.toDataUrl('image/png', 0.85);
    if (!dataUrl.startsWith('data:image/png;base64,')) {
      return const ScreenshotResult(error: '截屏数据格式异常');
    }
    final base64 = dataUrl.replaceFirst('data:image/png;base64,', '');
    return ScreenshotResult(base64: base64, bytes: base64Decode(base64));
  } catch (e) {
    return ScreenshotResult(error: '截屏失败：$e');
  }
}
