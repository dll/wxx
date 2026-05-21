import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/widgets.dart';

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
/// Web 上优先使用异步 toImage（更稳定），移动端使用同步 toImageSync。
/// 最多重试 3 次，每次等待一帧渲染完成。
Future<ScreenshotResult> captureScreenshot({double pixelRatio = 1.0}) async {
  for (int attempt = 0; attempt < 3; attempt++) {
    try {
      final ctx = screenshotKey.currentContext;
      if (ctx == null) {
        // 还未挂载，等待后重试
        await Future<void>.delayed(const Duration(milliseconds: 100));
        continue;
      }
      final boundary = ctx.findRenderObject() as RenderRepaintBoundary?;
      if (boundary == null) {
        await Future<void>.delayed(const Duration(milliseconds: 100));
        continue;
      }

      // 等当前帧渲染完成
      await SchedulerBinding.instance.endOfFrame;
      if (boundary.debugNeedsPaint) {
        await Future<void>.delayed(const Duration(milliseconds: 50));
      }

      ui.Image? image;
      try {
        if (kIsWeb) {
          // Web：异步 toImage 更稳定，避免 LateInitializationError
          image = await boundary.toImage(pixelRatio: pixelRatio);
        } else {
          // 移动端：同步快照更快更稳定
          try {
            image = boundary.toImageSync(pixelRatio: pixelRatio);
          } catch (_) {
            image = await boundary.toImage(pixelRatio: pixelRatio);
          }
        }
      } catch (e) {
        if (kDebugMode) debugPrint('toImage 失败 (attempt $attempt): $e');
        await Future<void>.delayed(const Duration(milliseconds: 100));
        continue;
      }

      try {
        final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
        if (byteData == null) {
          if (kDebugMode) debugPrint('toByteData 返回 null (attempt $attempt)');
          await Future<void>.delayed(const Duration(milliseconds: 100));
          continue;
        }
        return ScreenshotResult(bytes: byteData.buffer.asUint8List());
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

  // 所有重试耗尽
  final ctx = screenshotKey.currentContext;
  if (ctx == null) {
    return const ScreenshotResult(error: '截图区域未挂载（重试3次后仍失败）');
  }
  return const ScreenshotResult(error: '截图引擎异常（重试3次后仍失败）');
}
