// Android / 非 Web 平台兜底实现（暂不依赖 webview_flutter 插件）
import 'package:flutter/material.dart';

/// 非 Web 平台的空控制器（接口兼容）。
class BaiduCampusMapController {
  void setStep(int index) {}
  void refresh(List<Map<String, dynamic>> steps, int cur) {}
}

/// Android 平台提示卡片；后续可替换为 webview_flutter 实现。
class BaiduCampusMapEmbed extends StatelessWidget {
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

  final String baiduAk;
  final List<Map<String, dynamic>> steps;
  final int currentStep;
  final bool editMode;
  final ValueChanged<int>? onStepSelected;
  final void Function(int, double, double)? onMarkerMoved;
  final BaiduCampusMapController? controller;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLow,
        borderRadius: BorderRadius.circular(16),
      ),
      child: const Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(Icons.map_outlined, size: 48, color: Colors.grey),
          SizedBox(height: 8),
          Text('地图仅在 Web 端可用', style: TextStyle(color: Colors.grey)),
          SizedBox(height: 4),
          Text('Android 地图功能建设中',
              style: TextStyle(fontSize: 12, color: Colors.grey)),
        ]),
      ),
    );
  }
}
