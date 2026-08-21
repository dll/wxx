import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../providers/vopc_provider.dart';
import '../../utils/capability_utils.dart';

class VopcPage extends StatefulWidget {
  const VopcPage({super.key});
  @override
  State<VopcPage> createState() => _VopcPageState();
}

class _VopcPageState extends State<VopcPage> {
  @override
  void initState() {
    super.initState();
    Future.microtask(_start);
  }

  Future<void> _start() async {
    final p = context.read<VopcProvider>();
    await p.loadProjects();
    await p.loadInvitations();
  }

  @override
  Widget build(BuildContext context) {
    final p = context.watch<VopcProvider>();
    final canCreate = CapabilityUtils.has(Capability.vopcProjectCreate);
    return Scaffold(
        appBar: AppBar(title: const Text('vOPC 虚拟创业')),
        floatingActionButton: canCreate
            ? FloatingActionButton.extended(
                onPressed: _create,
                icon: const Icon(Icons.add),
                label: const Text('创建草稿'))
            : null,
        body: RefreshIndicator(
            onRefresh: _start,
            child: ListView(padding: const EdgeInsets.all(20), children: [
              Text('项目孵化工作台', style: Theme.of(context).textTheme.headlineSmall),
              const SizedBox(height: 8),
              const Text('从真实问题到可运行成果，由服务端门禁保障每一步。'),
              const SizedBox(height: 20),
              if (p.invitations.any((e) => e['status'] == 'pending')) ...[
                Text('待处理邀请', style: Theme.of(context).textTheme.titleLarge),
                ...p.invitations
                    .where((e) => e['status'] == 'pending')
                    .map((e) => Card(
                            child: ListTile(
                          leading: const Icon(Icons.mark_email_unread_outlined),
                          title: Text(e['project_name']?.toString() ?? '项目邀请'),
                          subtitle:
                              Text('${e['project_role']} · ${e['message']}'),
                          trailing: Wrap(children: [
                            TextButton(
                                onPressed: () => p.respondInvitation(
                                    (e['id'] as num).toInt(), 'decline'),
                                child: const Text('拒绝')),
                            FilledButton(
                                onPressed: () => p
                                        .respondInvitation(
                                            (e['id'] as num).toInt(), 'accept')
                                        .then((ok) {
                                      if (ok) p.loadProjects();
                                    }),
                                child: const Text('加入')),
                          ]),
                        ))),
                const SizedBox(height: 16),
              ],
              if (p.loading) const LinearProgressIndicator(),
              if (!p.loading && p.error != null)
                _ErrorCard(
                    message: p.error!, code: p.statusCode, onRetry: _start),
              if (!p.loading && p.error == null && p.projects.isEmpty)
                const Card(
                    child: Padding(
                        padding: EdgeInsets.all(28),
                        child: Column(children: [
                          Icon(Icons.rocket_launch_outlined, size: 48),
                          SizedBox(height: 12),
                          Text('还没有项目，从一个真实问题开始。')
                        ]))),
              ...p.projects.map((e) => Card(
                  child: ListTile(
                      leading: const Icon(Icons.rocket_launch_outlined),
                      title: Text(e.name),
                      subtitle:
                          Text('${e.stage} · ${e.status} · ${e.riskLevel}'),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () => context.push('/vopc/projects/${e.id}'))))
            ])));
  }

  Future<void> _create() async {
    final name = TextEditingController();
    final summary = TextEditingController();
    final ok = await showDialog<bool>(
        context: context,
        builder: (c) => AlertDialog(
                title: const Text('创建项目草稿'),
                content: SizedBox(
                    width: 420,
                    child: Column(mainAxisSize: MainAxisSize.min, children: [
                      TextField(
                          controller: name,
                          decoration:
                              const InputDecoration(labelText: '项目名称 *')),
                      TextField(
                          controller: summary,
                          decoration: const InputDecoration(labelText: '项目摘要'))
                    ])),
                actions: [
                  TextButton(
                      onPressed: () => Navigator.pop(c, false),
                      child: const Text('取消')),
                  FilledButton(
                      onPressed: () => Navigator.pop(c, true),
                      child: const Text('创建'))
                ]));
    if (ok == true && mounted) {
      final id = await context.read<VopcProvider>().createProject({
        'name': name.text.trim(),
        'summary': summary.text.trim(),
        'project_type': '自由探索项目',
        'project_source': 'self_proposed',
        'risk_level': 'R0',
        'data_type': '公开数据'
      });
      if (id != null && mounted) context.push('/vopc/projects/$id');
    }
    name.dispose();
    summary.dispose();
  }
}

