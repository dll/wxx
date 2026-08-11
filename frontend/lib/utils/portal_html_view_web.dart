import 'dart:html' as html;
import 'dart:ui_web' as ui_web;

import 'package:flutter/widgets.dart';

/// 供平台视图工厂读取的当前门户 HTML
String currentPortalHtml =
    '<html><body style="font-family:sans-serif;color:#666;display:flex;align-items:center;justify-content:center;height:100vh;">加载中…</body></html>';

bool _registered = false;

/// 注册 Web 端门户 iframe 平台视图工厂（首次调用时注册一次）
void _ensureRegistered() {
  if (_registered) return;
  _registered = true;
  ui_web.platformViewRegistry.registerViewFactory('web-portal-iframe', (int viewId) {
    final el = html.IFrameElement()
      ..style.width = '100%'
      ..style.height = '100%'
      ..style.border = 'none'
      ..srcdoc = currentPortalHtml;
    return el;
  });
}

/// 更新 iframe 内容
void setPortalHtml(String htmlContent) {
  currentPortalHtml = htmlContent;
}

/// Web：以 iframe 渲染后端代理返回的门户页面（HTML）
Widget buildPortalHtmlView(String html) {
  setPortalHtml(html);
  _ensureRegistered();
  return const HtmlElementView(viewType: 'web-portal-iframe');
}
