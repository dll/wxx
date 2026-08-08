// 非 Android 平台：无 App Links 支持，空实现。

/// 回调签名：收到 qr-login deep link 时回调 sessionId
typedef QrLinkCallback = void Function(String sessionId);

/// 注册扫码登录 deep link 监听（非移动端空实现）。
/// [onQrLogin] 收到 qr-login?qr=<sessionId> 时回调。
void initQrDeepLink(QrLinkCallback onQrLogin) {}
