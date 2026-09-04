import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/data_src_badge.dart';
import '../../widgets/error_view.dart';
import '../../widgets/secretary_dashboard_sections.dart';

/// 书记教育成果大屏（school_admin 全校 / college_admin 本院）
///
/// 一屏看「入学→在校→离校→就业」全生命周期育人成果：
/// 竞赛获奖 / 入党 / 学业 / 谈心 / 后勤 / 毕业去向（就业率·考研率）。
/// 全部真实数据聚合；就业率/考研率在无真实数据时诚实显示「待接入」。
class SecretaryOutcomeDashboardPage extends StatefulWidget {
  const SecretaryOutcomeDashboardPage({super.key});

  @override
  State<SecretaryOutcomeDashboardPage> createState() =>
      _SecretaryOutcomeDashboardPageState();
}

class _SecretaryOutcomeDashboardPageState
    extends State<SecretaryOutcomeDashboardPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SecretaryProvider>().fetchDashboard();
      context.read<SecretaryProvider>().fetchPartyDashboard();
      context.read<SecretaryProvider>().fetchCollabDashboard();
      context.read<SecretaryProvider>().fetchNurtureKPI();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('教育成果大屏'),
        actions: [
          IconButton(
            tooltip: '治理督办工单',
            icon: const Icon(Icons.assignment_late_outlined),
            onPressed: () =>
                GoRouter.of(context).go('/secretary/ticket-manage'),
          ),
        ],
      ),
      body: Consumer<SecretaryProvider>(
        builder: (_, provider, __) {
          if (provider.dashboardLoading && provider.dashboard == null) {
            return const Center(child: CircularProgressIndicator());
          }
          if (provider.error.isNotEmpty && provider.dashboard == null) {
            return ErrorView.error(
              message: provider.error,
              onRetry: () => provider.fetchDashboard(),
            );
          }
          final d = provider.dashboard;
          if (d == null) return ErrorView.empty(message: '暂无数据');
          return RefreshIndicator(
            onRefresh: () async {
              await provider.fetchDashboard();
              await provider.fetchPartyDashboard();
              await provider.fetchCollabDashboard();
              await provider.fetchNurtureKPI();
            },
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _LifecycleHeader(college: d['college'] as String? ?? '全校'),
                const SizedBox(height: 12),
                _buildKPICards(provider.nurtureKPIs),
                const SizedBox(height: 12),
                _buildCompetition(d['competition']),
                const SizedBox(height: 12),
                PartyDashboardSection(
                    party: provider.partyDashboard ?? d['party']),
                const SizedBox(height: 12),
                CollabDashboardSection(collab: provider.collabDashboard),
                const SizedBox(height: 12),
                _buildAcademic(d['academic']),
                const SizedBox(height: 12),
                _buildCare(d['counseling'], d['facility']),
                const SizedBox(height: 12),
                _buildOutcome(d['outcome'], d['meta']),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _LifecycleHeader({required String college}) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            Icon(Icons.school, color: Colors.indigo.shade700),
            const SizedBox(width: 8),
            Expanded(
              child: Text('书记教育成果 · ${college == '全校' ? '全校视角' : college}',
                  style: const TextStyle(
                      fontSize: 16, fontWeight: FontWeight.bold)),
            ),
            const Icon(Icons.auto_graph, color: Colors.indigo),
          ],
        ),
      ),
    );
  }

  // 育人成效 KPI 指标卡（D5-1 功能补齐，2026-08-16）。
  // 每项：{ key, label, value, unit, data_source(real/trend/not_available), source_desc, upload_target, upload_hint }。
  // real → 显示真实数值；trend → 五维纵向趋势条（Δ箭头+样本量）；
  // not_available → 显示「数据待补充」+ 上传支撑材料入口（绝不伪造数字）。
  // 特殊（P1-2 诚实修正）：key==nurture.growth_trend 的 not_available 靠时间积累而非补料，
  // 隐藏「上传材料/生成工单」两入口，只显「数据积累中，需满 N 周」提示。
  Widget _buildKPICards(List<Map<String, dynamic>> kpis) {
    if (kpis.isEmpty) return const SizedBox.shrink();
    return _SectionCard(
      title: '育人成效指标',
      icon: Icons.insights,
      src: '',
      children: [
        for (final k in kpis)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 6),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('${k['label'] ?? ''}',
                          style: const TextStyle(
                              fontWeight: FontWeight.bold, fontSize: 13)),
                      const SizedBox(height: 2),
                      Text(_kpiSourceDesc(k['source_desc']),
                          style: TextStyle(
                              fontSize: 11, color: Colors.grey.shade600)),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                if (k['data_source'] == 'real')
                  Text(
                    '${k['value'] ?? '-'}${k['unit'] ?? ''}',
                    style: TextStyle(
                        fontWeight: FontWeight.bold,
                        fontSize: 16,
                        color: Colors.indigo.shade700),
                  )
                else if (k['data_source'] == 'trend')
                  // 真实纵向趋势：五维 Δ 箭头 + 样本量（仅趋势/相关性，不作因果）
                  _buildTrendValues(k)
                else if (k['key'] == 'nurture.growth_trend')
                  // growth_trend 靠纵向时间积累而非补料：诚实提示，不给补料/工单入口
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      const Text('数据积累中，需满 4 周',
                          style: TextStyle(
                              color: Colors.orange, fontSize: 12)),
                      const SizedBox(height: 2),
                      Text('系统已开始对数字孪生快照做历史留痕，\n连续记录满 4 周后生成成长趋势。',
                          textAlign: TextAlign.right,
                          style: TextStyle(
                              fontSize: 11, color: Colors.grey.shade600)),
                    ],
                  )
                else
                  // not_available：数据待补充 + 上传支撑材料（复用 kb 上传心智，不伪造数字）
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      const Text('数据待补充',
                          style: TextStyle(
                              color: Colors.orange, fontSize: 12)),
                      const SizedBox(height: 2),
                      InkWell(
                        onTap: () => _uploadNurtureMaterial(k),
                        child: const Text('上传材料到知识库',
                            style: TextStyle(
                                color: Colors.indigo,
                                fontSize: 12,
                                decoration: TextDecoration.underline)),
                      ),
                      const SizedBox(height: 2),
                      InkWell(
                        onTap: () => _createSupplementTicket(k),
                        child: const Text('生成补料督办工单',
                            style: TextStyle(
                                color: Colors.deepOrange,
                                fontSize: 12,
                                decoration: TextDecoration.underline)),
                      ),
                    ],
                  ),
              ],
            ),
          ),
      ],
    );
  }

  // trend 卡的 value 为五维 Δ map；渲染五根横向趋势条（Δ箭头 + 样本量）。
  // sample_count==0 时后端已回落 not_available，此处只处理真实有数据的情况。
  Widget _buildTrendValues(Map<String, dynamic> k) {
    final val = k['value'] as Map<String, dynamic>?;
    final sample = (k['sample_count'] as int?) ?? 0;
    const dims = {
      'academic': '学业',
      'ability': '能力',
      'ideological': '思想',
      'emotional': '情感',
      'social': '社交',
    };
    final rows = <Widget>[];
    dims.forEach((key, name) {
      final delta = (val?[key] as num?)?.toDouble() ?? 0.0;
      rows.add(Padding(
        padding: const EdgeInsets.only(bottom: 2),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(name, style: const TextStyle(fontSize: 11)),
            const SizedBox(width: 4),
            if (delta.abs() < 1e-6)
              const Text('→', style: TextStyle(fontSize: 12, color: Colors.grey))
            else if (delta > 0)
              Text('↑${delta.toStringAsFixed(1)}',
                  style: const TextStyle(
                      fontSize: 12,
                      color: Colors.green,
                      fontWeight: FontWeight.bold))
            else
              Text('↓${delta.abs().toStringAsFixed(1)}',
                  style: const TextStyle(
                      fontSize: 12,
                      color: Colors.redAccent,
                      fontWeight: FontWeight.bold)),
          ],
        ),
      ));
    });
    rows.add(Text('基于 $sample 名有历史学生样本',
        style: TextStyle(fontSize: 10, color: Colors.grey.shade600)));
    return Column(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: rows,
    );
  }

  String _kpiSourceDesc(dynamic srcDesc) {
    final s = srcDesc as String? ?? '';
    return s.isEmpty ? '数据来源：以真实登记表为准' : s;
  }

  // not_available KPI 的上传支撑材料入口：按 upload_target 跳转（kb 上传心智）。
  void _uploadNurtureMaterial(Map<String, dynamic> kpi) {
    final target = kpi['upload_target'] as String? ?? _kpiUploadTarget();
    // 暂以知识库上传为落地（复用 kb/upload 心智）
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('请上传「${kpi['label'] ?? '该指标'}」的支撑材料到知识库，审核通过后转为真实数据'),
        duration: const Duration(seconds: 3),
      ),
    );
    debugPrint('[D5-1] nurture KPI upload target: $target');
  }

  String _kpiUploadTarget() => '/api/v1/kb/upload';

  // D5-3「洞察→工单」：从 not_available 补料 KPI 生成督办工单（D5-1 联动）。
  void _createSupplementTicket(Map<String, dynamic> kpi) {
    final key = kpi['key'] as String? ?? '';
    if (key.isEmpty) return;
    final prov = context.read<SecretaryProvider>();
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('生成补料督办工单'),
        content: Text('确定生成督办工单「${kpi['label'] ?? ''}」，催办上传支撑材料到知识库？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () async {
              Navigator.pop(ctx);
              final result = await prov.createTicketFromKPI(kpiKey: key);
              if (!ctx.mounted) return;
              ScaffoldMessenger.of(ctx).showSnackBar(SnackBar(
                content: Text(result.ok
                    ? '已生成补料督办工单：${kpi['label']}'
                    : '生成失败：${result.msg}'),
              ));
            },
            child: const Text('生成'),
          ),
        ],
      ),
    );
  }

  Widget _buildCompetition(dynamic comp) {
    if (comp is! Map) return const SizedBox.shrink();
    final total = (comp['total_awards'] ?? 0).toString();
    final byLevel = (comp['by_level'] as Map?)?.cast<String, dynamic>() ?? {};
    final byNat = (comp['by_nationality'] as Map?)?.cast<String, dynamic>() ?? {};
    final advisors = (comp['advisor_rank'] as List?) ?? [];
    return _SectionCard(
      title: '学科竞赛获奖',
      icon: Icons.emoji_events,
      src: '${comp['data_source']}',
      children: [
        _StatRow(label: '获奖总数', value: total, big: true),
        _ChipRow(
          items: {
            '国家级': '${byNat['national'] ?? 0}',
            '省级': '${byNat['provincial'] ?? 0}',
            '校级': '${byNat['school'] ?? 0}',
          },
        ),
        if ((byLevel['first'] ?? 0) > 0 ||
            (byLevel['second'] ?? 0) > 0 ||
            (byLevel['third'] ?? 0) > 0)
          _ChipRow(items: {
            '一等奖': '${byLevel['first'] ?? 0}',
            '二等奖': '${byLevel['second'] ?? 0}',
            '三等奖': '${byLevel['third'] ?? 0}',
          }),
        if (advisors.isNotEmpty) ...[
          const SizedBox(height: 8),
          const Text('指导教师榜（带竞赛）',
              style: TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
          const SizedBox(height: 4),
          ...advisors.take(5).map((a) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Row(
                  children: [
                    Icon(Icons.person, size: 16, color: Colors.orange.shade700),
                    const SizedBox(width: 4),
                    Expanded(
                        child: Text('${a['name']}',
                            style: const TextStyle(fontSize: 13))),
                    Text('${a['awards']} 项',
                        style: const TextStyle(
                            fontSize: 13, fontWeight: FontWeight.bold)),
                  ],
                ),
              )),
        ],
      ],
    );
  }

  Widget _buildAcademic(dynamic ac) {
    if (ac is! Map) return const SizedBox.shrink();
    return _SectionCard(
      title: '学业',
      icon: Icons.menu_book,
      src: '${ac['data_source']}',
      children: [
        _StatRow(
          label: '成绩记录数',
          value: '${ac['grade_count'] ?? 0}',
        ),
        _StatRow(
          label: '课程通过率',
          value: '${(ac['pass_rate'] as num? ?? 0).toStringAsFixed(1)}%',
          big: true,
        ),
      ],
    );
  }

  Widget _buildCare(dynamic counsel, dynamic facility) {
    final talk = (counsel is Map) ? '${counsel['talk_total'] ?? 0}' : '0';
    final fac = (facility is Map) ? '${facility['record_total'] ?? 0}' : '0';
    return _SectionCard(
      title: '育人协同（谈心·后勤）',
      icon: Icons.favorite,
      src: 'real',
      children: [
        _StatRow(label: '谈心记录', value: talk),
        _StatRow(label: '后勤服务记录', value: fac),
      ],
    );
  }

  Widget _buildOutcome(dynamic out, dynamic meta) {
    if (out is! Map) {
      return const _SectionCard(
        title: '毕业去向（就业·升学）',
        icon: Icons.work,
        src: 'not_available',
        children: [
          Text('毕业去向数据待接入（需教辅录入并经审核）',
              style: TextStyle(color: Colors.grey)),
        ],
      );
    }
    final src = '${out['data_source']}' == 'real' ? 'real' : 'not_available';
    final isNA = src != 'real';
    final typeMeta =
        (meta is Map && meta['outcome_types'] is Map)
            ? (meta['outcome_types'] as Map).cast<String, dynamic>()
            : <String, dynamic>{};
    final byType = (out['rate_by_type'] as Map?)?.cast<String, dynamic>() ?? {};
    return _SectionCard(
      title: '毕业去向（就业·升学）',
      icon: Icons.work,
      src: src,
      children: [
        if (isNA)
          const Text('毕业去向数据待接入（需教辅录入并经审核）',
              style: TextStyle(color: Colors.grey))
        else ...[
          _StatRow(
            label: '就业率',
            value: '${(out['employment_rate'] as num? ?? 0).toStringAsFixed(1)}%',
            big: true,
          ),
          _StatRow(
            label: '升学（考研/出国）率',
            value:
                '${(out['postgrad_rate'] as num? ?? 0).toStringAsFixed(1)}%',
            big: true,
          ),
          if (byType.isNotEmpty) ...[
            const SizedBox(height: 8),
            const Text('去向构成',
                style: TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
            ...byType.entries.map((e) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Row(
                    children: [
                      Expanded(
                          child: Text(
                              typeMeta[e.key] ?? e.key,
                              style: const TextStyle(fontSize: 13))),
                      Text('${e.value} 人',
                          style: const TextStyle(
                              fontSize: 13, fontWeight: FontWeight.bold)),
                    ],
                  ),
                )),
          ],
        ],
      ],
    );
  }
}

