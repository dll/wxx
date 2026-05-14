import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';

/// 教师 - AI 备课助手
class LessonPrepPage extends StatefulWidget {
  const LessonPrepPage({super.key});
  @override
  State<LessonPrepPage> createState() => _LessonPrepPageState();
}

class _LessonPrepPageState extends State<LessonPrepPage> {
  final _topicCtrl = TextEditingController();
  final _courseIdCtrl = TextEditingController();

  @override
  void dispose() {
    _topicCtrl.dispose();
    _courseIdCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 备课助手')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('备课主题', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(controller: _topicCtrl, decoration: const InputDecoration(labelText: '授课主题', border: OutlineInputBorder(), hintText: '例如：数据结构-二叉树遍历')),
                const SizedBox(height: 12),
                TextField(controller: _courseIdCtrl, decoration: const InputDecoration(labelText: '课程编号（可选）', border: OutlineInputBorder())),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: provider.loading ? null : () {
                      if (_topicCtrl.text.trim().isNotEmpty) {
                        provider.generateLessonPlan(_topicCtrl.text.trim(), courseId: _courseIdCtrl.text.trim().isEmpty ? null : _courseIdCtrl.text.trim());
                      }
                    },
                    icon: const Icon(Icons.auto_awesome),
                    label: const Text('生成教案'),
                  ),
                ),
              ]),
            ),
          ),
          if (provider.loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (provider.lessonPlan != null && !provider.loading) ...[
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(provider.lessonPlan!.topic, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  Text('目标：${provider.lessonPlan!.outline}', style: theme.textTheme.bodyMedium),
                  const Divider(height: 24),
                  Text('重点难点', style: theme.textTheme.titleSmall),
                  const SizedBox(height: 8),
                  ...provider.lessonPlan!.keyPoints.map((k) => Padding(padding: const EdgeInsets.only(bottom: 4), child: Row(children: [const Icon(Icons.circle, size: 6), const SizedBox(width: 8), Expanded(child: Text(k))]))),
                  const Divider(height: 24),
                  Text('教学策略', style: theme.textTheme.titleSmall),
                  const SizedBox(height: 8),
                  ...provider.lessonPlan!.strategies.map((a) => Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Row(children: [const Icon(Icons.check_circle_outline, size: 16), const SizedBox(width: 8), Expanded(child: Text(a))]),
                  )),
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
