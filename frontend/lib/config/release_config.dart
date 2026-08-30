/// 发布信息配置。
///
/// `scripts/build-all.ps1` 会在发布构建时同步更新本文件、pubspec 版本
/// 与 Web 静态发布清单，确保 Web 首页二维码指向最新 APK。
class ReleaseConfig {
  static const String version = '0.0.28';
  static const int buildNumber = 28;
  static const String releaseDate = '2026-08-24';
  static const String apkFileName = '蔚小芯-v0.0.28.apk';
  static const String apkDownloadUrl = 'https://github.com/dll/wxx/releases/latest/download/weixiaoxin.apk';
  static const String webUrl = 'https://wxx-agent.online';

  /// 生成外部二维码服务（api.qrserver.com）的图片 URL。
  /// [data] 传原始内容，内部统一做 URI 编码。
  /// 统一入口便于日后更换二维码服务或改为本地生成（qr_flutter）。
  static String qrCodeUrl(String data, {int size = 220}) =>
      'https://api.qrserver.com/v1/create-qr-code/?size=${size}x$size&data=${Uri.encodeComponent(data)}&margin=10';
}
