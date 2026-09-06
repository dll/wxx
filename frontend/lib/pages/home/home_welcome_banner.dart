import 'package:flutter/material.dart';

/// 首页欢迎横幅，仅负责展示与进入问芯的交互。
class HomeWelcomeBanner extends StatelessWidget {
  const HomeWelcomeBanner({
    super.key,
    required this.displayName,
    required this.gradeThemeName,
    required this.accent,
    required this.onOpenChat,
  });

  final String displayName;
  final String gradeThemeName;
  final Color accent;
  final VoidCallback onOpenChat;
  String _greeting(DateTime now) {
    final hour = now.hour;
    return hour < 6
        ? '夜深了'
        : hour < 12
            ? '上午好'
            : hour < 14
                ? '中午好'
                : hour < 18
                    ? '下午好'
                    : '晚上好';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final greeting = _greeting(DateTime.now());
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(colors: [
          accent.withOpacity(0.14),
          theme.colorScheme.surfaceContainerLow,
          theme.colorScheme.surfaceContainerLow
        ], begin: Alignment.topLeft, end: Alignment.bottomRight),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: accent.withOpacity(0.18)),
      ),
      child: Stack(children: [
        Positioned(
            right: -24, top: -30, child: _glow(120, accent.withOpacity(0.18))),
        Positioned(
            right: 40,
            bottom: -36,
            child: _glow(90, theme.colorScheme.secondary.withOpacity(0.12))),
        Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Expanded(
                child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                  Row(children: [
                    Icon(Icons.auto_awesome, size: 18, color: accent),
                    const SizedBox(width: 6),
                    Text('$gradeThemeName主题',
                        style: theme.textTheme.labelMedium?.copyWith(
                            color: accent, fontWeight: FontWeight.w700))
                  ]),
                  const SizedBox(height: 6),
                  Text('$greeting，$displayName',
                      style: theme.textTheme.titleLarge
                          ?.copyWith(fontWeight: FontWeight.w800)),
                  const SizedBox(height: 2),
                  Text('今日安排与智能服务都在这里',
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant)),
                ])),
            Container(
                width: 52,
                height: 52,
                decoration: BoxDecoration(
                    color: accent.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(16)),
                child: Icon(Icons.school_outlined, color: accent, size: 28))
          ]),
          const SizedBox(height: 16),
          Material(
              color: theme.colorScheme.surface.withOpacity(0.9),
              borderRadius: BorderRadius.circular(14),
              child: InkWell(
                  onTap: onOpenChat,
                  borderRadius: BorderRadius.circular(14),
                  child: Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 14, vertical: 12),
                      child: Row(children: [
                        Icon(Icons.auto_awesome,
                            size: 20, color: theme.colorScheme.secondary),
                        const SizedBox(width: 10),
                        Expanded(
                            child: Text('问小芯：政策、流程、学习与校园生活',
                                style: theme.textTheme.bodyMedium?.copyWith(
                                    color:
                                        theme.colorScheme.onSurfaceVariant))),
                        Icon(Icons.mic_none,
                            size: 20, color: theme.colorScheme.primary)
                      ])))),
        ]),
      ]),
    );
  }

  static Widget _glow(double size, Color color) => Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: RadialGradient(colors: [color, color.withOpacity(0)])));
}
