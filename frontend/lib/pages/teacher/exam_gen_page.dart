import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';

/// 教师 - AI 考试出题
class ExamGenPage extends StatefulWidget {
  const ExamGenPage({super.key});
  @override
  State<ExamGenPage> createState() => _ExamGenPageState();
}

class _ExamGenPageState extends State<ExamGenPage> {
  final _subjectCtrl = TextEditingController();
  final _countCtrl = TextEditingController(text: '10');
  String _difficulty = 'medium';

  @override
  void dispose() {
    _subjectCtrl.dispose();
    _countCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 考试出题')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('出题参数', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(controller: _subjectCtrl, decoration: const InputDecoration(labelText: '考试科目/知识点', border: OutlineInputBorder())),
                const SizedBox(height: 12),
                TextField(controller: _countCtrl, decoration: const InputDecoration(labelText: '题目数量', border: OutlineInputBorder()), keyboardType: TextInputType.number),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: _difficulty,
                  decoration: const InputDecoration(labelText: '难度', border: OutlineInputBorder()),
                  items: const [
                    DropdownMenuItem(value: 'easy', child: Text('简单')),
                    DropdownMenuItem(value: 'medium', child: Text('中等')),
                    DropdownMenuItem(value: 'hard', child: Text('困难')),
                  ],
                  onChanged: (v) => setState(() => _difficulty = v ?? 'medium'),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: provider.loading ? null : () {
                      provider.generateExam({'subject': _subjectCtrl.text, 'count': int.tryParse(_countCtrl.text) ?? 10, 'difficulty': _difficulty});
                    },
                    icon: const Icon(Icons.quiz),
                    label: const Text('生成试卷'),
                  ),
                ),
              ]),
            ),
          ),
          if (provider.loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (provider.examPaper != null && !provider.loading) ...[
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Icon(Icons.description, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('生成结果', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  ]),
                  const SizedBox(height: 12),
                  Text(provider.examPaper!['content'] ?? '试卷内容加载中...', style: theme.textTheme.bodyMedium),
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
