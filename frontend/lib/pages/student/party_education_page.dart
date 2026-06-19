import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_new_features_provider.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 入党教育页面（数据驱动版）
class PartyEducationPage extends StatefulWidget {
  const PartyEducationPage({super.key});
  @override
  State<PartyEducationPage> createState() => _PartyEducationPageState();
}

class _PartyEducationPageState extends State<PartyEducationPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudentNewFeaturesProvider>();
      p.fetchPartyStages();
      p.fetchMyPartyProgress();
      p.fetchStudyRecords();
      p.fetchPartyStats();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('入党教育')),
      body: Consumer<StudentNewFeaturesProvider>(
        builder: (_, p, __) {
          if (p.loading) return const Center(child: CircularProgressIndicator());
          return RefreshIndicator(
            onRefresh: () async {
              await p.fetchMyPartyProgress();
              await p.fetchStudyRecords();
            },
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _buildProgressCard(context, p, theme),
                const SizedBox(height: 12),
                _buildStagesCard(context, p, theme),
                const SizedBox(height: 12),
                _buildStudyRecordsCard(context, p, theme),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _buildProgressCard(BuildContext context, StudentNewFeaturesProvider p, ThemeData theme) {
    final progress = p.myPartyProgress;
    final currentStage = progress?['current_stage'] ?? '未开始';
    final appliedAt = progress?['applied_at'] ?? '';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Icon(Icons.flag, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Text('我的入党进度', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            ]),
            const Divider(),
            _infoRow('当前阶段', _stageLabel(currentStage)),
            if (appliedAt.isNotEmpty) _infoRow('申请时间', appliedAt),
          ],
        ),
      ),
    );
  }

  Widget _buildStagesCard(BuildContext context, StudentNewFeaturesProvider p, ThemeData theme) {
    if (p.partyStages.isEmpty) return const SizedBox.shrink();
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('入党流程', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            const Divider(),
            ...p.partyStages.map((s) => ListTile(
              dense: true,
              leading: CircleAvatar(
                radius: 14,
                child: Text(s.sortOrder.toString(), style: const TextStyle(fontSize: 12)),
              ),
              title: Text(s.name, style: const TextStyle(fontSize: 14)),
              subtitle: Text(s.description, maxLines: 1, overflow: TextOverflow.ellipsis),
            )),
          ],
        ),
      ),
    );
  }

  Widget _buildStudyRecordsCard(BuildContext context, StudentNewFeaturesProvider p, ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Text('学习记录', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              const Spacer(),
              TextButton.icon(
                onPressed: () => _showAddStudyRecordDialog(context),
                icon: const Icon(Icons.add, size: 16),
                label: const Text('新增'),
              ),
            ]),
            const Divider(),
            if (p.studyRecords.isEmpty)
              const Center(child: Padding(padding: EdgeInsets.all(20), child: Text('暂无学习记录')))
            else
              ...p.studyRecords.map((r) => ListTile(
                dense: true,
                leading: Icon(Icons.book_outlined, color: theme.colorScheme.primary),
                title: Text(r['title']?.toString() ?? '', style: const TextStyle(fontSize: 14)),
                subtitle: Text(r['record_date']?.toString() ?? '', style: const TextStyle(fontSize: 12)),
              )),
          ],
        ),
      ),
    );
  }

  void _showAddStudyRecordDialog(BuildContext context) {
    final titleCtrl = TextEditingController();
    final contentCtrl = TextEditingController();
    final formKey = GlobalKey<FormState>();

    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('新增学习记录'),
        content: Form(
          key: formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: titleCtrl,
                decoration: const InputDecoration(labelText: '学习主题', border: OutlineInputBorder()),
                validator: (v) => v == null || v.isEmpty ? '请输入主题' : null,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: contentCtrl,
                maxLines: 3,
                decoration: const InputDecoration(labelText: '学习内容', border: OutlineInputBorder()),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              if (!formKey.currentState!.validate()) return;
              final api = ApiService();
              final res = await api.post(ApiConfig.partyStudyRecordAdd,
                data: {'title': titleCtrl.text, 'content': contentCtrl.text},
              );
              if (context.mounted) {
                Navigator.pop(context);
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(res.data['code'] == 0 ? '添加成功' : '添加失败')),
                );
                context.read<StudentNewFeaturesProvider>().fetchStudyRecords();
              }
            },
            child: const Text('添加'),
          ),
        ],
      ),
    );
  }

  Widget _infoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          SizedBox(width: 80, child: Text(label, style: const TextStyle(color: Colors.grey, fontSize: 13))),
          Expanded(child: Text(value, style: const TextStyle(fontSize: 13))),
        ],
      ),
    );
  }

  String _stageLabel(String code) {
    switch (code) {
      case 'applicant': return '入党申请人';
      case 'activist': return '入党积极分子';
      case 'development': return '发展对象';
      case 'probation': return '预备党员';
      case 'member': return '正式党员';
      default: return code;
    }
  }
}
