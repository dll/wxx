import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';

/// 校歌曲库 — 校歌、院歌、经典曲目
class AnthemPage extends StatefulWidget {
  const AnthemPage({super.key});

  @override
  State<AnthemPage> createState() => _AnthemPageState();
}

class _AnthemPageState extends State<AnthemPage> {
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
              itemBuilder: (_, i) => _AnthemCard(data: items[i] as Map<String, dynamic>, theme: theme),
            ),
    );
  }
}

class _AnthemCard extends StatelessWidget {
  final Map<String, dynamic> data;
  final ThemeData theme;
  const _AnthemCard({required this.data, required this.theme});

  @override
  Widget build(BuildContext context) {
    final cat = data['category'] as String? ?? '';
    final catLabel = cat == 'school' ? '校歌' : (cat == 'college' ? '院歌' : '经典');
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 64, height: 64,
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [Color(0xFF6750A4), Color(0xFF4F378B)],
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
                    Text(data['title'] as String? ?? '',
                        style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                    const SizedBox(width: 8),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.primaryContainer,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(catLabel, style: theme.textTheme.labelSmall),
                    ),
                  ]),
                  const SizedBox(height: 4),
                  Text('时长 ${data['duration']}s · 更新 ${data['updated_at']}',
                      style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                  const SizedBox(height: 8),
                  Text(
                    data['lyric'] as String? ?? '',
                    style: theme.textTheme.bodySmall,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            IconButton.filled(
              icon: const Icon(Icons.play_arrow),
              onPressed: () {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('开始播放：${data['title']}（音频接入下一阶段）')),
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}
