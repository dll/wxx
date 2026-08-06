import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';
import '../../widgets/md_text.dart';

/// 教师 - AI 作业批改
class GradingPage extends StatefulWidget {
  const GradingPage({super.key});
  @override
  State<GradingPage> createState() => _GradingPageState();
}

class _GradingPageState extends State<GradingPage> {
  final _studentCtrl = TextEditingController();
  final _contentCtrl = TextEditingController();

  @override
  void dispose() {
    _studentCtrl.dispose();
    _contentCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 作业批改')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('作业信息', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(controller: _studentCtrl, decoration: const InputDecoration(labelText: '学生姓名/学号', border: OutlineInputBorder())),
                const SizedBox(height: 12),
                TextField(controller: _contentCtrl, decoration: const InputDecoration(labelText: '作业内容', border: OutlineInputBorder()), maxLines: 5),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: provider.loading ? null : () {
                      provider.submitGrading({'student': _studentCtrl.text, 'content': _contentCtrl.text});
                    },
                    icon: const Icon(Icons.grading),
                    label: const Text('AI 批改'),
                  ),
                ),
              ]),
            ),
          ),
          if (provider.loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (provider.gradingResult != null && !provider.loading) ...[
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Icon(Icons.rate_review, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('批改结果', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  ]),
                  const SizedBox(height: 12),
                  if (provider.gradingResult!['score'] != null)
                    Text('得分：${provider.gradingResult!['score']}', style: theme.textTheme.headlineSmall?.copyWith(color: theme.colorScheme.primary)),
                  const SizedBox(height: 8),
                  MdText(provider.gradingResult!['feedback'] ?? provider.gradingResult!['content'] ?? '', style: theme.textTheme.bodyMedium),
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
