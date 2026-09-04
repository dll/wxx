import 'package:flutter/material.dart';
import 'dart:async';
import 'dart:math';

/// 通用音频播放器组件
/// 支持播放/暂停、进度条、音量控制，用于校歌曲库和校园广播
class AudioPlayerWidget extends StatefulWidget {
  final String audioUrl;
  final String title;
  final String? subtitle;
  final String? coverUrl;
  final bool isLive; // true = 直播模式（无进度条），false = 点播模式
  final VoidCallback? onError;

  const AudioPlayerWidget({
    super.key,
    required this.audioUrl,
    required this.title,
    this.subtitle,
    this.coverUrl,
    this.isLive = false,
    this.onError,
  });

  @override
  State<AudioPlayerWidget> createState() => _AudioPlayerWidgetState();
}

class _AudioPlayerWidgetState extends State<AudioPlayerWidget>
    with SingleTickerProviderStateMixin {
  bool _isPlaying = false;
  bool _isLoading = false;
  double _volume = 1.0;
  double _progress = 0.0;

  // 模拟播放进度（实际项目中应使用真实音频播放器如 just_audio）
  Timer? _progressTimer;
  late AnimationController _pulseAnim;

  @override
  void initState() {
    super.initState();
    _pulseAnim = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
  }

  @override
  void dispose() {
    _progressTimer?.cancel();
    _pulseAnim.dispose();
    super.dispose();
  }

  void _togglePlay() {
    setState(() {
      if (_isPlaying) {
        _isPlaying = false;
        _isLoading = false;
        _progressTimer?.cancel();
        _pulseAnim.stop();
      } else {
        _isLoading = true;
        // 模拟加载延迟
        Future.delayed(const Duration(milliseconds: 600), () {
          if (!mounted) return;
          setState(() {
            _isLoading = false;
            _isPlaying = true;
          });
          _pulseAnim.repeat(reverse: true);
          _startProgressSimulation();
        });
      }
    });
  }

  void _startProgressSimulation() {
    _progressTimer?.cancel();
    _progressTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (!mounted || !_isPlaying) {
        timer.cancel();
        return;
      }
      setState(() {
        _progress += 0.005; // 模拟 ~200s 的音频
        if (_progress >= 1.0) {
          _progress = 0.0;
          _isPlaying = false;
          _pulseAnim.stop();
          timer.cancel();
        }
      });
    });
  }

  String _formatDuration(double progress) {
    // 假设总时长 200 秒
    const totalSecs = 200;
    final currentSecs = (progress * totalSecs).round();
    final mins = currentSecs ~/ 60;
    final secs = currentSecs % 60;
    return '${mins.toString().padLeft(2, '0')}:${secs.toString().padLeft(2, '0')}';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
          color: _isPlaying
              ? theme.colorScheme.primary.withOpacity( 0.3)
              : theme.colorScheme.outlineVariant.withOpacity( 0.3),
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // 标题栏
            Row(
              children: [
                // 封面/图标
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(12),
                    gradient: LinearGradient(
                      colors: widget.isLive
                          ? [Colors.red.shade400, Colors.red.shade700]
                          : [theme.colorScheme.primary, theme.colorScheme.tertiary],
                    ),
                  ),
                  child: AnimatedBuilder(
                    animation: _pulseAnim,
                    builder: (_, child) {
                      return Icon(
                        widget.isLive
                            ? Icons.radio
                            : Icons.music_note,
                        color: Colors.white,
                        size: 24 + (_isPlaying ? _pulseAnim.value * 4 : 0),
                      );
                    },
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        widget.title,
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      if (widget.subtitle != null)
                        Text(
                          widget.subtitle!,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      if (widget.isLive)
                        Row(
                          children: [
                            Container(
                              width: 8,
                              height: 8,
                              decoration: const BoxDecoration(
                                color: Colors.red,
                                shape: BoxShape.circle,
                              ),
                            ),
                            const SizedBox(width: 4),
                            Text(
                              '直播中',
                              style: theme.textTheme.labelSmall?.copyWith(
                                color: Colors.red,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ],
                        ),
                    ],
                  ),
                ),
                // 音量控制
                PopupMenuButton<double>(
                  icon: Icon(
                    _volume == 0
                        ? Icons.volume_off
                        : _volume < 0.5
                            ? Icons.volume_down
                            : Icons.volume_up,
                    size: 20,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  itemBuilder: (_) => [
                    const PopupMenuItem(value: 0.0, child: Text('静音')),
                    const PopupMenuItem(value: 0.3, child: Text('30%')),
                    const PopupMenuItem(value: 0.6, child: Text('60%')),
                    const PopupMenuItem(value: 1.0, child: Text('100%')),
                  ],
                  onSelected: (v) => setState(() => _volume = v),
                ),
              ],
            ),

            // 进度条（非直播模式）
            if (!widget.isLive) ...[
              const SizedBox(height: 12),
              Row(
                children: [
                  Text(
                    _formatDuration(_progress),
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.outline,
                    ),
                  ),
                  Expanded(
                    child: SliderTheme(
                      data: SliderThemeData(
                        trackHeight: 3,
                        thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 6),
                        activeTrackColor: theme.colorScheme.primary,
                        inactiveTrackColor: theme.colorScheme.surfaceContainerHighest,
                        thumbColor: theme.colorScheme.primary,
                      ),
                      child: Slider(
                        value: _progress,
                        onChanged: (v) => setState(() => _progress = v),
                      ),
                    ),
                  ),
                  Text(
                    '03:20',
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.outline,
                    ),
                  ),
                ],
              ),
            ],

            // 播放控制按钮
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  icon: const Icon(Icons.replay_10, size: 22),
                  onPressed: () {
                    setState(() {
                      _progress = max(0, _progress - 0.05);
                    });
                  },
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 16),
                Container(
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    gradient: widget.isLive
                        ? LinearGradient(colors: [Colors.red.shade400, Colors.red.shade600])
                        : LinearGradient(colors: [
                            theme.colorScheme.primary,
                            theme.colorScheme.primary.withOpacity( 0.8),
                          ]),
                  ),
                  child: IconButton(
                    onPressed: _togglePlay,
                    icon: _isLoading
                        ? const SizedBox(
                            width: 24,
                            height: 24,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : Icon(
                            _isPlaying ? Icons.pause : Icons.play_arrow,
                            size: 32,
                            color: Colors.white,
                          ),
                  ),
                ),
                const SizedBox(width: 16),
                IconButton(
                  icon: const Icon(Icons.forward_10, size: 22),
                  onPressed: () {
                    setState(() {
                      _progress = min(1.0, _progress + 0.05);
                    });
                  },
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 歌词逐字滚动组件（用于校歌曲库）
class LyricsScrollingWidget extends StatefulWidget {
  final String lyrics;
  final bool isPlaying;

  const LyricsScrollingWidget({
    super.key,
    required this.lyrics,
    this.isPlaying = false,
  });

  @override
  State<LyricsScrollingWidget> createState() => _LyricsScrollingWidgetState();
}

class _LyricsScrollingWidgetState extends State<LyricsScrollingWidget> {
  final ScrollController _scrollCtrl = ScrollController();
  int _currentLine = 0;
  Timer? _scrollTimer;

  List<String> get _lines =>
      widget.lyrics.split('\n').where((l) => l.trim().isNotEmpty).toList();

  @override
  void didUpdateWidget(LyricsScrollingWidget old) {
    super.didUpdateWidget(old);
    if (widget.isPlaying && !old.isPlaying) {
      _startAutoScroll();
    } else if (!widget.isPlaying && old.isPlaying) {
      _scrollTimer?.cancel();
    }
  }

  void _startAutoScroll() {
    _scrollTimer?.cancel();
    _scrollTimer = Timer.periodic(const Duration(seconds: 4), (timer) {
      if (!mounted || !widget.isPlaying) {
        timer.cancel();
        return;
      }
      setState(() {
        _currentLine = (_currentLine + 1) % _lines.length;
        if (_scrollCtrl.hasClients) {
          final offset = _currentLine * 32.0;
          _scrollCtrl.animateTo(
            offset.clamp(0, _scrollCtrl.position.maxScrollExtent),
            duration: const Duration(milliseconds: 500),
            curve: Curves.easeInOut,
          );
        }
      });
    });
  }

  @override
  void dispose() {
    _scrollTimer?.cancel();
    _scrollCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final lines = _lines;

    return Container(
      height: 160,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        color: theme.colorScheme.surfaceContainerHighest.withOpacity( 0.3),
      ),
      child: ListView.builder(
        controller: _scrollCtrl,
        padding: const EdgeInsets.symmetric(vertical: 12),
        itemCount: lines.length,
        itemBuilder: (_, i) {
          final isCurrent = i == _currentLine && widget.isPlaying;
          return Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
            child: Text(
              lines[i],
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: isCurrent ? 16 : 14,
                fontWeight: isCurrent ? FontWeight.bold : FontWeight.normal,
                color: isCurrent
                    ? theme.colorScheme.primary
                    : theme.colorScheme.onSurfaceVariant,
              ),
            ),
          );
        },
      ),
    );
  }
}
