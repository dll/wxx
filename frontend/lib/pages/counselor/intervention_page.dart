import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - 干预方案生成
class InterventionPage extends StatefulWidget {
  const InterventionPage({super.key});
  @override
  State<InterventionPage> createState() => _InterventionPageState();
}

class _InterventionPageState extends State<InterventionPage> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 干预方案')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('输入学生信息', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(
                  controller: _controller,
                  decoration: const InputDecoration(labelText: '学生学号', border: OutlineInputBorder(), hintText: '请输入需要生成干预方案的学生学号'),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: provider.loading ? null : () {
                      if (_controller.text.trim().isNotEmpty) {
                        provider.generateIntervention(_controller.text.trim());
                      }
                    },
                    icon: const Icon(Icons.auto_fix_high),
                    label: const Text('生成干预方案'),
                  ),
                ),
              ]),
            ),
          ),
          if (provider.loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (provider.intervention != null && !provider.loading) ...[
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Icon(Icons.lightbulb, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('干预方案', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  ]),
                  const SizedBox(height: 12),
                  Text(provider.intervention!['plan'] ?? provider.intervention!['content'] ?? '方案生成中...', style: theme.textTheme.bodyMedium),
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
