/// 页面刷新统一入口
///
/// 按平台选择实现：
/// - Web：强制整页刷新（window.location.reload）
/// - Android/iOS：无浏览器页面可刷新，no-op（由调用方做本地状态重置）
///
/// 用途：删除对话等操作完成后，Web 端整页刷新可彻底规避
/// CanvasKit 渲染冻结导致的空白页问题（见 docs/蔚小芯前端重新部署.md）。
library;
export 'page_reload_web.dart' if (dart.library.io) 'page_reload_mobile.dart';
