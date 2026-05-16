import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/culture_provider.dart';

/// 校园活动 — 报名 + 推送
class EventsPage extends StatefulWidget {
  const EventsPage({super.key});

  @override
  State<EventsPage> createState() => _EventsPageState();
}

class _EventsPageState extends State<EventsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CultureProvider>().fetchEvents();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = context.watch<CultureProvider>();
    final data = p.events;
    final upcoming = ((data?['upcoming'] as List?) ?? const []).cast<Map<String, dynamic>>();
    return Scaffold(
      appBar: AppBar(title: const Text('校园活动')),
      body: p.loading && data == null
          ? const Center(child: CircularProgressIndicator())
          : ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: upcoming.length,
              itemBuilder: (_, i) {
                final it = upcoming[i];
                final cap = it['capacity'] as int? ?? 1;
                final reg = it['registered'] as int? ?? 0;
                final progress = cap > 0 ? (reg / cap).clamp(0.0, 1.0) : 0.0;
                final tags = (it['tags'] as List?) ?? const [];
                return Card(
                  margin: const EdgeInsets.only(bottom: 12),
                  child: Padding(
                    padding: const EdgeInsets.all(14),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(children: [
                          _CategoryChip(category: it['category'] as String? ?? '', theme: theme),
                          const Spacer(),
                          Text('已报名 $reg / $cap',
                              style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                        ]),
                        const SizedBox(height: 8),
                        Text(it['title'] as String? ?? '',
                            style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600)),
                        const SizedBox(height: 6),
                        Text('${it['start_at']} → ${it['end_at']}', style: theme.textTheme.bodySmall),
                        Text('地点：${it['venue']} · 主办：${it['organizer']}', style: theme.textTheme.bodySmall),
                        const SizedBox(height: 10),
                        ClipRRect(
                          borderRadius: BorderRadius.circular(8),
                          child: LinearProgressIndicator(value: progress, minHeight: 6),
                        ),
                        const SizedBox(height: 10),
                        Wrap(spacing: 6, children: [
                          for (final t in tags)
                            Chip(label: Text(t.toString(), style: const TextStyle(fontSize: 11)), visualDensity: VisualDensity.compact),
                        ]),
                        const SizedBox(height: 8),
                        Align(
                          alignment: Alignment.centerRight,
                          child: FilledButton.icon(
                            icon: const Icon(Icons.app_registration, size: 18),
                            label: const Text('立即报名'),
                            onPressed: () {
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text('已发起报名：${it['title']}（提交流程下一阶段对接）')),
                              );
                            },
                          ),
                        ),
                      ],
                    ),
                  ),
                );
              },
            ),
    );
  }
}

class _CategoryChip extends StatelessWidget {
  final String category;
  final ThemeData theme;
  const _CategoryChip({required this.category, required this.theme});

  static const _colorMap = {
    '学术': Color(0xFF1976D2),
    '志愿': Color(0xFF388E3C),
    '文化': Color(0xFFD32F2F),
    '体育': Color(0xFFF57C00),
  };

  @override
  Widget build(BuildContext context) {
    final color = _colorMap[category] ?? theme.colorScheme.primary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(category, style: theme.textTheme.labelSmall?.copyWith(color: color, fontWeight: FontWeight.w600)),
    );
  }
}
