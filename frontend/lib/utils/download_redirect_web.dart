import 'dart:html' as html;

void redirectDownloadFallbackIfNeededImpl() {
  final location = html.window.location;
  final path = location.pathname ?? '';
  final hash = location.hash;
  if (!path.startsWith('/downloads/')) return;
  if (!path.toLowerCase().endsWith('.apk')) return;
  if (hash.isEmpty) return;

  // 如果下载路径被 SPA fallback 接管，清除 #/chat、#/home 等路由片段，直达 APK。
  // 使用当前域名的 origin，避免从 www.wxx-agent.online 被强制跳到 pages.dev（域名窜跳根因）。
  location.replace('${location.origin}$path');
}
