import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class ProcessEditPage extends StatefulWidget {
  const ProcessEditPage({super.key});
  @override
  State<ProcessEditPage> createState() => _ProcessEditPageState();
}

class _ProcessEditPageState extends State<ProcessEditPage> {
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      await context.read<StudentFeatureProvider>().askAI(ApiConfig.counselorProcessEdit);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('流程编辑')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () {},
        icon: const Icon(Icons.add),
        label: const Text('新建流程'),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _processTile(theme, '缓考申请', 3, true, '2026-05-01'),
                _processTile(theme, '请假审批', 4, true, '2026-04-20'),
                _processTile(theme, '转专业申请', 5, true, '2026-03-15'),
                _processTile(theme, '奖学金评定', 6, false, '草稿'),
              ],
            ),
    );
  }

  Widget _processTile(ThemeData theme, String name, int steps, bool published, String date) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: published ? Colors.green.withOpacity( 0.1) : Colors.grey.withOpacity( 0.1),
          child: Icon(Icons.account_tree, color: published ? Colors.green : Colors.grey),
        ),
        title: Text(name),
        subtitle: Text('$steps 个步骤 · $date'),
        trailing: Row(mainAxisSize: MainAxisSize.min, children: [
          if (published)
            const Chip(label: Text('已发布'))
          else
            Chip(label: const Text('草稿'), backgroundColor: Colors.orange.withOpacity( 0.2)),
          const SizedBox(width: 4),
          const Icon(Icons.chevron_right),
        ]),
        onTap: () {},
      ),
    );
  }
}
