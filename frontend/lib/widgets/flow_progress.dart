import 'dart:async';
import 'package:flutter/material.dart';

/// 办事流程加载进度指示器
/// 分阶段显示当前操作，给用户清晰的等待反馈
class FlowProgressIndicator extends StatefulWidget {
  final String flowName; // "入学流程" 或 "离校流程"

  const FlowProgressIndicator({super.key, required this.flowName});

  @override
  State<FlowProgressIndicator> createState() => _FlowProgressIndicatorState();
}

class _FlowProgressIndicatorState extends State<FlowProgressIndicator> {
  final List<String> _stages = [
    '正在检索知识库...',
    '正在匹配流程数据...',
    '正在整理流程步骤...',
    '正在生成回答...',
  ];

  int _stageIndex = 0;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    // 每 2.5 秒推进一个阶段
    _timer = Timer.periodic(const Duration(milliseconds: 2500), (_) {
      if (mounted) {
        setState(() {
          if (_stageIndex < _stages.length - 1) {
            _stageIndex++;
          }
        });
      }
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // 动画图标
            _AnimatedFlowIcon(theme: theme),
            const SizedBox(height: 32),
            // 流程名称
            Text(
              widget.flowName,
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            // 当前阶段文字（带动画过渡）
            AnimatedSwitcher(
              duration: const Duration(milliseconds: 400),
              child: Text(
                _stages[_stageIndex],
                key: ValueKey(_stages[_stageIndex]),
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
            const SizedBox(height: 24),
            // 阶段进度条
            SizedBox(
              width: 200,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(4),
                child: LinearProgressIndicator(
                  value: (_stageIndex + 0.5) / _stages.length,
                  minHeight: 4,
                  backgroundColor: theme.colorScheme.surfaceContainerHighest,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 流程动画图标 — 旋转齿轮 + 搜索/文档
class _AnimatedFlowIcon extends StatefulWidget {
  final ThemeData theme;
  const _AnimatedFlowIcon({required this.theme});

  @override
  State<_AnimatedFlowIcon> createState() => _AnimatedFlowIconState();
}

class _AnimatedFlowIconState extends State<_AnimatedFlowIcon>
    with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 2),
    );
    _ctrl.repeat();
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return RotationTransition(
      turns: _ctrl,
      child: Icon(
        Icons.sync,
        size: 48,
        color: widget.theme.colorScheme.primary.withValues(alpha: 0.6),
      ),
    );
  }
}
