import 'screenshot_result.dart';

/// 非 Web 平台暂不支持自动截屏
Future<ScreenshotResult> captureCurrentPage() async {
  return const ScreenshotResult(error: '当前平台暂不支持自动截屏');
}
