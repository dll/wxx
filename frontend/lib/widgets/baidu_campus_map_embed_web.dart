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

  /// 切换到指定校区完整范围取景（huifeng / langya / all）。
  void fitCampus(String campusId) =>
      _st?._send({'type': 'fit_campus', 'campusId': campusId});

  /// 2D/3D 视角切换（true=3D 倾斜透视+建筑，false=2D 俯视）。
  void set3D(bool enabled) => _st?._send({'type': 'set_3d', 'enabled': enabled});
}

/// 多地图服务商校园导航嵌入组件（Web 端）。
///
/// 支持三家地图：百度 / 高德 / 腾讯。通过 [provider] 切换，内部根据
/// provider 加载对应 HTML（baidu_campus_map.html / amap_campus_map.html /
/// tencent_campus_map.html）并注入对应 AK。三套 HTML 共用同一套 postMessage
/// 通信协议（init/set_step/refresh/fit_campus/set_3d ↔ ready/step_selected/
/// marker_moved），切换服务商时父级用 ValueKey 强制重建以重新加载 iframe。
class BaiduCampusMapEmbed extends StatefulWidget {
  final String baiduAk;
  final String amapAk; // 高德地图 JS API key
  final String tencentAk; // 腾讯地图 JS API key
  final String provider; // baidu / amap / tencent
  final List<Map<String, dynamic>> steps; // {id, title, location, lat, lng}
  final int currentStep;
  final bool editMode;
  final String campusId; // huifeng / langya
  final ValueChanged<int>? onStepSelected;
  final void Function(int index, double lat, double lng)? onMarkerMoved;
  final BaiduCampusMapController? controller;

  const BaiduCampusMapEmbed({
    super.key,
    required this.baiduAk,
    required this.amapAk,
    required this.tencentAk,
    required this.provider,
    required this.steps,
    this.currentStep = 0,
    this.editMode = false,
    this.campusId = 'huifeng',
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
    // 根据 provider 选择对应地图 HTML（三套 HTML 共用同一 postMessage 协议）
    final htmlPath = switch (widget.provider) {
      'amap' => '/assets/amap_campus_map.html?v=1',
      'tencent' => '/assets/tencent_campus_map.html?v=1',
      _ => '/assets/baidu_campus_map.html?v=3',
    };
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (_) {
      // 用容器 div 包裹 iframe：容器 100% 填满 Flutter 的 flt-platform-view，
      // iframe 再 100% 填满容器。这样 iframe 高度始终跟随 Flutter 布局，
      // 不再依赖固定像素值，彻底避免容器高度与 iframe 高度不同步导致的塌陷。
      final host = html.DivElement()
        ..style.width = '100%'
        ..style.height = '100%'
        ..style.position = 'absolute'
        ..style.top = '0'
        ..style.left = '0';
      final f = html.IFrameElement()
        // 带版本查询串：地图页文件名固定，靠 ?v= 破除 CDN/浏览器旧缓存
        ..src = htmlPath
        ..style.border = '0'
        ..style.width = '100%'
        ..style.height = '100%'
        ..style.position = 'absolute'
        ..style.top = '0'
        ..style.left = '0'
        ..allow = 'geolocation'
        ..referrerPolicy = 'strict-origin-when-cross-origin';
      host.append(f);
      _iframe = f;
      return host;
    });
  }

  void _sendInit() {
    // 根据 provider 选择对应地图 AK
    final ak = switch (widget.provider) {
      'amap' => widget.amapAk,
      'tencent' => widget.tencentAk,
      _ => widget.baiduAk,
    };
    _send({
      'type': 'init',
      'ak': ak,
      'provider': widget.provider,
      'steps': widget.steps,
      'currentStep': widget.currentStep,
      'mode': widget.editMode ? 'edit' : 'view',
      'campusId': widget.campusId,
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
    // steps 每次父级 setState 都是新 List（引用不同），
    // 必须按内容比较，否则每次点击步骤/完成按钮都触发全量标注重建。
    if (old.editMode != widget.editMode ||
        (old.steps != widget.steps &&
            _stepsJson(old.steps) != _stepsJson(widget.steps))) {
      _send({
        'type': 'refresh',
        'steps': widget.steps,
        'currentStep': widget.currentStep,
      });
    }
    // 校区切换：重新取景到对应校区完整范围
    if (old.campusId != widget.campusId) {
      _send({'type': 'fit_campus', 'campusId': widget.campusId});
    }
    if (old.controller != widget.controller) {
      old.controller?._detach();
      widget.controller?._attach(this);
    }
  }

  /// 将步骤列表序列化用于内容比较（忽略每次新建 List 的引用差异）。
  static String _stepsJson(List<Map<String, dynamic>> steps) {
    final buf = StringBuffer();
    for (final s in steps) {
      buf
        ..write(s['title'])
        ..write('|')
        ..write(s['location'])
        ..write('|')
        ..write(s['lat'])
        ..write(',')
        ..write(s['lng'])
        ..write(';');
    }
    return buf.toString();
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
    // 直接让 HtmlElementView 填满父级 tight 约束（来自外层 Positioned.fill）。
    // 不再用 LayoutBuilder 包裹：Flutter Web 下 LayoutBuilder 与 platform view
    // 配合时，flt-platform-view 容器的 CSS 高度可能不与 SizedBox 同步，导致
    // iframe 塌陷成窄条。SizedBox.expand 强制 HtmlElementView 填满父级约束，
    // iframe 用 width/height:100% 填满容器（见 _registerView），高度链稳定。
    return SizedBox.expand(
      child: HtmlElementView(viewType: _viewType),
    );
  }
}