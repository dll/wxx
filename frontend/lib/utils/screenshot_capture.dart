import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/widgets.dart';

import 'screenshot_capture_web.dart' if (dart.library.io) 'screenshot_capture_io.dart' as platform;

/// 全局 GlobalKey — 挂在 MainShell 顶层 RepaintBoundary 上
/// 通过它可以在任意位置抓取「当前主页面」的画面帧（不含弹窗）
final GlobalKey screenshotKey = GlobalKey(debugLabel: 'screenshot');

/// 截图结果
class ScreenshotResult {
  final Uint8List? bytes;
  final String? error;

  const ScreenshotResult({this.bytes, this.error});

  bool get success => bytes != null && bytes!.isNotEmpty;
}

/// 抓取主页面当前帧（跨 Web/Android/iOS 通用）
///
/// - Web：直接从 Flutter 渲染的 <canvas> 元素抓取，绕开 RenderRepaintBoundary 的 CanvasKit 兼容问题
/// - 移动端：使用 RenderRepaintBoundary.toImage（Flutter 标准路径）
Future<ScreenshotResult> captureScreenshot({double pixelRatio = 1.0}) async {
  if (kIsWeb) {
    return platform.captureWebScreenshot();
  }
  return _captureNative(pixelRatio: pixelRatio);
}

/// 移动端 / 桌面端：使用 Flutter RenderRepaintBoundary
Future<ScreenshotResult> _captureNative({double pixelRatio = 1.0}) async {
  for (int attempt = 0; attempt < 3; attempt++) {
    try {
      final ctx = screenshotKey.currentContext;
      if (ctx == null) {
        await Future<void>.delayed(const Duration(milliseconds: 100));
        continue;
      }
      final boundary = ctx.findRenderObject() as RenderRepaintBoundary?;
      if (boundary == null) {
        await Future<void>.delayed(const Duration(milliseconds: 100));
        continue;
      }

      await SchedulerBinding.instance.endOfFrame;
      if (boundary.debugNeedsPaint) {
        await Future<void>.delayed(const Duration(milliseconds: 50));
      }

      ui.Image? image;
      try {
        image = boundary.toImageSync(pixelRatio: pixelRatio);
      } catch (_) {
        image = await boundary.toImage(pixelRatio: pixelRatio);
      }

      try {
        final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
        if (byteData == null) {
          await Future<void>.delayed(const Duration(milliseconds: 100));
          continue;
        }
        // 只取实际有效区段，避免 buffer 尾部填充字节导致 PNG 解码失败
        return ScreenshotResult(
            bytes: byteData.buffer.asUint8List(
                byteData.offsetInBytes, byteData.lengthInBytes));
      } finally {
        image.dispose();
      }
    } catch (e) {
      if (kDebugMode) debugPrint('captureScreenshot 失败 (attempt $attempt): $e');
      if (attempt == 2) {
        return ScreenshotResult(error: '截图失败：$e');
      }
      await Future<void>.delayed(const Duration(milliseconds: 100));
    }
  }
  return const ScreenshotResult(error: '截图失败：超过重试次数');
}
