import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';

/// 学术讲座 — 即将开始 + 回放
class LecturesPage extends StatefulWidget {
  const LecturesPage({super.key});

  @override
  State<LecturesPage> createState() => _LecturesPageState();
}

class _LecturesPageState extends State<LecturesPage> with SingleTickerProviderStateMixin {
  late TabController _tab;

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 2, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CultureProvider>().fetchLectures();
    });
  }

  @override
  void dispose() {
    _tab.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final p = context.watch<CultureProvider>();
    final data = p.lectures;
    final upcoming = (data?['upcoming'] as List?) ?? const [];
    final replay = (data?['replay'] as List?) ?? const [];
    return Scaffold(
      appBar: AppBar(
        title: const Text('学术讲座'),
        bottom: TabBar(controller: _tab, tabs: const [
          Tab(text: '即将开始'),
          Tab(text: '回放'),
        ]),
      ),
      body: p.loading && data == null
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(controller: _tab, children: [
              _buildList(context, upcoming.cast<Map<String, dynamic>>(), upcoming: true),
              _buildList(context, replay.cast<Map<String, dynamic>>(), upcoming: false),
            ]),
    );
  }

  Widget _buildList(BuildContext context, List<Map<String, dynamic>> items, {required bool upcoming}) {
    final theme = Theme.of(context);
    if (items.isEmpty) return const Center(child: Text('暂无讲座'));
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: items.length,
      itemBuilder: (_, i) {
        final it = items[i];
        final tags = (it['tags'] as List?) ?? const [];
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  Icon(upcoming ? Icons.event : Icons.replay_circle_filled,
                      color: upcoming ? theme.colorScheme.primary : Colors.deepOrange),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(it['title'] as String? ?? '',
                        style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                  ),
                ]),
                const SizedBox(height: 8),
                Text('讲者：${it['speaker']}', style: theme.textTheme.bodySmall),
                if (upcoming) ...[
                  Text('时间：${it['start_at']} · ${it['duration']} 分钟', style: theme.textTheme.bodySmall),
                  Text('地点：${it['venue']}', style: theme.textTheme.bodySmall),
                  Text('平台：${it['platform']}', style: theme.textTheme.bodySmall),
                ] else ...[
                  Text('回放时长：${it['duration']} 分钟 · 已观看 ${it['played']}', style: theme.textTheme.bodySmall),
                ],
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  children: [
                    for (final t in tags)
                      Chip(label: Text(t.toString(), style: const TextStyle(fontSize: 11)), visualDensity: VisualDensity.compact),
                  ],
                ),
                const SizedBox(height: 8),
                Align(
                  alignment: Alignment.centerRight,
                  child: FilledButton.tonalIcon(
                    icon: Icon(upcoming ? Icons.open_in_new : Icons.play_circle_outline, size: 18),
                    label: Text(upcoming ? '前往直播' : '观看回放'),
                    onPressed: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text('外链：${it['link']}（请复制到浏览器打开）')),
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}
