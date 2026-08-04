import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../models/models.dart';
import '../../providers/process_provider.dart';

class ProcessEditorPage extends StatefulWidget {
  final ProcessDefinition? definition;
  const ProcessEditorPage({super.key, this.definition});

  @override
  State<ProcessEditorPage> createState() => _ProcessEditorPageState();
}

class _ProcessEditorPageState extends State<ProcessEditorPage> {
  late final bool _isEdit = widget.definition != null;
  late final TextEditingController _resourceId;
  late final TextEditingController _title;
  late final TextEditingController _summary;
  late final TextEditingController _content;
  late final TextEditingController _sourceLink;
  late final TextEditingController _sourceVersion;
  late final TextEditingController _effectiveAt;
  late final TextEditingController _expiredAt;
  late final TextEditingController _tags;
  late final TextEditingController _ownerId;
  late String _ownerScope;
  late final Set<String> _roleScope;
  late final List<_StepDraft> _steps;
  late final List<_ReminderDraft> _reminders;

  static const _roles = [
    'student',
    'student_union',
    'counselor',
    'teacher',
    'assistant',
    'college_admin',
    'school_admin',
    'sys_admin',
  ];

  @override
  void initState() {
    super.initState();
    final d = widget.definition;
    _resourceId = TextEditingController(text: d?.resourceId ?? '');
    _title = TextEditingController(text: d?.title ?? '');
    _summary = TextEditingController(text: d?.summary ?? '');
    _content = TextEditingController(text: d?.content ?? '');
    _sourceLink = TextEditingController(text: d?.sourceLink ?? '');
    _sourceVersion = TextEditingController(text: d?.sourceVersion ?? '');
    _effectiveAt = TextEditingController(text: d?.effectiveAt ?? '');
    _expiredAt = TextEditingController(text: d?.expiredAt ?? '');
    _tags = TextEditingController(text: d?.tags.join(', ') ?? '');
    _ownerId = TextEditingController(text: d?.ownerId ?? '');
    _ownerScope = d?.ownerScope ?? 'school';
    _roleScope = {};
    final roles = _decodeRoles(d?.roleScope ?? '');
    for (final r in roles) {
      _roleScope.add(r);
    }
    _steps = (d?.steps ?? [])
        .map((s) => _StepDraft(
              stepOrder: s.stepOrder,
              title: s.title,
              materials: s.materialsList.join(', '),
              entryUrl: s.entryUrl,
              deadline: s.deadline,
              location: s.location,
              notes: s.notes,
              contact: s.contact,
              phone: s.phone,
              contactWechat: s.contactWechat,
              officeHours: s.officeHours,
              faq: s.faqList.map((f) => '${f.q} | ${f.a}').join('\n'),
            ))
        .toList();
    _reminders = (d?.reminders ?? [])
        .map((r) => _ReminderDraft(
              stepOrder: r.stepOrder.toString(),
              remindAt: r.remindAt,
              title: r.title,
              content: r.content,
              isEnabled: r.isEnabled,
            ))
        .toList();
  }

