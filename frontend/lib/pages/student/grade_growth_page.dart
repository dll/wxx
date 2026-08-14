import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../main.dart';
import '../../providers/student_feature_provider.dart';
import 'package:provider/provider.dart';

/// 分年级成长计划（大二/大三/大四粘性增强）
///
/// 按当前入学年份推导年级，展示"本阶段该做什么 + 关键节点倒计时 + 高频入口"，
/// 对标大一"开学待办"，让高年级学生每天打开都有专属的"下一步"。
class GradeGrowthPage extends StatelessWidget {
  const GradeGrowthPage({super.key});

  static const _gradeMilestones = {
    // ── 大二：追梦蓝 ──
    2: _GradePlan(
      name: '大二 · 追梦',
      color: Color(0xFF1565C0),
      tagline: '打牢专业基础，点亮目标方向',
      focus: ['课程绩点保持/提升', '四六级、计算机等级等考证', '入党积极分子培养', '提前接触学科竞赛', '加入/负责学生组织'],
      milestones: [
        _Milestone('英语四六级报名', Icons.translate, '6月/12月笔试，3月/9月报名'),
        _Milestone('计算机等级考试（NCRE）', Icons.computer, '3月/9月，关注学院通知'),
        _Milestone('学科竞赛校赛/省赛', Icons.emoji_events, '蓝桥杯、大学生创新创业等'),
        _Milestone('奖学金评优材料提交', Icons.card_giftcard, '学期末关注学工通知'),
      ],
      entryRoutes: [
        _RouteEntry('我的课表', Icons.calendar_month, '/student/study-plan'),
        _RouteEntry('学业服务', Icons.menu_book, '/student/study'),
        _RouteEntry('学科竞赛', Icons.emoji_events, '/competition'),
        _RouteEntry('入党教育', Icons.flag, '/party-education'),
        _RouteEntry('学习打卡', Icons.fitness_center, '/student/checkin'),
      ],
    ),
    // ── 大三：奋斗橙 ──
    3: _GradePlan(
      name: '大三 · 奋斗',
      color: Color(0xFFEF6C00),
      tagline: '聚焦赛道，升学就业两手抓',
      focus: ['明确考研/就业/考公方向', '竞赛与科研（大创、论文、项目）', '专业方向课程深耕', '暑期实习/实践积累', '四六级/专业证书刷分'],
      milestones: [
        _Milestone('大创/科研项目申报', Icons.science, '通常每年春季申报'),
        _Milestone('暑期实习/实践投递', Icons.work, '5-7月集中投递'),
        _Milestone('考研初步规划', Icons.school, '大三下确定目标院校与专业'),
        _Milestone('竞赛国赛/省赛备赛', Icons.emoji_events, '按赛事节点推进'),
      ],
      entryRoutes: [
        _RouteEntry('学习计划', Icons.fact_check, '/student/study-plan'),
        _RouteEntry('学科竞赛', Icons.emoji_events, '/competition'),
        _RouteEntry('就业服务', Icons.work, '/student/career'),
        _RouteEntry('学习打卡', Icons.fitness_center, '/student/checkin'),
        _RouteEntry('心理健康', Icons.favorite, '/student/mental'),
      ],
    ),
    // ── 大四：创业紫 ──
    4: _GradePlan(
      name: '大四 · 创业',
      color: Color(0xFF6A1B9A),
      tagline: '毕业冲刺，奔向人生下一站',
      focus: ['求职/考研/考公关键年', '毕业设计选题与推进', '三方协议/实习', '档案、户口、离校手续', '校友联络'],
      milestones: [
        _Milestone('毕业设计选题开题', Icons.description, '大四上完成选题与开题'),
        _Milestone('秋招/春招投递', Icons.work, '9-11月秋招，3-4月春招'),
        _Milestone('考研初试（12月）', Icons.school, '12月底全国统考'),
        _Milestone('离校手续与档案办理', Icons.archive, '毕业季按通知办理'),
      ],
      entryRoutes: [
        _RouteEntry('就业服务', Icons.work, '/student/career'),
        _RouteEntry('我的课表', Icons.calendar_month, '/student/study-plan'),
        _RouteEntry('毕设选题', Icons.description, '/graduation'),
        _RouteEntry('心理健康', Icons.favorite, '/student/mental'),
      ],
    ),
  };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final themeNotifier = context.watch<ThemeNotifier>();
    final grade = themeNotifier.grade;

    final plan = _gradeMilestones[grade];

