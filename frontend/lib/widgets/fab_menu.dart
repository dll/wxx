import 'dart:ui' show ImageFilter;
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../utils/storage.dart';

/// 悬浮菜单 — 精美展开动画 + 磨砂玻璃风格 + 可拖拽
/// 在所有认证页面右下角显示，展开后展示 问题反馈 / 语音导航 / 数字孪生
class FabMenu extends StatefulWidget {
  const FabMenu({super.key});

  @override
  State<FabMenu> createState() => _FabMenuState();
}

class _FabMenuState extends State<FabMenu> with TickerProviderStateMixin {
  bool _isOpen = false;
  late final AnimationController _expandCtrl;
  late final Animation<double> _overlayAnim;
  late final Animation<double> _rotateAnim;
  late final List<Animation<double>> _slideAnims;
  late final List<Animation<double>> _fadeAnims;

  // 可拖拽位置（相对于右下角）
  double _dx = 16;
  double _dy = 80;

  /// 子菜单项：问题反馈 / 语音导航 / 数字孪生
  static const _items = <_FabItem>[
    _FabItem(
        icon: Icons.feedback_outlined,
        label: '问题反馈',
        color: Color(0xFF6750A4),
        action: _FabAction.feedback),
    _FabItem(
        icon: Icons.mic,
        label: '语音导航',
        color: Color(0xFFE65100),
        action: _FabAction.voice),
    _FabItem(
        icon: Icons.person_pin_circle_outlined,
        label: '数字孪生',
        color: Color(0xFF1565C0),
        action: _FabAction.twin),
  ];

  @override
  void initState() {
    super.initState();
    _expandCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 420),
    );
    _overlayAnim = CurvedAnimation(
      parent: _expandCtrl,
      curve: const Interval(0.0, 0.5, curve: Curves.easeOut),
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
    final theme = Theme.of(context);
    return Stack(
      children: [
        // ── 半透明遮罩层 ──
        if (_isOpen)
          Positioned.fill(
            child: GestureDetector(
              onTap: _close,
              child: FadeTransition(
                opacity: _overlayAnim,
                child: Container(color: Colors.black38),
              ),
            ),
          ),
        // ── FAB 及子菜单 ──
        Positioned(
          right: _dx,
          bottom: _dy,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              // 子菜单（倒序遍历，从下往上排列）
              for (int i = _items.length - 1; i >= 0; i--)
                _buildSubItem(_items[i], i, theme),
              const SizedBox(height: 8),
              // 主按钮
              _buildMainFab(theme),
            ],
          ),
        ),
      ],
    );
  }

  /// 子菜单项：磨砂玻璃标签 + 圆形图标按钮（紧凑型）
  Widget _buildSubItem(_FabItem item, int index, ThemeData theme) {
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
                _FrostedLabel(text: item.label, theme: theme),
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
  Widget _buildMainFab(ThemeData theme) {
    final openColor = theme.colorScheme.error;
    final closedColor = theme.colorScheme.primary;
    final activeColor = _isOpen ? openColor : closedColor;

    return AnimatedBuilder(
      animation: _rotateAnim,
      builder: (_, child) => Transform.rotate(
        angle: _rotateAnim.value * 0.75, // 135°
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
                color: activeColor.withOpacity( 0.45),
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
      case _FabAction.feedback:
        // 问题反馈：未登录跳登录页，已登录跳我的反馈
        if (!Storage.isLoggedIn) {
          context.go('/login');
        } else {
          context.go('/my-feedbacks');
        }
      case _FabAction.voice:
        // 语音导航：跳对话页并聚焦语音输入
        context.go('/chat');
      case _FabAction.twin:
        // 数字孪生：学生角色跳数字画像
        context.go('/student/digital-twin');
    }
  }
}

/// 子菜单动作枚举
enum _FabAction { feedback, voice, twin }

/// 子菜单项定义
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

/// 磨砂玻璃标签（紧凑型）
class _FrostedLabel extends StatelessWidget {
  final String text;
  final ThemeData theme;
  const _FrostedLabel({required this.text, required this.theme});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(6),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 10, sigmaY: 10),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface.withOpacity( 0.82),
            borderRadius: BorderRadius.circular(6),
            border: Border.all(
              color: theme.colorScheme.outlineVariant.withOpacity( 0.25),
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withOpacity( 0.08),
                blurRadius: 8,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: Text(
            text,
            style: theme.textTheme.labelMedium?.copyWith(
              fontWeight: FontWeight.w600,
              fontSize: 12,
            ),
          ),
        ),
      ),
    );
  }
}

/// 彩色圆形图标按钮（紧凑型）
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
            color: color.withOpacity( 0.4),
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
