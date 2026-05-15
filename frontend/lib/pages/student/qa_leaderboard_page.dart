import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

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
      context.read<StudentFeatureProvider>().askAI(ApiConfig.qaLeaderboard);
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
    return Scaffold(
      appBar: AppBar(
        title: const Text('问答排行榜'),
        bottom: TabBar(controller: _tabController, tabs: const [
          Tab(text: '热门问题'),
          Tab(text: '活跃答主'),
          Tab(text: '知识贡献'),
        ]),
      ),
      body: provider.aiLoading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(controller: _tabController, children: [
              _buildRankList(theme, '热门问题', Icons.whatshot, Colors.orange),
              _buildRankList(theme, '活跃答主', Icons.person, Colors.blue),
              _buildRankList(theme, '知识贡献', Icons.star, Colors.amber),
            ]),
    );
  }

  Widget _buildRankList(ThemeData theme, String title, IconData icon, Color color) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: 3,
      itemBuilder: (context, index) {
        final medals = ['🥇', '🥈', '🥉'];
        return Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: Text(medals[index], style: const TextStyle(fontSize: 24)),
            title: Text('第${index + 1}名 · $title'),
            trailing: Icon(icon, color: color),
          ),
        );
      },
    );
  }
}