class VopcProjectPage extends StatefulWidget {
  final int projectId;
  const VopcProjectPage({super.key, required this.projectId});
  @override
  State<VopcProjectPage> createState() => _VopcProjectPageState();
}

class _VopcProjectPageState extends State<VopcProjectPage> {
  @override
  void initState() {
    super.initState();
    Future.microtask(
        () => context.read<VopcProvider>().loadDetail(widget.projectId));
  }

  @override
  Widget build(BuildContext context) {
    final p = context.watch<VopcProvider>();
    final canManage = CapabilityUtils.has(Capability.vopcProjectManage);
    return Scaffold(
        appBar: AppBar(title: const Text('项目工作台')),
        floatingActionButton:
            canManage && p.detail != null && _canCreateTask(p.detail!)
                ? FloatingActionButton.extended(
                    onPressed: p.taskMutating ? null : _createTask,
                    icon: const Icon(Icons.add_task_outlined),
                    label: const Text('新建任务'))
                : null,
        body: p.loading
            ? const Center(child: CircularProgressIndicator())
            : p.error != null && p.detail == null
                ? Center(
                    child: _ErrorCard(
                        message: p.error!,
                        code: p.statusCode,
                        onRetry: () => p.loadDetail(widget.projectId)))
                : p.detail == null
                    ? const Center(child: Text('暂无项目数据'))
                    : ListView(padding: const EdgeInsets.all(20), children: [
                        Text(p.detail!.name,
                            style: Theme.of(context).textTheme.headlineSmall),
                        const SizedBox(height: 8),
                        Text(p.detail!.summary.isEmpty
                            ? '尚未填写摘要'
                            : p.detail!.summary),
                        const SizedBox(height: 20),
                        Wrap(spacing: 8, children: [
                          Chip(label: Text(p.detail!.stage)),
                          Chip(label: Text(p.detail!.status)),
                          Chip(label: Text(p.detail!.projectType)),
                          Chip(label: Text(p.detail!.riskLevel))
                        ]),
                        const SizedBox(height: 24),
                        _buildDecisions(context, p, canManage),
                        const SizedBox(height: 24),
                        _buildMembers(context, p, canManage),
                        const SizedBox(height: 24),
                        _buildArtifacts(context, p, canManage),
                        const SizedBox(height: 24),
                        _buildMilestones(context, p, canManage),
                        const SizedBox(height: 24),
                        _buildTasks(context, p, canManage),
                      ]));
  }

  bool _canCreateTask(VopcProject project) {
    final stage = int.tryParse(project.stage.replaceFirst('S', '')) ?? -1;
    const blocked = {'paused', 'risk_frozen', 'terminated', 'archived'};
    return stage >= 1 && stage < 9 && !blocked.contains(project.status);
  }

