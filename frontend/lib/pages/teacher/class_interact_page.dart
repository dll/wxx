import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/teacher_feature_provider.dart';
import '../../widgets/md_text.dart';

/// 教师 - AI 课堂互动
class ClassInteractPage extends StatefulWidget {
  const ClassInteractPage({super.key});
  @override
  State<ClassInteractPage> createState() => _ClassInteractPageState();
}

class _ClassInteractPageState extends State<ClassInteractPage> {
  final _topicCtrl = TextEditingController();
  String _type = 'quiz';

  @override
  void dispose() {
    _topicCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<TeacherFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 课堂互动')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('互动设置', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(controller: _topicCtrl, decoration: const InputDecoration(labelText: '互动主题', border: OutlineInputBorder())),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: _type,
                  decoration: const InputDecoration(labelText: '互动类型', border: OutlineInputBorder()),
                  items: const [
                    DropdownMenuItem(value: 'quiz', child: Text('随堂测验')),
                    DropdownMenuItem(value: 'discussion', child: Text('讨论话题')),
                    DropdownMenuItem(value: 'brainstorm', child: Text('头脑风暴')),
                  ],
                  onChanged: (v) => setState(() => _type = v ?? 'quiz'),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: provider.loading ? null : () {
                      provider.startInteraction({'topic': _topicCtrl.text, 'type': _type});
                    },
                    icon: const Icon(Icons.play_circle),
                    label: const Text('发起互动'),
                  ),
                ),
              ]),
            ),
          ),
          if (provider.loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (provider.interactionData != null && !provider.loading) ...[
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Icon(Icons.live_help, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('互动内容', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  ]),
                  const SizedBox(height: 12),
                  MdText(provider.interactionData!['content'] ?? '生成中...', style: theme.textTheme.bodyMedium),
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
