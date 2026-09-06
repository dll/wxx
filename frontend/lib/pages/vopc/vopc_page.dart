import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'vopc_error_card.dart';
import 'vopc_meta_chip.dart';
import 'vopc_task_card.dart';
import 'vopc_section_widgets.dart';
import 'vopc_hall_project_card.dart';
import 'vopc_reality_extension_card.dart';
import 'vopc_invitation_card.dart';
import 'vopc_project_card.dart';
import 'vopc_hero.dart';
import 'vopc_flow_strip.dart';
import 'vopc_core_idea_card.dart';
import 'vopc_empty_card.dart';
import 'vopc_stage_progress.dart';
import 'vopc_quiz_card.dart';
import 'vopc_user_search_dialog.dart';
import 'vopc_learning_sheet.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../providers/vopc_provider.dart';
import '../../utils/capability_utils.dart';
import '../../utils/vopc_access.dart';

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
    await p.loadLearning();
    await p.loadGuides();
    await p.loadOutcomesSummary();
  }

  @override
  Widget build(BuildContext context) {
    final p = context.watch<VopcProvider>();
    final pending = p.invitations.where((e) => e['status'] == 'pending').length;
    return Scaffold(
      appBar: AppBar(
        title: const Text('vOPC 虚拟创业'),
        actions: [
          IconButton(
            tooltip: '刷新',
            onPressed: p.loading ? null : _start,
            icon: const Icon(Icons.refresh),
          ),
          IconButton(
            tooltip: '运行软件项目模拟演示',
            onPressed: p.loading ? null : _createDemo,
            icon: const Icon(Icons.play_circle_outline),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _create,
        icon: const Icon(Icons.add),
        label: const Text('创建项目'),
      ),
      body: RefreshIndicator(
        onRefresh: _start,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 96),
          children: [
            VopcHero(
              title: '虚拟 OPC 教学模块',
              subtitle: '通过分层的复杂度，理解一人公司如何把一个想法推进为可审阅、可复盘的虚拟成果。',
              projectCount: p.projects.length,
              pendingCount: pending,
            ),
            const SizedBox(height: 20),
            const _OpcIntroSection(),
            const SizedBox(height: 20),
            if (pending > 0) ...[
              const VopcSectionHeader(
                title: '待处理邀请',
                subtitle: '加入团队，协作推进项目',
                icon: Icons.mark_email_unread_outlined,
              ),
              ...p.invitations
                  .where((e) => e['status'] == 'pending')
                  .map((e) => VopcInvitationCard(
                        invitation: e,
                        onDecline: () => p.respondInvitation(
                            (e['id'] as num).toInt(), 'decline'),
                        onAccept: () => p
                            .respondInvitation(
                                (e['id'] as num).toInt(), 'accept')
                            .then((ok) {
                          if (ok) p.loadProjects();
                        }),
                      )),
              const SizedBox(height: 20),
            ],
            if (p.loading) const LinearProgressIndicator(),
            if (!p.loading && p.error != null)
              VopcErrorCard(
                  message: p.error!, code: p.statusCode, onRetry: _start),
            if (!p.loading && p.error == null) ...[
              VopcSectionHeader(
                title: '我的虚拟项目',
                subtitle: p.projects.isEmpty
                    ? '还没有虚拟项目，从一个想法开始走通 OPC 主线'
                    : '${p.projects.length} 个虚拟项目持续孵化中',
                icon: Icons.rocket_launch_outlined,
              ),
              if (p.projects.isEmpty)
                const VopcEmptyCard()
              else
                ...p.projects.map((e) => VopcProjectCard(
                      project: e,
                      onTap: () => context.push('/vopc/projects/${e.id}'),
                      onEdit: () => _editProject(e),
                      onDelete: () => _deleteProject(e),
                    )),
              const SizedBox(height: 24),
              _buildProjectHall(context, p),
              const SizedBox(height: 24),
              _buildOutcomes(context, p),
              const SizedBox(height: 24),
              const VopcRealityExtensionCard(),
            ],
          ],
        ),
      ),
    );
  }

  Future<void> _create() async {
    final data = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (_) => const _CreateProjectDialog(),
    );
    if (data == null || !mounted) return;
    final id = await context.read<VopcProvider>().createProject(data);
    if (!mounted) return;
    if (id != null) {
      context.push('/vopc/projects/$id');
    } else {
      final message = context.read<VopcProvider>().error ?? '项目创建失败';
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(message)));
    }
  }

  Future<void> _createDemo() async {
    final p = context.read<VopcProvider>();
    final id = await p.createDemoProject();
    if (!mounted) return;
    if (id != null) {
      context.push('/vopc/projects/$id');
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(p.error ?? '模拟项目创建失败')));
    }
  }

  Future<void> _editProject(VopcProject project) async {
    final data = await showDialog<Map<String, dynamic>>(
        context: context,
        builder: (_) => _CreateProjectDialog(initial: project));
    if (data == null || !mounted) return;
    final p = context.read<VopcProvider>();
    final ok = await p.updateProject(project.id, data);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '草稿已保存' : (p.error ?? '保存失败'))));
    }
  }

  Future<void> _deleteProject(VopcProject project) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (c) => AlertDialog(
        title: const Text('删除项目'),
        content: Text('确定要删除项目「${project.name}」吗？删除为强制操作，任意状态均可删除，该操作不可恢复。'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(c, false),
              child: const Text('取消')),
          FilledButton(
              style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.error),
              onPressed: () => Navigator.pop(c, true),
              child: const Text('删除')),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final p = context.read<VopcProvider>();
    final ok = await p.deleteProject(project.id);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '项目已删除' : (p.error ?? '删除失败'))));
    }
  }

  /// B1 项目大厅：展示可浏览的可见虚拟练习项目（非成员只读）。
  Widget _buildProjectHall(BuildContext context, VopcProvider p) {
    // vOPC 可见性模型为 private / invite_only / college / restricted（无 public）。
    // 大厅展示非 private（可被学院授权用户浏览）的虚拟练习项目，只读不可写。
    final hall = p.projects.where((e) => e.visibility != 'private').toList();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      VopcSectionHeader(
        title: '项目大厅',
        subtitle:
            hall.isEmpty ? '暂无学院可见的虚拟练习项目' : '${hall.length} 个项目可浏览（只读，不可写）',
        icon: Icons.storefront_outlined,
      ),
      if (hall.isEmpty)
        Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(children: [
              const Icon(Icons.visibility_outlined,
                  size: 40, color: Colors.grey),
              const SizedBox(height: 10),
              Text('暂无学院可见项目', style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 4),
              const Text('当项目主理人将项目设为学院可见（非 private）后，会在这里展示供浏览学习。',
                  style: TextStyle(color: Colors.grey, fontSize: 12)),
            ]),
          ),
        )
      else
        ...hall.map((e) => VopcHallProjectCard(
              project: e,
              onTap: () => context.push('/vopc/projects/${e.id}'),
            )),
    ]);
  }

  /// B1 成果与复盘：聚合 artifacts 摘要 + close-records。
  Widget _buildOutcomes(BuildContext context, VopcProvider p) {
    final summary = p.outcomesSummary;
    final artifactCount = (summary['artifact_count'] as num?)?.toInt() ?? 0;
    final versionCount = (summary['version_count'] as num?)?.toInt() ?? 0;
    final closeRecords =
        (summary['close_records'] as List?)?.cast<Map<String, dynamic>>() ??
            const <Map<String, dynamic>>[];
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      VopcSectionHeader(
        title: '成果与复盘',
        subtitle: p.outcomesLoading
            ? '正在汇总…'
            : '$artifactCount 项成果 · $versionCount 个版本 · ${closeRecords.length} 条复盘/结项记录',
        icon: Icons.reviews_outlined,
      ),
      if (p.outcomesLoading)
        const LinearProgressIndicator()
      else if (artifactCount == 0 && closeRecords.isEmpty)
        Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(children: [
              const Icon(Icons.emoji_events_outlined,
                  size: 40, color: Colors.grey),
              const SizedBox(height: 10),
              Text('还没有成果或复盘记录',
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 4),
              const Text('在项目中登记成果、推进里程碑，结项复盘后会在这里沉淀出你的 OPC 成果档案。',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Colors.grey, fontSize: 12)),
            ]),
          ),
        )
      else ...[
        if (artifactCount > 0)
          Row(children: [
            Icon(Icons.inventory_2_outlined,
                color: Theme.of(context).colorScheme.primary, size: 18),
            const SizedBox(width: 6),
            Text('累计 $artifactCount 项成果（$versionCount 个版本）',
                style: Theme.of(context).textTheme.bodyMedium),
          ]),
        if (closeRecords.isNotEmpty) ...[
          const SizedBox(height: 10),
          ...closeRecords.take(5).map((r) => Card(
                margin: const EdgeInsets.only(bottom: 8),
                child: ListTile(
                  dense: true,
                  leading: Icon(
                    _closeActionIcon(r['action']?.toString()),
                    color: Theme.of(context).colorScheme.primary,
                  ),
                  title: Text(
                      '${r['project_name']?.toString() ?? ''} · ${_closeActionLabel(r['action']?.toString())}'),
                  subtitle: Text(r['reason']?.toString() ?? ''),
                  trailing: Text(r['created_at']?.toString() ?? '',
                      style: const TextStyle(fontSize: 11)),
                ),
              )),
          if (closeRecords.length > 5)
            Text('等 ${closeRecords.length} 条记录，前往对应项目工作台查看全部',
                style: const TextStyle(color: Colors.grey, fontSize: 12)),
        ],
      ],
    ]);
  }

  static String _closeActionLabel(String? action) => switch (action) {
        'close' => '结项',
        'pause' => '暂停',
        'resume' => '继续',
        'pivot' => '转向',
        'terminate' => '终止',
        'archive' => '归档',
        _ => action ?? '记录',
      };

  static IconData _closeActionIcon(String? action) => switch (action) {
        'close' => Icons.task_alt_rounded,
        'pause' => Icons.pause_circle_outline,
        'resume' => Icons.play_circle_outline,
        'pivot' => Icons.u_turn_left_rounded,
        'terminate' => Icons.cancel_outlined,
        'archive' => Icons.archive_outlined,
        _ => Icons.history_rounded,
      };
}

