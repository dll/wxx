import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

import 'portal_html_view_web.dart'
    if (dart.library.io) 'portal_html_view_stub.dart';

export 'portal_html_view_web.dart'
    if (dart.library.io) 'portal_html_view_stub.dart';

/// 门户内嵌浏览组件：Web 用 iframe 渲染代理 HTML，其他平台占位
Widget portalHtmlView(String html) {
  if (kIsWeb) return buildPortalHtmlView(html);
  return const Center(child: Text('当前平台暂不支持内嵌门户浏览'));
}
