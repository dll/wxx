/// 校园服务内嵌网页（按平台选择实现）：
///   Web       → inline_web_page_web.dart    (HtmlElementView + iframe)
///   Android/iOS → inline_web_page_android.dart (webview_flutter)
///   其它       → inline_web_page_stub.dart  (占位提示)
///
/// 用途：VR全景 / 学校官网 / 迎新网站 / 招生抖音 等校园服务 tab，
/// 切换时在页面内直接显示目标站点，不再新开浏览器；需要新窗口打开时
/// 由调用方用 url_launcher 显式触发。
export 'inline_web_page_stub.dart'
    if (dart.library.html) 'inline_web_page_web.dart'
    if (dart.library.io) 'inline_web_page_android.dart';