class _CreateProjectDialog extends StatefulWidget {
  final VopcProject? initial;
  const _CreateProjectDialog({this.initial});
  @override
  State<_CreateProjectDialog> createState() => _CreateProjectDialogState();
}

class _CreateProjectDialogState extends State<_CreateProjectDialog> {
  final formKey = GlobalKey<FormState>();
  final name = TextEditingController();
  final summary = TextEditingController();
  final problem = TextEditingController();
  final target = TextEditingController();
  final outcome = TextEditingController();
  final validation = TextEditingController();
  final product = TextEditingController();
  final cycle = TextEditingController();
  final acceptance = TextEditingController();
  final mentor = TextEditingController();
  final resource = TextEditingController();
  String projectType = '自由探索项目';
  String source = 'self_proposed';
  String dataType = '公开数据';
  bool realTrial = false;
  bool externalPublish = false;
  bool funds = false;
  String teamMode = 'auto';

  @override
  void initState() {
    super.initState();
    final p = widget.initial;
    if (p == null) return;
    name.text = p.name;
    summary.text = p.summary;
    problem.text = p.problem;
    target.text = p.targetUsers;
    outcome.text = p.expectedOutcome;
    validation.text = p.validationPlan;
    product.text = p.productForm;
    cycle.text = p.projectCycle;
    acceptance.text = p.acceptanceCriteria;
    mentor.text = p.mentorNeeds;
    resource.text = p.resourceNeeds;
    projectType = p.projectType;
    source = p.projectSource;
    dataType = p.dataType;
    realTrial = p.realUserTrial;
    externalPublish = p.externalPublish;
    funds = p.fundsInvolved;
    teamMode = p.teamMode;
  }

  @override
  void dispose() {
    for (final c in [
      name,
      summary,
      problem,
      target,
      outcome,
      validation,
      product,
      cycle,
      acceptance,
      mentor,
      resource
    ]) {
      c.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: Text(widget.initial == null ? '创建软件项目' : '编辑项目草稿'),
        content: SizedBox(
          width: 560,
          child: Form(
            key: formKey,
            child: SingleChildScrollView(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('先完成 G0 想法卡，再选择自动或手工组建软件项目团队。',
                        style: Theme.of(context).textTheme.bodySmall),
                    const SizedBox(height: 14),
                    DropdownButtonFormField<String>(
                      value: teamMode,
                      decoration: const InputDecoration(labelText: '团队组建方式'),
                      items: const [
                        DropdownMenuItem(
                            value: 'auto', child: Text('自动组队（推荐）')),
                        DropdownMenuItem(value: 'manual', child: Text('手工组队')),
                      ],
                      onChanged: (v) => setState(() => teamMode = v ?? 'auto'),
                    ),
                    const SizedBox(height: 8),
                    _field(name, '项目名称 *', required: true),
                    _field(summary, '项目摘要'),
                    _field(problem, '要解决的真实问题'),
                    _field(target, '目标用户'),
                    _field(outcome, '预期成果'),
                    _field(validation, '验证计划'),
                    _field(product, '产品/成果形态'),
                    _field(cycle, '项目周期'),
                    _field(acceptance, '验收标准'),
                    DropdownButtonFormField<String>(
                        value: projectType,
                        decoration: const InputDecoration(labelText: '项目类型'),
                        items: const [
                          '软件与 AI 产品',
                          '内容与知识产品',
                          '校园服务创新',
                          '创新创业项目',
                          '科研与技术实验',
                          '公益与社会实践',
                          '教学改革项目',
                          '自由探索项目'
                        ]
                            .map((v) =>
                                DropdownMenuItem(value: v, child: Text(v)))
                            .toList(),
                        onChanged: (v) => setState(() => projectType = v!)),
                    DropdownButtonFormField<String>(
                        value: source,
                        decoration: const InputDecoration(labelText: '项目来源'),
                        items: const [
                          DropdownMenuItem(
                              value: 'self_proposed', child: Text('自拟项目')),
                          DropdownMenuItem(
                              value: 'client_requirement', child: Text('甲方需求'))
                        ],
                        onChanged: (v) => setState(() => source = v!)),
                    DropdownButtonFormField<String>(
                        value: dataType,
                        decoration: const InputDecoration(labelText: '数据类型'),
                        items: const [
                          '公开数据',
                          '校内非敏感数据',
                          '个人数据',
                          '敏感个人数据',
                          '学籍数据',
                          '心理健康数据',
                          '医疗健康数据'
                        ]
                            .map((v) =>
                                DropdownMenuItem(value: v, child: Text(v)))
                            .toList(),
                        onChanged: (v) => setState(() => dataType = v!)),
                    _field(mentor, '导师需求'),
                    _field(resource, '资源需求'),
                    SwitchListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text('真实用户试用'),
                        value: realTrial,
                        onChanged: (v) => setState(() => realTrial = v)),
                    SwitchListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text('计划外部发布'),
                        value: externalPublish,
                        onChanged: (v) => setState(() => externalPublish = v)),
                    SwitchListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text('涉及资金/真实支付'),
                        subtitle: const Text('将自动判定为 R3，默认禁止'),
                        value: funds,
                        onChanged: (v) => setState(() => funds = v)),
                  ]),
            ),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context), child: const Text('取消')),
          FilledButton.icon(
            onPressed: _save,
            icon: const Icon(Icons.save_outlined),
            label: const Text('保存草稿'),
          ),
        ],
      );

  Widget _field(TextEditingController controller, String label,
          {bool required = false}) =>
      Padding(
        padding: const EdgeInsets.only(bottom: 10),
        child: TextFormField(
          controller: controller,
          minLines: 1,
          maxLines: label == '项目名称 *' || label == '项目周期' ? 1 : 3,
          validator: required
              ? (v) => (v?.trim().isEmpty ?? true) ? '请填写项目名称' : null
              : null,
          decoration:
              InputDecoration(labelText: label, alignLabelWithHint: true),
        ),
      );

  void _save() {
    if (!(formKey.currentState?.validate() ?? false)) return;
    Navigator.pop(context, {
      'name': name.text.trim(),
      'summary': summary.text.trim(),
      'problem_statement': problem.text.trim(),
      'target_users': target.text.trim(),
      'expected_outcome': outcome.text.trim(),
      'validation_plan': validation.text.trim(),
      'product_form': product.text.trim(),
      'project_cycle': cycle.text.trim(),
      'acceptance_criteria': acceptance.text.trim(),
      'project_type': projectType,
      'project_source': source,
      'risk_level': 'R0',
      'data_type': dataType,
      'mentor_needs': mentor.text.trim(),
      'resource_needs': resource.text.trim(),
      'real_user_trial': realTrial,
      'external_publish': externalPublish,
      'funds_involved': funds,
      'team_mode': teamMode,
    });
  }
}

