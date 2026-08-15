import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/data_src_badge.dart';

/// 党课/活动登记（教师/教辅，蓝图第3块，2026-08-16）
///
/// 教师/教辅登记其组织的党课/积极分子活动：主题 / 类型 / 时长 / 日期 / 参与学生。
/// 数据落 party_study_records(created_by=登记人) → 书记党建看板立即可见。
class PartyActivityRegisterPage extends StatefulWidget {
  const PartyActivityRegisterPage({super.key});

  @override
  State<PartyActivityRegisterPage> createState() =>
      _PartyActivityRegisterPageState();
}

class _PartyActivityRegisterPageState extends State<PartyActivityRegisterPage> {
  final _form = GlobalKey<FormState>();
  final _titleCtrl = TextEditingController();
  final _contentCtrl = TextEditingController();
  final _durationCtrl = TextEditingController();
  final _dateCtrl = TextEditingController();
  String _studyType = 'theory';
  final List<int> _studentIds = [];
  final Set<String> _selectedTypes = <String>{};
  bool _submitting = false;
  List<dynamic>? _records;

  static const _typeNames = {
    'theory': '理论学习',
    'practice': '实践活动',
    'meeting': '组织生活',
    'volunteer': '志愿服务',
  };

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final r = await context.read<SecretaryProvider>().fetchMyPartyRecords();
    if (mounted) setState(() => _records = r);
  }

  Future<void> _submit() async {
    if (!(_form.currentState?.validate() ?? false)) return;
    if (_selectedTypes.isEmpty) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请选择学习/活动类型')));
      return;
    }
    setState(() => _submitting = true);
    final r = await context
        .read<SecretaryProvider>()
        .registerPartyRecord(
          title: _titleCtrl.text.trim(),
          studyType: _studyType,
          content: _contentCtrl.text.trim(),
          duration: int.tryParse(_durationCtrl.text.trim()) ?? 0,
          studyDate: _dateCtrl.text.trim(),
          studentIds: _studentIds.isEmpty ? null : _studentIds,
        );
    setState(() => _submitting = false);
    if (r != null && r['code'] == 0) {
      _titleCtrl.clear();
      _contentCtrl.clear();
      _durationCtrl.clear();
      _dateCtrl.clear();
      _studentIds.clear();
      _selectedTypes.clear();
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('登记成功（已计入书记党建看板）')));
      await _load();
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(context.read<SecretaryProvider>().error)));
    }
  }

  Future<void> _delete(int id) async {
    final r = await context.read<SecretaryProvider>().deletePartyRecord(id);
    if (r != null && r['code'] == 0) {
      await _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<SecretaryProvider>();
    return Scaffold(
      appBar: AppBar(title: const Text('党课/活动登记')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            // 登记表单
            Card(
              child: Padding(
                padding: const EdgeInsets.all(14),
                child: Form(
                  key: _form,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Row(
                        children: [
                          Icon(Icons.flag, color: Colors.red),
                          SizedBox(width: 8),
                          Text('登记党课 / 积极分子活动',
                              style: TextStyle(
                                  fontSize: 15, fontWeight: FontWeight.bold)),
                        ],
                      ),
                      const SizedBox(height: 10),
                      TextFormField(
                        controller: _titleCtrl,
                        decoration: const InputDecoration(
                            labelText: '活动/党课主题 *',
                            hintText: '如：二十大精神专题党课',
                            border: OutlineInputBorder()),
                        validator: (v) =>
                            (v == null || v.trim().isEmpty) ? '请输入主题' : null,
                      ),
                      const SizedBox(height: 10),
                      Wrap(
                        spacing: 6,
                        children: _typeNames.entries.map((e) {
                          final sel = _selectedTypes.contains(e.key);
                          return ChoiceChip(
                            label: Text(e.value),
                            selected: sel,
                            onSelected: (_) {
                              setState(() {
                                _selectedTypes.clear();
                                _selectedTypes.add(e.key);
                                _studyType = e.key;
                              });
                            },
                          );
                        }).toList(),
                      ),
                      const SizedBox(height: 10),
                      Row(
                        children: [
                          Expanded(
                            child: TextFormField(
                              controller: _durationCtrl,
                              keyboardType: TextInputType.number,
                              decoration: const InputDecoration(
                                  labelText: '时长(分钟)',
                                  border: OutlineInputBorder()),
                            ),
                          ),
                          const SizedBox(width: 10),
                          Expanded(
                            child: TextFormField(
                              controller: _dateCtrl,
                              decoration: const InputDecoration(
                                  labelText: '日期 * (YYYY-MM-DD)',
                                  hintText: '2026-08-16',
                                  border: OutlineInputBorder()),
                              validator: (v) =>
                                  (v == null || v.trim().isEmpty)
                                      ? '请输入日期'
                                      : null,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 10),
                      TextFormField(
                        controller: _contentCtrl,
                        maxLines: 3,
                        decoration: const InputDecoration(
                            labelText: '内容/参与学生说明（可填学号，逗号分隔）',
                            border: OutlineInputBorder()),
                      ),
                      const SizedBox(height: 14),
                      SizedBox(
                        width: double.infinity,
                        child: FilledButton.icon(
                          onPressed: _submitting ? null : _submit,
                          icon: const Icon(Icons.add),
                          label: Text(_submitting ? '提交中...' : '登记'),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 12),
            // 我的登记列表
            Card(
              child: Padding(
                padding: const EdgeInsets.all(14),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        const Icon(Icons.list_alt, color: Colors.indigo),
                        const SizedBox(width: 8),
                        const Expanded(
                          child: Text('我的登记记录',
                              style: TextStyle(
                                  fontSize: 15, fontWeight: FontWeight.bold)),
                        ),
                        DataSrcBadge(src: (provider.error.isEmpty && _records != null) ? 'real' : 'not_available'),
                      ],
                    ),
                    const SizedBox(height: 8),
                    if (_records == null && provider.error.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.all(8),
                        child: Text(provider.error,
                            style: const TextStyle(color: Colors.grey)),
                      )
                    else if (_records == null || _records!.isEmpty)
                      const Padding(
                        padding: EdgeInsets.all(8),
                        child: Text('暂无登记记录',
                            style: TextStyle(color: Colors.grey)),
                      )
                    else
                      ..._records!.map((e) => _RecordTile(
                            rec: e,
                            onDelete: () => _delete(
                                ((e as Map)['id'] as num?)?.toInt() ?? 0),
                          )),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _RecordTile extends StatelessWidget {
  final dynamic rec;
  final VoidCallback onDelete;
  const _RecordTile({required this.rec, required this.onDelete});

  static const _typeNames = {
    'theory': '理论学习',
    'practice': '实践活动',
    'meeting': '组织生活',
    'volunteer': '志愿服务',
  };

  @override
  Widget build(BuildContext context) {
    final m = rec as Map;
    final type = '${m['study_type'] ?? ''}';
    final title = '${m['title'] ?? ''}';
    final dur = (m['duration'] as num?)?.toInt() ?? 0;
    final date = '${m['study_date'] ?? ''}';
    final content = '${m['content'] ?? ''}';
    return ListTile(
      dense: true,
      contentPadding: EdgeInsets.zero,
      leading: const Icon(Icons.event_available, color: Colors.green),
      title: Text(title,
          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
      subtitle: Text(
        '${_typeNames[type] ?? type} · ${dur > 0 ? '$dur 分钟' : ''} · $date\n$content',
        maxLines: 3,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: IconButton(
        icon: const Icon(Icons.delete_outline, color: Colors.redAccent),
        onPressed: onDelete,
      ),
    );
  }
}
