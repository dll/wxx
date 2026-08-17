import 'package:flutter/material.dart';
import 'data_src_badge.dart';

/// 书记专项可视化共享区块（D1-1 功能补齐，2026-08-16）
///
/// 从 `secretary_outcome_dashboard_page.dart` 的私有渲染逻辑抽取并公开，
/// 供「教育成果大屏」聚合页与「党建育人专项页 / 协同育人专项页」复用，
/// 避免重造渲染逻辑。仅做数据 → 卡片渲染，不拉取数据、不改业务逻辑。
///
/// - [PartyDashboardSection]：党建育人（思想政治）区块
/// - [CollabDashboardSection]：协同育人总览区块
/// - （复用）[DashboardSectionCard] / [DashboardStatRow] / [DashboardChipRow]
///
/// 数据来源：沿用 `data_source` 三态诚实边界，经 [DataSrcBadge] 呈现，
/// 绝不伪造数值。

/// 区块通用卡片容器（由原 _SectionCard 公开化）。
class DashboardSectionCard extends StatelessWidget {
  final String title;
  final IconData icon;
  final String src;
  final List<Widget> children;
  const DashboardSectionCard({
    super.key,
    required this.title,
    required this.icon,
    required this.src,
    required this.children,
  });

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

/// 标签-值 行（由原 _StatRow 公开化）。
class DashboardStatRow extends StatelessWidget {
  final String label;
  final String value;
  final bool big;
  const DashboardStatRow({
    super.key,
    required this.label,
    required this.value,
    this.big = false,
  });

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

/// 标签 chip 组（由原 _ChipRow 公开化）。
class DashboardChipRow extends StatelessWidget {
  final Map<String, String> items;
  const DashboardChipRow({super.key, required this.items});

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

/// 党建育人（思想政治）区块。由原 `_buildParty` 抽取。
///
/// [party] 为 `SecretaryProvider.partyDashboard`（Map，含 members / stage_distribution /
/// stage_total / study_records / study_hours / study_by_type / data_source）。
/// 非 Map 或缺失时返回空容器。
class PartyDashboardSection extends StatelessWidget {
  final dynamic party;
  const PartyDashboardSection({super.key, required this.party});

  /// 阶段中文名映射。
  static const Map<String, String> _stageNames = {
    'applicant': '申请入党',
    'activist': '入党积极分子',
    'development': '发展对象',
    'probation': '预备党员',
    'member': '正式党员',
  };

  @override
  Widget build(BuildContext context) {
    final party = this.party;
    if (party is! Map) return const SizedBox.shrink();
    final members = (party['members'] as Map?)?.cast<String, dynamic>() ?? {};
    final stage = (party['stage_distribution'] as Map?)?.cast<String, dynamic>() ?? {};
    final studyByType = (party['study_by_type'] as Map?)?.cast<String, dynamic>() ?? {};
    final stageTotal = (party['stage_total'] as num?)?.toInt() ?? 0;
    final studyCount = (party['study_records'] as num?)?.toInt() ?? 0;
    final studyHours = (party['study_hours'] as num?)?.toInt() ?? 0;
    return DashboardSectionCard(
      title: '党建育人（思想政治）',
      icon: Icons.flag,
      src: '${party['data_source']}',
      children: [
        DashboardStatRow(label: '入党申请总人数', value: '$stageTotal'),
        DashboardStatRow(label: '正式党员', value: '${members['member'] ?? 0}'),
        DashboardStatRow(label: '预备党员', value: '${members['probation'] ?? 0}'),
        if (stage.isNotEmpty)
          DashboardChipRow(items: {
            for (final e in stage.entries) _stageNames[e.key] ?? e.key: '${e.value}',
          }),
        DashboardStatRow(label: '党课/学习记录', value: '$studyCount 人次'),
        DashboardStatRow(label: '学习时长', value: '$studyHours 小时'),
        if (studyByType.isNotEmpty)
          DashboardChipRow(items: {
            for (final e in studyByType.entries)
              e.key: '${e.value['count'] ?? 0} 人次',
          }),
      ],
    );
  }
}

/// 协同育人总览区块。由原 `_buildCollab` 抽取。
///
/// [collab] 为 `SecretaryProvider.collabDashboard`（Map，含 students_total /
/// talk_records / facility_records / party_registrations / course_schedules /
/// by_role / data_source）。非 Map 或缺失时返回空容器。
class CollabDashboardSection extends StatelessWidget {
  final dynamic collab;
  const CollabDashboardSection({super.key, required this.collab});

  /// 角色中文名映射。
  static const Map<String, String> _roleNames = {
    'counselor': '辅导员',
    'teacher': '教师',
    'assistant': '教辅',
    'student_union': '学生会',
    'college_admin': '学院管理员',
  };

  @override
  Widget build(BuildContext context) {
    final collab = this.collab;
    if (collab is! Map) return const SizedBox.shrink();
    final src = '${collab['data_source']}';
    final roleSum = (collab['by_role'] as Map?)?.cast<String, dynamic>() ?? {};
    return DashboardSectionCard(
      title: '协同育人总览（教师/教辅付出）',
      icon: Icons.groups,
      src: src,
      children: [
        DashboardStatRow(label: '本院学生数', value: '${collab['students_total'] ?? 0}'),
        DashboardStatRow(label: '谈心记录', value: '${collab['talk_records'] ?? 0}'),
        DashboardStatRow(label: '后勤服务', value: '${collab['facility_records'] ?? 0}'),
        DashboardStatRow(label: '党课/活动登记', value: '${collab['party_registrations'] ?? 0}'),
        DashboardStatRow(label: '教学课表节次', value: '${collab['course_schedules'] ?? 0}'),
        if (roleSum.isNotEmpty) ...[
          const SizedBox(height: 8),
          const Text('育人动作按角色',
              style: TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
          DashboardChipRow(items: {
            for (final e in roleSum.entries) _roleNames[e.key] ?? e.key: '${e.value}',
          }),
        ],
      ],
    );
  }
}
