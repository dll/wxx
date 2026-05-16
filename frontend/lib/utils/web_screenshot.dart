// 当前页面截屏入口
// - Web: 通过 dart:html 抓取 Flutter <canvas>
// - 非 Web: 走 stub，返回不支持
export 'web_screenshot_stub.dart'
    if (dart.library.html) 'web_screenshot_web.dart';
