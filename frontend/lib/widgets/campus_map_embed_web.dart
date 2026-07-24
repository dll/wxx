import 'dart:html' as html;
import 'dart:ui_web' as ui_web;

import 'package:flutter/material.dart';

class CampusMapEmbed extends StatefulWidget {
  final String url;
  final String title;

  const CampusMapEmbed({super.key, required this.url, required this.title});

  @override
  State<CampusMapEmbed> createState() => _CampusMapEmbedState();
}

class _CampusMapEmbedState extends State<CampusMapEmbed> {
  late String _viewType;

  @override
  void initState() {
    super.initState();
    _registerView();
  }

  @override
  void didUpdateWidget(covariant CampusMapEmbed oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url) {
      _registerView();
    }
  }

  void _registerView() {
    _viewType =
        'campus-map-${widget.url.hashCode}-${DateTime.now().microsecondsSinceEpoch}';
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (int viewId) {
      return html.IFrameElement()
        ..src = widget.url
        ..style.border = '0'
        ..style.width = '100%'
        ..style.height = '100%'
        ..allowFullscreen = true
        ..referrerPolicy = 'no-referrer-when-downgrade';
    });
  }

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(18),
      child: HtmlElementView(viewType: _viewType),
    );
  }
}