/// L1 概念层入口：OPC 核心思想一句话 + 五步核心流程图（idea → validate →
/// build → deliver → feedback）+ 进入学习入口。
/// 数据优先取 VopcProvider.learning（来自 GET /vopc/learning），
/// 为空时回退到内置默认内容，保证任意环境都能渲染。
class _OpcIntroSection extends StatelessWidget {
  const _OpcIntroSection();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final learning = context.watch<VopcProvider>().learning;
    final cards = _learningList(learning, 'knowledge_cards');
    final steps = _learningList(learning, 'flow_steps');

    return Card(
      margin: EdgeInsets.zero,
      elevation: 0,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(
              color: theme.colorScheme.outlineVariant.withOpacity(.55))),
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(
                    color: theme.colorScheme.primary.withOpacity(.1),
                    borderRadius: BorderRadius.circular(10)),
                child: Icon(Icons.school_outlined,
                    size: 19, color: theme.colorScheme.primary)),
            const SizedBox(width: 10),
            Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('OPC 入门 · L1 概念层',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w700)),
              Text('先理解核心思想，再进入项目流程',
                  style: theme.textTheme.labelSmall
                      ?.copyWith(color: theme.colorScheme.outline)),
            ]),
          ]),
          const SizedBox(height: 14),
          if (cards.isNotEmpty)
            VopcCoreIdeaCard(
                title: cards.first['title'] ?? 'OPC 核心思想',
                body: cards.first['body'] ?? _defaultCoreIdea),
          const SizedBox(height: 14),
          if (steps.isNotEmpty)
            VopcFlowStrip(steps: steps)
          else
            const VopcFlowStrip(steps: _defaultFlowSteps),
          const SizedBox(height: 14),
          Row(children: [
            FilledButton.tonalIcon(
              onPressed: () => _openLearning(context),
              icon: const Icon(Icons.menu_book_outlined),
              label: const Text('进入 OPC 学习'),
            ),
          ]),
        ]),
      ),
    );
  }

  void _openLearning(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetCtx) {
        final learning = sheetCtx.read<VopcProvider>().learning;
        return VopcLearningSheet(
            learning: learning,
            defaultCards: _defaultCards,
            defaultSteps: _defaultFlowSteps);
      },
    );
  }
}

/// 从 learning 数据中提取某段列表（knowledge_cards / flow_steps），
/// 数据为 Map 或 List，兜底为空返回空列表。
List<Map<String, dynamic>> _learningList(
    Map<String, dynamic> learning, String key) {
  final raw = learning[key];
  if (raw is List) {
    return raw
        .whereType<Map>()
        .map((e) => Map<String, dynamic>.from(e))
        .toList();
  }
  return const [];
}

const String _defaultCoreIdea =
    'OPC（一人公司）：一个人像一家公司一样，独立承担产品、市场、交付等多数关键角色，把一个小想法在「需求 → 成果 → 反馈 → 复盘」的闭环里推进为可审阅、可复盘的成果。';

const List<Map<String, String>> _defaultFlowSteps = [
  {'key': 'idea', 'title': '想法', 'desc': '找到要解决的问题，明确价值假设'},
  {'key': 'validate', 'title': '验证', 'desc': '低成本验证假设是否成立'},
  {'key': 'build', 'title': '构建', 'desc': '做出最小可审阅的虚拟产出'},
  {'key': 'deliver', 'title': '交付', 'desc': '整理成果并约定验收标准'},
  {'key': 'feedback', 'title': '反馈', 'desc': '收集反馈，复盘并决定下一步'},
];

