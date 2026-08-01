// Web 端百度地图嵌入组件（HtmlElementView + postMessage 通信）
// 底图为真实百度地图，报到流程以脉冲标注 + 抽象折线叠加其上。
import 'dart:html' as html;
import 'dart:ui_web' as ui_web;

import 'package:flutter/material.dart';

/// 向嵌入地图发送指令的控制器。
class BaiduCampusMapController {
  _BaiduCampusMapWebState? _st;
  void _attach(_BaiduCampusMapWebState s) => _st = s;
  void _detach() => _st = null;

  /// 切换当前高亮步骤，地图同步平移。
  void setStep(int index) => _st?._send({'type': 'set_step', 'index': index});

  /// 管理员编辑后刷新全部标注。
  void refresh(List<Map<String, dynamic>> steps, int cur) =>
      _st?._send({'type': 'refresh', 'steps': steps, 'currentStep': cur});
}

class BaiduCampusMapEmbed extends StatefulWidget {
  final String baiduAk;
  final List<Map<String, dynamic>> steps; // {id, title, location, lat, lng}
  final int currentStep;
  final bool editMode;
  final ValueChanged<int>? onStepSelected;
  final void Function(int index, double lat, double lng)? onMarkerMoved;
  final BaiduCampusMapController? controller;

  const BaiduCampusMapEmbed({
    super.key,
    required this.baiduAk,
    required this.steps,
    this.currentStep = 0,
    this.editMode = false,
    this.onStepSelected,
    this.onMarkerMoved,
    this.controller,
  });

  @override
  State<BaiduCampusMapEmbed> createState() => _BaiduCampusMapWebState();
}

class _BaiduCampusMapWebState extends State<BaiduCampusMapEmbed> {
  late final String _viewType;
  html.IFrameElement? _iframe;
  html.EventListener? _listener;
  bool _ready = false;

  @override
  void initState() {
    super.initState();
    widget.controller?._attach(this);
    _viewType = 'baidu-map-${DateTime.now().microsecondsSinceEpoch}';
    _setupListener();
    _registerView();
  }

  void _setupListener() {
    _listener = (html.Event e) {
      final data = (e as html.MessageEvent).data;
      if (data is! Map) return;
      final type = data['type'] as String?;
      switch (type) {
        case 'ready':
          _sendInit();
        case 'step_selected':
          final idx = (data['index'] as num?)?.toInt() ?? 0;
          widget.onStepSelected?.call(idx);
        case 'marker_moved':
          final idx = (data['index'] as num?)?.toInt() ?? 0;
          final lat = (data['lat'] as num?)?.toDouble() ?? 0.0;
          final lng = (data['lng'] as num?)?.toDouble() ?? 0.0;
          widget.onMarkerMoved?.call(idx, lat, lng);
      }
    };
    html.window.addEventListener('message', _listener!);
  }

  void _registerView() {
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (_) {
      final f = html.IFrameElement()
        // 带版本查询串：地图页文件名固定，靠 ?v= 破除 CDN/浏览器旧缓存
        ..src = '/assets/baidu_campus_map.html?v=2'
        ..style.border = '0'
        // 初始给一个明确的像素高度，避免 iframe 在 Flutter Web 布局
        // 确定前塌缩为 0，导致百度地图按 0 高度渲染（窄条/点挤一堆）
        ..style.width = '100%'
        ..style.height = '600px'
        ..allow = 'geolocation'
        ..referrerPolicy = 'strict-origin-when-cross-origin';
      _iframe = f;
      return f;
    });
  }

  void _sendInit() {
    _send({
      'type': 'init',
      'ak': widget.baiduAk,
      'steps': widget.steps,
      'currentStep': widget.currentStep,
      'mode': widget.editMode ? 'edit' : 'view',
    });
    _ready = true;
  }

  void _send(Map<String, dynamic> msg) =>
      _iframe?.contentWindow?.postMessage(msg, '*');

  @override
  void didUpdateWidget(covariant BaiduCampusMapEmbed old) {
    super.didUpdateWidget(old);
    if (!_ready) return;
    if (old.currentStep != widget.currentStep) {
      _send({'type': 'set_step', 'index': widget.currentStep});
    }
    if (old.steps != widget.steps || old.editMode != widget.editMode) {
      _send({'type': 'refresh', 'steps': widget.steps,
        'currentStep': widget.currentStep});
    }
    if (old.controller != widget.controller) {
      old.controller?._detach();
      widget.controller?._attach(this);
    }
  }

  @override
  void dispose() {
    widget.controller?._detach();
    if (_listener != null) {
      html.window.removeEventListener('message', _listener!);
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // 给 HtmlElementView 一个明确的尺寸，保证 iframe 有真实高度，
    // 否则 Flutter Web 下 iframe 高度链断裂，百度地图塌成窄条（1cm 高、点挤一堆）。
    return LayoutBuilder(
      builder: (context, constraints) {
        final w = constraints.maxWidth;
        final h = constraints.maxHeight;
        // 同步 iframe 像素尺寸（若已创建）
        if (_iframe != null && w > 0 && h > 0) {
          _iframe!.style.width = '${w}px';
          _iframe!.style.height = '${h}px';
        }
        return SizedBox(
          width: w,
          height: h > 0 ? h : 600,
          child: HtmlElementView(viewType: _viewType),
        );
      },
    );
  }
}