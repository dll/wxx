import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 - 谈心谈话记录
class TalkRecordPage extends StatefulWidget {
  const TalkRecordPage({super.key});
  @override
  State<TalkRecordPage> createState() => _TalkRecordPageState();
}

class _TalkRecordPageState extends State<TalkRecordPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<CounselorFeatureProvider>().fetchTalkRecords();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<CounselorFeatureProvider>();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('谈心谈话记录')),
      floatingActionButton: FloatingActionButton(
        onPressed: () => _showAddDialog(context, provider),
        child: const Icon(Icons.add),
      ),
      body: provider.loading
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty
              ? Center(child: Text(provider.error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(provider, theme),
    );
  }

  Widget _buildContent(CounselorFeatureProvider provider, ThemeData theme) {
    final records = provider.talkRecords;
    if (records.isEmpty) return const Center(child: Text('暂无谈话记录'));
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: records.length,
      itemBuilder: (context, index) {
        final r = records[index];
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: ListTile(
            leading: CircleAvatar(child: Text(r.studentName.isNotEmpty ? r.studentName[0] : '?')),
            title: Text(r.studentName),
            subtitle: Text(r.summary, maxLines: 2, overflow: TextOverflow.ellipsis),
            trailing: Text(r.date, style: theme.textTheme.bodySmall),
          ),
        );
      },
    );
  }

  void _showAddDialog(BuildContext context, CounselorFeatureProvider provider) {
    final nameCtrl = TextEditingController();
    final contentCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('新增谈话记录'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(controller: nameCtrl, decoration: const InputDecoration(labelText: '学生姓名')),
          const SizedBox(height: 12),
          TextField(controller: contentCtrl, decoration: const InputDecoration(labelText: '谈话内容'), maxLines: 3),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(onPressed: () {
            provider.saveTalkRecord({'student_name': nameCtrl.text, 'content': contentCtrl.text});
            Navigator.pop(ctx);
          }, child: const Text('保存')),
        ],
      ),
    );
  }
}
