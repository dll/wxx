// Web 端百度地图嵌入组件（HtmlElementView + postMessage 通信）
// 底图为真实百度地图，报到流程以脉冲标注 + 折线叠加其上。
import 'dart:async';
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

  /// 底图图层切换（'standard'=标准矢量图，'satellite'=卫星影像图）。
  void setLayer(String layer) =>
      _st?._send({'type': 'set_layer', 'layer': layer});
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
  String _currentHtmlPath = '';

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
    // 根据 provider 选择对应地图 HTML（三套 HTML 共用同一 postMessage 协议）。
    // provider/campusId 变化时通过 didUpdateWidget 更新 iframe.src 重新加载，
    // 不依赖 ValueKey 重建（Flutter Web HtmlElementView 下 key 重建不可靠）。
    final htmlPath = _buildHtmlPath();
    _currentHtmlPath = htmlPath;
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (viewId) {
      // 工厂返回的元素会被 Flutter append 到自动创建的 flt-platform-view 里。
      // Flutter Web canvaskit 下 flt-platform-view 的 CSS 宽高由 RenderBox
      // 控制，但部分场景下与 RenderBox 同步存在延迟或为 0×0。这里用轮询
      // 主动读取 host 父元素（flt-platform-view）的实际像素尺寸，并显式
      // 写到 host 与 iframe 上，保证 iframe 不会塌缩为 0。
      final host = html.DivElement()
        ..style.width = '100%'
        ..style.height = '100%'
        ..style.display = 'block'
        ..style.overflow = 'hidden';
      final f = html.IFrameElement()
        ..src = htmlPath
        ..style.border = '0'
        ..style.display = 'block'
        ..style.width = '100%'
        ..style.height = '100%'
        ..allow = 'geolocation'
        ..referrerPolicy = 'strict-origin-when-cross-origin';
      host.append(f);
      _iframe = f;
      _observePlatformView(host, f);
      return host;
    });
  }

  /// 根据 provider/campusId/AK 计算 iframe 的 HTML 路径。
  String _buildHtmlPath() {
    final akParam = switch (widget.provider) {
      'amap' => widget.amapAk,
      'tencent' => widget.tencentAk,
      _ => widget.baiduAk,
    };
    final campusParam = widget.campusId.isEmpty ? 'huifeng' : widget.campusId;
    return switch (widget.provider) {
      'amap' =>
        '/assets/amap_campus_map.html?v=6&ak=${Uri.encodeComponent(akParam)}&campus=$campusParam',
      'tencent' =>
        '/assets/tencent_campus_map.html?v=6&ak=${Uri.encodeComponent(akParam)}&campus=$campusParam',
      _ =>
        '/assets/baidu_campus_map.html?v=12&ak=${Uri.encodeComponent(akParam)}&campus=$campusParam',
    };
  }

  /// 轮询读取 host 父元素（flt-platform-view）的像素尺寸，显式同步到
  /// host 与 iframe。如果 flt-platform-view 尺寸为 0（Flutter Web 已知 bug），
  /// 向上查找有尺寸的祖先元素作为 fallback，并强制设置 flt-platform-view
  /// 的 CSS 尺寸。确保 iframe 始终有非零尺寸，避免浏览器中止加载。
  void _observePlatformView(html.Element host, html.IFrameElement iframe) {
    int attempts = 0;
    void check() {
      attempts++;
      final parent = host.parent;
      if (parent == null) {
        if (attempts < 60) Timer(const Duration(milliseconds: 50), check);
        return;
      }
      var rect = parent.getBoundingClientRect();
      var w = rect.width.toInt();
      var h = rect.height.toInt();
      // flt-platform-view 尺寸为 0 时，向上查找有尺寸的祖先元素
      if (w == 0 || h == 0) {
        html.Element? p = parent.parent;
        while (p != null) {
          final r = p.getBoundingClientRect();
          if (r.width > 0 && r.height > 0) {
            w = r.width.toInt();
            h = r.height.toInt();
            break;
          }
          p = p.parent;
        }
      }
      if (w > 0 && h > 0) {
        iframe.style.width = '${w}px';
        iframe.style.height = '${h}px';
        host.style.width = '${w}px';
        host.style.height = '${h}px';
        // 强制设置 flt-platform-view 的 CSS 尺寸（覆盖 Flutter 的 0×0）
        parent.style
          ..width = '${w}px'
          ..height = '${h}px';
      }
      // 持续监听尺寸变化（窗口缩放、布局重排），最多轮询 60 次 ≈ 6s
      if (attempts < 60) Timer(const Duration(milliseconds: 100), check);
    }
    Timer(const Duration(milliseconds: 50), check);
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
    // provider 或 campusId 变化：直接更新 iframe.src 重新加载对应地图。
    // Flutter Web HtmlElementView 下 ValueKey 重建不可靠（iframe 不会重新
    // 创建），改用 src 更新让浏览器重新加载 HTML，新 HTML 的 window.load
    // 会自动初始化并用 URL 的 ak/campus 参数，ready 后 _sendInit 同步 steps。
    if (old.provider != widget.provider ||
        old.campusId != widget.campusId) {
      final newHtmlPath = _buildHtmlPath();
      if (newHtmlPath != _currentHtmlPath && _iframe != null) {
        _currentHtmlPath = newHtmlPath;
        _ready = false;
        _iframe!.src = newHtmlPath;
        if (old.controller != widget.controller) {
          old.controller?._detach();
          widget.controller?._attach(this);
        }
        return; // src 变化后等新 ready，跳过本次后续消息
      }
    }
    if (!_ready) return;
    if (old.currentStep != widget.currentStep) {
      _send({'type': 'set_step', 'index': widget.currentStep});
    }
    // steps 每次父级 setState 都是新 List（引用不同），
    // 必须按内容比较，否则每次点击步骤/完成按钮都触发全量标注重建。
    // editMode 变化也走 refresh：HTML 收到后更新 _mode 并重建标注，
    // 新标注在 _mode==='edit' 时绑定拖拽，实现编辑模式切换。
    if (old.editMode != widget.editMode ||
        (old.steps != widget.steps &&
            _stepsJson(old.steps) != _stepsJson(widget.steps))) {
      _send({
        'type': 'refresh',
        'steps': widget.steps,
        'currentStep': widget.currentStep,
        'mode': widget.editMode ? 'edit' : 'view',
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
    // 使用 LayoutBuilder 获取父级约束的显式尺寸，传给 SizedBox。
    // 这样 flt-platform-view 的 CSS 宽高 = SizedBox 的宽高，不会出现 0×0。
    // 配合外层 Stack(fit: StackFit.expand)，constraints.maxWidth/Height
    // 必为有限值，SizedBox 拿到精确尺寸后 HtmlElementView 渲染稳定。
    return LayoutBuilder(
      builder: (context, constraints) {
        final w = constraints.maxWidth.isFinite ? constraints.maxWidth : 0.0;
        final h = constraints.maxHeight.isFinite ? constraints.maxHeight : 0.0;
        return SizedBox(
          width: w,
          height: h,
          child: HtmlElementView(viewType: _viewType),
        );
      },
    );
  }
}