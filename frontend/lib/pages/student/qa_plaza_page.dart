import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class QAPlazaPage extends StatefulWidget {
  const QAPlazaPage({super.key});
  @override
  State<QAPlazaPage> createState() => _QAPlazaPageState();
}

class _QAPlazaPageState extends State<QAPlazaPage> {
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      final api = context.read<StudentFeatureProvider>();
      final res = await api.askAI(ApiConfig.qaPlaza);
      setState(() => _loading = false);
    } catch (e) {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('问答广场')),
      floatingActionButton: FloatingActionButton(
        onPressed: () {},
        child: const Icon(Icons.add),
      ),
      body: RefreshIndicator(
        onRefresh: () => provider.askAI(ApiConfig.qaPlaza),
        child: provider.aiLoading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [theme.colorScheme.primary, theme.colorScheme.primary.withValues(alpha: 0.7)],
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(children: [
                      Icon(Icons.forum, color: theme.colorScheme.onPrimary, size: 32),
                      const SizedBox(width: 16),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('问答广场', style: TextStyle(color: theme.colorScheme.onPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
                        const SizedBox(height: 4),
                        Text('AI 增强的校园问答社区', style: TextStyle(color: theme.colorScheme.onPrimary.withValues(alpha: 0.8), fontSize: 13)),
                      ])),
                    ]),
                  ),
                  const SizedBox(height: 16),
                  if (provider.aiResponse.isNotEmpty)
                    Card(child: Padding(padding: const EdgeInsets.all(16), child: SelectableText(provider.aiResponse, style: theme.textTheme.bodyMedium)))
                  else
                    ..._buildDemoContent(theme),
                ],
              ),
      ),
    );
  }

  List<Widget> _buildDemoContent(ThemeData theme) {
    final questions = [
      {'title': '转专业需要什么条件？', 'answers': '5', 'views': '128', 'tags': '政策'},
      {'title': '图书馆自习室怎么预约？', 'answers': '3', 'views': '89', 'tags': '生活'},
      {'title': 'ACM竞赛如何入门？', 'answers': '8', 'views': '256', 'tags': '竞赛'},
    ];
    return questions.map((q) => Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        title: Text(q['title']!),
        subtitle: Text('${q['answers']} 回答 · ${q['views']} 浏览'),
        trailing: Chip(label: Text(q['tags']!, style: const TextStyle(fontSize: 11))),
      ),
    )).toList();
  }
}