  Widget _buildDecisions(
      BuildContext context, VopcProvider provider, bool canManage) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text('决策中心 · ${provider.decisions.length}',
            style: Theme.of(context).textTheme.titleLarge),
        const Spacer(),
        if (canManage)
          FilledButton.tonalIcon(
              onPressed: provider.decisionMutating ? null : _createDecision,
              icon: const Icon(Icons.add),
              label: const Text('创建决策'))
      ]),
      const SizedBox(height: 8),
      ...provider.decisions.map((d) => Card(
          child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Expanded(
                          child: Text(d.title,
                              style: Theme.of(context).textTheme.titleMedium)),
                      _MetaChip(d.status)
                    ]),
                    if (d.background.isNotEmpty) Text(d.background),
                    if (d.options.isNotEmpty) Text('选项：${d.options}'),
                    if (d.decision.isNotEmpty) Text('决定：${d.decision}'),
                    if (d.rationale.isNotEmpty) Text('理由：${d.rationale}'),
                    if (canManage && d.status == 'pending')
                      Wrap(spacing: 8, children: [
                        OutlinedButton(
                            onPressed: () => _actDecision(d, 'resolve'),
                            child: const Text('形成决议')),
                        TextButton(
                            onPressed: () => _actDecision(d, 'cancel'),
                            child: const Text('取消'))
                      ])
                  ]))))
    ]);
  }

  Widget _buildMembers(BuildContext context, VopcProvider p, bool canManage) =>
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Text('真人成员 · ${p.members.length}',
              style: Theme.of(context).textTheme.titleLarge),
          const Spacer(),
          if (canManage)
            FilledButton.tonal(
                onPressed: _inviteMember, child: const Text('邀请成员'))
        ]),
        ...p.members.map((m) => ListTile(
            leading: const Icon(Icons.person_outline),
            title: Text(m['display_name']?.toString() ?? '用户 #${m['user_id']}'),
            subtitle: Text('${m['project_role']} · ${m['status']}')))
      ]);

  Widget _buildArtifacts(
          BuildContext context, VopcProvider p, bool canManage) =>
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Text('成果仓库 · ${p.artifacts.length}',
              style: Theme.of(context).textTheme.titleLarge),
          const Spacer(),
          if (canManage)
            FilledButton.tonal(
                onPressed: _createArtifact, child: const Text('登记成果'))
        ]),
        ...p.artifacts.map((a) => Card(
            child: ListTile(
                leading: const Icon(Icons.inventory_2_outlined),
                title: Text(a.name),
                subtitle: Text('${a.type} · ${a.versionCount} 个版本'),
                trailing: canManage
                    ? IconButton(
                        onPressed: () => _createArtifactVersion(a),
                        icon: const Icon(Icons.add_link))
                    : null)))
      ]);

  Widget _buildMilestones(
          BuildContext context, VopcProvider p, bool canManage) =>
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Text('正式里程碑材料', style: Theme.of(context).textTheme.titleLarge),
          const Spacer(),
          if (canManage)
            FilledButton.tonal(
                onPressed: _submitMilestone, child: const Text('提交材料'))
        ]),
        ...p.milestoneSubmissions.map((s) => Card(
            child: ListTile(
                title: Text('${s.stage} · ${s.status}'),
                subtitle: Text(s.evidence),
                trailing: s.status == 'pending' &&
                        CapabilityUtils.has(Capability.vopcMilestoneReview)
                    ? PopupMenuButton<String>(
                        onSelected: (r) => _reviewMilestone(s, r),
                        itemBuilder: (_) => const [
                              PopupMenuItem(value: 'pass', child: Text('通过')),
                              PopupMenuItem(value: 'return', child: Text('退回'))
                            ])
                    : null)))
      ]);

  Future<Map<String, String>?> _textDialog(
      String title, List<String> labels) async {
    final cs = labels.map((_) => TextEditingController()).toList();
    final result = await showDialog<Map<String, String>>(
        context: context,
        builder: (c) => AlertDialog(
                title: Text(title),
                content: SizedBox(
                    width: 460,
                    child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: List.generate(
                            labels.length,
                            (i) => TextField(
                                controller: cs[i],
                                decoration:
                                    InputDecoration(labelText: labels[i]))))),
                actions: [
                  TextButton(
                      onPressed: () => Navigator.pop(c),
                      child: const Text('取消')),
                  FilledButton(
                      onPressed: () => Navigator.pop(c, {
                            for (var i = 0; i < labels.length; i++)
                              labels[i]: cs[i].text.trim()
                          }),
                      child: const Text('确定'))
                ]));
    for (final c in cs) {
      c.dispose();
    }
    return result;
  }

  Future<void> _createDecision() async {
    final d = await _textDialog('创建决策', ['标题', '背景', '选项', '建议决定', '理由']);
    if (d != null && mounted) {
      await context.read<VopcProvider>().createDecision(widget.projectId, {
        'title': d['标题'],
        'background': d['背景'],
        'options': d['选项'],
        'decision': d['建议决定'],
        'rationale': d['理由']
      });
    }
  }

  Future<void> _actDecision(VopcDecision d, String action) async {
    final v =
        await _textDialog(action == 'resolve' ? '形成决议' : '取消决策', ['决定', '理由']);
    if (v != null && mounted) {
      await context.read<VopcProvider>().actDecision(
          widget.projectId, d.id, action,
          decision: v['决定'], rationale: v['理由']);
    }
  }

  Future<void> _inviteMember() async {
    final d = await _textDialog(
        '邀请成员', ['用户 ID', '项目角色（member/co_owner/mentor/reviewer）', '说明']);
    final id = int.tryParse(d?['用户 ID'] ?? '');
    if (d != null && id != null && mounted) {
      await context.read<VopcProvider>().inviteMember(widget.projectId, id,
          d['项目角色（member/co_owner/mentor/reviewer）']!, d['说明']!);
    }
  }

  Future<void> _createArtifact() async {
    final d = await _textDialog(
        '登记成果', ['名称', '类型（document/repository/link/dataset）', '描述', '许可']);
    if (d != null && mounted) {
      await context.read<VopcProvider>().createArtifact(widget.projectId, {
        'name': d['名称'],
        'artifact_type': d['类型（document/repository/link/dataset）'],
        'description': d['描述'],
        'license': d['许可'],
        'visibility': 'private'
      });
    }
  }

  Future<void> _createArtifactVersion(VopcArtifact a) async {
    final d = await _textDialog('登记成果版本', [
      '版本号',
      '来源类型（link/repository/storage_ref/dataset_ref）',
      '安全引用 URL/标识',
      '校验和',
      '版本说明'
    ]);
    if (d != null && mounted) {
      await context
          .read<VopcProvider>()
          .createArtifactVersion(widget.projectId, a.id, {
        'version': d['版本号'],
        'source_kind': d['来源类型（link/repository/storage_ref/dataset_ref）'],
        'source_ref': d['安全引用 URL/标识'],
        'checksum': d['校验和'],
        'release_notes': d['版本说明']
      });
    }
  }

  Future<void> _submitMilestone() async {
    final d =
        await _textDialog('提交正式里程碑材料', ['目标阶段（如 S2）', '证据说明', '指定评审用户 ID（可选）']);
    final rid = int.tryParse(d?['指定评审用户 ID（可选）'] ?? '');
    if (d != null && mounted) {
      await context.read<VopcProvider>().submitMilestone(widget.projectId, {
        'stage': d['目标阶段（如 S2）'],
        'evidence': d['证据说明'],
        'artifact_version_ids': <int>[],
        if (rid != null) 'reviewer_user_id': rid
      });
    }
  }

  Future<void> _reviewMilestone(
      VopcMilestoneSubmission s, String result) async {
    final d = await _textDialog(result == 'pass' ? '通过里程碑' : '退回里程碑', ['评审意见']);
    if (d != null && mounted) {
      await context
          .read<VopcProvider>()
          .reviewMilestone(widget.projectId, s.id, result, d['评审意见']!);
    }
  }

  Widget _buildTasks(
      BuildContext context, VopcProvider provider, bool canManage) {
    final theme = Theme.of(context);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Icon(Icons.task_alt_outlined, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        Text('任务 · ${provider.tasks.length}',
            style: theme.textTheme.titleLarge),
        const Spacer(),
        IconButton(
            tooltip: '刷新任务',
            onPressed: provider.tasksLoading
                ? null
                : () => provider.loadTasks(widget.projectId),
            icon: const Icon(Icons.refresh))
      ]),
      if (provider.tasksLoading) const LinearProgressIndicator(),
      if (provider.error != null) ...[
        const SizedBox(height: 10),
        _ErrorCard(
            message: provider.error!,
            code: provider.statusCode,
            onRetry: () => provider.loadTasks(widget.projectId)),
      ],
      if (!provider.tasksLoading &&
          provider.error == null &&
          provider.tasks.isEmpty)
        const Card(
            child: Padding(
                padding: EdgeInsets.all(24),
                child: Center(child: Text('暂无任务，项目进入 S1 后可创建。')))),
      const SizedBox(height: 8),
      LayoutBuilder(builder: (context, constraints) {
        final width = constraints.maxWidth >= 760
            ? (constraints.maxWidth - 12) / 2
            : constraints.maxWidth;
        return Wrap(
            spacing: 12,
            runSpacing: 12,
            children: provider.tasks
                .map((task) => SizedBox(
                    width: width,
                    child: _TaskCard(
                        task: task,
                        enabled: canManage && !provider.taskMutating,
                        onStatus: (status) => _updateStatus(task, status))))
                .toList());
      }),
      if (!canManage && provider.tasks.isNotEmpty)
        Padding(
            padding: const EdgeInsets.only(top: 12),
            child: Text('你当前拥有只读权限。', style: theme.textTheme.bodySmall)),
    ]);
  }

  Future<void> _createTask() async {
    final title = TextEditingController();
    final criteria = TextEditingController();
    final assignee = TextEditingController();
    var priority = 'normal';
    final formKey = GlobalKey<FormState>();
    final data = await showDialog<Map<String, dynamic>>(
        context: context,
        builder: (dialogContext) => StatefulBuilder(
            builder: (context, setDialogState) => AlertDialog(
                    title: const Text('新建任务'),
                    content: SizedBox(
                        width: 460,
                        child: Form(
                            key: formKey,
                            child: SingleChildScrollView(
                                child: Column(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
                                  TextFormField(
                                      controller: title,
                                      autofocus: true,
                                      maxLength: 200,
                                      decoration: const InputDecoration(
                                          labelText: '任务标题 *'),
                                      validator: (value) =>
                                          value == null || value.trim().isEmpty
                                              ? '请填写任务标题'
                                              : null),
                                  TextFormField(
                                      controller: criteria,
                                      minLines: 2,
                                      maxLines: 4,
                                      decoration: const InputDecoration(
                                          labelText: '验收标准 *'),
                                      validator: (value) =>
                                          value == null || value.trim().isEmpty
                                              ? '请填写可验证的验收标准'
                                              : null),
                                  DropdownButtonFormField<String>(
                                      value: priority,
                                      decoration: const InputDecoration(
                                          labelText: '优先级'),
                                      items: const [
                                        DropdownMenuItem(
                                            value: 'low', child: Text('低')),
                                        DropdownMenuItem(
                                            value: 'normal', child: Text('普通')),
                                        DropdownMenuItem(
                                            value: 'high', child: Text('高')),
                                        DropdownMenuItem(
                                            value: 'urgent', child: Text('紧急')),
                                      ],
                                      onChanged: (value) => setDialogState(
                                          () => priority = value ?? 'normal')),
                                  TextFormField(
                                      controller: assignee,
                                      keyboardType: TextInputType.number,
                                      decoration: const InputDecoration(
                                          labelText: '负责人用户 ID（可选）',
                                          helperText: '仅可填写项目内有效成员；留空表示暂不分派'),
                                      validator: (value) {
                                        final text = value?.trim() ?? '';
                                        if (text.isEmpty) return null;
                                        final id = int.tryParse(text);
                                        return id == null || id <= 0
                                            ? '请输入有效的成员用户 ID'
                                            : null;
                                      })
                                ])))),
                    actions: [
                      TextButton(
                          onPressed: () => Navigator.pop(dialogContext),
                          child: const Text('取消')),
                      FilledButton(
                          onPressed: () {
                            if (!formKey.currentState!.validate()) return;
                            final body = <String, dynamic>{
                              'title': title.text.trim(),
                              'acceptance_criteria': criteria.text.trim(),
                              'priority': priority,
                            };
                            final id = int.tryParse(assignee.text.trim());
                            if (id != null) body['assignee_user_id'] = id;
                            Navigator.pop(dialogContext, body);
                          },
                          child: const Text('创建'))
                    ])));
    title.dispose();
    criteria.dispose();
    assignee.dispose();
    if (data == null || !mounted) return;
    final ok =
        await context.read<VopcProvider>().createTask(widget.projectId, data);
    if (ok && mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('任务已创建')));
    }
  }

  Future<void> _updateStatus(VopcTask task, String status) async {
    final ok = await context
        .read<VopcProvider>()
        .updateTaskStatus(widget.projectId, task.id, status);
    if (ok && mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('任务状态已更新')));
    }
  }
}

