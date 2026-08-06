import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../models/avatar_config.dart';
import 'avatar_painter.dart';

/// 数字人形象卡片 — 3D 透视 + 呼吸动画 + 磨砂背景 + 五维分数叠加层。
class AvatarCard extends StatefulWidget {
  final AvatarConfig config;
  final double height;

  const AvatarCard({super.key, required this.config, this.height = 300});

  @override
  State<AvatarCard> createState() => _AvatarCardState();
}

class _AvatarCardState extends State<AvatarCard>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  AvatarConfig get config => widget.config;

  @override
  void initState() {
    super.initState();
    // 呼吸动画：2.5 秒周期，1.0 ↔ 1.03 轻微缩放
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 2500),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final secondary = theme.colorScheme.tertiary;

    return Container(
      height: widget.height,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(24),
        boxShadow: [
          // 4 层阴影叠加：环境光 → 主光源 → 补光 → 轮廓光（伪 3D 纵深）
          BoxShadow(
            color: primary.withOpacity(0.18),
            blurRadius: 32,
            offset: const Offset(0, 14),
          ),
          BoxShadow(
            color: primary.withOpacity(0.10),
            blurRadius: 12,
            offset: const Offset(0, 5),
          ),
          BoxShadow(
            color: Colors.white.withOpacity(0.30),
            blurRadius: 6,
            offset: const Offset(-3, -3),
          ),
          BoxShadow(
            color: primary.withOpacity(0.20),
            blurRadius: 3,
            offset: const Offset(0, 1),
          ),
        ],
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(24),
        child: Stack(
          fit: StackFit.expand,
          children: [
            // 磨砂玻璃背景
            BackdropFilter(
              filter: ui.ImageFilter.blur(sigmaX: 6, sigmaY: 6),
              child: Container(
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    colors: [
                      primary.withOpacity(0.16),
                      secondary.withOpacity(0.12),
                      primary.withOpacity(0.06),
                    ],
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                  ),
                  border: Border.all(
                    color: Colors.white.withOpacity(0.25),
                  ),
                ),
              ),
            ),

            // 3D 透视容器（轻微 X 轴旋转营造立体感）
            AnimatedBuilder(
              animation: _controller,
              builder: (context, _) {
                final t = _controller.value;
                final scale = 1.0 + t * 0.03;
                return Transform(
                  alignment: Alignment.bottomCenter,
                  transform: Matrix4.identity()
                    ..setEntry(3, 2, 0.0012) // 透视投影
                    ..rotateX(0.04 + t * 0.03) // 呼吸微倾
                    ..scale(scale),
                  child: CustomPaint(
                    painter: AvatarPainter(
                      config: widget.config,
                      primary: primary,
                      secondary: secondary,
                    ),
                  ),
                );
              },
            ),

            // 顶部信息叠加层
            Positioned(
              top: 14,
              left: 16,
              right: 16,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _buildInfoChip(
                    theme,
                    Icons.person_outline,
                    widget.config.displayName,
                    light: true,
                  ),
                  const Spacer(),
                  _buildScoreChip(theme, widget.config.overall),
                ],
              ),
            ),

            // 底部五维分数叠加层
            Positioned(
              left: 16,
              right: 16,
              bottom: 12,
              child: _buildDimensionBar(theme),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoChip(ThemeData theme, IconData icon, String text,
      {bool light = false}) {
    final fg = light ? Colors.white : theme.colorScheme.onSurface;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.20),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 13, color: fg.withOpacity(0.9)),
          const SizedBox(width: 4),
          Text(
            text,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: fg,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildScoreChip(ThemeData theme, double overall) {
    final color = overall >= 80
        ? const Color(0xFF2E7D32)
        : overall >= 60
            ? const Color(0xFFE65100)
            : const Color(0xFFC62828);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.85),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        '综合 ${overall.toStringAsFixed(0)} 分',
        style: const TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w700,
          color: Colors.white,
        ),
      ),
    );
  }

  /// 底部五维迷你进度条
  Widget _buildDimensionBar(ThemeData theme) {
    final dims = [
      ('学业', config.academic),
      ('能力', config.ability),
      ('思想', config.ideological),
      ('情感', config.emotional),
      ('社交', config.social),
    ];
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.30),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final (label, score) in dims) ...[
            Row(
              children: [
                SizedBox(
                  width: 34,
                  child: Text(
                    label,
                    style: const TextStyle(
                        fontSize: 10, color: Colors.white70),
                  ),
                ),
                Expanded(
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(999),
                    child: LinearProgressIndicator(
                      value: (score / 100).clamp(0.0, 1.0),
                      minHeight: 5,
                      backgroundColor: Colors.white.withOpacity(0.15),
                      valueColor: AlwaysStoppedAnimation(
                        Color.lerp(
                          const Color(0xFF64B5F6),
                          const Color(0xFF42A5F5),
                          (score / 100).clamp(0.0, 1.0),
                        )!,
                      ),
                    ),
                  ),
                ),
                SizedBox(
                  width: 28,
                  child: Text(
                    score.toStringAsFixed(0),
                    textAlign: TextAlign.right,
                    style: const TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                        color: Colors.white),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 3),
          ],
        ],
      ),
    );
  }
}
