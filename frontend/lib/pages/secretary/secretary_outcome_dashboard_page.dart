import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/secretary_provider.dart';
import '../../widgets/data_src_badge.dart';
import '../../widgets/error_view.dart';

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
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('教育成果大屏')),
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
            },
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _LifecycleHeader(college: d['college'] as String? ?? '全校'),
                const SizedBox(height: 12),
                _buildCompetition(d['competition']),
                const SizedBox(height: 12),
                _buildParty(provider.partyDashboard ?? d['party']),
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
              child: Text('书记教育成果 · ${college == '全校' ? '全校视角' : '$college'}',
                  style: const TextStyle(
                      fontSize: 16, fontWeight: FontWeight.bold)),
            ),
            const Icon(Icons.auto_graph, color: Colors.indigo),
          ],
        ),
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

  Widget _buildParty(dynamic party) {
    if (party is! Map) return const SizedBox.shrink();
    final members = (party['members'] as Map?)?.cast<String, dynamic>() ?? {};
    final stage = (party['stage_distribution'] as Map?)?.cast<String, dynamic>() ?? {};
    final studyByType = (party['study_by_type'] as Map?)?.cast<String, dynamic>() ?? {};
    final stageTotal = (party['stage_total'] as num?)?.toInt() ?? 0;
    final studyCount = (party['study_records'] as num?)?.toInt() ?? 0;
    final studyHours = (party['study_hours'] as num?)?.toInt() ?? 0;
    // 阶段中文名
    const stageNames = {
      'applicant': '申请入党',
      'activist': '入党积极分子',
      'development': '发展对象',
      'probation': '预备党员',
      'member': '正式党员',
    };
    return _SectionCard(
      title: '党建育人（思想政治）',
      icon: Icons.flag,
      src: '${party['data_source']}',
      children: [
        _StatRow(label: '入党申请总人数', value: '$stageTotal'),
        _StatRow(label: '正式党员', value: '${members['member'] ?? 0}'),
        _StatRow(label: '预备党员', value: '${members['probation'] ?? 0}'),
        if (stage.isNotEmpty)
          _ChipRow(items: {
            for (final e in stage.entries)
              stageNames[e.key] ?? e.key: '${e.value}',
          }),
        _StatRow(label: '党课/学习记录', value: '$studyCount 人次'),
        _StatRow(label: '学习时长', value: '$studyHours 小时'),
        if (studyByType.isNotEmpty)
          _ChipRow(items: {
            for (final e in studyByType.entries)
              e.key: '${e.value['count'] ?? 0} 人次',
          }),
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
      return _SectionCard(
        title: '毕业去向（就业·升学）',
        icon: Icons.work,
        src: 'not_available',
        children: const [
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