/// 五步核心流程图（水平流 strip，窄屏自动换行）。
const List<Map<String, String>> _defaultCards = [
  {
    'title': 'OPC 是什么',
    'body':
        'OPC（One-Person Company，一人公司）：一个人像一家公司一样，独立承担产品、市场、交付等多数角色，把一个小想法推进为可交付的成果。'
  },
  {'title': '为什么成立', 'body': '单点专注、成本低、决策快、反馈短。关键不是「一个人干所有事」，而是「承担所有关键责任」。'},
  {'title': '最小闭环', 'body': '一个 OPC ≈ 需求方 + 产品/服务 + 交付 + 反馈。四者缺一不可，循环闭环即生意。'},
  {'title': '核心心态', 'body': '先验证再投入，先交付再完善；每一步都要能回溯、能复盘、能讲清楚。'},
];

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
    Future.microtask(() async {
      final p = context.read<VopcProvider>();
      await p.loadDetail(widget.projectId);
      await p.loadAIRoles(widget.projectId);
      await p.loadTimeline(widget.projectId);
      // L3 治理层数据（最佳努力，失败不阻塞）
      await p.loadCloseRecords(widget.projectId);
      await p.loadRisks(widget.projectId);
      await p.loadSpecialApprovals(widget.projectId);
      await p.loadRiskAppeals(widget.projectId);
      await p.loadMilestoneWaivers(widget.projectId);
      await p.loadClientEvidence(widget.projectId);
      await p.loadRubrics(widget.projectId);
      await p.loadFiles(widget.projectId);
      await p.loadAITasks(widget.projectId);
    });
  }

  @override
  Widget build(BuildContext context) {
    final p = context.watch<VopcProvider>();
    final detail = p.detail;
    final canManage = detail?.canManage == true;
    return Scaffold(
        appBar: AppBar(
          title: const Text('项目工作台'),
          actions: [
            IconButton(
              tooltip: '刷新',
              onPressed:
                  p.loading ? null : () => p.loadDetail(widget.projectId),
              icon: const Icon(Icons.refresh),
            )
          ],
        ),
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
                    child: VopcErrorCard(
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
                        Wrap(spacing: 8, runSpacing: 8, children: [
                          Chip(label: Text(p.detail!.stage)),
                          Chip(label: Text(p.detail!.status)),
                          Chip(label: Text(p.detail!.projectType)),
                          Chip(label: Text(p.detail!.riskLevel))
                        ]),
                        const SizedBox(height: 16),
                        VopcStageProgress(
                            currentStage: p.detail!.stage,
                            status: p.detail!.status),
                        const SizedBox(height: 16),
                        _buildCurrentStage(context, p),
                        const SizedBox(height: 24),
                        if (canManage && p.detail!.stage == 'G0') ...[
                          const SizedBox(height: 16),
                          Card(
                            color: Theme.of(context)
                                .colorScheme
                                .primaryContainer
                                .withOpacity(.42),
                            child: Padding(
                              padding: const EdgeInsets.all(16),
                              child: Row(children: [
                                Expanded(
                                    child: Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.start,
                                        children: [
                                      Text('下一步：完善并提交立项',
                                          style: Theme.of(context)
                                              .textTheme
                                              .titleMedium
                                              ?.copyWith(
                                                  fontWeight: FontWeight.w700)),
                                      const SizedBox(height: 4),
                                      const Text(
                                          '提交前需填写完整项目资料；提交后所有阶段只能经正式评审推进。'),
                                    ])),
                                const SizedBox(width: 12),
                                TextButton.icon(
                                    onPressed: _editDraft,
                                    icon: const Icon(Icons.edit_outlined),
                                    label: const Text('编辑草稿')),
                                FilledButton.icon(
                                    onPressed: _submitProject,
                                    icon: const Icon(Icons.arrow_forward),
                                    label: const Text('提交立项')),
                              ]),
                            ),
                          ),
                        ],
                        const SizedBox(height: 24),
                        _buildAIRoles(context, p, canManage),
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
                        const SizedBox(height: 24),
                        _buildGovernance(context, p, canManage),
                      ]));
  }

  Future<void> _submitProject() async {
    final p = context.read<VopcProvider>();
    final project = p.detail;
    if (project == null || project.stage != 'G0') return;
    // 提交前前端完整性自检：缺失字段明确列出并引导编辑，避免直接收到后端笼统 422。
    final missing = <String>[];
    const required = {
      'summary': '项目摘要',
      'problem': '要解决的真实问题',
      'targetUsers': '目标用户',
      'expectedOutcome': '预期成果',
      'validationPlan': '验证计划',
      'productForm': '产品/成果形态',
      'projectCycle': '项目周期',
      'acceptanceCriteria': '验收标准',
    };
    String? fieldOf(String s) {
      switch (s) {
        case 'summary':
          return project.summary;
        case 'problem':
          return project.problem;
        case 'targetUsers':
          return project.targetUsers;
        case 'expectedOutcome':
          return project.expectedOutcome;
        case 'validationPlan':
          return project.validationPlan;
        case 'productForm':
          return project.productForm;
        case 'projectCycle':
          return project.projectCycle;
        case 'acceptanceCriteria':
          return project.acceptanceCriteria;
        default:
          return null;
      }
    }

    required.forEach((key, label) {
      final v = fieldOf(key);
      if (v == null || v.trim().isEmpty) missing.add(label);
    });
    if (missing.isNotEmpty) {
      await showDialog<void>(
        context: context,
        builder: (c) => AlertDialog(
          title: const Text('提交立项前需补齐以下信息'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              ...missing.map((label) => Text('• $label')),
              const SizedBox(height: 12),
              const Text('请点击「编辑草稿」填写完整后再提交立项。'),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(c), child: const Text('知道了')),
          ],
        ),
      );
      return;
    }
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (c) => AlertDialog(
        title: const Text('提交项目立项'),
        content: const Text(
            '系统将检查项目摘要、问题、目标用户、预期成果、验证计划、产品形态、周期和验收标准。信息不完整时会提示需要补充的字段。'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(c, false),
              child: const Text('取消')),
          FilledButton(
              onPressed: () => Navigator.pop(c, true),
              child: const Text('确认提交')),
        ],
      ),
    );
    if (confirmed != true) return;
    final ok = await p.submitProject(widget.projectId);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '项目已提交立项' : (p.error ?? '立项提交失败'))));
  }

  Future<void> _editDraft() async {
    final project = context.read<VopcProvider>().detail;
    if (project == null) return;
    final data = await showDialog<Map<String, dynamic>>(
        context: context,
        builder: (_) => _CreateProjectDialog(initial: project));
    if (data == null || !mounted) return;
    final p = context.read<VopcProvider>();
    final ok = await p.updateProject(widget.projectId, data);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '草稿已保存' : (p.error ?? '草稿保存失败'))));
  }

  bool _canCreateTask(VopcProject project) {
    final stage = int.tryParse(project.stage.replaceFirst('G', '')) ?? -1;
    const blocked = {'paused', 'risk_frozen', 'terminated', 'archived'};
    return stage >= 1 && stage < 4 && !blocked.contains(project.status);
  }

  Widget _buildCurrentStage(BuildContext context, VopcProvider p) {
    final project = p.detail!;
    const titles = {
      'G0': '完善想法卡并确认团队',
      'G1': '明确目标与软件方案',
      'G2': '推进任务并形成阶段成果',
      'G3': '模拟验证并整理用户反馈',
      'G4': '复盘成果并做出闭环决定',
    };
    return Card(
      color: Theme.of(context).colorScheme.primaryContainer.withOpacity(.35),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(Icons.track_changes,
                color: Theme.of(context).colorScheme.primary),
            const SizedBox(width: 8),
            Text('当前阶段驾驶舱',
                style: Theme.of(context)
                    .textTheme
                    .titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
            const Spacer(),
            VopcMetaChip(project.teamMode == 'manual' ? '手工组队' : '自动组队'),
            if (project.isDemo) ...[
              const SizedBox(width: 6),
              const VopcMetaChip('模拟演示')
            ],
          ]),
          const SizedBox(height: 8),
          Text('${project.stage} · ${titles[project.stage] ?? '查看项目状态'}',
              style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 4),
          Text(
              '团队 ${p.aiRoles.length + p.members.length} 人/向导 · 任务 ${p.tasks.length} · 成果 ${p.artifacts.length} · 时间线 ${p.timeline.length} 条',
              style: Theme.of(context).textTheme.bodySmall),
          if (p.timeline.isNotEmpty) ...[
            const SizedBox(height: 10),
            Text('最近动态：${p.timeline.first['action'] ?? '项目事件'}',
                style: Theme.of(context).textTheme.bodySmall),
          ],
          if (project.isDemo && project.stage != 'G4') ...[
            const SizedBox(height: 12),
            FilledButton.tonalIcon(
              onPressed: () async {
                final ok = await p.advanceSimulation(project.id);
                if (context.mounted && !ok) {
                  ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(p.error ?? '模拟推进失败')));
                }
              },
              icon: const Icon(Icons.play_arrow),
              label: const Text('推进模拟下一阶段'),
            ),
          ],
        ]),
      ),
    );
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
                      VopcMetaChip(d.status)
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

  Widget _buildAIRoles(BuildContext context, VopcProvider p, bool canManage) =>
      Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Text('AI 团队成员 · ${p.aiRoles.length}',
              style: Theme.of(context).textTheme.titleLarge),
          const Spacer(),
        ]),
        const SizedBox(height: 4),
        const Text('虚拟 AI 化身，辅助你完成各阶段孵化工作',
            style: TextStyle(color: Colors.grey, fontSize: 12)),
        const SizedBox(height: 8),
        if (p.aiRoles.isEmpty)
          const Text('暂无 AI 岗位（创建项目时自动生成 4 个默认岗位）',
              style: TextStyle(color: Colors.grey))
        else
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: p.aiRoles
                .map<Widget>((r) => Chip(
                      avatar: Icon(
                          r['role_key'] == 'project_manager'
                              ? Icons.supervisor_account
                              : r['role_key'] == 'execution'
                                  ? Icons.handyman
                                  : Icons.smart_toy_outlined,
                          size: 18),
                      label: Text(
                          '${r['role_name']}${r['enabled'] == true ? '' : '（禁用）'}'),
                    ))
                .toList(),
          ),
      ]);

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
    final p = context.read<VopcProvider>();
    final searchCtrl = TextEditingController();
    final result = await showDialog<Map<String, String>>(
        context: context,
        builder: (c) => VopcUserSearchDialog(
            controller: searchCtrl,
            findUsers: (q) =>
                p.searchUsers(q, excludeProjectId: widget.projectId)));
    searchCtrl.dispose();
    if (result == null || !mounted) return;
    final userId = int.tryParse(result['user_id'] ?? '');
    if (userId == null) return;
    await p.inviteMember(widget.projectId, userId, result['role'] ?? 'member',
        result['note'] ?? '');
    if (mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(p.error ?? '已发送邀请')));
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
      'SHA-256 校验和（64 位十六进制）',
      '适用阶段（G0-G4）',
      '版本说明'
    ]);
    if (d != null && mounted) {
      await context
          .read<VopcProvider>()
          .createArtifactVersion(widget.projectId, a.id, {
        'version': d['版本号'],
        'source_kind': d['来源类型（link/repository/storage_ref/dataset_ref）'],
        'source_ref': d['安全引用 URL/标识'],
        'checksum': d['SHA-256 校验和（64 位十六进制）'],
        'intended_stage': d['适用阶段（G0-G4）'],
        'release_notes': d['版本说明']
      });
    }
  }

  Future<void> _submitMilestone() async {
    final p = context.read<VopcProvider>();
    final versions = <Map<String, dynamic>>[];
    for (final artifact in p.artifacts) {
      final items = await p.loadArtifactVersions(widget.projectId, artifact.id);
      versions.addAll(items.map((v) => {...v, 'artifact_name': artifact.name}));
    }
    if (!mounted) return;
    if (versions.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('请先登记成果并创建至少一个版本，再提交里程碑')));
      return;
    }
    final result = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (_) => _MilestoneSubmissionDialog(versions: versions),
    );
    if (result == null || !mounted) return;
    final ok = await p.submitMilestone(widget.projectId, result);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(ok ? '里程碑材料已提交评审' : (p.error ?? '里程碑提交失败'))));
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
        VopcErrorCard(
            message: provider.error!,
            code: provider.statusCode,
            onRetry: () => provider.loadTasks(widget.projectId)),
      ],
      if (!provider.tasksLoading &&
          provider.error == null &&
          provider.tasks.isEmpty)
        Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(children: [
              Icon(Icons.playlist_add_check_circle_outlined,
                  size: 42, color: theme.colorScheme.primary),
              const SizedBox(height: 10),
              Text('暂无任务',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w700)),
              const SizedBox(height: 5),
              Text(provider.detail?.stage == 'G0'
                  ? '请先点击页面上方“提交立项”，项目进入 G1 后即可创建任务。'
                  : canManage
                      ? '点击右下角“新建任务”，开始分解和推进工作。'
                      : '项目负责人尚未创建任务。'),
              if (canManage && provider.detail?.stage == 'G0') ...[
                const SizedBox(height: 12),
                FilledButton.tonalIcon(
                  onPressed: _submitProject,
                  icon: const Icon(Icons.arrow_upward),
                  label: const Text('前往提交立项'),
                ),
              ] else if (canManage &&
                  provider.detail != null &&
                  _canCreateTask(provider.detail!)) ...[
                const SizedBox(height: 12),
                FilledButton.tonalIcon(
                  onPressed: provider.taskMutating ? null : _createTask,
                  icon: const Icon(Icons.add_task_outlined),
                  label: const Text('新建第一个任务'),
                ),
              ],
            ]),
          ),
        ),
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
                    child: VopcTaskCard(
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

  // ── L3 治理层 UI（结项/风险/里程碑门禁/私有文件/虚拟向导） ──

  Widget _buildGovernance(
      BuildContext context, VopcProvider p, bool canManage) {
    final theme = Theme.of(context);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Icon(Icons.gpp_good_outlined, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        Text('治理与门禁 · L3', style: theme.textTheme.titleLarge),
      ]),
      const SizedBox(height: 4),
      const Text('结项状态、风险分级、里程碑门禁、私有文件与虚拟向导审阅',
          style: TextStyle(color: Colors.grey, fontSize: 12)),
      const SizedBox(height: 12),
      _buildCloseGovernance(context, p, canManage),
      const SizedBox(height: 20),
      _buildRiskGovernance(context, p, canManage),
      const SizedBox(height: 20),
      _buildMilestoneGate(context, p, canManage),
      const SizedBox(height: 20),
      _buildPrivateFiles(context, p, canManage),
      const SizedBox(height: 20),
      _buildAITaskReview(context, p, canManage),
    ]);
  }

  Widget _buildCloseGovernance(
      BuildContext context, VopcProvider p, bool canManage) {
    final status = p.detail?.status ?? '';
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text('结项 / 生命周期 · ${p.closeRecords.length}',
            style: Theme.of(context).textTheme.titleMedium),
        const Spacer(),
        if (canManage)
          FilledButton.tonalIcon(
            onPressed: _openCloseAction,
            icon: const Icon(Icons.flag_outlined),
            label: const Text('状态流转'),
          ),
      ]),
      const SizedBox(height: 6),
      Row(children: [
        VopcMetaChip('当前状态：$status'),
        const SizedBox(width: 6),
        if (status == 'risk_frozen') const VopcMetaChip('已冻结'),
      ]),
      const SizedBox(height: 8),
      if (p.closeRecords.isEmpty)
        const Text('暂无结项/异常状态记录',
            style: TextStyle(color: Colors.grey, fontSize: 12))
      else
        ...p.closeRecords.take(5).map((r) => Card(
              margin: const EdgeInsets.only(bottom: 6),
              child: ListTile(
                dense: true,
                title: Text('${r['action']} → ${r['new_status']}'),
                subtitle: Text(r['reason']?.toString() ?? ''),
                trailing: Text(r['created_at']?.toString() ?? '',
                    style: const TextStyle(fontSize: 11)),
              ),
            )),
    ]);
  }

  Widget _buildRiskGovernance(
      BuildContext context, VopcProvider p, bool canManage) {
    final isRiskManager = CapabilityUtils.has(Capability.vopcRiskManage);
    final isMentor = CapabilityUtils.has(Capability.vopcMentorReview);
    final canApproveRisk = isRiskManager || isMentor;
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text('风险治理 · ${p.risks.length}',
            style: Theme.of(context).textTheme.titleMedium),
        const Spacer(),
        if (canManage)
          FilledButton.tonalIcon(
            onPressed: _createRisk,
            icon: const Icon(Icons.report_problem_outlined),
            label: const Text('登记风险'),
          ),
        if (isRiskManager) ...[
          const SizedBox(width: 8),
          OutlinedButton(
            onPressed: () => _freezeAction(
                p.detail?.status == 'risk_frozen' ? 'unfreeze' : 'freeze'),
            child: Text(p.detail?.status == 'risk_frozen' ? '解冻项目' : '冻结项目'),
          ),
        ],
      ]),
      const SizedBox(height: 6),
      if ((p.detail?.riskLevel ?? '') == 'R2' ||
          (p.detail?.riskLevel ?? '') == 'R3')
        const Padding(
          padding: EdgeInsets.only(bottom: 6),
          child: Text('未批不可推进：R2 需导师/管理员审核，R3 需学校制度专项审批',
              style: TextStyle(color: Colors.orange, fontSize: 12)),
        ),
      if (p.risks.isEmpty)
        const Text('暂无登记风险', style: TextStyle(color: Colors.grey, fontSize: 12))
      else
        ...p.risks.map((r) => Card(
              margin: const EdgeInsets.only(bottom: 6),
              child: ListTile(
                dense: true,
                leading: VopcMetaChip(r['risk_level']?.toString() ?? 'R0'),
                title: Text(r['title']?.toString() ?? ''),
                subtitle: Text(
                    '${r['status']} · ${r['description']?.toString() ?? ''}'),
                trailing: canApproveRisk && r['status'] == 'open'
                    ? Wrap(spacing: 4, children: [
                        TextButton(
                            onPressed: () => _approveRisk(r, 'approve'),
                            child: const Text('通过')),
                        TextButton(
                            onPressed: () => _approveRisk(r, 'reject'),
                            child: const Text('拒绝')),
                      ])
                    : null,
              ),
            )),
      // 专项审批记录
      if (p.specialApprovals.isNotEmpty) ...[
        const SizedBox(height: 4),
        const Text('专项审批记录',
            style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
        ...p.specialApprovals.map((s) => ListTile(
            dense: true,
            leading: const Icon(Icons.verified_outlined, size: 18),
            title: Text('${s['approver']}'),
            subtitle: Text(
                '${s['reason']}${s['ref'] != null && s['ref'] != '' ? ' · 依据：${s['ref']}' : ''}'))),
      ],
      if (isRiskManager)
        FilledButton.tonal(
            onPressed: _createSpecialApproval, child: const Text('登记专项审批（R3）')),
      // 申诉
      if (p.riskAppeals.isNotEmpty) ...[
        const SizedBox(height: 6),
        Text('风险申诉 · ${p.riskAppeals.length}',
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
        ...p.riskAppeals.map((a) => ListTile(
            dense: true,
            title: Text(a['status']?.toString() ?? ''),
            subtitle: Text(a['reason']?.toString() ?? ''),
            trailing: isRiskManager && a['status'] == 'pending'
                ? Wrap(spacing: 4, children: [
                    TextButton(
                        onPressed: () => _resolveAppeal(a, 'upheld'),
                        child: const Text('维持')),
                    TextButton(
                        onPressed: () => _resolveAppeal(a, 'dismissed'),
                        child: const Text('驳回')),
                  ])
                : null)),
      ],
      if (canManage)
        TextButton.icon(
            onPressed: _createRiskAppeal,
            icon: const Icon(Icons.contact_support_outlined, size: 18),
            label: const Text('发起风险申诉')),
    ]);
  }

  Widget _buildMilestoneGate(
      BuildContext context, VopcProvider p, bool canManage) {
    final canReview = CapabilityUtils.has(Capability.vopcMilestoneReview);
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text(
            '里程碑门禁（G0-G4） · ${p.rubrics.isEmpty ? '量表未加载' : '${p.rubrics.length} 维度'}',
            style: Theme.of(context).textTheme.titleMedium),
        const Spacer(),
        if (canManage)
          FilledButton.tonalIcon(
            onPressed: _createWaiver,
            icon: const Icon(Icons.rule_outlined),
            label: const Text('申请豁免'),
          ),
      ]),
      const SizedBox(height: 6),
      // 评分量表（阶段 + 维度 + 通过分）
      if (p.rubrics.isNotEmpty)
        Wrap(
          spacing: 6,
          runSpacing: 6,
          children: p.rubrics
              .map((r) =>
                  VopcMetaChip('${r['stage']}·${r['title']} ≥${r['min_pass']}'))
              .toList(),
        ),
      if (p.milestoneSubmissions.isNotEmpty) ...[
        const SizedBox(height: 8),
        ...p.milestoneSubmissions.map((s) => Card(
              margin: const EdgeInsets.only(bottom: 6),
              child: ListTile(
                dense: true,
                title: Text('${s.stage} · ${s.status}'),
                subtitle: Text(s.evidence),
                trailing: canReview && s.status == 'condition_pending'
                    ? PopupMenuButton<String>(
                        onSelected: (a) async {
                          if (a == 'finalize') {
                            await p.finalizeMilestone(widget.projectId, s.id);
                          }
                        },
                        itemBuilder: (_) => const [
                              PopupMenuItem(
                                  value: 'finalize', child: Text('确认闭环'))
                            ])
                    : canReview && s.status == 'pending'
                        ? PopupMenuButton<String>(
                            onSelected: (r) => _reviewMilestone(s, r),
                            itemBuilder: (_) => const [
                                  PopupMenuItem(
                                      value: 'pass', child: Text('通过')),
                                  PopupMenuItem(
                                      value: 'return', child: Text('退回')),
                                  PopupMenuItem(
                                      value: 'condition_pending',
                                      child: Text('条件通过')),
                                ])
                        : null,
              ),
            )),
      ],
      // 豁免列表
      if (p.milestoneWaivers.isNotEmpty) ...[
        const SizedBox(height: 4),
        Text('里程碑豁免 · ${p.milestoneWaivers.length}',
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
        ...p.milestoneWaivers.map((w) => ListTile(
            dense: true,
            title: Text('${w['stage']} · ${w['status']}'),
            subtitle: Text(w['reason']?.toString() ?? ''),
            trailing: (CapabilityUtils.has(Capability.vopcMentorReview) ||
                        CapabilityUtils.has(Capability.vopcRiskManage)) &&
                    w['status'] == 'pending'
                ? Wrap(spacing: 4, children: [
                    TextButton(
                        onPressed: () => _reviewWaiver(w, 'approve'),
                        child: const Text('批准')),
                    TextButton(
                        onPressed: () => _reviewWaiver(w, 'reject'),
                        child: const Text('驳回')),
                  ])
                : null)),
      ],
      // 甲方结构化证据
      if (p.clientEvidence.isNotEmpty) ...[
        const SizedBox(height: 4),
        Text('甲方结构化证据 · ${p.clientEvidence.length}',
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
        ...p.clientEvidence.map((e) => ListTile(
            dense: true,
            leading: const Icon(Icons.badge_outlined, size: 18),
            title: Text('${e['stage']} · ${e['client_rep']}'),
            subtitle: Text('${e['conclusion']}'))),
      ],
    ]);
  }

  Widget _buildPrivateFiles(
      BuildContext context, VopcProvider p, bool canManage) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text('私有文件（受控鉴权） · ${p.files.length}',
            style: Theme.of(context).textTheme.titleMedium),
        const Spacer(),
        if (canManage)
          FilledButton.tonalIcon(
            onPressed: _uploadFile,
            icon: const Icon(Icons.upload_file_outlined),
            label: const Text('上传文件'),
          ),
      ]),
      const SizedBox(height: 8),
      if (p.files.isEmpty)
        const Text('暂无私有文件', style: TextStyle(color: Colors.grey, fontSize: 12))
      else
        ...p.files.map((f) => Card(
              margin: const EdgeInsets.only(bottom: 6),
              child: ListTile(
                dense: true,
                leading: const Icon(Icons.lock_outline, size: 18),
                title: Text(f['file_name']?.toString() ??
                    f['object_key']?.toString() ??
                    ''),
                subtitle: Text(
                    '${f['storage_status']} · ${_formatBytes(f['size_bytes'])}'),
                trailing: IconButton(
                  tooltip: '下载',
                  onPressed: () => _downloadFile(f),
                  icon: const Icon(Icons.download_outlined),
                ),
              ),
            )),
    ]);
  }

  Widget _buildAITaskReview(
      BuildContext context, VopcProvider p, bool canManage) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Text('虚拟向导调阅 · ${p.aiTasks.length}',
            style: Theme.of(context).textTheme.titleMedium),
        const Spacer(),
        if (canManage)
          FilledButton.tonalIcon(
            onPressed: _createAITask,
            icon: const Icon(Icons.auto_awesome_outlined),
            label: const Text('引导（新建向导稿）'),
          ),
      ]),
      const SizedBox(height: 8),
      if (p.aiTasks.isEmpty)
        const Text('暂无虚拟向导任务，点击「引导」生成结构草稿',
            style: TextStyle(color: Colors.grey, fontSize: 12))
      else
        ...p.aiTasks.map((t) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Expanded(
                          child: Text('${t.roleKey} · ${t.model}',
                              style: Theme.of(context).textTheme.titleSmall)),
                      VopcMetaChip(
                          'provider=${t.model == 'virtual_guide' ? 'template' : t.model}'),
                    ]),
                    const SizedBox(height: 4),
                    Wrap(spacing: 6, runSpacing: 4, children: [
                      VopcMetaChip(
                          '版次 revision=${t.revision.isEmpty ? '0' : t.revision}'),
                      VopcMetaChip(
                          '修改率 ${(t.modificationRate * 100).toStringAsFixed(0)}%'),
                      if (t.finalDecision != null)
                        VopcMetaChip('已审阅：${t.finalDecision}'),
                    ]),
                    if (t.outputContent.isNotEmpty) ...[
                      const SizedBox(height: 6),
                      Text(t.outputContent,
                          maxLines: 5,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                              fontSize: 12, color: Colors.black54)),
                    ],
                    if (canManage && t.finalDecision == null) ...[
                      const SizedBox(height: 8),
                      Wrap(spacing: 8, runSpacing: 6, children: [
                        FilledButton(
                            onPressed: () => _reviewAITask(t, 'accept'),
                            child: const Text('接受 accept')),
                        OutlinedButton(
                            onPressed: () => _reviewAITask(t, 'revise'),
                            child: const Text('修改 revise')),
                        OutlinedButton(
                            onPressed: () => _reviewAITask(t, 'reject'),
                            child: const Text('退回 reject')),
                        OutlinedButton(
                            onPressed: () => _reviewAITask(t, 'overrule'),
                            child: const Text('否决 overrule')),
                      ]),
                    ],
                  ],
                ),
              ),
            )),
    ]);
  }

  // ── 治理动作 ──

  Future<void> _openCloseAction() async {
    final action = await showDialog<String>(
      context: context,
      builder: (c) => SimpleDialog(
        title: const Text('选择生命周期动作'),
        children: ['close', 'pause', 'resume', 'pivot', 'terminate', 'archive']
            .map((a) => SimpleDialogOption(
                  onPressed: () => Navigator.pop(c, a),
                  child: Text(_VopcPageState._closeActionLabel(a)),
                ))
            .toList(),
      ),
    );
    if (action == null || !mounted) return;
    final note = await _textDialog('状态流转：$action', [
      '理由（必填）',
      if (action == 'terminate') '失败证据',
      if (action == 'close' || action == 'pivot') '人类决策依据',
      if (action == 'close') '成果包/复盘要点'
    ]);
    if (note == null || !mounted) return;
    final p = context.read<VopcProvider>();
    final ok = await p.closeProject(widget.projectId, {
      'action': action,
      'reason': note['理由（必填）'] ?? '',
      if (action == 'terminate') 'failure_evidence': note['失败证据'] ?? '',
      if (action == 'close' || action == 'pivot')
        'human_decision': note['人类决策依据'] ?? '',
      if (action == 'close') 'outcome_package': note['成果包/复盘要点'] ?? '',
    });
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '状态已流转' : (p.error ?? '流转失败'))));
    }
  }

  Future<void> _createRisk() async {
    final d = await _textDialog('登记风险', ['风险等级（R0/R1/R2/R3）', '标题', '描述']);
    if (d != null && mounted) {
      await context.read<VopcProvider>().createRisk(widget.projectId, {
        'risk_level': d['风险等级（R0/R1/R2/R3）'] ?? 'R0',
        'title': d['标题'],
        'description': d['描述'],
      });
    }
  }

  Future<void> _approveRisk(Map<String, dynamic> r, String decision) async {
    final d = await _textDialog('风险审批：$decision', ['理由']);
    if (d != null && mounted) {
      final ok = await context.read<VopcProvider>().approveRisk(
          widget.projectId, (r['id'] as num).toInt(), decision, d['理由'] ?? '');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(ok
                ? '审批已提交'
                : (context.read<VopcProvider>().error ?? '审批失败'))));
      }
    }
  }

  Future<void> _freezeAction(String action) async {
    final d = await _textDialog(action == 'freeze' ? '冻结项目' : '解冻项目', ['理由']);
    if (d != null && mounted) {
      final ok = await context
          .read<VopcProvider>()
          .freezeProject(widget.projectId, action, d['理由'] ?? '');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(ok
                ? (action == 'freeze' ? '项目已冻结' : '项目已解冻')
                : (context.read<VopcProvider>().error ?? '操作失败'))));
      }
    }
  }

  Future<void> _createSpecialApproval() async {
    final d = await _textDialog('登记专项审批（R3）', ['审批主体', '批准理由', '依据/批文编号']);
    if (d != null && mounted) {
      await context
          .read<VopcProvider>()
          .createSpecialApproval(widget.projectId, {
        'approver': d['审批主体'],
        'reason': d['批准理由'],
        'ref': d['依据/批文编号'],
      });
    }
  }

  Future<void> _createRiskAppeal() async {
    final d = await _textDialog('发起风险申诉', ['申诉理由']);
    if (d != null && mounted) {
      await context
          .read<VopcProvider>()
          .createRiskAppeal(widget.projectId, d['申诉理由'] ?? '');
    }
  }

  Future<void> _resolveAppeal(Map<String, dynamic> a, String decision) async {
    final d = await _textDialog('裁定申诉：$decision', ['裁定说明']);
    if (d != null && mounted) {
      await context.read<VopcProvider>().resolveRiskAppeal(widget.projectId,
          (a['id'] as num).toInt(), decision, d['裁定说明'] ?? '');
    }
  }

  Future<void> _createWaiver() async {
    final d = await _textDialog('申请里程碑豁免', ['阶段（G0-G4）', '必交证据', '理由']);
    if (d != null && mounted) {
      await context
          .read<VopcProvider>()
          .createMilestoneWaiver(widget.projectId, {
        'stage': d['阶段（G0-G4）'],
        'required_evidence': d['必交证据'],
        'reason': d['理由'],
      });
    }
  }

  Future<void> _reviewWaiver(Map<String, dynamic> w, String action) async {
    final d = await _textDialog('豁免审批：$action', ['审批意见']);
    if (d != null && mounted) {
      await context.read<VopcProvider>().reviewMilestoneWaiver(
          widget.projectId, (w['id'] as num).toInt(), action, d['审批意见'] ?? '');
    }
  }

  Future<void> _uploadFile() async {
    final result = await FilePicker.platform.pickFiles(withData: true);
    if (result == null || result.files.isEmpty || !mounted) return;
    final file = result.files.first;
    final bytes = file.bytes;
    if (bytes == null) return;
    final p = context.read<VopcProvider>();
    final ok = await p.uploadProjectFile(widget.projectId,
        filename: file.name, bytes: bytes);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ok ? '文件已上传' : (p.error ?? '上传失败'))));
    }
  }

  Future<void> _downloadFile(Map<String, dynamic> f) async {
    final key = f['object_key']?.toString();
    if (key == null || key.isEmpty) return;
    final p = context.read<VopcProvider>();
    final bytes = await p.downloadProjectFile(widget.projectId, key);
    if (!mounted) return;
    if (bytes == null) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(p.error ?? '下载失败')));
      return;
    }
    // 简单落地：提示已取回字节（受控鉴权已由后端完成）。
    ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('已取回文件（${bytes.length} 字节），受控鉴权通过')));
  }

  Future<void> _createAITask() async {
    final p = context.read<VopcProvider>();
    final roles = p.aiRoles
        .where((r) => r['enabled'] == true)
        .map((r) => r['role_key']?.toString() ?? '')
        .where((k) => k.isNotEmpty)
        .toList();
    final data = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (c) => _AITaskCreateDialog(roles: roles),
    );
    if (data == null || !mounted) return;
    final task = await p.createAITask(widget.projectId, data);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(task != null ? '虚拟向导草稿已生成' : (p.error ?? '生成失败'))));
    }
  }

  Future<void> _reviewAITask(VopcAITask t, String decision) async {
    final labels = <String>['审阅备注'];
    if (decision == 'revise') labels.add('修订指示 revision');
    final d = await _textDialog('审阅虚拟向导：$decision', labels);
    if (d == null || !mounted) return;
    final ok = await context.read<VopcProvider>().reviewAITask(
        widget.projectId, t.id, decision,
        note: d['审阅备注'] ?? '', revision: d['修订指示 revision'] ?? '');
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(
              ok ? '已审阅' : (context.read<VopcProvider>().error ?? '审阅失败'))));
    }
  }

  static String? _formatBytes(dynamic v) {
    if (v is! num) return null;
    final n = v.toDouble();
    if (n < 1024) return '${n.toInt()} B';
    if (n < 1024 * 1024) return '${(n / 1024).toStringAsFixed(1)} KB';
    return '${(n / 1024 / 1024).toStringAsFixed(1)} MB';
  }
}