  @override
  void dispose() {
    _resourceId.dispose();
    _title.dispose();
    _summary.dispose();
    _content.dispose();
    _sourceLink.dispose();
    _sourceVersion.dispose();
    _effectiveAt.dispose();
    _expiredAt.dispose();
    _tags.dispose();
    _ownerId.dispose();
    for (final s in _steps) {
      s.dispose();
    }
    for (final r in _reminders) {
      r.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit ? '编辑办事流程' : '新建办事流程'),
        actions: [
          TextButton.icon(
            onPressed: _saving ? null : _save,
            icon: const Icon(Icons.save_outlined),
            label: const Text('保存'),
          ),
        ],
      ),
      body: Form(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            _sectionTitle(theme, '基础信息', Icons.description_outlined),
            TextFormField(
              controller: _resourceId,
              enabled: !_isEdit,
              decoration: const InputDecoration(
                labelText: '资源ID（留空自动生成）',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: DropdownButtonFormField<String>(
                    value: _ownerScope,
                    decoration: const InputDecoration(
                      labelText: '归属范围',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    items: const [
                      DropdownMenuItem(value: 'school', child: Text('学校')),
                      DropdownMenuItem(value: 'college', child: Text('学院')),
                      DropdownMenuItem(value: 'class', child: Text('班级')),
                    ],
                    onChanged: (v) =>
                        setState(() => _ownerScope = v ?? 'school'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextFormField(
                    controller: _ownerId,
                    decoration: const InputDecoration(
                      labelText: '归属ID',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _title,
              decoration: const InputDecoration(
                labelText: '流程标题 *',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _summary,
              maxLines: 2,
              decoration: const InputDecoration(
                labelText: '摘要',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _content,
              maxLines: 6,
              decoration: const InputDecoration(
                labelText: '办理说明 *',
                border: OutlineInputBorder(),
                alignLabelWithHint: true,
              ),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _tags,
              decoration: const InputDecoration(
                labelText: '标签（逗号分隔）',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 12,
              children: [
                SizedBox(
                  width: 220,
                  child: TextFormField(
                    controller: _sourceLink,
                    decoration: const InputDecoration(
                      labelText: '来源链接',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                ),
                SizedBox(
                  width: 180,
                  child: TextFormField(
                    controller: _effectiveAt,
                    decoration: const InputDecoration(
                      labelText: '生效时间',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                ),
                SizedBox(
                  width: 180,
                  child: TextFormField(
                    controller: _expiredAt,
                    decoration: const InputDecoration(
                      labelText: '失效时间',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: _roles.map((role) {
                return FilterChip(
                  label: Text(_roleLabel(role)),
                  selected: _roleScope.contains(role),
                  onSelected: (v) {
                    setState(() {
                      if (v) {
                        _roleScope.add(role);
                      } else {
                        _roleScope.remove(role);
                      }
                    });
                  },
                );
              }).toList(),
            ),
            const SizedBox(height: 24),
            Row(
              children: [
                Expanded(
                  child:
                      _sectionTitle(theme, '办理步骤', Icons.account_tree_outlined),
                ),
                IconButton.filledTonal(
                  tooltip: '添加步骤',
                  onPressed: () => setState(() => _steps.add(_StepDraft())),
                  icon: const Icon(Icons.add),
                ),
              ],
            ),
            ..._steps.asMap().entries.map((entry) {
              final index = entry.key;
              return Card(
                margin: const EdgeInsets.only(bottom: 8),
                child: Padding(
                  padding: const EdgeInsets.all(10),
                  child: _StepEditor(
                    draft: entry.value,
                    onRemove: () => setState(() => _steps.removeAt(index)),
                  ),
                ),
              );
            }),
            const SizedBox(height: 24),
            Row(
              children: [
                Expanded(
                  child: _sectionTitle(theme, '提醒节点', Icons.alarm),
                ),
                IconButton.filledTonal(
                  tooltip: '添加提醒',
                  onPressed: () =>
                      setState(() => _reminders.add(_ReminderDraft())),
                  icon: const Icon(Icons.add),
                ),
              ],
            ),
            ..._reminders.asMap().entries.map((entry) {
              final index = entry.key;
              return Card(
                margin: const EdgeInsets.only(bottom: 8),
                child: Padding(
                  padding: const EdgeInsets.all(10),
                  child: _ReminderEditor(
                    draft: entry.value,
                    onRemove: () => setState(() => _reminders.removeAt(index)),
                  ),
                ),
              );
            }),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }

  bool _saving = false;

  Future<void> _save() async {
    if (_title.text.trim().isEmpty || _content.text.trim().isEmpty) {
      _toast('标题和办理说明不能为空');
      return;
    }
    setState(() => _saving = true);
    final steps = _steps.map((s) {
      final materials = s.materials.text
          .split(',')
          .map((e) => e.trim())
          .where((e) => e.isNotEmpty)
          .toList();
      final faq = <Map<String, String>>[];
      for (final line in s.faq.text.split('\n')) {
        final parts = line.split('|').map((e) => e.trim()).toList();
        if (parts.length >= 2 && parts[0].isNotEmpty) {
          faq.add({'q': parts[0], 'a': parts.sublist(1).join('|').trim()});
        }
      }
      return {
        'step_order': s.stepOrder,
        'title': s.title.text.trim(),
        'materials': jsonEncode(materials),
        'entry_url': s.entryUrl.text.trim(),
        'deadline': s.deadline.text.trim(),
        'location': s.location.text.trim(),
        'notes': s.notes.text.trim(),
        'contact': s.contact.text.trim(),
        'phone': s.phone.text.trim(),
        'contact_wechat': s.contactWechat.text.trim(),
        'office_hours': s.officeHours.text.trim(),
        'media_urls': '[]',
        'faq': jsonEncode(faq),
      };
    }).toList();
    final reminders = _reminders.map((r) {
      return {
        'step_order': int.tryParse(r.stepOrder.text.trim()) ?? 0,
        'remind_at': r.remindAt.text.trim(),
        'title': r.title.text.trim(),
        'content': r.content.text.trim(),
        'is_enabled': r.isEnabled,
      };
    }).toList();
    final roleScope =
        _roleScope.isEmpty ? const <String>[] : _roleScope.toList();
    final payload = <String, dynamic>{
      'resource_id': _resourceId.text.trim(),
      'owner_scope': _ownerScope,
      'owner_id': _ownerId.text.trim(),
      'role_scope': roleScope,
      'title': _title.text.trim(),
      'summary': _summary.text.trim(),
      'content': _content.text.trim(),
      'source_link': _sourceLink.text.trim(),
      'source_version': _sourceVersion.text.trim(),
      'effective_at':
          _effectiveAt.text.trim().isEmpty ? null : _effectiveAt.text.trim(),
      'expired_at':
          _expiredAt.text.trim().isEmpty ? null : _expiredAt.text.trim(),
      'tags': _tags.text
          .split(',')
          .map((e) => e.trim())
          .where((e) => e.isNotEmpty)
          .toList(),
      'steps': steps,
      'reminders': reminders,
    };
    final provider = context.read<ProcessProvider>();
    final ok = _isEdit
        ? await provider.updateProcess(widget.definition!.resourceId, payload)
        : await provider.createProcess(payload);
    if (!mounted) return;
    setState(() => _saving = false);
    if (ok) {
      _toast(_isEdit ? '流程已更新' : '流程已创建');
      Navigator.of(context).pop(true);
    } else {
      _toast(provider.error ?? '保存失败');
    }
  }

  void _toast(String message) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(message)));
  }

  Widget _sectionTitle(ThemeData theme, String title, IconData icon) {
    return Row(
      children: [
        Icon(icon, size: 18, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        Text(title,
            style: theme.textTheme.titleMedium
                ?.copyWith(fontWeight: FontWeight.w600)),
      ],
    );
  }

  static List<String> _decodeRoles(String raw) {
    if (raw.isEmpty || raw == '[]') return const [];
    try {
      final decoded = jsonDecode(raw);
      if (decoded is List) return decoded.map((e) => e.toString()).toList();
    } catch (_) {}
    return const [];
  }

  static String _roleLabel(String role) {
    const map = {
      'student': '学生',
      'student_union': '学生会',
      'counselor': '辅导员',
      'teacher': '教师',
      'assistant': '教辅',
      'college_admin': '学院管理员',
      'school_admin': '学校管理员',
      'sys_admin': '系统管理员',
    };
    return map[role] ?? role;
  }
}

class _StepDraft {
  int stepOrder;
  final TextEditingController stepOrderController;
  final TextEditingController title;
  final TextEditingController materials;
  final TextEditingController entryUrl;
  final TextEditingController deadline;
  final TextEditingController location;
  final TextEditingController notes;
  final TextEditingController contact;
  final TextEditingController phone;
  final TextEditingController contactWechat;
  final TextEditingController officeHours;
  final TextEditingController faq;

  _StepDraft({
    int? stepOrder,
    String title = '',
    String materials = '',
    String entryUrl = '',
    String deadline = '',
    String location = '',
    String notes = '',
    String contact = '',
    String phone = '',
    String contactWechat = '',
    String officeHours = '',
    String faq = '',
  })  : stepOrder = stepOrder ?? 0,
        stepOrderController = TextEditingController(text: '${stepOrder ?? 0}'),
        title = TextEditingController(text: title),
        materials = TextEditingController(text: materials),
        entryUrl = TextEditingController(text: entryUrl),
        deadline = TextEditingController(text: deadline),
        location = TextEditingController(text: location),
        notes = TextEditingController(text: notes),
        contact = TextEditingController(text: contact),
        phone = TextEditingController(text: phone),
        contactWechat = TextEditingController(text: contactWechat),
        officeHours = TextEditingController(text: officeHours),
        faq = TextEditingController(text: faq);

  void dispose() {
    stepOrderController.dispose();
    title.dispose();
    materials.dispose();
    entryUrl.dispose();
    deadline.dispose();
    location.dispose();
    notes.dispose();
    contact.dispose();
    phone.dispose();
    contactWechat.dispose();
    officeHours.dispose();
    faq.dispose();
  }
}

class _StepEditor extends StatelessWidget {
  final _StepDraft draft;
  final VoidCallback onRemove;

  const _StepEditor({required this.draft, required this.onRemove});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            SizedBox(
              width: 70,
              child: TextField(
                controller: draft.stepOrderController,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: '序号',
                  isDense: true,
                  border: OutlineInputBorder(),
                ),
                onChanged: (v) =>
                    draft.stepOrder = int.tryParse(v) ?? draft.stepOrder,
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: TextField(
                controller: draft.title,
                decoration: const InputDecoration(
                  labelText: '步骤标题 *',
                  isDense: true,
                  border: OutlineInputBorder(),
                ),
              ),
            ),
            IconButton(
              tooltip: '删除步骤',
              onPressed: onRemove,
              icon: const Icon(Icons.delete_outline),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            _field(draft.materials, '材料（逗号分隔）', 220),
            _field(draft.deadline, '截止时间', 180),
            _field(draft.location, '办理地点', 220),
            _field(draft.contact, '联系人', 160),
            _field(draft.phone, '电话', 160),
            _field(draft.entryUrl, '办理入口', 240),
            _field(draft.officeHours, '办公时间', 220),
          ],
        ),
        const SizedBox(height: 8),
        TextField(
          controller: draft.notes,
          maxLines: 2,
          decoration: const InputDecoration(
            labelText: '说明',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: draft.faq,
          maxLines: 3,
          decoration: const InputDecoration(
            labelText: 'FAQ（每行：问题 | 答案）',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
      ],
    );
  }

  Widget _field(TextEditingController c, String label, double width) {
    return SizedBox(
      width: width,
      child: TextField(
        controller: c,
        decoration: InputDecoration(
          labelText: label,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
      ),
    );
  }
}

class _ReminderDraft {
  final TextEditingController stepOrder;
  final TextEditingController remindAt;
  final TextEditingController title;
  final TextEditingController content;
  bool isEnabled;

  _ReminderDraft({
    String stepOrder = '',
    String remindAt = '',
    String title = '',
    String content = '',
    this.isEnabled = true,
  })  : stepOrder = TextEditingController(text: stepOrder),
        remindAt = TextEditingController(text: remindAt),
        title = TextEditingController(text: title),
        content = TextEditingController(text: content);

  void dispose() {
    stepOrder.dispose();
    remindAt.dispose();
    title.dispose();
    content.dispose();
  }
}

class _ReminderEditor extends StatefulWidget {
  final _ReminderDraft draft;
  final VoidCallback onRemove;

  const _ReminderEditor({required this.draft, required this.onRemove});

  @override
  State<_ReminderEditor> createState() => _ReminderEditorState();
}

class _ReminderEditorState extends State<_ReminderEditor> {
  @override
  Widget build(BuildContext context) {
    final draft = widget.draft;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Switch(
          value: draft.isEnabled,
          onChanged: (v) => setState(() => draft.isEnabled = v),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              SizedBox(
                width: 110,
                child: TextField(
                  controller: draft.stepOrder,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    labelText: '步骤',
                    isDense: true,
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
              SizedBox(
                width: 180,
                child: TextField(
                  controller: draft.remindAt,
                  decoration: const InputDecoration(
                    labelText: '时间节点 *',
                    isDense: true,
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
              SizedBox(
                width: 200,
                child: TextField(
                  controller: draft.title,
                  decoration: const InputDecoration(
                    labelText: '提醒标题 *',
                    isDense: true,
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
              SizedBox(
                width: 260,
                child: TextField(
                  controller: draft.content,
                  decoration: const InputDecoration(
                    labelText: '提醒内容',
                    isDense: true,
                    border: OutlineInputBorder(),
                  ),
                ),
              ),
            ],
          ),
        ),
        IconButton(
          tooltip: '删除提醒',
          onPressed: widget.onRemove,
          icon: const Icon(Icons.delete_outline),
        ),
      ],
    );
  }
}
