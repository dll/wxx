import 'dart:async';
import 'package:flutter/material.dart';

/// 首页日期时间横幅 — 点击展开「校历」周任务
class DateTimeBanner extends StatefulWidget {
  const DateTimeBanner({super.key});

  @override
  State<DateTimeBanner> createState() => _DateTimeBannerState();
}

class _DateTimeBannerState extends State<DateTimeBanner> {
  late DateTime _now;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _now = DateTime.now();
    _timer = Timer.periodic(const Duration(seconds: 30), (_) {
      if (mounted) setState(() => _now = DateTime.now());
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final weekDay = _weekDayName(_now.weekday);
    final dateStr =
        '${_now.year}年${_now.month.toString().padLeft(2, '0')}月${_now.day.toString().padLeft(2, '0')}日';
    final timeStr =
        '${_now.hour.toString().padLeft(2, '0')}:${_now.minute.toString().padLeft(2, '0')}';

    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: () => _showCalendar(context),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              colors: [
                theme.colorScheme.secondaryContainer,
                theme.colorScheme.secondaryContainer.withValues(alpha: 0.65),
              ],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(16),
            boxShadow: [
              BoxShadow(
                color: theme.colorScheme.shadow.withValues(alpha: 0.08),
                blurRadius: 10,
                offset: const Offset(0, 3),
              ),
            ],
          ),
          child: Row(
            children: [
              Icon(Icons.calendar_today_rounded,
                  size: 28, color: theme.colorScheme.onSecondaryContainer),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Text(dateStr,
                          style: TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                            color: theme.colorScheme.onSecondaryContainer,
                          )),
                      const SizedBox(width: 8),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.primary.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(weekDay,
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              color: theme.colorScheme.primary,
                            )),
                      ),
                    ]),
                    const SizedBox(height: 2),
                    Text(timeStr,
                        style: TextStyle(
                          fontSize: 22,
                          fontWeight: FontWeight.bold,
                          fontFeatures: const [FontFeature.tabularFigures()],
                          color: theme.colorScheme.onSecondaryContainer,
                        )),
                  ],
                ),
              ),
              Icon(Icons.chevron_right,
                  color: theme.colorScheme.onSecondaryContainer.withValues(alpha: 0.6)),
            ],
          ),
        ),
      ),
    );
  }

  static String _weekDayName(int w) =>
      const ['星期一', '星期二', '星期三', '星期四', '星期五', '星期六', '星期日'][w - 1];

  void _showCalendar(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (_) => const _CalendarSheet(),
    );
  }
}

