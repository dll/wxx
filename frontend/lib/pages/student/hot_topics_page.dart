import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

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
      context.read<StudentFeatureProvider>().askAI(ApiConfig.hotTopics);
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('热点关注')),
      body: RefreshIndicator(
        onRefresh: () => provider.askAI(ApiConfig.hotTopics),
        child: provider.aiLoading
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
                        Text('AI 实时聚合校园热点话题', style: TextStyle(color: Colors.white.withValues(alpha: 0.8), fontSize: 13)),
                      ])),
                    ]),
                  ),
                  const SizedBox(height: 16),
                  if (provider.aiResponse.isNotEmpty)
                    Card(child: Padding(padding: const EdgeInsets.all(16), child: SelectableText(provider.aiResponse, style: theme.textTheme.bodyMedium)))
                  else
                    ..._buildTopics(theme),
                ],
              ),
      ),
    );
  }

  List<Widget> _buildTopics(ThemeData theme) {
    final topics = [
      {'title': '期中考试安排', 'heat': '95', 'trend': 'rising'},
      {'title': '暑期实习招聘', 'heat': '82', 'trend': 'rising'},
      {'title': '校园网升级', 'heat': '68', 'trend': 'stable'},
      {'title': '社团招新', 'heat': '55', 'trend': 'falling'},
    ];
    return topics.map((t) {
      final icon = t['trend'] == 'rising' ? Icons.trending_up : t['trend'] == 'falling' ? Icons.trending_down : Icons.trending_flat;
      final color = t['trend'] == 'rising' ? Colors.red : t['trend'] == 'falling' ? Colors.green : Colors.grey;
      return Card(
        margin: const EdgeInsets.only(bottom: 8),
        child: ListTile(
          leading: CircleAvatar(backgroundColor: color.withValues(alpha: 0.1), child: Icon(icon, color: color, size: 20)),
          title: Text(t['title']!),
          trailing: Text('热度 ${t['heat']}', style: TextStyle(color: color, fontWeight: FontWeight.bold)),
        ),
      );
    }).toList();
  }
}
