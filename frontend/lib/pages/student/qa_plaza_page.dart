import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';

class QAPlazaPage extends StatefulWidget {
  const QAPlazaPage({super.key});
  @override
  State<QAPlazaPage> createState() => _QAPlazaPageState();
}

class _QAPlazaPageState extends State<QAPlazaPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<StudentFeatureProvider>().fetchQAPlaza();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudentFeatureProvider>();
    final questions = provider.qaQuestions;
    return Scaffold(
      appBar: AppBar(title: const Text('问答广场')),
      body: RefreshIndicator(
        onRefresh: () => provider.fetchQAPlaza(),
        child: provider.loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [theme.colorScheme.primary, theme.colorScheme.primary.withOpacity(0.7)],
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(children: [
                      Icon(Icons.forum, color: theme.colorScheme.onPrimary, size: 32),
                      const SizedBox(width: 16),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('问答广场', style: TextStyle(color: theme.colorScheme.onPrimary, fontSize: 18, fontWeight: FontWeight.bold)),
                        const SizedBox(height: 4),
                        Text('AI 增强的校园问答社区', style: TextStyle(color: theme.colorScheme.onPrimary.withOpacity(0.8), fontSize: 13)),
                      ])),
                    ]),
                  ),
                  const SizedBox(height: 16),
                  if (questions.isEmpty)
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Column(children: [
                          Icon(Icons.forum_outlined, size: 48, color: theme.colorScheme.primary.withOpacity(0.4)),
                          const SizedBox(height: 12),
                          Text('暂无问答内容', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                        ]),
                      ),
                    )
                  else
                    ...questions.map((q) => Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      child: ExpansionTile(
                        title: Text(q['title'] ?? '', style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w500)),
                        subtitle: Padding(
                          padding: const EdgeInsets.only(top: 4),
                          child: Text(
                            '${q['author'] ?? '同学'} · ${q['answers'] ?? 0} 回答 · ${q['views'] ?? 0} 浏览',
                            style: theme.textTheme.bodySmall,
                          ),
                        ),
                        leading: CircleAvatar(
                          backgroundColor: theme.colorScheme.primaryContainer,
                          child: Icon(Icons.question_answer, color: theme.colorScheme.onPrimaryContainer, size: 20),
                        ),
                        children: [
                          if ((q['ai_answer'] ?? '').toString().isNotEmpty)
                            Padding(
                              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                              child: Container(
                                width: double.infinity,
                                padding: const EdgeInsets.all(12),
                                decoration: BoxDecoration(
                                  color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: Text('💡 AI 解答：${q['ai_answer']}',
                                    style: theme.textTheme.bodySmall),
                              ),
                            ),
                        ],
                      ),
                    )),
                ],
              ),
      ),
    );
  }
}
