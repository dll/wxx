import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../utils/storage.dart';

class ServiceCenterPage extends StatelessWidget {
  const ServiceCenterPage({super.key});

  @override
  Widget build(BuildContext context) {
    final role = Storage.role;
    final groups = _groupsFor(role);

    return Scaffold(
      appBar: AppBar(title: const Text('服务中心')),
      body: ListView.separated(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 32),
        itemCount: groups.length,
        separatorBuilder: (_, __) => const SizedBox(height: 24),
        itemBuilder: (context, index) {
          final group = groups[index];
          return _ServiceGroup(group: group);
        },
      ),
    );
  }

  List<_ServiceGroupData> _groupsFor(String? role) {
    const common = _ServiceGroupData('校园服务', '导航 · 知识 · 应用', [
      const _ServiceItem('知识大厅', '政策、流程与可信来源',
          Icons.menu_book_outlined, '/browse', Color(0xFF1565C0)),
      const _ServiceItem('办事服务', '查看流程和办理进度',
          Icons.assignment_outlined, '/enrollment', Color(0xFF6A1B9A)),
      const _ServiceItem('校园导航', '地图、全景与校园入口',
          Icons.map_outlined, '/campus', Color(0xFF00838F)),
      const _ServiceItem('应用中心', '校内外系统快捷入口',
          Icons.apps_outlined, '/apps', Color(0xFF2E7D32)),
      const _ServiceItem('AI 简讯', '教学、工具与行业动态',
          Icons.newspaper_outlined, '/ai-briefings', Color(0xFFE65100)),
    ]);

    if (role == 'teacher') {
      return [
        const _ServiceGroupData('教学工具', '备课 · 出题 · 互动 · 分析', [
          _ServiceItem('AI 备课', '生成并完善教学方案',
              Icons.auto_awesome, '/teacher/lesson-prep', Color(0xFF1565C0)),
          _ServiceItem('AI 出题', '按课程目标生成试题',
              Icons.quiz_outlined, '/teacher/exam-gen', Color(0xFF2E7D32)),
          _ServiceItem('课堂互动', '设计课堂问题与活动',
              Icons.live_help_outlined, '/teacher/class-interact',
              Color(0xFF6A1B9A)),
          _ServiceItem('AI 批改', '辅助批改与反馈',
              Icons.grading, '/teacher/grading', Color(0xFFE65100)),
          _ServiceItem('学情热力图', '定位班级学习差异',
              Icons.grid_on_outlined, '/teacher/heatmap', Color(0xFFC62828)),
          _ServiceItem('教学反思', '沉淀课堂复盘',
              Icons.self_improvement, '/teacher/reflection', Color(0xFF00838F)),
        ]),
        common,
      ];
    }

    if (role == 'counselor') {
      return [
        const _ServiceGroupData('辅导员工作', '跟进 · 管理 · 预警', [
          _ServiceItem('今日关注', '待确认与待跟进事项',
              Icons.visibility_outlined, '/counselor/daily-focus',
              Color(0xFFE65100)),
          _ServiceItem('学生列表', '查看所管理学生',
              Icons.people_alt_outlined, '/counselor/student-list',
              Color(0xFF1565C0)),
          _ServiceItem('办事管理', '维护办事流程',
              Icons.edit_note_outlined, '/process-manage', Color(0xFF6A1B9A)),
          _ServiceItem('情感预警', '查看并处理风险提示',
              Icons.warning_amber_rounded, '/emotion', Color(0xFFC62828)),
        ]),
        common,
      ];
    }

    if (role == 'student' || role == 'student_union') {
      return [
        const _ServiceGroupData('学习与成长', '学业 · 计划 · 生涯 · 心理', [
          _ServiceItem('学业服务', '课程、成绩与学习资源',
              Icons.school_outlined, '/student/study', Color(0xFF1565C0)),
          _ServiceItem('学习计划', '目标、任务与课表',
              Icons.checklist_rtl, '/student/study-plan', Color(0xFF00838F)),
          _ServiceItem('就业服务', '岗位、政策与生涯支持',
              Icons.work_outline, '/student/career', Color(0xFFE65100)),
          _ServiceItem('心理健康', '测评、咨询与关怀资源',
              Icons.favorite_outline, '/student/mental', Color(0xFFC62828)),
          _ServiceItem('学科竞赛', '竞赛报名与作品提交',
              Icons.emoji_events_outlined, '/competition', Color(0xFF6A1B9A)),
          _ServiceItem('大学规划', '四年学业与成长规划',
              Icons.timeline, '/plan', Color(0xFF2E7D32)),
        ]),
        common,
      ];
    }

    return [common];
  }
}

class _ServiceGroup extends StatelessWidget {
  final _ServiceGroupData group;

  const _ServiceGroup({required this.group});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Container(
              width: 34,
              height: 34,
              decoration: BoxDecoration(
                color: theme.colorScheme.primary.withOpacity(0.12),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(group.icon,
                  size: 19, color: theme.colorScheme.primary),
            ),
            const SizedBox(width: 10),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(group.title,
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w700)),
                Text(group.subtitle,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.outline,
                    )),
              ],
            ),
          ],
        ),
        const SizedBox(height: 14),
        LayoutBuilder(
          builder: (context, constraints) {
            final columns = constraints.maxWidth >= 900
                ? 3
                : constraints.maxWidth >= 560
                    ? 2
                    : 1;
            const gap = 12.0;
            final width =
                (constraints.maxWidth - gap * (columns - 1)) / columns;
            return Wrap(
              spacing: gap,
              runSpacing: gap,
              children: [
                for (final item in group.items)
                  SizedBox(
                    width: width,
                    child: _ServiceTile(item: item),
                  ),
              ],
            );
          },
        ),
      ],
    );
  }
}

class _ServiceTile extends StatelessWidget {
  final _ServiceItem item;

  const _ServiceTile({required this.item});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(
            color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: InkWell(
        onTap: () => context.push(item.route),
        borderRadius: BorderRadius.circular(14),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Container(
                width: 46,
                height: 46,
                decoration: BoxDecoration(
                  color: item.color.withOpacity(0.12),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(item.icon, color: item.color),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(item.title, style: theme.textTheme.titleMedium),
                    const SizedBox(height: 2),
                    Text(
                      item.subtitle,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right,
                  color: theme.colorScheme.outlineVariant),
            ],
          ),
        ),
      ),
    );
  }
}

class _ServiceGroupData {
  final String title;
  final String subtitle;
  final IconData icon;
  final List<_ServiceItem> items;

  const _ServiceGroupData(this.title, this.subtitle, this.items,
      [this.icon = Icons.grid_view_rounded]);
}

class _ServiceItem {
  final String title;
  final String subtitle;
  final IconData icon;
  final String route;
  final Color color;

  const _ServiceItem(this.title, this.subtitle, this.icon, this.route,
      [this.color = _defaultColor]);

  static const _defaultColor = Color(0xFF1565C0);
}
