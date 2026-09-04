import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/data_src_badge.dart';
import '../../utils/capability_utils.dart';
import '../../utils/storage.dart';

/// 毕业去向管理（教辅：录入+审核；学生：自报）
/// 对齐书记教育成果闭环的「登记入口」：学生自报待教辅审核，教辅录入后需审核才计入统计。
class OutcomeManagePage extends StatefulWidget {
  const OutcomeManagePage({super.key});

  @override
  State<OutcomeManagePage> createState() => _OutcomeManagePageState();
}

class _OutcomeManagePageState extends State<OutcomeManagePage> {
  final _formKey = GlobalKey<FormState>();
  final _studentIdCtrl = TextEditingController();
  final _studentNameCtrl = TextEditingController();
  final _collegeCtrl = TextEditingController();
  final _majorCtrl = TextEditingController();
  final _yearCtrl = TextEditingController();
  final _employerCtrl = TextEditingController();
  final _positionCtrl = TextEditingController();
  final _remarkCtrl = TextEditingController();
  String _outcomeType = 'employment';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<SecretaryProvider>();
      p.fetchOutcomeTypes();
      p.fetchOutcomes(status: 'pending');
      p.fetchPendingCount();
    });
  }

  bool get _canReview => CapabilityUtils.has(Capability.outcomeReview);
  // 学生（仅自报，无录入/审核权）：不显示学生ID/姓名/学院手动录入项
  bool get _isStudent => Storage.role == 'student';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('毕业去向登记')),
      body: Consumer<SecretaryProvider>(
        builder: (_, p, __) {
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              _buildForm(p),
              const SizedBox(height: 16),
              if (_canReview) ...[
                _buildPendingBar(p),
                const SizedBox(height: 16),
              ],
              _buildOutcomeList(p),
            ],
          );
        },
      ),
    );
  }

  Widget _buildForm(SecretaryProvider p) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(children: [
                Icon(Icons.assignment_add,
                    color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                const Text('登记毕业去向（真实数据，需审核）',
                    style:
                        TextStyle(fontSize: 15, fontWeight: FontWeight.bold)),
                const Spacer(),
                const DataSrcBadge(src: 'real'),
              ]),
              const SizedBox(height: 12),
              if (!_isStudent) ...[
                TextFormField(
                  controller: _studentIdCtrl,
                  decoration: const InputDecoration(
                      labelText: '学生ID', border: OutlineInputBorder()),
                  keyboardType: TextInputType.number,
                ),
                const SizedBox(height: 10),
                TextFormField(
                  controller: _studentNameCtrl,
                  decoration: const InputDecoration(
                      labelText: '学生姓名', border: OutlineInputBorder()),
                ),
                const SizedBox(height: 10),
                TextFormField(
                  controller: _collegeCtrl,
                  decoration: const InputDecoration(
                      labelText: '学院', border: OutlineInputBorder()),
                ),
                const SizedBox(height: 10),
              ],
              TextFormField(
                controller: _yearCtrl,
                decoration: const InputDecoration(
                    labelText: '毕业届别（如2026）', border: OutlineInputBorder()),
                keyboardType: TextInputType.number,
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                value: _outcomeType,
                decoration: const InputDecoration(
                    labelText: '去向类型', border: OutlineInputBorder()),
                items: (p.outcomeTypes.isEmpty
                        ? {'employment': '就业', 'postgrad': '国内升读研'}
                        : p.outcomeTypes)
                    .entries
                    .map((e) => DropdownMenuItem(
                        value: e.key, child: Text(e.value)))
                    .toList(),
                onChanged: (v) => setState(() => _outcomeType = v ?? 'employment'),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _employerCtrl,
                decoration: const InputDecoration(
                    labelText: '去向单位 / 升学院校',
                    hintText: '就业填单位、考研填院校',
                    border: OutlineInputBorder()),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _positionCtrl,
                decoration: const InputDecoration(
                    labelText: '岗位 / 专业方向', border: OutlineInputBorder()),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _remarkCtrl,
                decoration: const InputDecoration(
                    labelText: '备注', border: OutlineInputBorder()),
                maxLines: 2,
              ),
              const SizedBox(height: 14),
              SizedBox(
                width: double.infinity,
                child: FilledButton.icon(
                  icon: const Icon(Icons.send),
                  label: const Text('提交（待教辅审核）'),
                  onPressed: () => _submit(p),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _submit(SecretaryProvider p) async {
    if (!_formKey.currentState!.validate()) return;
    final msg = await p.submitOutcome(
      studentId: int.tryParse(_studentIdCtrl.text) ?? 0,
      studentName: _studentNameCtrl.text,
      college: _collegeCtrl.text,
      major: _majorCtrl.text,
      graduateYear: int.tryParse(_yearCtrl.text) ?? 0,
      outcomeType: _outcomeType,
      employerName: _employerCtrl.text,
      position: _positionCtrl.text,
      remark: _remarkCtrl.text,
    );
    if (mounted) {
      if (msg == null) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('提交成功，待审核')));
        _clearForm();
        p.fetchOutcomes(status: 'pending');
        p.fetchPendingCount();
      } else {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('提交失败：$msg')));
      }
    }
  }

  void _clearForm() {
    _studentIdCtrl.clear();
    _studentNameCtrl.clear();
    _collegeCtrl.clear();
    _majorCtrl.clear();
    _yearCtrl.clear();
    _employerCtrl.clear();
    _positionCtrl.clear();
    _remarkCtrl.clear();
  }

  Widget _buildPendingBar(SecretaryProvider p) {
    return Card(
      color: Colors.orange.shade50,
      child: ListTile(
        leading: Icon(Icons.pending_actions, color: Colors.orange.shade800),
        title: Text('待审核记录：${p.pendingCount} 条',
            style: const TextStyle(fontWeight: FontWeight.bold)),
        subtitle: const Text('审核通过后计入教育成果统计'),
        trailing: TextButton(
          onPressed: () => p.fetchOutcomes(status: 'pending'),
          child: const Text('刷新'),
        ),
      ),
    );
  }

  Widget _buildOutcomeList(SecretaryProvider p) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('去向记录',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            if (p.outcomesLoading)
              const Center(child: Padding(
                padding: EdgeInsets.all(12),
                child: CircularProgressIndicator(),
              ))
            else if (p.outcomes.isEmpty)
              const Padding(
                padding: EdgeInsets.all(12),
                child: Text('暂无记录', style: TextStyle(color: Colors.grey)),
              )
            else
              ...p.outcomes.map((o) => _OutcomeTile(
                    record: o,
                    canReview: _canReview,
                    onReview: (status) => _review(p, o.id, status),
                  )),
          ],
        ),
      ),
    );
  }

  Future<void> _review(SecretaryProvider p, int id, String status) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) {
        final note = TextEditingController();
        return AlertDialog(
          title: Text(status == 'approved' ? '通过审核' : '驳回'),
          content: TextField(
            controller: note,
            maxLines: 2,
            decoration: const InputDecoration(
                labelText: '审核意见（可选）', border: OutlineInputBorder()),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(ctx, false),
                child: const Text('取消')),
            FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: const Text('确认'),
            ),
          ],
        );
      },
    );
    if (ok != true || !mounted) return;
    const note = '';
    final msg = await p.reviewOutcome(id, status, note);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(msg == null ? '已审核' : '审核失败：$msg')));
      p.fetchOutcomes(status: 'pending');
      p.fetchPendingCount();
    }
  }
}