class _SectionCard extends StatelessWidget {
  final String title;
  final IconData icon;
  final String src;
  final List<Widget> children;
  const _SectionCard(
      {required this.title,
      required this.icon,
      required this.src,
      required this.children});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(
                    child: Text(title,
                        style: const TextStyle(
                            fontSize: 15, fontWeight: FontWeight.bold))),
                DataSrcBadge(src: src),
              ],
            ),
            const SizedBox(height: 10),
            ...children,
          ],
        ),
      ),
    );
  }
}

class _StatRow extends StatelessWidget {
  final String label;
  final String value;
  final bool big;
  const _StatRow(
      {required this.label, required this.value, this.big = false});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: const TextStyle(fontSize: 14)),
          Text(value,
              style: TextStyle(
                  fontSize: big ? 22 : 16,
                  fontWeight: FontWeight.bold,
                  color: big
                      ? Theme.of(context).colorScheme.primary
                      : Colors.black87)),
        ],
      ),
    );
  }
}

class _ChipRow extends StatelessWidget {
  final Map<String, String> items;
  const _ChipRow({required this.items});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Wrap(
        spacing: 8,
        children: items.entries
            .map((e) => Chip(
                  label: Text('${e.key} ${e.value}',
                      style: const TextStyle(fontSize: 12)),
                  visualDensity: VisualDensity.compact,
                  backgroundColor: Colors.blueGrey.shade50,
                ))
            .toList(),
      ),
    );
  }
}