/// 校历底部抽屉 — 显示本周 7 天任务
class _CalendarSheet extends StatelessWidget {
  const _CalendarSheet();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final now = DateTime.now();
    // 本周一
    final monday = now.subtract(Duration(days: now.weekday - 1));
    final tasks = _weekTasks(monday);

    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.4,
      maxChildSize: 0.95,
      expand: false,
      builder: (_, controller) => Column(
        children: [
          // 顶部抓手
          Container(
            margin: const EdgeInsets.only(top: 8, bottom: 12),
            width: 40, height: 4,
            decoration: BoxDecoration(
              color: theme.colorScheme.outlineVariant,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          // 标题
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20),
            child: Row(
              children: [
                Icon(Icons.event_note, color: theme.colorScheme.primary),
                const SizedBox(width: 10),
                Text('校历 · 本周任务',
                    style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600)),
                const Spacer(),
                Text(
                  '${monday.month}/${monday.day} – ${monday.add(const Duration(days: 6)).month}/${monday.add(const Duration(days: 6)).day}',
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Expanded(
            child: ListView.builder(
              controller: controller,
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
              itemCount: 7,
              itemBuilder: (_, i) {
                final day = monday.add(Duration(days: i));
                final isToday = _sameDay(day, now);
                return _DayCard(
                  day: day,
                  isToday: isToday,
                  tasks: tasks[i],
                  theme: theme,
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  static bool _sameDay(DateTime a, DateTime b) =>
      a.year == b.year && a.month == b.month && a.day == b.day;

  /// 占位周任务种子（后续可对接 /api/v1/student/schedule 与 /culture/events）
  List<List<_TaskItem>> _weekTasks(DateTime monday) {
    return [
      [
        _TaskItem('08:00', '高等数学', '理工楼 A201', _TaskKind.course),
        _TaskItem('14:00', '党史学习', '智慧教室', _TaskKind.study),
        _TaskItem('19:00', '院辩论队选拔', '第二会议室', _TaskKind.activity),
      ],
      [
        _TaskItem('10:00', '英语听力测试', '语音室', _TaskKind.exam),
        _TaskItem('15:30', '专业实训', '计算机房', _TaskKind.course),
      ],
      [
        _TaskItem('09:00', '操作系统', '理工楼 B305', _TaskKind.course),
        _TaskItem('14:00', '志愿者岗前培训', '青协办公室', _TaskKind.activity),
      ],
      [
        _TaskItem('全天', '心理健康讲座', '大礼堂', _TaskKind.study),
      ],
      [
        _TaskItem('10:00', '数据库原理', '理工楼 A310', _TaskKind.course),
        _TaskItem('19:00', '校园广播·音乐之声', '广播台直播', _TaskKind.activity),
      ],
      [
        _TaskItem('全天', '校园文化节', '中心广场', _TaskKind.activity),
      ],
      [
        _TaskItem('19:00', '本周学习周报', '系统自动生成', _TaskKind.study),
      ],
    ];
  }
}

class _DayCard extends StatelessWidget {
  final DateTime day;
  final bool isToday;
  final List<_TaskItem> tasks;
  final ThemeData theme;
  const _DayCard({
    required this.day,
    required this.isToday,
    required this.tasks,
    required this.theme,
  });

  static const _weekDays = ['一', '二', '三', '四', '五', '六', '日'];

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: isToday
            ? theme.colorScheme.primaryContainer
            : theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.45),
        borderRadius: BorderRadius.circular(12),
        border: isToday
            ? Border.all(color: theme.colorScheme.primary, width: 1.5)
            : null,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 日期圆环
          Container(
            width: 52, height: 52,
            decoration: BoxDecoration(
              color: isToday ? theme.colorScheme.primary : theme.colorScheme.surface,
              shape: BoxShape.circle,
              border: Border.all(
                color: isToday ? theme.colorScheme.primary : theme.colorScheme.outlineVariant,
                width: 1,
              ),
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(
                  _weekDays[day.weekday - 1],
                  style: TextStyle(
                    fontSize: 11,
                    color: isToday ? Colors.white70 : theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                Text(
                  '${day.day}',
                  style: TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.bold,
                    color: isToday ? Colors.white : theme.colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: tasks.isEmpty
                ? Padding(
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    child: Text('今日无安排',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        )),
                  )
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      for (int i = 0; i < tasks.length; i++) ...[
                        if (i > 0) const SizedBox(height: 8),
                        _TaskRow(task: tasks[i], theme: theme),
                      ],
                    ],
                  ),
          ),
        ],
      ),
    );
  }
}

class _TaskRow extends StatelessWidget {
  final _TaskItem task;
  final ThemeData theme;
  const _TaskRow({required this.task, required this.theme});

  @override
  Widget build(BuildContext context) {
    final color = task.kind.color;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.15),
            borderRadius: BorderRadius.circular(4),
          ),
          child: Text(task.time,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: color,
                fontFeatures: const [FontFeature.tabularFigures()],
              )),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(task.title,
                  style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600)),
              const SizedBox(height: 1),
              Text(task.location,
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  )),
            ],
          ),
        ),
        Icon(task.kind.icon, color: color, size: 16),
      ],
    );
  }
}

class _TaskItem {
  final String time;
  final String title;
  final String location;
  final _TaskKind kind;
  const _TaskItem(this.time, this.title, this.location, this.kind);
}

enum _TaskKind { course, exam, activity, study }

extension _TaskKindMeta on _TaskKind {
  Color get color {
    switch (this) {
      case _TaskKind.course:
        return const Color(0xFF1565C0);
      case _TaskKind.exam:
        return const Color(0xFFD32F2F);
      case _TaskKind.activity:
        return const Color(0xFF388E3C);
      case _TaskKind.study:
        return const Color(0xFF7B1FA2);
    }
  }

  IconData get icon {
    switch (this) {
      case _TaskKind.course:
        return Icons.menu_book_outlined;
      case _TaskKind.exam:
        return Icons.assignment_outlined;
      case _TaskKind.activity:
        return Icons.celebration_outlined;
      case _TaskKind.study:
        return Icons.school_outlined;
    }
  }
}
