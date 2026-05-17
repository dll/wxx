import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';
import '../../widgets/audio_player.dart';

/// 校歌曲库 — 校歌、院歌、经典曲目
class AnthemPage extends StatefulWidget {
  const AnthemPage({super.key});

  @override
  State<AnthemPage> createState() => _AnthemPageState();
}

class _AnthemPageState extends State<AnthemPage> {
  int? _expandedIndex;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CultureProvider>().fetchAnthems();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = context.watch<CultureProvider>();
    final items = (p.anthems?['items'] as List?) ?? const [];
    return Scaffold(
      appBar: AppBar(title: const Text('校歌曲库')),
      body: p.loading && items.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: items.length,
              itemBuilder: (_, i) => _AnthemCard(
                data: items[i] as Map<String, dynamic>,
                theme: theme,
                isExpanded: _expandedIndex == i,
                onTap: () => setState(() {
                  _expandedIndex = _expandedIndex == i ? null : i;
                }),
              ),
            ),
    );
  }
}

class _AnthemCard extends StatelessWidget {
  final Map<String, dynamic> data;
  final ThemeData theme;
  final bool isExpanded;
  final VoidCallback onTap;

  const _AnthemCard({
    required this.data,
    required this.theme,
    required this.isExpanded,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final cat = data['category'] as String? ?? '';
    final catLabel = cat == 'school' ? '校歌' : (cat == 'college' ? '院歌' : '经典');
    final title = data['title'] as String? ?? '';
    final lyric = data['lyric'] as String? ?? '';
    final audioUrl = data['audio_url'] as String? ?? '';

    final catColors = {
      'school': [const Color(0xFF6750A4), const Color(0xFF4F378B)],
      'college': [const Color(0xFF1565C0), const Color(0xFF0D47A1)],
      'classic': [const Color(0xFF2E7D32), const Color(0xFF1B5E20)],
    };
    final gradientColors = catColors[cat] ?? catColors['classic']!;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    width: 64, height: 64,
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: gradientColors,
                        begin: Alignment.topLeft, end: Alignment.bottomRight,
                      ),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: const Icon(Icons.music_note, color: Colors.white, size: 32),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(children: [
                          Expanded(
                            child: Text(title,
                                style: theme.textTheme.titleMedium?.copyWith(
                                    fontWeight: FontWeight.w600)),
                          ),
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.primaryContainer,
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(catLabel,
                                style: theme.textTheme.labelSmall),
                          ),
                        ]),
                        const SizedBox(height: 4),
                        Text(
                          '时长 ${data['duration']}s · 更新 ${data['updated_at']}',
                          style: theme.textTheme.labelSmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          lyric,
                          style: theme.textTheme.bodySmall,
                          maxLines: isExpanded ? null : 2,
                          overflow:
                              isExpanded ? null : TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                ],
              ),

              // 展开后显示播放器 + 歌词滚动
              if (isExpanded) ...[
                const SizedBox(height: 12),
                AudioPlayerWidget(
                  audioUrl: audioUrl,
                  title: title,
                  subtitle: catLabel,
                  isLive: false,
                  onError: () {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text('音频加载失败：$title')),
                    );
                  },
                ),
                if (lyric.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  LyricsScrollingWidget(
                    lyrics: lyric,
                    isPlaying: true,
                  ),
                ],
              ],
            ],
          ),
        ),
      ),
    );
  }
}