class _OutcomeTile extends StatelessWidget {
  final dynamic record;
  final bool canReview;
  final void Function(String) onReview;
  const _OutcomeTile(
      {required this.record, required this.canReview, required this.onReview});

  @override
  Widget build(BuildContext context) {
    final statusColor = record.status == 'approved'
        ? Colors.green
        : record.status == 'rejected'
            ? Colors.red
            : Colors.orange;
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: statusColor.withOpacity(0.15),
          child: Icon(Icons.person, color: statusColor),
        ),
        title: Text('${record.studentName}',
            style: const TextStyle(fontWeight: FontWeight.bold)),
        subtitle: Text(
          '${record.outcomeType} · ${record.employerName}\n'
          '${record.college} · ${record.graduateYear}届',
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: canReview && record.status == 'pending'
            ? Row(mainAxisSize: MainAxisSize.min, children: [
                IconButton(
                    icon: const Icon(Icons.check_circle,
                        color: Colors.green),
                    onPressed: () => onReview('approved')),
                IconButton(
                    icon: const Icon(Icons.cancel, color: Colors.red),
                    onPressed: () => onReview('rejected')),
              ])
            : Text(record.status == 'approved'
                ? '已通过'
                : record.status == 'rejected'
                    ? '已驳回'
                    : '待审'),
      ),
    );
  }
}
