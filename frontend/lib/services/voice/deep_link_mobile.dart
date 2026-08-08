// Android 平台：使用 app_links 监听 App Links URL（https://wxx-agent.online/qr-login?qr=xxx）
// 用户扫码后若已安装 APK，Android 系统直接唤起本应用并传递该 URL。
import 'package:app_links/app_links.dart';

/// 回调签名：收到 qr-login deep link 时回调 sessionId
typedef QrLinkCallback = void Function(String sessionId);

/// 注册扫码登录 deep link 监听。
/// 收到 qr-login?qr=<sessionId> 时回调 sessionId。
void initQrDeepLink(QrLinkCallback onQrLogin) {
  final appLinks = AppLinks();

  // 冷启动：应用被唤起时的初始链接
  appLinks.getInitialLink().then((uri) {
    if (uri != null) _handleUri(uri, onQrLogin);
  }).catchError((_) {});

  // 热启动：应用运行中收到新链接
  appLinks.uriLinkStream.listen((uri) {
    _handleUri(uri, onQrLogin);
  }).onError((_) {});
}

void _handleUri(Uri uri, QrLinkCallback onQrLogin) {
  // 仅处理 /qr-login 路径
  if (uri.path != '/qr-login' && !uri.path.startsWith('/qr-login')) return;
  final sessionId = uri.queryParameters['qr'];
  if (sessionId == null || sessionId.isEmpty) return;
  onQrLogin(sessionId);
}
