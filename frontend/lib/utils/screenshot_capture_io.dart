import 'screenshot_capture.dart';

/// 非 Web 平台：占位（不会被调用）
Future<ScreenshotResult> captureWebScreenshot() async {
  return const ScreenshotResult(error: 'Web 截图仅支持 Web 平台');
}
