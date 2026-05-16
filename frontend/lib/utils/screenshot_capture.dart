import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/rendering.dart';
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

/// 抓取主页面当前帧
///
/// 基于 RepaintBoundary.toImage，跨 Web/Android/iOS 通用，无需 dart:html。
/// [pixelRatio] 截图分辨率倍率，默认 2.0；超出此值会显著增大文件体积
Future<ScreenshotResult> captureScreenshot({double pixelRatio = 2.0}) async {
  try {
    final ctx = screenshotKey.currentContext;
    if (ctx == null) {
      return const ScreenshotResult(error: '截图区域未挂载');
    }
    final boundary = ctx.findRenderObject() as RenderRepaintBoundary?;
    if (boundary == null) {
      return const ScreenshotResult(error: '未找到 RepaintBoundary');
    }
    // Web 平台首次渲染需等下一帧
    if (boundary.debugNeedsPaint) {
      await Future<void>.delayed(const Duration(milliseconds: 30));
    }
    final image = await boundary.toImage(pixelRatio: pixelRatio);
    final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
    if (byteData == null) {
      return const ScreenshotResult(error: '截图编码失败');
    }
    return ScreenshotResult(bytes: byteData.buffer.asUint8List());
  } catch (e) {
    if (kDebugMode) {
      debugPrint('captureScreenshot 失败: $e');
    }
    return ScreenshotResult(error: '截图失败：$e');
  }
}
