// Android / iOS / Desktop 平台：用 webview_flutter 加载已部署的地图 HTML。
// 根据 provider 加载对应地图：百度 / 高德 / 腾讯，三套 HTML 共用同一 postMessage 协议。
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// 非 Web 平台地图控制器（与 Web 端接口一致）。
class BaiduCampusMapController {
  _BaiduCampusMapAndroidState? _st;
  void _attach(_BaiduCampusMapAndroidState s) => _st = s;
  void _detach() => _st = null;

  /// 切换高亮步骤，地图同步平移。
  void setStep(int index) =>
      _st?._send({'type': 'set_step', 'index': index});

  /// 管理员编辑后刷新标注。
  void refresh(List<Map<String, dynamic>> steps, int cur) =>
      _st?._send({'type': 'refresh', 'steps': steps, 'currentStep': cur});

  /// 切换到指定校区完整范围取景（huifeng / langya / all）。
  void fitCampus(String campusId) =>
      _st?._send({'type': 'fit_campus', 'campusId': campusId});

  /// 2D/3D 视角切换。
  void set3D(bool enabled) => _st?._send({'type': 'set_3d', 'enabled': enabled});

  /// 底图图层切换（'standard'=标准矢量图，'satellite'=卫星影像图）。
  void setLayer(String layer) =>
      _st?._send({'type': 'set_layer', 'layer': layer});
}

class BaiduCampusMapEmbed extends StatefulWidget {
  final String baiduAk;
  final String amapAk; // 高德地图 AK
  final String tencentAk; // 腾讯地图 AK
  final String provider; // baidu / amap / tencent
  final List<Map<String, dynamic>> steps;
  final int currentStep;
  final bool editMode;
  final String campusId; // huifeng / langya
  final ValueChanged<int>? onStepSelected;
  final void Function(int, double, double)? onMarkerMoved;
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
  State<BaiduCampusMapEmbed> createState() => _BaiduCampusMapAndroidState();
}

class _BaiduCampusMapAndroidState extends State<BaiduCampusMapEmbed> {
  late final WebViewController _wc;
  bool _ready = false;

  /// 根据 provider 返回对应地图 HTML 的线上地址。
  /// 三套 HTML 均部署在 CF Pages 的 /assets/ 下，Baidu AK 域名校验通过。
  String get _mapUrl {
    final base = 'https://www.wxx-agent.online/assets';
    return switch (widget.provider) {
      'amap' => '$base/amap_campus_map.html?v=2',
      'tencent' => '$base/tencent_campus_map.html?v=2',
      _ => '$base/baidu_campus_map.html?v=4',
    };
  }

  /// 根据 provider 返回对应 AK。
  String get _ak {
    return switch (widget.provider) {
      'amap' => widget.amapAk,
      'tencent' => widget.tencentAk,
      _ => widget.baiduAk,
    };
  }

  @override
  void initState() {
    super.initState();
    widget.controller?._attach(this);
    _wc = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(Colors.transparent)
      ..addJavaScriptChannel(
        'FlutterBridge',
        onMessageReceived: _onMessage,
      )
      ..loadRequest(Uri.parse(_mapUrl));
  }

  void _onMessage(JavaScriptMessage m) {
    final Map<String, dynamic> data;
    try {
      data = jsonDecode(m.message) as Map<String, dynamic>;
    } catch (_) {
      return;
    }
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
  }

  void _sendInit() {
    _send({
      'type': 'init',
      'ak': _ak,
      'provider': widget.provider,
      'steps': widget.steps,
      'currentStep': widget.currentStep,
      'mode': widget.editMode ? 'edit' : 'view',
      'campusId': widget.campusId,
    });
    _ready = true;
  }

  void _send(Map<String, dynamic> msg) {
    final json = jsonEncode(msg);
    _wc.runJavaScript('window.postMessage($json,"*")');
  }

  @override
  void didUpdateWidget(covariant BaiduCampusMapEmbed old) {
    super.didUpdateWidget(old);
    if (!_ready) return;
    if (old.currentStep != widget.currentStep) {
      _send({'type': 'set_step', 'index': widget.currentStep});
    }
    if (old.steps != widget.steps || old.editMode != widget.editMode) {
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

  @override
  void dispose() {
    widget.controller?._detach();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => WebViewWidget(controller: _wc);
}
