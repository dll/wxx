import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/error_view.dart';

/// 督办工单管理（书记/学院管理员，D5-3「洞察→工单」治理回环）
///
/// 一屏查看治理洞察督办件的总览与明细：
/// - 顶部状态统计（待办/处理中/已完成/已关闭）
/// - 「从育人 KPI 生成补料工单」入口（D5-1 联动：仅对 data_source=not_available 的
///   真实缺失指标生成「催办补料」工单，后端校验，绝不伪造数字）
/// - 工单列表 + 分派责任人 + 状态推进
class GovTicketManagePage extends StatefulWidget {
  const GovTicketManagePage({super.key});

  @override
  State<GovTicketManagePage> createState() => _GovTicketManagePageState();
}

class _GovTicketManagePageState extends State<GovTicketManagePage> {
  String _statusFilter = '';
  String _categoryFilter = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SecretaryProvider>().fetchTicketStats();
      context.read<SecretaryProvider>().fetchTickets();
    });
  }

  Future<void> _reload(SecretaryProvider p) async {
    await p.fetchTicketStats();
    await p.fetchTickets(status: _statusFilter, category: _categoryFilter);
  }

  /// 从育人 KPI（not_available + upload_target=kb）生成补料督办工单（D5-1 联动）
  Future<void> _openCreateFromKPI(SecretaryProvider p) async {
    final kpis = p.nurtureKPIs
        .where((k) =>
            k['data_source'] == 'not_available' && k['upload_target'] == 'kb')
        .toList();
    if (kpis.isEmpty) {
      // 兜底拉一次 KPI
      await p.fetchNurtureKPI();
      if (!mounted) return;
    }
    final fresh = context.read<SecretaryProvider>().nurtureKPIs
        .where((k) =>
            k['data_source'] == 'not_available' && k['upload_target'] == 'kb')
        .toList();
    if (fresh.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('当前没有需要补料的育人指标')));
      return;
    }
    final kpi = await _pickKPI(fresh);
    if (kpi == null || !mounted) return;
    final priority = await _pickPriority();
    if (priority == null || !mounted) return;

    final result = await context.read<SecretaryProvider>().createTicketFromKPI(
          kpiKey: (kpi['key'] as String?) ?? '',
          priority: priority,
        );
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(result.ok
          ? '已生成补料督办工单：${kpi['label']}'
          : '生成失败：${result.msg}'),
    ));
    if (result.ok) _reload(context.read<SecretaryProvider>());
  }

  Future<Map<String, dynamic>?> _pickKPI(List<Map<String, dynamic>> kpis) async {
    return showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      builder: (ctx) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            const Padding(
              padding: EdgeInsets.all(12),
              child: Text('选择需补料的育人指标（一键生成催办工单）',
                  style: TextStyle(fontWeight: FontWeight.bold)),
            ),
            for (final k in kpis)
              ListTile(
                leading: const Icon(Icons.add_task, color: Colors.orange),
                title: Text('${k['label'] ?? ''}'),
                subtitle: Text(
                    '${k['source_desc'] ?? ''}\n${k['upload_hint'] ?? ''}',
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis),
                onTap: () => Navigator.pop(ctx, k),
              ),
          ],
        ),
      ),
    );
  }

  Future<String?> _pickPriority() {
    return showDialog<String>(
      context: context,
      builder: (ctx) => SimpleDialog(
        title: const Text('选择督办优先级'),
        children: [
          for (final e in {'normal': '普通', 'high': '高', 'low': '低'}.entries)
            SimpleDialogOption(
              onPressed: () => Navigator.pop(ctx, e.key),
              child: Text('${e.value}（${e.key}）'),
            ),
        ],
      ),
    );
  }

  // 状态快捷推进：pending->processing->completed（书记可任意）
  Future<void> _advance(SecretaryProvider p, GovTicket t) async {
    final target = switch (t.status) {
      'pending' => 'processing',
      'processing' => 'completed',
      _ => null,
    };
    if (target == null) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('该工单已完结')));
      return;
    }
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('推进「${t.title}」'),
        content: Text('将状态从「${t.statusLabel}」更新为「${_statusText(target)}」，确定？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确定')),
        ],
      ),
    );
    if (confirm != true || !mounted) return;
    final result = await context
        .read<SecretaryProvider>()
        .updateTicketStatus(id: t.id, status: target, asManager: true);
    if (!mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(result.ok ? '已推进' : result.msg)));
    if (result.ok) _reload(context.read<SecretaryProvider>());
  }

  static String _statusText(String s) => switch (s) {
        'processing' => '处理中',
        'completed' => '已完成',
        'closed' => '已关闭',
        _ => s,
      };

  // 分派责任人：选择角色 + 录入责任人 id 与姓名（后端落 assignee_id/name/role）
  Future<void> _assign(SecretaryProvider p, GovTicket t) async {
    if (t.status == 'completed' || t.status == 'closed') {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已完结工单不可再分派')));
      return;
    }
    final role = await showDialog<String>(
      context: context,
      builder: (ctx) => SimpleDialog(
        title: const Text('选择责任人角色'),
        children: [
          for (final e in {'counselor': '辅导员', 'assistant': '教辅', 'party': '党群'}.entries)
            SimpleDialogOption(
              onPressed: () => Navigator.pop(ctx, e.key),
              child: Text(e.value),
            ),
        ],
      ),
    );
    if (role == null || !mounted) return;

    final idCtl = TextEditingController(text: t.assigneeId > 0 ? '${t.assigneeId}' : '');
    final nameCtl = TextEditingController(text: t.assigneeName);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('填入责任人（id + 姓名）'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: idCtl,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(hintText: '责任人用户 id（数字）'),
            ),
            TextField(
              controller: nameCtl,
              decoration: const InputDecoration(hintText: '责任人姓名'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确定')),
        ],
      ),
    );
    final id = int.tryParse(idCtl.text.trim());
    final name = nameCtl.text.trim();
    if (ok != true || id == null || id <= 0 || name.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请填写有效的责任人 id 与姓名')));
      return;
    }
    if (!mounted) return;
    final result = await context.read<SecretaryProvider>().assignTicket(
          id: t.id,
          assigneeId: id,
          assigneeName: name,
          assigneeRole: role,
        );
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(result.ok ? '分派成功' : '分派失败：${result.msg}')));
    if (result.ok) _reload(context.read<SecretaryProvider>());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('治理督办工单'),
        actions: [
          IconButton(
            tooltip: '从育人KPI生成补料工单',
            onPressed: () =>
                _openCreateFromKPI(context.read<SecretaryProvider>()),
            icon: const Icon(Icons.add_task),
          ),
          IconButton(
            tooltip: '刷新',
            onPressed: () => _reload(context.read<SecretaryProvider>()),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openCreateFromKPI(context.read<SecretaryProvider>()),
        icon: const Icon(Icons.add_task),
        label: const Text('生成督办工单'),
      ),
      body: Consumer<SecretaryProvider>(
        builder: (_, p, __) {
          if (p.ticketsLoading && p.tickets.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (p.tickets.isEmpty) {
            return ErrorView.empty(
                message: '暂无督办工单\n点击右下角从育人KPI一键生成补料工单');
          }
          return RefreshIndicator(
            onRefresh: () => _reload(p),
            child: ListView(
              padding: const EdgeInsets.all(12),
              children: [
                _StatsBar(stats: p.ticketStats),
                const SizedBox(height: 8),
                _FilterBar(
                  status: _statusFilter,
                  category: _categoryFilter,
                  onStatus: (v) {
                    setState(() => _statusFilter = v);
                    _reload(p);
                  },
                  onCategory: (v) {
                    setState(() => _categoryFilter = v);
                    _reload(p);
                  },
                ),
                const SizedBox(height: 4),
                for (final t in p.tickets) _TicketCard(ticket: t, onAdvance: () => _advance(p, t), onAssign: () => _assign(p, t)),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _StatsBar extends StatelessWidget {
  final Map<String, int> stats;
  const _StatsBar({required this.stats});

  @override
  Widget build(BuildContext context) {
    final labels = {
      'pending': ('待办', Colors.orange),
      'processing': ('处理中', Colors.blue),
      'completed': ('已完成', Colors.green),
      'closed': ('已关闭', Colors.grey),
    };
    return Row(
      children: labels.entries.map((e) {
        final (label, color) = e.value;
        final v = stats[e.key] ?? 0;
        return Expanded(
          child: Card(
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Column(
                children: [
                  Text('$v',
                      style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: color)),
                  Text(label, style: const TextStyle(fontSize: 12)),
                ],
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}

class _FilterBar extends StatelessWidget {
  final String status;
  final String category;
  final ValueChanged<String> onStatus;
  final ValueChanged<String> onCategory;
  const _FilterBar(
      {required this.status,
      required this.category,
      required this.onStatus,
      required this.onCategory});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 4,
      runSpacing: 4,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        _chip('全部状态', status.isEmpty ? '' : 'all', status.isEmpty, (_) => onStatus('')),
        _chip('待办', 'pending', status == 'pending', (_) => onStatus('pending')),
        _chip('处理中', 'processing', status == 'processing',
            (_) => onStatus('processing')),
        _chip('已完成', 'completed', status == 'completed',
            (_) => onStatus('completed')),
        const SizedBox(width: 8),
        _chip('补料类', 'supplement', category == 'supplement',
            (_) => onCategory(category == 'supplement' ? '' : 'supplement')),
      ],
    );
  }

  Widget _chip(String label, String value, bool selected, ValueChanged<bool> cb) {
    return ChoiceChip(
      label: Text(label, style: const TextStyle(fontSize: 12)),
      selected: selected,
      onSelected: cb,
    );
  }
}

class _TicketCard extends StatelessWidget {
  final GovTicket ticket;
  final VoidCallback onAdvance;
  final VoidCallback onAssign;
  const _TicketCard(
      {required this.ticket, required this.onAdvance, required this.onAssign});

  @override
  Widget build(BuildContext context) {
    final color = switch (ticket.status) {
      'pending' => Colors.orange,
      'processing' => Colors.blue,
      'completed' => Colors.green,
      'closed' => Colors.grey,
      _ => Colors.grey,
    };
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                      color: color.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(6)),
                  child: Text(ticket.statusLabel,
                      style: TextStyle(color: color, fontSize: 12)),
                ),
                const SizedBox(width: 6),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                      color: Colors.indigo.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(6)),
                  child: Text(ticket.categoryLabel,
                      style: const TextStyle(
                          color: Colors.indigo, fontSize: 12)),
                ),
                const Spacer(),
                if (ticket.priority == 'high')
                  const Icon(Icons.priority_high,
                      color: Colors.red, size: 16),
              ],
            ),
            const SizedBox(height: 6),
            Text(ticket.title,
                style: const TextStyle(
                    fontWeight: FontWeight.bold, fontSize: 14)),
            const SizedBox(height: 4),
            Text(ticket.remark.isNotEmpty
                ? ticket.remark
                : ticket.sourceDesc,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
            const SizedBox(height: 8),
            Row(
              children: [
                Icon(Icons.person, size: 14, color: Colors.grey.shade500),
                const SizedBox(width: 2),
                Text(ticket.assigneeName.isEmpty ? '未分派' : ticket.assigneeName,
                    style: const TextStyle(fontSize: 12)),
                for (final (k, v, cb) in [
                  ('推进', ticket.status != 'completed' && ticket.status != 'closed', onAdvance),
                  ('分派', ticket.status != 'completed' && ticket.status != 'closed', onAssign),
                ])
                  if (v)
                    Padding(
                      padding: const EdgeInsets.only(left: 8),
                      child: OutlinedButton(
                        onPressed: cb,
                        style: OutlinedButton.styleFrom(
                            minimumSize: const Size(0, 30),
                            padding: const EdgeInsets.symmetric(horizontal: 10)),
                        child: Text(k, style: const TextStyle(fontSize: 12)),
                      ),
                    ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
