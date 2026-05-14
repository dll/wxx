import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - 谈话话术推荐
class TalkTipsPage extends StatefulWidget {
  const TalkTipsPage({super.key});
  @override
  State<TalkTipsPage> createState() => _TalkTipsPageState();
}

class _TalkTipsPageState extends State<TalkTipsPage> {
  String _scene = '';
  String _studentType = '';

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('谈话话术推荐')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('场景选择', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  decoration: const InputDecoration(labelText: '谈话场景', border: OutlineInputBorder()),
                  items: const [
                    DropdownMenuItem(value: 'academic', child: Text('学业困难')),
                    DropdownMenuItem(value: 'mental', child: Text('心理疏导')),
                    DropdownMenuItem(value: 'career', child: Text('职业规划')),
                    DropdownMenuItem(value: 'discipline', child: Text('纪律问题')),
                  ],
                  onChanged: (v) => setState(() => _scene = v ?? ''),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  decoration: const InputDecoration(labelText: '学生类型', border: OutlineInputBorder()),
                  items: const [
                    DropdownMenuItem(value: 'introvert', child: Text('内向型')),
                    DropdownMenuItem(value: 'extrovert', child: Text('外向型')),
                    DropdownMenuItem(value: 'resistant', child: Text('抵触型')),
                  ],
                  onChanged: (v) => setState(() => _studentType = v ?? ''),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: provider.loading ? null : () => provider.fetchTalkTips(scene: _scene, studentType: _studentType),
                    icon: const Icon(Icons.tips_and_updates),
                    label: const Text('获取话术推荐'),
                  ),
                ),
              ]),
            ),
          ),
          if (provider.loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (provider.talkTips.isNotEmpty && !provider.loading) ...[
            const SizedBox(height: 16),
            Text('推荐话术', style: theme.textTheme.titleSmall),
            const SizedBox(height: 8),
            ...provider.talkTips.asMap().entries.map((e) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: CircleAvatar(radius: 14, child: Text('${e.key + 1}', style: const TextStyle(fontSize: 12))),
                title: Text(e.value, style: theme.textTheme.bodyMedium),
              ),
            )),
          ],
        ],
      ),
    );
  }
}
