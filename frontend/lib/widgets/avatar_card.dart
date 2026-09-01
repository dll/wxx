import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../models/avatar_config.dart';
import '../services/voice/voice_service.dart';
import 'avatar_painter.dart';

/// 数字人形象卡片 — 动画特效 + 磨砂背景 + 五维分数叠加层。
class AvatarCard extends StatefulWidget {
  final AvatarConfig config;
  final double height;

  const AvatarCard({super.key, required this.config, this.height = 280});

  @override
  State<AvatarCard> createState() => _AvatarCardState();
}

class _AvatarCardState extends State<AvatarCard>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  late final AnimationController _controller;
  late final VoiceService _voice;
  Timer? _cycleTimer;
  int _dimensionIndex = 0;
  bool _visible = true;
  bool _speaking = false;

  AvatarConfig get config => widget.config;

  @override
  void initState() {
    super.initState();
    // 动画：3 秒周期往返，驱动眨眼/漂浮/粒子/光环脉冲
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 3000),
    )..repeat();
    _voice = VoiceService();
    _startDimensionCycle();
    // 监听应用生命周期：页面不可见时暂停动画，减少持续重绘
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _stopDimensionCycle();
    _controller.dispose();
    _voice.dispose();
    super.dispose();
  }

  /// 启动维度轮播：每 4 秒聚焦下一个维度。
  /// 用可取消的 Timer 而非 while 循环，dispose 与切后台时都能确定性停止。
  void _startDimensionCycle() {
    _cycleTimer ??= Timer.periodic(const Duration(seconds: 4), (_) {
      setState(
          () => _dimensionIndex = (_dimensionIndex + 1) % _dimensions.length);
    });
  }

  void _stopDimensionCycle() {
    _cycleTimer?.cancel();
    _cycleTimer = null;
  }

  List<({String name, double score})> get _dimensions => [
        (name: '学业', score: config.academic),
        (name: '能力', score: config.ability),
        (name: '思想', score: config.ideological),
        (name: '情感', score: config.emotional),
        (name: '社交', score: config.social),
      ];

  Future<void> _speakCurrentDimension() async {
    if (_speaking) {
      _voice.stopPlayback();
      setState(() => _speaking = false);
      return;
    }
    final d = _dimensions[_dimensionIndex];
    setState(() => _speaking = true);
    final data = await _voice.textToSpeech(
        '蔚小芯成长画像播报：${d.name}维度，当前记录分数约${d.score.toStringAsFixed(0)}分。这个指标用于帮助你了解成长方向，不代表对你的定义。');
    if (data != null && data.isNotEmpty) await _voice.playAudio(data);
    if (mounted) setState(() => _speaking = false);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    final visible = state == AppLifecycleState.resumed;
    if (visible == _visible) return;
    _visible = visible;
    if (visible) {
      _controller.repeat();
      _startDimensionCycle();
    } else {
      _controller.stop();
      // 不可见时同时停掉维度轮播，避免后台无谓重建
      _stopDimensionCycle();
    }
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

            // 动画驱动的数字人（眨眼/漂浮/粒子/光环脉冲）
            AnimatedBuilder(
              animation: _controller,
              builder: (context, _) {
                final t = _controller.value;
                return CustomPaint(
                  painter: AvatarPainter(
                    config: widget.config,
                    primary: primary,
                    secondary: secondary,
                    t: t,
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

            // 底部五维循环指标 + 语音播报
            Positioned(
              left: 16,
              right: 16,
              bottom: 12,
              child: _buildDimensionBar(theme, _dimensionIndex),
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

  /// 底部指标循环展示：每 4 秒聚焦一个维度，同时保留五维总览。
  Widget _buildDimensionBar(ThemeData theme, int activeIndex) {
    final dims = _dimensions;
    final active = dims[activeIndex];
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.30),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(children: [
            const Icon(Icons.insights, size: 14, color: Colors.white70),
            const SizedBox(width: 5),
            Expanded(
                child: AnimatedSwitcher(
                    duration: const Duration(milliseconds: 450),
                    child: Text(
                        '${active.name} · ${active.score.toStringAsFixed(0)}分',
                        key: ValueKey(activeIndex),
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 12,
                            fontWeight: FontWeight.w700)))),
            IconButton(
                onPressed: _speakCurrentDimension,
                icon: Icon(
                    _speaking
                        ? Icons.stop_circle_outlined
                        : Icons.volume_up_outlined,
                    color: Colors.white,
                    size: 18),
                tooltip: _speaking ? '停止播报' : '播报当前维度',
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 28, minHeight: 28)),
          ]),
          const SizedBox(height: 5),
          ...dims.map((d) => Padding(
                padding: const EdgeInsets.only(bottom: 3),
                child: Row(
                  children: [
                    SizedBox(
                      width: 34,
                      child: Text(
                        d.name,
                        style: const TextStyle(
                            fontSize: 10, color: Colors.white70),
                      ),
                    ),
                    Expanded(
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(999),
                        child: LinearProgressIndicator(
                          value: (d.score / 100).clamp(0.0, 1.0),
                          minHeight: 5,
                          backgroundColor: Colors.white.withOpacity(0.15),
                          valueColor: AlwaysStoppedAnimation(
                            Color.lerp(
                              const Color(0xFF64B5F6),
                              const Color(0xFF42A5F5),
                              (d.score / 100).clamp(0.0, 1.0),
                            )!,
                          ),
                        ),
                      ),
                    ),
                    SizedBox(
                      width: 28,
                      child: Text(
                        d.score.toStringAsFixed(0),
                        textAlign: TextAlign.right,
                        style: const TextStyle(
                            fontSize: 10,
                            fontWeight: FontWeight.w600,
                            color: Colors.white),
                      ),
                    ),
                  ],
                ),
              )),
        ],
      ),
    );
  }
}
