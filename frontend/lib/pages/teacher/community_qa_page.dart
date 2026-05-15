import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class CommunityQAPage extends StatefulWidget {
  const CommunityQAPage({super.key});
  @override
  State<CommunityQAPage> createState() => _CommunityQAPageState();
}

class _CommunityQAPageState extends State<CommunityQAPage> with SingleTickerProviderStateMixin {
  late TabController _tabController;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _load();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      await context.read<StudentFeatureProvider>().askAI(ApiConfig.teacherCommunityQA);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('社区问答'),
        bottom: TabBar(controller: _tabController, tabs: const [
          Tab(text: '待回答'),
          Tab(text: '我的回答'),
        ]),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(controller: _tabController, children: [
              _pendingList(theme),
              _myAnswersList(theme),
            ]),
    );
  }

  Widget _pendingList(ThemeData theme) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _questionCard(theme, '数据结构期末重点有哪些？', '编程新手', '2小时前', ['学业', '考试']),
        _questionCard(theme, '操作系统进程调度算法比较？', '求知者', '5小时前', ['学业']),
        _questionCard(theme, '毕业设计选题建议？', '大三学生', '1天前', ['学业', '毕设']),
      ],
    );
  }

  Widget _myAnswersList(ThemeData theme) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _answerCard(theme, '如何学好数据结构？', 12, 3, true),
        _answerCard(theme, 'C++和Java选哪个？', 8, 1, true),
        _answerCard(theme, '考研数学怎么准备？', 5, 0, false),
      ],
    );
  }

  Widget _questionCard(ThemeData theme, String title, String author, String time, List<String> tags) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          Row(children: [
            Text('$author · $time', style: theme.textTheme.bodySmall),
            const Spacer(),
            ...tags.map((t) => Padding(
              padding: const EdgeInsets.only(left: 4),
              child: Chip(label: Text(t, style: const TextStyle(fontSize: 11)), padding: EdgeInsets.zero, visualDensity: VisualDensity.compact),
            )),
          ]),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton.tonal(onPressed: () {}, child: const Text('回答')),
          ),
        ]),
      ),
    );
  }

  Widget _answerCard(ThemeData theme, String question, int likes, int comments, bool adopted) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        title: Text(question),
        subtitle: Row(children: [
          Icon(Icons.thumb_up, size: 14, color: theme.colorScheme.primary),
          const SizedBox(width: 4),
          Text('$likes'),
          const SizedBox(width: 12),
          const Icon(Icons.comment, size: 14),
          const SizedBox(width: 4),
          Text('$comments'),
        ]),
        trailing: adopted
            ? const Chip(label: Text('已采纳'), backgroundColor: Color(0x1A4CAF50))
            : const Chip(label: Text('待采纳')),
      ),
    );
  }
}
