import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';

/// 校园广播 — 节目单 + 当前直播
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
                    _NowPlayingCard(data: data['now_playing'] as Map<String, dynamic>?, theme: theme),
                    const SizedBox(height: 20),
                    Text('今日节目单', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    ...((data['schedule'] as List?) ?? []).map(
                      (s) => Card(
                        margin: const EdgeInsets.only(bottom: 6),
                        child: ListTile(
                          leading: Icon(Icons.schedule, color: theme.colorScheme.primary),
                          title: Text(s['title'] as String? ?? ''),
                          subtitle: Text('${s['time']} · ${s['category']}'),
                        ),
                      ),
                    ),
                    const SizedBox(height: 20),
                    Text('往期回放', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                    const SizedBox(height: 8),
                    ...((data['recent_episodes'] as List?) ?? []).map(
                      (e) => Card(
                        margin: const EdgeInsets.only(bottom: 6),
                        child: ListTile(
                          leading: const Icon(Icons.podcasts, color: Colors.deepPurple),
                          title: Text(e['title'] as String? ?? ''),
                          subtitle: Text('${e['published_at']} · ${e['duration']}秒'),
                          trailing: const Icon(Icons.play_circle_outline),
                          onTap: () {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(content: Text('播放：${e['title']}')),
                            );
                          },
                        ),
                      ),
                    ),
                  ],
                ),
    );
  }
}

class _NowPlayingCard extends StatelessWidget {
  final Map<String, dynamic>? data;
  final ThemeData theme;
  const _NowPlayingCard({required this.data, required this.theme});

  @override
  Widget build(BuildContext context) {
    if (data == null) return const SizedBox.shrink();
    final isLive = data!['is_live'] == true;
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFFD32F2F), Color(0xFFB71C1C)],
          begin: Alignment.topLeft, end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
        boxShadow: [BoxShadow(color: Colors.red.withValues(alpha: 0.3), blurRadius: 18, offset: const Offset(0, 6))],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            if (isLive)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(4)),
                child: Row(mainAxisSize: MainAxisSize.min, children: const [
                  Icon(Icons.circle, color: Colors.red, size: 8),
                  SizedBox(width: 4),
                  Text('LIVE', style: TextStyle(color: Colors.red, fontWeight: FontWeight.bold, fontSize: 11)),
                ]),
              ),
            const SizedBox(width: 8),
            Text(data!['host'] as String? ?? '', style: const TextStyle(color: Colors.white70, fontSize: 12)),
          ]),
          const SizedBox(height: 12),
          Text(data!['title'] as String? ?? '',
              style: const TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold)),
          const SizedBox(height: 6),
          Text('${data!['start_at']} → ${data!['end_at']}', style: const TextStyle(color: Colors.white70, fontSize: 12)),
          const SizedBox(height: 12),
          Row(children: [
            Expanded(
              child: ElevatedButton.icon(
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.white,
                  foregroundColor: Colors.red,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
                ),
                icon: const Icon(Icons.play_arrow),
                label: Text(isLive ? '收听直播' : '收听'),
                onPressed: () {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('音频流接入下一阶段实现')),
                  );
                },
              ),
            ),
          ]),
        ],
      ),
    );
  }
}
