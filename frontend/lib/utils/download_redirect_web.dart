import 'dart:html' as html;

import '../config/release_config.dart';

void redirectDownloadFallbackIfNeededImpl() {
  final location = html.window.location;
  final path = location.pathname ?? '';
  final hash = location.hash;
  if (!path.startsWith('/downloads/')) return;
  if (!path.toLowerCase().endsWith('.apk')) return;
  if (hash.isEmpty) return;

  // 如果下载路径被 SPA fallback 接管，清除 #/chat/#/home 等路由片段，直达 APK。
  location.replace(ReleaseConfig.apkDownloadUrl);
}
