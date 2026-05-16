import 'dart:async';
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';

/// 显示「专注模式」全屏遮罩
///
/// 浏览器禁止 JS 主动最小化窗口，因此我们用全屏磨砂遮罩模拟「隐藏内容」体验：
/// - 任意点击或按键即恢复（再次点击 FabMenu 的图标）
/// - 显示一行简短提示「点击任意位置恢复」
/// - 倒计时 8 秒自动恢复（防误触卡死）
Future<void> showFocusMode(BuildContext context) async {
  final overlay = Overlay.of(context, rootOverlay: true);
  final entry = OverlayEntry(builder: (_) => const _FocusOverlay());
  overlay.insert(entry);
  // 兜底：30 秒强制收回，避免极端情况下卡死
  Timer? autoTimer;
  void close() {
    autoTimer?.cancel();
    if (entry.mounted) entry.remove();
  }

  // 把 close 暴露给 OverlayEntry 内部
  _focusModeCloser = close;
  autoTimer = Timer(const Duration(seconds: 30), close);
}

/// 全局关闭回调（由 _FocusOverlay 内部调用）
VoidCallback? _focusModeCloser;

class _FocusOverlay extends StatefulWidget {
  const _FocusOverlay();

  @override
  State<_FocusOverlay> createState() => _FocusOverlayState();
}

class _FocusOverlayState extends State<_FocusOverlay> with SingleTickerProviderStateMixin {
  late final AnimationController _ctrl;
  late final Animation<double> _fade;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: const Duration(milliseconds: 300));
    _fade = CurvedAnimation(parent: _ctrl, curve: Curves.easeOut);
    _ctrl.forward();
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  void _close() {
    _focusModeCloser?.call();
  }

  @override
  Widget build(BuildContext context) {
    return Positioned.fill(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: _close,
        child: FadeTransition(
          opacity: _fade,
          child: ClipRect(
            child: BackdropFilter(
              filter: ImageFilter.blur(sigmaX: 28, sigmaY: 28),
              child: Container(
                color: Colors.black.withValues(alpha: 0.78),
                alignment: Alignment.center,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.nightlight_round, color: Colors.white70, size: 64),
                    const SizedBox(height: 24),
                    const Text(
                      '专注模式',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 28,
                        fontWeight: FontWeight.w600,
                        letterSpacing: 4,
                      ),
                    ),
                    const SizedBox(height: 14),
                    Text(
                      '页面内容已隐藏，点击任意位置恢复',
                      style: TextStyle(
                        color: Colors.white.withValues(alpha: 0.72),
                        fontSize: 14,
                        letterSpacing: 1,
                      ),
                    ),
                    const SizedBox(height: 36),
                    ElevatedButton.icon(
                      style: ElevatedButton.styleFrom(
                        backgroundColor: Colors.white.withValues(alpha: 0.18),
                        foregroundColor: Colors.white,
                        elevation: 0,
                        padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 14),
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(28)),
                      ),
                      onPressed: _close,
                      icon: const Icon(Icons.visibility_outlined, size: 20),
                      label: const Text('恢复显示', style: TextStyle(fontSize: 15)),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
