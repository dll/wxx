/// 发布信息配置。
///
/// `scripts/build-all.ps1` 会在发布构建时同步更新本文件、pubspec 版本
/// 与 Web 静态发布清单，确保 Web 首页二维码指向最新 APK。
class ReleaseConfig {
  static const String version = '0.0.4';
  static const int buildNumber = 4;
  static const String releaseDate = '2026-07-20';
  static const String apkFileName = '蔚小芯-v0.0.4.apk';
  static const String apkDownloadUrl = 'https://wxx-agent.pages.dev/downloads/%E8%94%9A%E5%B0%8F%E8%8A%AF-v0.0.4.apk';
}
