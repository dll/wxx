import 'dart:html' as html;
import 'dart:ui_web' as ui_web;

import 'package:flutter/material.dart';

/// Web 端：用 HtmlElementView 内嵌 iframe 显示目标站点（不新开浏览器）。
class InlineWebPage extends StatefulWidget {
  final String url;
  const InlineWebPage({super.key, required this.url});

  @override
  State<InlineWebPage> createState() => _InlineWebPageState();
}

class _InlineWebPageState extends State<InlineWebPage> {
  static int _counter = 0;
  late final String _viewType;
  html.IFrameElement? _iframe;

  @override
  void initState() {
    super.initState();
    _viewType = 'inline-web-${_counter++}';
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (int viewId) {
      _iframe = html.IFrameElement()
        ..src = widget.url
        ..style.width = '100%'
        ..style.height = '100%'
        ..style.border = 'none'
        ..allow = 'fullscreen'
        ..setAttribute('sandbox', 'allow-scripts allow-same-origin allow-forms allow-popups');
      return _iframe!;
    });
  }

  @override
  Widget build(BuildContext context) {
    return HtmlElementView(viewType: _viewType);
  }
}
