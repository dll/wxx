import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/widgets.dart';

/// 全局 GlobalKey — 挂在 MainShell 顶层 RepaintBoundary 上
/// 通过它可以在任意位置抓取「当前主页面」的画面帧（不含弹窗）
final GlobalKey screenshotKey = GlobalKey();

/// 截图结果
class ScreenshotResult {
  final Uint8List? bytes;
  final String? error;

  const ScreenshotResult({this.bytes, this.error});

  bool get success => bytes != null && bytes!.isNotEmpty;
}

/// 抓取主页面当前帧（跨 Web/Android/iOS 通用）
///
/// 关键防御：
/// 1. 等 endOfFrame 让 RepaintBoundary 完成渲染，避免 Web 上 LateInitializationError
/// 2. 优先 toImageSync（Flutter 3.7+），同步路径在 Web 上更稳定
/// 3. 失败时兜底走 await toImage
/// 4. 用完立即 dispose 释放显存
Future<ScreenshotResult> captureScreenshot({double pixelRatio = 1.0}) async {
  try {
    final ctx = screenshotKey.currentContext;
    if (ctx == null) {
      return const ScreenshotResult(error: '截图区域未挂载');
    }
    final boundary = ctx.findRenderObject() as RenderRepaintBoundary?;
    if (boundary == null) {
      return const ScreenshotResult(error: '未找到 RepaintBoundary');
    }

    // 等当前帧渲染完成（关键：解决 Web 上 LateInitializationError）
    await SchedulerBinding.instance.endOfFrame;
    if (boundary.debugNeedsPaint) {
      await Future<void>.delayed(const Duration(milliseconds: 50));
    }

    ui.Image? image;
    try {
      // Flutter 3.7+ 同步快照，Web 上比 toImage 稳定
      image = boundary.toImageSync(pixelRatio: pixelRatio);
    } catch (e1) {
      if (kDebugMode) debugPrint('toImageSync 失败，回退 toImage: $e1');
      try {
        image = await boundary.toImage(pixelRatio: pixelRatio);
      } catch (e2) {
        if (kDebugMode) debugPrint('toImage 失败: $e2');
        return ScreenshotResult(error: '截图引擎异常：$e2');
      }
    }

    try {
      final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
      if (byteData == null) {
        return const ScreenshotResult(error: '截图编码失败');
      }
      return ScreenshotResult(bytes: byteData.buffer.asUint8List());
    } finally {
      image.dispose();
    }
  } catch (e) {
    if (kDebugMode) debugPrint('captureScreenshot 失败: $e');
    return ScreenshotResult(error: '截图失败：$e');
  }
}
