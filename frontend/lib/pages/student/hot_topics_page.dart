import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';

class HotTopicsPage extends StatefulWidget {
  const HotTopicsPage({super.key});
  @override
  State<HotTopicsPage> createState() => _HotTopicsPageState();
}

class _HotTopicsPageState extends State<HotTopicsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchHotTopics();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    final topics = provider.hotTopics;
    return Scaffold(
      appBar: AppBar(title: const Text('热点关注')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchHotTopics(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(colors: [Colors.orange.shade700, Colors.orange.shade400]),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(children: [
                      const Icon(Icons.local_fire_department, color: Colors.white, size: 32),
                      const SizedBox(width: 16),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        const Text('热点关注', style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold)),
                        const SizedBox(height: 4),
                        Text('AI 实时聚合校园热点话题', style: TextStyle(color: Colors.white.withOpacity(0.8), fontSize: 13)),
                      ])),
                    ]),
                  ),
                  const SizedBox(height: 16),
                  if (topics.isEmpty)
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Column(children: [
                          Icon(Icons.local_fire_department, size: 48, color: Colors.orange.withOpacity(0.4)),
                          const SizedBox(height: 12),
                          Text('暂无热点话题', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                        ]),
                      ),
                    )
                  else
                    ...topics.map((t) {
                      final trend = t['trend'] ?? 'stable';
                      final icon = trend == 'rising'
                          ? Icons.trending_up
                          : trend == 'falling'
                              ? Icons.trending_down
                              : Icons.trending_flat;
                      final color = trend == 'rising'
                          ? Colors.red
                          : trend == 'falling'
                              ? Colors.green
                              : Colors.grey;
                      return Card(
                        margin: const EdgeInsets.only(bottom: 8),
                        child: ExpansionTile(
                          leading: CircleAvatar(backgroundColor: color.withOpacity(0.1), child: Icon(icon, color: color, size: 20)),
                          title: Text(t['title'] ?? ''),
                          subtitle: (t['summary'] ?? '').toString().isNotEmpty
                              ? Text((t['summary'] ?? '').toString(), maxLines: 2, overflow: TextOverflow.ellipsis, style: theme.textTheme.bodySmall)
                              : null,
                          trailing: Text('热度 ${t['heat'] ?? 0}', style: TextStyle(color: color, fontWeight: FontWeight.bold)),
                        ),
                      );
                    }),
                ],
              ),
      ),
    );
  }
}
