import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';

class QALeaderboardPage extends StatefulWidget {
  const QALeaderboardPage({super.key});
  @override
  State<QALeaderboardPage> createState() => _QALeaderboardPageState();
}

class _QALeaderboardPageState extends State<QALeaderboardPage> with SingleTickerProviderStateMixin {
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchQALeaderboard();
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    final lb = provider.leaderboard;
    return Scaffold(
      appBar: AppBar(
        title: const Text('问答排行榜'),
        bottom: TabBar(controller: _tabController, tabs: const [
          Tab(text: '热门问题'),
          Tab(text: '活跃答主'),
          Tab(text: '知识贡献'),
        ]),
      ),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(controller: _tabController, children: [
              _buildRankList(
                theme,
                '热门问题',
                Icons.whatshot,
                Colors.orange,
                (lb?['hot_questions'] as List?)?.cast<Map<String, dynamic>>() ?? [],
                (e) => '${e['title'] ?? ''}',
                (e) => '${e['count'] ?? e['views'] ?? 0} 次提问',
              ),
              _buildRankList(
                theme,
                '活跃答主',
                Icons.person,
                Colors.blue,
                (lb?['top_answerers'] as List?)?.cast<Map<String, dynamic>>() ?? [],
                (e) => '${e['name'] ?? ''}',
                (e) => '${e['answers'] ?? 0} 回答 · ${e['adopted'] ?? 0} 采纳',
              ),
              _buildRankList(
                theme,
                '知识贡献',
                Icons.star,
                Colors.amber,
                (lb?['contributors'] as List?)?.cast<Map<String, dynamic>>() ?? [],
                (e) => '${e['name'] ?? ''}',
                (e) => '${e['contributions'] ?? 0} 贡献 · 质量 ${e['quality_score'] ?? '-'}',
              ),
            ]),
    );
  }

  Widget _buildRankList(ThemeData theme, String title, IconData icon, Color color,
      List<Map<String, dynamic>> items, String Function(Map<String, dynamic>) nameOf,
      String Function(Map<String, dynamic>) subOf) {
    if (items.isEmpty) {
      return Center(
        child: Text('暂无$title', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
      );
    }
    final medals = ['🥇', '🥈', '🥉'];
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: items.length,
      itemBuilder: (context, index) {
        final e = items[index];
        return Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: index < medals.length
                ? Text(medals[index], style: const TextStyle(fontSize: 24))
                : CircleAvatar(radius: 16, child: Text('${index + 1}', style: const TextStyle(fontSize: 13))),
            title: Text(nameOf(e), style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500)),
            subtitle: Text(subOf(e), style: theme.textTheme.bodySmall),
            trailing: Icon(icon, color: color),
          ),
        );
      },
    );
  }
}
