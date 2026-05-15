import 'dart:math' show pi;
import 'package:flutter/material.dart';
import 'feedback_dialog.dart';

/// 悬浮菜单（FAB），展开后显示"反馈"和"语音"两个选项
class FabMenu extends StatefulWidget {
  const FabMenu({super.key});

  @override
  State<FabMenu> createState() => _FabMenuState();
}

class _FabMenuState extends State<FabMenu> with TickerProviderStateMixin {
  bool _isOpen = false;
  late final AnimationController _controller;
  late final Animation<double> _rotateAnim;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: const Duration(milliseconds: 200),
      vsync: this,
    );
    _rotateAnim = Tween<double>(begin: 0, end: 0.125).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _toggle() {
    setState(() {
      _isOpen = !_isOpen;
      if (_isOpen) {
        _controller.forward();
      } else {
        _controller.reverse();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Stack(
      children: [
        // 展开的菜单项（在 FAB 上方）
        if (_isOpen) ...[
          // 语音按钮
          Positioned(
            bottom: 70,
            right: 0,
            child: _buildMenuItem(
              icon: Icons.mic,
              label: '语音',
              color: theme.colorScheme.secondary,
              onTap: () {
                _toggle();
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('语音功能：请长按说话或说出"蔚小芯"唤醒')),
                );
              },
            ),
          ),
          // 反馈按钮
          Positioned(
            bottom: 130,
            right: 0,
            child: _buildMenuItem(
              icon: Icons.feedback_outlined,
              label: '反馈',
              color: theme.colorScheme.primary,
              onTap: () {
                _toggle();
                showFeedbackDialog(context);
              },
            ),
          ),
        ],
        // 主 FAB
        AnimatedBuilder(
          animation: _rotateAnim,
          builder: (context, child) {
            return Transform.rotate(
              angle: _rotateAnim.value * 2 * pi,
              child: child,
            );
          },
          child: FloatingActionButton(
            onPressed: _toggle,
            tooltip: _isOpen ? '关闭菜单' : '更多功能',
            child: Icon(_isOpen ? Icons.close : Icons.more_horiz),
          ),
        ),
      ],
    );
  }

  Widget _buildMenuItem({
    required IconData icon,
    required String label,
    required Color color,
    required VoidCallback onTap,
  }) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerHighest.withValues(alpha: 0.9),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(label, style: Theme.of(context).textTheme.labelSmall),
        ),
        const SizedBox(height: 4),
        FloatingActionButton.small(
          heroTag: null,
          onPressed: onTap,
          backgroundColor: color,
          child: Icon(icon),
        ),
      ],
    );
  }
}