class _AITaskCreateDialog extends StatefulWidget {
  final List<String> roles;
  const _AITaskCreateDialog({required this.roles});
  @override
  State<_AITaskCreateDialog> createState() => _AITaskCreateDialogState();
}

class _AITaskCreateDialogState extends State<_AITaskCreateDialog> {
  String _roleKey = '';
  final _instruction = TextEditingController();

  @override
  void dispose() {
    _instruction.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_roleKey.isEmpty && widget.roles.isNotEmpty) {
      _roleKey = widget.roles.first;
    }
    return AlertDialog(
      title: const Text('虚拟向导引导'),
      content: SizedBox(
        width: 460,
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          DropdownButtonFormField<String>(
            value: widget.roles.contains(_roleKey) ? _roleKey : null,
            decoration: const InputDecoration(labelText: '向导岗位'),
            items: widget.roles
                .map((r) => DropdownMenuItem(value: r, child: Text(r)))
                .toList(),
            onChanged: (v) => setState(() => _roleKey = v ?? ''),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _instruction,
            minLines: 2,
            maxLines: 5,
            decoration: const InputDecoration(labelText: '任务指令'),
          ),
        ]),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(
          onPressed: _instruction.text.trim().isEmpty
              ? null
              : () => Navigator.pop(context, {
                    'role_key': _roleKey,
                    'instruction': _instruction.text.trim(),
                  }),
          child: const Text('生成草稿'),
        ),
      ],
    );
  }
}

