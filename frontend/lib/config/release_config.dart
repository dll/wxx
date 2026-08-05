/// 发布信息配置。
///
/// `scripts/build-all.ps1` 会在发布构建时同步更新本文件、pubspec 版本
/// 与 Web 静态发布清单，确保 Web 首页二维码指向最新 APK。
class ReleaseConfig {
  static const String version = '0.0.16';
  static const int buildNumber = 16;
  static const String releaseDate = '2026-08-05';
  static const String apkFileName = '蔚小芯-v0.0.16.apk';

  /// APK 由 GitHub Release 分发，不再打进 Web 静态包
  ///（57MB 超出 Cloudflare Pages 单文件 25MB 限制）。
  /// `latest/download` 为固定入口，发新版无需改本常量。
  static const String apkDownloadUrl =
      'https://github.com/dll/wxx/releases/latest/download/weixiaoxin.apk';

  static const String webUrl = 'https://www.wxx-agent.online';
}
