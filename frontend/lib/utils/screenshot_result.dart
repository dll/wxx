import 'dart:typed_data';

/// 当前页面截屏结果
class ScreenshotResult {
  final Uint8List? bytes;
  final String? base64;
  final String? error;

  const ScreenshotResult({this.bytes, this.base64, this.error});

  bool get success => bytes != null && bytes!.isNotEmpty;
}