class _MilestoneSubmissionDialog extends StatefulWidget {
  final List<Map<String, dynamic>> versions;
  const _MilestoneSubmissionDialog({required this.versions});

  @override
  State<_MilestoneSubmissionDialog> createState() =>
      _MilestoneSubmissionDialogState();
}

class _MilestoneSubmissionDialogState
    extends State<_MilestoneSubmissionDialog> {
  final stage = TextEditingController();
  final evidence = TextEditingController();
  final reviewer = TextEditingController();
  final selected = <int>{};

  @override
  void dispose() {
    stage.dispose();
    evidence.dispose();
    reviewer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('提交正式里程碑材料'),
        content: SizedBox(
          width: 520,
          child: SingleChildScrollView(
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              TextField(
                  controller: stage,
                  onChanged: (_) => setState(() {}),
                  decoration: const InputDecoration(labelText: '目标阶段（如 G2）')),
              TextField(
                  controller: evidence,
                  onChanged: (_) => setState(() {}),
                  minLines: 2,
                  maxLines: 5,
                  decoration: const InputDecoration(labelText: '证据说明')),
              TextField(
                  controller: reviewer,
                  keyboardType: TextInputType.number,
                  decoration:
                      const InputDecoration(labelText: '指定评审用户 ID（可选）')),
              const SizedBox(height: 12),
              const Align(
                  alignment: Alignment.centerLeft, child: Text('绑定成果版本（至少一项）')),
              ...widget.versions.map((v) {
                final id = (v['id'] as num).toInt();
                return CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  value: selected.contains(id),
                  title: Text('${v['artifact_name']} · ${v['version']}'),
                  subtitle: Text(v['release_notes']?.toString() ?? ''),
                  onChanged: (checked) => setState(() {
                    if (checked == true) {
                      selected.add(id);
                    } else {
                      selected.remove(id);
                    }
                  }),
                );
              }),
            ]),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context), child: const Text('取消')),
          FilledButton(
            onPressed: stage.text.trim().isEmpty ||
                    evidence.text.trim().isEmpty ||
                    selected.isEmpty
                ? null
                : () {
                    final reviewerId = int.tryParse(reviewer.text.trim());
                    Navigator.pop(context, {
                      'stage': stage.text.trim(),
                      'evidence': evidence.text.trim(),
                      'artifact_version_ids': selected.toList(),
                      if (reviewerId != null) 'reviewer_user_id': reviewerId,
                    });
                  },
            child: const Text('提交评审'),
          ),
        ],
      );
}