class _TaskCard extends StatelessWidget {
  final VopcTask task;
  final bool enabled;
  final ValueChanged<String> onStatus;

  const _TaskCard(
      {required this.task, required this.enabled, required this.onStatus});

  @override
  Widget build(BuildContext context) {
    final next = _nextStatuses[task.status] ?? const <String>[];
    return Card(
        margin: EdgeInsets.zero,
        child: Padding(
            padding: const EdgeInsets.all(14),
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Expanded(
                    child: Text(task.title,
                        style: Theme.of(context).textTheme.titleMedium)),
                _MetaChip(_priorityLabels[task.priority] ?? task.priority),
                const SizedBox(width: 6),
                _MetaChip(_statusLabels[task.status] ?? task.status),
              ]),
              const SizedBox(height: 10),
              Text('验收：${task.acceptanceCriteria}'),
              if (task.assigneeUserId != null ||
                  task.assigneeAiRole != null) ...[
                const SizedBox(height: 6),
                Text(task.assigneeUserId != null
                    ? '负责人：用户 #${task.assigneeUserId}'
                    : '负责人：AI ${task.assigneeAiRole}')
              ],
              if (next.isNotEmpty) ...[
                const SizedBox(height: 10),
                Wrap(
                    spacing: 8,
                    runSpacing: 6,
                    children: next
                        .map((status) => OutlinedButton(
                            onPressed: enabled ? () => onStatus(status) : null,
                            child: Text(_actionLabels[status] ?? status)))
                        .toList())
              ]
            ])));
  }

  static const _nextStatuses = <String, List<String>>{
    'todo': ['in_progress', 'cancelled'],
    'in_progress': ['todo', 'review', 'cancelled'],
    'review': ['in_progress', 'done'],
  };
  static const _statusLabels = {
    'todo': '待开始',
    'in_progress': '进行中',
    'review': '待验收',
    'done': '已完成',
    'cancelled': '已取消',
  };
  static const _priorityLabels = {
    'low': '低',
    'normal': '普通',
    'high': '高',
    'urgent': '紧急',
  };
  static const _actionLabels = {
    'todo': '退回待开始',
    'in_progress': '开始 / 退回',
    'review': '提交验收',
    'done': '验收通过',
    'cancelled': '取消任务',
  };
}

class _MetaChip extends StatelessWidget {
  final String label;
  const _MetaChip(this.label);
  @override
  Widget build(BuildContext context) => Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(999)),
      child: Text(label, style: Theme.of(context).textTheme.labelSmall));
}

class _ErrorCard extends StatelessWidget {
  final String message;
  final int? code;
  final Future<void> Function() onRetry;
  const _ErrorCard(
      {required this.message, required this.code, required this.onRetry});
  @override
  Widget build(BuildContext context) => Card(
      color: Theme.of(context).colorScheme.errorContainer,
      child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(children: [
            Icon(code == 403 ? Icons.lock_outline : Icons.error_outline),
            const SizedBox(height: 8),
            Text(code == null ? message : 'HTTP $code · $message'),
            const SizedBox(height: 12),
            OutlinedButton(onPressed: onRetry, child: const Text('重试'))
          ])));
}
