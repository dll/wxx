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
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
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
    const common = _ServiceGroupData('校园服务', [
      const _ServiceItem(
          '知识大厅', '政策、流程与可信来源', Icons.menu_book_outlined, '/browse'),
      const _ServiceItem(
          '办事服务', '查看流程和办理进度', Icons.assignment_outlined, '/enrollment'),
      const _ServiceItem('校园导航', '地图、全景与校园入口', Icons.map_outlined, '/campus'),
      const _ServiceItem('应用中心', '校内外系统快捷入口', Icons.apps_outlined, '/apps'),
      const _ServiceItem(
          'AI 简讯', '教学、工具与行业动态', Icons.newspaper_outlined, '/ai-briefings'),
    ]);

    if (role == 'teacher') {
      return [
        const _ServiceGroupData('教学工具', [
          _ServiceItem(
              'AI 备课', '生成并完善教学方案', Icons.auto_awesome, '/teacher/lesson-prep'),
          _ServiceItem(
              'AI 出题', '按课程目标生成试题', Icons.quiz_outlined, '/teacher/exam-gen'),
          _ServiceItem('课堂互动', '设计课堂问题与活动', Icons.live_help_outlined,
              '/teacher/class-interact'),
          _ServiceItem('AI 批改', '辅助批改与反馈', Icons.grading, '/teacher/grading'),
          _ServiceItem(
              '学情热力图', '定位班级学习差异', Icons.grid_on_outlined, '/teacher/heatmap'),
          _ServiceItem(
              '教学反思', '沉淀课堂复盘', Icons.self_improvement, '/teacher/reflection'),
        ]),
        common,
      ];
    }

    if (role == 'counselor') {
      return [
        const _ServiceGroupData('辅导员工作', [
          _ServiceItem('今日关注', '待确认与待跟进事项', Icons.visibility_outlined,
              '/counselor/daily-focus'),
          _ServiceItem('学生列表', '查看所管理学生', Icons.people_alt_outlined,
              '/counselor/student-list'),
          _ServiceItem(
              '办事管理', '维护办事流程', Icons.edit_note_outlined, '/process-manage'),
          _ServiceItem(
              '情感预警', '查看并处理风险提示', Icons.warning_amber_rounded, '/emotion'),
        ]),
        common,
      ];
    }

    if (role == 'student' || role == 'student_union') {
      return [
        const _ServiceGroupData('学习与成长', [
          _ServiceItem(
              '学业服务', '课程、成绩与学习资源', Icons.school_outlined, '/student/study'),
          _ServiceItem(
              '学习计划', '目标、任务与课表', Icons.checklist_rtl, '/student/study-plan'),
          _ServiceItem(
              '就业服务', '岗位、政策与生涯支持', Icons.work_outline, '/student/career'),
          _ServiceItem(
              '心理健康', '测评、咨询与关怀资源', Icons.favorite_outline, '/student/mental'),
          _ServiceItem(
              '学科竞赛', '竞赛报名与作品提交', Icons.emoji_events_outlined, '/competition'),
          _ServiceItem('大学规划', '四年学业与成长规划', Icons.timeline, '/plan'),
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
        Text(group.title, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
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
      child: InkWell(
        onTap: () => context.push(item.route),
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: theme.colorScheme.secondaryContainer,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(item.icon,
                    color: theme.colorScheme.onSecondaryContainer),
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
              const Icon(Icons.chevron_right),
            ],
          ),
        ),
      ),
    );
  }
}

class _ServiceGroupData {
  final String title;
  final List<_ServiceItem> items;

  const _ServiceGroupData(this.title, this.items);
}

class _ServiceItem {
  final String title;
  final String subtitle;
  final IconData icon;
  final String route;

  const _ServiceItem(this.title, this.subtitle, this.icon, this.route);
}
