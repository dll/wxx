import 'download_redirect_stub.dart'
    if (dart.library.html) 'download_redirect_web.dart';

/// 处理 Cloudflare Pages 将 /downloads/* fallback 到 Flutter 的情况。
void redirectDownloadFallbackIfNeeded() {
  redirectDownloadFallbackIfNeededImpl();
}