    return Scaffold(
      appBar: AppBar(title: const Text('本阶段成长计划')),
      body: plan == null
          ? _buildNoGrade(theme, themeNotifier)
          : RefreshIndicator(
              onRefresh: () async {
                // 刷新学业数据（若存在）
                final p = context.read<StudentFeatureProvider>();
                await p.fetchPersonalProfile();
              },
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  _buildHeader(theme, plan),
                  const SizedBox(height: 16),
                  _buildFocus(theme, plan),
                  const SizedBox(height: 16),
                  _buildMilestones(theme, plan),
                  const SizedBox(height: 16),
                  _buildQuickEntries(context, theme, plan),
                ],
              ),
            ),
    );
  }

  Widget _buildNoGrade(ThemeData theme, ThemeNotifier tn) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.school_outlined, size: 48, color: theme.colorScheme.outline),
            const SizedBox(height: 12),
            Text('暂无法识别你的年级', style: theme.textTheme.titleMedium),
            const SizedBox(height: 6),
            Text(
              '请在个人信息中确认入学年份，即可看到本阶段专属成长计划。',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader(ThemeData theme, _GradePlan plan) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [plan.color.withOpacity(0.16), plan.color.withOpacity(0.05)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: plan.color.withOpacity(0.25)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.auto_awesome, color: plan.color, size: 22),
              const SizedBox(width: 8),
              Text(
                plan.name,
                style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
              ),
              const Spacer(),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: plan.color.withOpacity(0.14),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Text(
                  '${plan.focus.length} 个重点',
                  style: TextStyle(fontSize: 12, color: plan.color, fontWeight: FontWeight.w700),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            plan.tagline,
            style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }

  Widget _buildFocus(ThemeData theme, _GradePlan plan) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('本阶段该做什么', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
        const SizedBox(height: 10),
        ...plan.focus.asMap().entries.map((e) => Container(
              margin: const EdgeInsets.only(bottom: 8),
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              decoration: BoxDecoration(
                color: theme.colorScheme.surface,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: theme.colorScheme.outlineVariant.withOpacity(0.5)),
              ),
              child: Row(children: [
                Icon(Icons.check_circle_outline, color: plan.color, size: 20),
                const SizedBox(width: 10),
                Expanded(child: Text(e.value, style: theme.textTheme.bodyMedium)),
              ]),
            )),
      ],
    );
  }

  Widget _buildMilestones(ThemeData theme, _GradePlan plan) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('关键节点', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
        const SizedBox(height: 10),
        ...plan.milestones.map((m) => Container(
              margin: const EdgeInsets.only(bottom: 10),
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: plan.color.withOpacity(0.06),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: plan.color.withOpacity(0.15)),
              ),
              child: Row(children: [
                Icon(m.icon, color: plan.color, size: 24),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(m.title, style: theme.textTheme.bodyLarge?.copyWith(fontWeight: FontWeight.w600)),
                      const SizedBox(height: 2),
                      Text(m.desc, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                    ],
                  ),
                ),
              ]),
            )),
      ],
    );
  }

  Widget _buildQuickEntries(BuildContext context, ThemeData theme, _GradePlan plan) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('快捷入口', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
        const SizedBox(height: 10),
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: plan.entryRoutes.map((r) => GestureDetector(
                onTap: () => context.go(r.route),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                  decoration: BoxDecoration(
                    color: r.color.withOpacity(0.08),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: r.color.withOpacity(0.2)),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(r.icon, size: 18, color: r.color),
                      const SizedBox(width: 6),
                      Text(r.label, style: TextStyle(fontSize: 13, color: r.color, fontWeight: FontWeight.w600)),
                    ],
                  ),
                ),
              )).toList(),
        ),
      ],
    );
  }
}

class _GradePlan {
  final String name;
  final Color color;
  final String tagline;
  final List<String> focus;
  final List<_Milestone> milestones;
  final List<_RouteEntry> entryRoutes;
  const _GradePlan({
    required this.name,
    required this.color,
    required this.tagline,
    required this.focus,
    required this.milestones,
    required this.entryRoutes,
  });
}

class _Milestone {
  final String title;
  final IconData icon;
  final String desc;
  const _Milestone(this.title, this.icon, this.desc);
}

class _RouteEntry {
  final String label;
  final IconData icon;
  final Color color;
  final String route;
  const _RouteEntry(this.label, this.icon, this.route) : color = const Color(0xFF1565C0);
}
