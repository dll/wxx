import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';
import '../../widgets/audio_player.dart';

/// 校园广播 — 节目单 + 当前直播 + 往期回放
class RadioPage extends StatefulWidget {
  const RadioPage({super.key});

  @override
  State<RadioPage> createState() => _RadioPageState();
}

class _RadioPageState extends State<RadioPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CultureProvider>().fetchRadio();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = context.watch<CultureProvider>();
    final data = p.radio;
    return Scaffold(
      appBar: AppBar(title: const Text('校园广播')),
      body: p.loading && data == null
          ? const Center(child: CircularProgressIndicator())
          : data == null
              ? const Center(child: Text('暂无广播数据'))
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    _NowPlayingCard(
                      data: data['now_playing'] as Map<String, dynamic>?,
                      theme: theme,
                    ),
                    const SizedBox(height: 20),
                    Text('今日节目单',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    ...((data['schedule'] as List?) ?? []).map(
                      (s) => Card(
                        margin: const EdgeInsets.only(bottom: 6),
                        child: ListTile(
                          leading: Icon(Icons.schedule,
                              color: theme.colorScheme.primary),
                          title: Text(s['title'] as String? ?? ''),
                          subtitle:
                              Text('${s['time']} · ${s['category']}'),
                        ),
                      ),
                    ),
                    const SizedBox(height: 20),
                    Text('往期回放',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    ...((data['recent_episodes'] as List?) ?? []).whereType<Map>().map(
                      (e) => _EpisodeCard(
                        data: Map<String, dynamic>.from(e),
                        theme: theme,
                      ),
                    ),
                  ],
                ),
    );
  }
}

/// 往期节目卡片（含音频播放器）
class _EpisodeCard extends StatefulWidget {
  final Map<String, dynamic> data;
  final ThemeData theme;

  const _EpisodeCard({required this.data, required this.theme});

  @override
  State<_EpisodeCard> createState() => _EpisodeCardState();
}

class _EpisodeCardState extends State<_EpisodeCard> {
  bool _showPlayer = false;

  @override
  Widget build(BuildContext context) {
    final title = widget.data['title'] as String? ?? '';
    final audioUrl = widget.data['audio_url'] as String? ?? '';
    final publishedAt = widget.data['published_at'] as String? ?? '';
    final duration = widget.data['duration'];

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          ListTile(
            leading: const Icon(Icons.podcasts, color: Colors.deepPurple),
            title: Text(title),
            subtitle: Text('$publishedAt · ${duration}s'),
            trailing: IconButton(
              icon: Icon(
                  _showPlayer ? Icons.expand_less : Icons.play_circle_outline),
              onPressed: () =>
                  setState(() => _showPlayer = !_showPlayer),
            ),
          ),
          if (_showPlayer)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              child: AudioPlayerWidget(
                audioUrl: audioUrl,
                title: title,
                subtitle: publishedAt,
                isLive: false,
              ),
            ),
        ],
      ),
    );
  }
}

/// 当前播放卡片（含直播播放器）
class _NowPlayingCard extends StatefulWidget {
  final Map<String, dynamic>? data;
  final ThemeData theme;
  const _NowPlayingCard({required this.data, required this.theme});

  @override
  State<_NowPlayingCard> createState() => _NowPlayingCardState();
}

class _NowPlayingCardState extends State<_NowPlayingCard> {
  bool _showPlayer = false;

  @override
  Widget build(BuildContext context) {
    if (widget.data == null) return const SizedBox.shrink();
    final data = widget.data!;
    final isLive = data['is_live'] == true;
    final title = data['title'] as String? ?? '';
    final streamUrl = data['stream_url'] as String? ?? '';

    return Container(
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFFD32F2F), Color(0xFFB71C1C)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
        boxShadow: [
          BoxShadow(
            color: Colors.red.withOpacity( 0.3),
            blurRadius: 18,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  if (isLive)
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 3),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Row(mainAxisSize: MainAxisSize.min, children: const [
                        Icon(Icons.circle, color: Colors.red, size: 8),
                        SizedBox(width: 4),
                        Text('LIVE',
                            style: TextStyle(
                                color: Colors.red,
                                fontWeight: FontWeight.bold,
                                fontSize: 11)),
                      ]),
                    ),
                  const SizedBox(width: 8),
                  Text(data['host'] as String? ?? '',
                      style: const TextStyle(
                          color: Colors.white70, fontSize: 12)),
                ]),
                const SizedBox(height: 12),
                Text(title,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 22,
                        fontWeight: FontWeight.bold)),
                const SizedBox(height: 6),
                Text('${data['start_at']} → ${data['end_at']}',
                    style: const TextStyle(
                        color: Colors.white70, fontSize: 12)),
                const SizedBox(height: 12),
                Row(children: [
                  Expanded(
                    child: ElevatedButton.icon(
                      style: ElevatedButton.styleFrom(
                        backgroundColor: Colors.white,
                        foregroundColor: Colors.red,
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(24)),
                      ),
                      icon: Icon(_showPlayer
                          ? Icons.radio
                          : Icons.play_arrow),
                      label: Text(isLive
                          ? (_showPlayer ? '隐藏播放器' : '收听直播')
                          : (_showPlayer ? '隐藏播放器' : '收听')),
                      onPressed: () =>
                          setState(() => _showPlayer = !_showPlayer),
                    ),
                  ),
                ]),
              ],
            ),
          ),
          // 展开后显示音频播放器
          if (_showPlayer)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 16),
              child: Material(
                borderRadius: BorderRadius.circular(12),
                child: AudioPlayerWidget(
                  audioUrl: streamUrl,
                  title: title,
                  subtitle: data['host'] as String? ?? '',
                  isLive: isLive,
                ),
              ),
            ),
        ],
      ),
    );
  }
}
