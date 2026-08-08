/// 发布信息配置。
///
/// `scripts/build-all.ps1` 会在发布构建时同步更新本文件、pubspec 版本
/// 与 Web 静态发布清单，确保 Web 首页二维码指向最新 APK。
class ReleaseConfig {
  static const String version = '0.0.26';
  static const int buildNumber = 26;
  static const String releaseDate = '2026-08-09';
  static const String apkFileName = '蔚小芯-v0.0.26.apk';
  static const String apkDownloadUrl = 'https://github.com/dll/wxx/releases/latest/download/weixiaoxin.apk';
  static const String webUrl = 'https://wxx-agent.online';
}
