import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/health_provider.dart';

/// 学生会工作台：集中呈现原「无前端入口」的学生会职能。
/// 1. 成员活跃度（真实报名/到场统计）
/// 2. 活动数据分析（真实报名/到场率）
/// 3. 招新助手 / 问卷生成 / 热点追踪（AI / 参考方案）
/// 数据均来自后端 /union/* 接口；真实统计标 data_source=real，其余标 reference 或 ai。
class UnionWorkbenchPage extends StatefulWidget {
  const UnionWorkbenchPage({super.key});
  @override
  State<UnionWorkbenchPage> createState() => _UnionWorkbenchPageState();
}

class _UnionWorkbenchPageState extends State<UnionWorkbenchPage> {
  Map<String, dynamic>? _members;
  Map<String, dynamic>? _analysis;
  Map<String, dynamic>? _hot;
  final _analysisCtrl = TextEditingController();
  final _recruitCtrl = TextEditingController(text: '计算机学院');
  final _topicCtrl = TextEditingController(text: '校园活动');
  Map<String, dynamic>? _recruit;
  Map<String, dynamic>? _questionnaire;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadAll());
  }

  @override
  void dispose() {
    _analysisCtrl.dispose();
    _recruitCtrl.dispose();
    _topicCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadAll() async {
    final p = context.read<HealthProvider>();
    setState(() => _loading = true);
    final members = await p.fetchUnionMembers();
    final hot = await p.fetchUnionHotTopics();
    if (mounted) {
      setState(() {
        _members = members;
        _hot = hot;
        _loading = false;
      });
    }
  }

  void _srcBadge(BuildContext c, String? src) {
    if (src == null) return;
    final real = src == 'real';
    showDialog(
      context: c,
      builder: (ctx) => AlertDialog(
        title: const Text('数据来源'),
        content: Text(real
            ? '✅ 真实数据：来自系统内活动报名/签到记录实时统计，非模拟。'
            : '⚠️ 参考/AI 生成：当前无对应真实数据源（招新报名表/问卷库等），内容为模板或 AI 生成，仅作参考。'),
        actions: [TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('知道了'))],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('学生会工作台')),
      body: RefreshIndicator(
        onRefresh: _loadAll,
        child: ListView(
          padding: const EdgeInsets.all(12),
          children: [
            _buildMembersCard(theme),
            const SizedBox(height: 10),
            _buildAnalysisCard(theme),
            const SizedBox(height: 10),
            _buildRecruitCard(theme),
            const SizedBox(height: 10),
            _buildQuestionnaireCard(theme),
            const SizedBox(height: 10),
            _buildHotCard(theme),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  Widget _srcRow(ThemeData theme, String? src) => Row(
        children: [
          const Spacer(),
          InkWell(
            onTap: src == null ? null : () => _srcBadge(context, src),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: (src == 'real' ? Colors.green : Colors.amber).withOpacity(0.12),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Text(src == 'real' ? '真实数据' : '参考/AI',
                  style: TextStyle(fontSize: 11,
                      color: src == 'real' ? Colors.green.shade700 : Colors.orange.shade800)),
            ),
          ),
        ],
      );

  // ── 1. 成员活跃度（真实数据）──
  Widget _buildMembersCard(ThemeData theme) {
    final members = (_members?['members'] as List?)?.cast<Map>() ?? [];
    final stats = (_members?['stats'] as Map?) ?? {};
    return _card(theme, title: '成员活跃度', icon: Icons.people_outline, badge: _members?['data_source'],
      body: _loading || members.isEmpty
          ? Text(members.isEmpty ? '暂无报名/到场记录，待活动产生数据后自动统计。' : '加载中…',
              style: theme.textTheme.bodySmall)
          : Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Wrap(spacing: 16, children: [
                _stat(theme, '${stats['total'] ?? 0}', '参与人数'),
                _stat(theme, '${stats['active'] ?? 0}', '较活跃(B+)'),
                _stat(theme, '${stats['excellent'] ?? 0}', '优秀(A)'),
              ]),
              const SizedBox(height: 8),
              ...members.take(30).map((m) => Padding(
                    padding: const EdgeInsets.symmetric(vertical: 3),
                    child: Row(children: [
                      Expanded(child: Text('${m['name']}', style: theme.textTheme.bodyMedium,
                          maxLines: 1, overflow: TextOverflow.ellipsis)),
                      Text('报名 ${m['signups']} · 到场 ${m['attends']}',
                          style: theme.textTheme.bodySmall),
                      const SizedBox(width: 8),
                      _perfChip(theme, '${m['performance']}'),
                    ]),
                  )),
            ]),
    );
  }

  Widget _perfChip(ThemeData theme, String p) {
    final color = p == 'A' ? Colors.green : (p == 'B' ? Colors.blue : Colors.grey);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(color: color.withOpacity(0.12), borderRadius: BorderRadius.circular(8)),
      child: Text(p, style: TextStyle(fontSize: 11, color: color.shade700, fontWeight: FontWeight.w700)),
    );
  }

  // ── 2. 活动数据分析 ──
  Widget _buildAnalysisCard(ThemeData theme) {
    return _card(theme, title: '活动数据分析', icon: Icons.bar_chart,
        body: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Expanded(
              child: TextField(
                controller: _analysisCtrl,
                decoration: const InputDecoration(
                  hintText: '输入活动名（如 新生杯足球赛）', isDense: true, border: OutlineInputBorder()),
              ),
            ),
            const SizedBox(width: 8),
            FilledButton(
              onPressed: () async {
                if (_analysisCtrl.text.trim().isEmpty) return;
                final d = await context.read<HealthProvider>()
                    .fetchActivityAnalysis(_analysisCtrl.text.trim());
                if (mounted) setState(() => _analysis = d);
              },
              child: const Text('分析'),
            ),
          ]),
          if (_analysis != null) ...[
            const SizedBox(height: 10),
            _srcRow(theme, _analysis?['data_source']),
            Text('${_analysis?['event_name'] ?? ''}',
                style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Wrap(spacing: 16, children: [
              _stat(theme, _fmtRate(_analysis?['attend_rate']), '到场率'),
              _stat(theme, _fmtRate(_analysis?['reg_rate']), '报名(或报名率)'),
            ]),
            const SizedBox(height: 4),
            Text('${_analysis?['report'] ?? ''}', style: theme.textTheme.bodySmall),
            if ((_analysis?['suggestions'] as List?)?.isNotEmpty ?? false) ...[
              const SizedBox(height: 6),
              ...(_analysis?['suggestions'] as List).map((s) => Padding(
                    padding: const EdgeInsets.symmetric(vertical: 2),
                    child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Text('• ', style: theme.textTheme.bodySmall),
                      Expanded(child: Text('$s', style: theme.textTheme.bodySmall)),
                    ]),
                  )),
            ],
          ] else
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text('输入活动名查询真实报名/到场分析。', style: theme.textTheme.bodySmall),
            ),
        ]),
    );
  }

  String _fmtRate(dynamic v) {
    if (v == null) return '0';
    final f = v is num ? v.toDouble() : double.tryParse('$v') ?? 0;
    // 后端返回 0-100 的百分比数字
    return '${f.toStringAsFixed(f == f.roundToDouble() ? 0 : 1)}%';
  }

  // ── 3. 招新助手 ──
  Widget _buildRecruitCard(ThemeData theme) {
    final stages = (_recruit?['stages'] as List?)?.cast<Map>() ?? [];
    return _card(theme, title: '招新助手', icon: Icons.campaign_outlined, badge: _recruit?['data_source'],
        body: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Expanded(child: TextField(controller: _recruitCtrl,
                decoration: const InputDecoration(hintText: '部门/学院', isDense: true, border: OutlineInputBorder()))),
            const SizedBox(width: 8),
            OutlinedButton(
              onPressed: () async {
                final d = await context.read<HealthProvider>()
                    .fetchUnionRecruitment(_recruitCtrl.text.trim());
                if (mounted) setState(() => _recruit = d);
              },
              child: const Text('生成方案'),
            ),
          ]),
          const SizedBox(height: 8),
          Text('${_recruit?['plan'] ?? '输入部门生成招新方案（参考模板）'}',
              style: theme.textTheme.bodySmall),
          if (stages.isNotEmpty)
            ...stages.map((s) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Text('• ${s['stage']}（${s['duration']}）：${(s['actions'] as List? ?? []).join('、')}',
                      style: theme.textTheme.bodySmall),
                )),
        ]),
    );
  }

  // ── 4. 问卷生成 ──
  Widget _buildQuestionnaireCard(ThemeData theme) {
    final qs = (_questionnaire?['questions'] as List?)?.cast<Map>() ?? [];
    return _card(theme, title: '问卷生成', icon: Icons.quiz_outlined, badge: _questionnaire?['data_source'],
        body: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Expanded(child: TextField(controller: _topicCtrl,
                decoration: const InputDecoration(hintText: '主题', isDense: true, border: OutlineInputBorder()))),
            const SizedBox(width: 8),
            OutlinedButton(
              onPressed: () async {
                final d = await context.read<HealthProvider>()
                    .fetchUnionQuestionnaire(_topicCtrl.text.trim());
                if (mounted) setState(() => _questionnaire = d);
              },
              child: const Text('生成问卷'),
            ),
          ]),
          const SizedBox(height: 8),
          if (qs.isNotEmpty) ...[
            Text('${_questionnaire?['title'] ?? ''}', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
            ...qs.asMap().entries.map((e) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Text('${e.key + 1}. ${e.value['q']}', style: theme.textTheme.bodySmall),
                )),
          ] else
            Text('输入主题生成问卷题目（参考模板）。', style: theme.textTheme.bodySmall),
        ]),
    );
  }

  // ── 5. 热点追踪 ──
  Widget _buildHotCard(ThemeData theme) {
    final topics = (_hot?['topics'] as List?)?.cast<Map>() ?? [];
    final sugg = (_hot?['suggestions'] as List?) ?? [];
    return _card(theme, title: '热点追踪', icon: Icons.local_fire_department_outlined, badge: _hot?['data_source'],
        body: _loading
            ? Text('加载中…', style: theme.textTheme.bodySmall)
            : topics.isEmpty
                ? Text('暂无热点数据。', style: theme.textTheme.bodySmall)
                : Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    ...topics.map((t) => Padding(
                          padding: const EdgeInsets.symmetric(vertical: 3),
                          child: Row(children: [
                            Expanded(child: Text('${t['topic']}', style: theme.textTheme.bodyMedium)),
                            Text('热度 ${t['heat']}',
                                style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.primary)),
                          ]),
                        )),
                    if (sugg.isNotEmpty) ...[
                      const Divider(height: 16),
                      ...sugg.map((s) => Text('• $s', style: theme.textTheme.bodySmall)),
                    ],
                  ]),
    );
  }

  Widget _stat(ThemeData theme, String v, String l) => Row(mainAxisSize: MainAxisSize.min, children: [
        Text(v, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800, color: theme.colorScheme.primary)),
        const SizedBox(width: 3),
        Text(l, style: theme.textTheme.bodySmall),
      ]);

  Widget _card(ThemeData theme, {required String title, required IconData icon, String? badge,
      required Widget body}) {
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(icon, size: 18, color: theme.colorScheme.primary),
            const SizedBox(width: 6),
            Text(title, style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
            const Spacer(),
            if (badge != null)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
                decoration: BoxDecoration(
                  color: (badge == 'real' ? Colors.green : Colors.amber).withOpacity(0.12),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: InkWell(
                  onTap: () => _srcBadge(context, badge),
                  child: Text(badge == 'real' ? '真实数据' : '参考/AI',
                      style: TextStyle(fontSize: 11, color: badge == 'real' ? Colors.green.shade700 : Colors.orange.shade800)),
                ),
              ),
          ]),
          const SizedBox(height: 10),
          body,
        ]),
      ),
    );
  }
}
