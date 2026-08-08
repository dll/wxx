// 扫码登录 deep link 统一入口
// 平台条件导出：Android 用 app_links 监听 App Links URL，其他平台空实现。
export 'deep_link_stub.dart'
    if (dart.library.io) 'deep_link_mobile.dart'
    if (dart.library.html) 'deep_link_stub.dart';

/// 回调签名：收到 qr-login deep link 时回调 sessionId
typedef QrLinkCallback = void Function(String sessionId);
