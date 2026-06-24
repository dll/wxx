import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import 'feedback_dialog.dart';
import 'focus_mode.dart';
import 'voice_dialog.dart';

/// 悬浮菜单 — 精美展开动画 + 磨砂玻璃风格 + 可拖拽
/// 在所有认证页面右下角显示，展开后展示 问题反馈 / 语音导航
class FabMenu extends StatefulWidget {
  const FabMenu({super.key});

  @override
  State<FabMenu> createState() => _FabMenuState();
}

class _FabMenuState extends State<FabMenu> with TickerProviderStateMixin {
  bool _isOpen = false;
  late final AnimationController _expandCtrl;
  late final Animation<double> _rotateAnim;
  late final List<Animation<double>> _slideAnims;
  late final List<Animation<double>> _fadeAnims;

  // 可拖拽位置（相对于右下角）
  double _dx = 16;
  double _dy = 120;

  static const _items = <_FabItem>[
    _FabItem(
      icon: Icons.navigation,
      label: '导航',
      color: Color(0xFF1677FF),
      action: _FabAction.campus,
    ),
    _FabItem(
      icon: Icons.feedback_outlined,
      label: '反馈',
      color: Color(0xFF6750A4),
      action: _FabAction.feedback,
    ),
    _FabItem(
      icon: Icons.mic,
      label: '语音',
      color: Color(0xFFE65100),
      action: _FabAction.voice,
    ),
    _FabItem(
      icon: Icons.visibility_off_outlined,
      label: '专注',
      color: Color(0xFF1B5E20),
      action: _FabAction.focus,
    ),
  ];

  @override
  void initState() {
    super.initState();
    _expandCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 420),
    );
    _rotateAnim = CurvedAnimation(
      parent: _expandCtrl,
      curve: Curves.easeInOut,
    );
    _slideAnims = List.generate(_items.length, (i) {
      return CurvedAnimation(
        parent: _expandCtrl,
        curve: Interval(
          0.08 + i * 0.09,
          0.55 + i * 0.14,
          curve: Curves.elasticOut,
        ),
      );
    });
    _fadeAnims = List.generate(_items.length, (i) {
      return CurvedAnimation(
        parent: _expandCtrl,
        curve: Interval(0.02 + i * 0.09, 0.30 + i * 0.14),
      );
    });
  }

  @override
  void dispose() {
    _expandCtrl.dispose();
    super.dispose();
  }

  void _toggle() {
    setState(() {
      _isOpen = !_isOpen;
      _isOpen ? _expandCtrl.forward() : _expandCtrl.reverse();
    });
  }

  void _close() {
    if (!_isOpen) return;
    setState(() => _isOpen = false);
    _expandCtrl.reverse();
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        // 透明触控层：点击空白处关闭，但不再压暗主内容
        if (_isOpen)
          Positioned.fill(
            child: GestureDetector(
              behavior: HitTestBehavior.translucent,
              onTap: _close,
            ),
          ),
        Positioned(
          right: _dx,
          bottom: _dy,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              for (int i = _items.length - 1; i >= 0; i--)
                _buildSubItem(_items[i], i),
              const SizedBox(height: 8),
              _buildMainFab(),
            ],
          ),
        ),
      ],
    );
  }

  /// 子菜单项：白底圆角 pill 标签（彩色文字）+ 圆形图标
  Widget _buildSubItem(_FabItem item, int index) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: const Offset(0.0, 0.35),
        end: Offset.zero,
      ).animate(_slideAnims[index]),
      child: FadeTransition(
        opacity: _fadeAnims[index],
        child: Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: GestureDetector(
            onTap: () {
              _close();
              _onTap(item);
            },
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _PillLabel(text: item.label, color: item.color),
                const SizedBox(width: 6),
                _IconCircle(icon: item.icon, color: item.color),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// 主 FAB — 旋转 + 渐变色 + 拖拽
  Widget _buildMainFab() {
    final theme = Theme.of(context);
    final openColor = theme.colorScheme.error;
    final closedColor = theme.colorScheme.primary;
    final activeColor = _isOpen ? openColor : closedColor;

    return AnimatedBuilder(
      animation: _rotateAnim,
      builder: (_, child) => Transform.rotate(
        angle: _rotateAnim.value * 0.75,
        child: child,
      ),
      child: GestureDetector(
        onTap: _toggle,
        onPanUpdate: (d) => setState(() {
          _dx = (_dx - d.delta.dx).clamp(8.0, 200.0);
          _dy = (_dy - d.delta.dy).clamp(8.0, 350.0);
        }),
        child: Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: _isOpen
                  ? [const Color(0xFFBA1A1A), const Color(0xFF690005)]
                  : [const Color(0xFF6750A4), const Color(0xFF4F378B)],
            ),
            boxShadow: [
              BoxShadow(
                color: activeColor.withValues(alpha: 0.45),
                blurRadius: 14,
                offset: const Offset(0, 5),
                spreadRadius: 1,
              ),
            ],
          ),
          child: AnimatedSwitcher(
            duration: const Duration(milliseconds: 200),
            child: Icon(
              _isOpen ? Icons.close : Icons.add,
              key: ValueKey(_isOpen),
              color: Colors.white,
              size: 24,
            ),
          ),
        ),
      ),
    );
  }

  void _onTap(_FabItem item) {
    switch (item.action) {
      case _FabAction.campus:
        _openNavigation(context);
      case _FabAction.feedback:
        showFeedbackDialog(context);
      case _FabAction.voice:
        showVoiceDialog(context);
      case _FabAction.focus:
        showFocusMode(context);
    }
  }

  Future<void> _openNavigation(BuildContext context) async {
    const url = 'https://uri.amap.com/navigation?to=118.2988,32.2921,%E6%BB%81%E5%B7%9E%E5%AD%A6%E9%99%A2&mode=car&coordinate=gaode';
    await launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);
  }
}

// ── 内部组件 ──

enum _FabAction { campus, feedback, voice, focus }

class _FabItem {
  final IconData icon;
  final String label;
  final Color color;
  final _FabAction action;
  const _FabItem({
    required this.icon,
    required this.label,
    required this.color,
    required this.action,
  });
}

/// 白底圆角 pill 标签（彩色文字）
class _PillLabel extends StatelessWidget {
  final String text;
  final Color color;
  const _PillLabel({required this.text, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(14),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.12),
            blurRadius: 6,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: 12,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

/// 彩色圆形图标按钮
class _IconCircle extends StatelessWidget {
  final IconData icon;
  final Color color;
  const _IconCircle({required this.icon, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        boxShadow: [
          BoxShadow(
            color: color.withValues(alpha: 0.4),
            blurRadius: 10,
            offset: const Offset(0, 3),
            spreadRadius: 1,
          ),
        ],
      ),
      child: Icon(icon, color: Colors.white, size: 18),
    );
  }
}
