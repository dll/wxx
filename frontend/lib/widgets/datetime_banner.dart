import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/study_plan_provider.dart';
import '../utils/date_utils.dart';

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
      if (!mounted) return;
      final next = DateTime.now();
      // 显示精度只到分钟，跳过同分钟内的 setState 以避免无意义重建
      if (next.year == _now.year &&
          next.month == _now.month &&
          next.day == _now.day &&
          next.hour == _now.hour &&
          next.minute == _now.minute) {
        return;
      }
      setState(() => _now = next);
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
    final dateStr =
        '${_now.year}年${_now.month.toString().padLeft(2, '0')}月${_now.day.toString().padLeft(2, '0')}日';

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
                theme.colorScheme.secondaryContainer.withOpacity( 0.65),
              ],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(16),
            boxShadow: [
              BoxShadow(
                color: theme.colorScheme.shadow.withOpacity( 0.08),
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
                          color: theme.colorScheme.primary.withOpacity( 0.15),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(TimeFormatter.weekdayFull(_now.weekday),
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              color: theme.colorScheme.primary,
                            )),
                      ),
                    ]),
                    const SizedBox(height: 2),
                    Text(TimeFormatter.hhmm(_now),
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
                  color: theme.colorScheme.onSecondaryContainer.withOpacity( 0.6)),
            ],
          ),
        ),
      ),
    );
  }

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
///
/// 数据源：StudyPlanProvider.timetable（GET /study/timetable），
/// 按 weekday 分组后映射为 _TaskItem 列表；同时合并校历事件（events）。
class _CalendarSheet extends StatefulWidget {
  const _CalendarSheet();

  @override
  State<_CalendarSheet> createState() => _CalendarSheetState();
}

class _CalendarSheetState extends State<_CalendarSheet> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudyPlanProvider>();
      // 仅在无缓存时拉取，避免每次打开都触发请求
      if (p.timetable == null) p.fetchTimetable();
      if (p.calendarData == null) p.fetchCalendar();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final provider = context.watch<StudyPlanProvider>();
    final today = DateTime.now();
    // 归零到当日 0 点，避免 DST / 跨午夜带来的时间分量误差
    final midnight = DateTime(today.year, today.month, today.day);
    final monday = midnight.subtract(Duration(days: midnight.weekday - 1));
    final sunday = monday.add(const Duration(days: 6));

    // 根据课表 + 校历事件构建本周 7 天任务
    final weekTasks = _buildWeekTasks(provider);

    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.4,
      maxChildSize: 0.95,
      expand: false,
      builder: (_, controller) {
        // 顶部加载/错误状态条
        final Widget header = Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Row(
            children: [
              Icon(Icons.event_note, color: theme.colorScheme.primary),
              const SizedBox(width: 10),
              Text('校历 · 本周任务',
                  style: theme.textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.w600)),
              const Spacer(),
              Text(
                '${monday.month}/${monday.day} - ${sunday.month}/${sunday.day}',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        );

        // 顶部状态条：仅当数据为空且仍在加载/出错时展示
        final bool noData = weekTasks.every((t) => t.isEmpty);
        final Widget? statusBanner =
            (provider.loading && provider.timetable == null)
                ? const _StatusBanner(
                    icon: SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                    text: '正在加载课表…',
                  )
                : (provider.error.isNotEmpty && provider.timetable == null)
                    ? _StatusBanner(
                        icon: Icon(Icons.error_outline,
                            color: theme.colorScheme.error, size: 18),
                        text: '课表加载失败',
                        actionText: '重试',
                        onAction: () => provider.fetchTimetable(),
                      )
                    : null;

        return Column(
          children: [
            Container(
              margin: const EdgeInsets.only(top: 8, bottom: 12),
              width: 40, height: 4,
              decoration: BoxDecoration(
                color: theme.colorScheme.outlineVariant,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            header,
            if (statusBanner != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                child: statusBanner,
              )
            else if (noData)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                child: _StatusBanner(
                  icon: Icon(Icons.event_available_outlined,
                      color: theme.colorScheme.outline, size: 18),
                  text: '本周暂无课表安排',
                ),
              ),
            const SizedBox(height: 8),
            Expanded(
              child: ListView.builder(
                controller: controller,
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                itemCount: 7,
                itemBuilder: (_, i) {
                  final day = monday.add(Duration(days: i));
                  return _DayCard(
                    day: day,
                    isToday: TimeFormatter.isSameDay(day, today),
                    tasks: weekTasks[i],
                  );
                },
              ),
            ),
          ],
        );
      },
    );
  }

  /// 构建 7 天的任务列表（索引 0=周一…6=周日）
  ///
  /// 数据来源：
  /// 1. StudyPlanProvider.timetable['schedule'] —— 课程，按 weekday 分组
  /// 2. StudyPlanProvider.calendarData['events'] / ['upcoming_events'] ——
  ///    校历事件，按 start_date 落在本周则归到对应日。
  List<List<_TaskItem>> _buildWeekTasks(StudyPlanProvider provider) {
    final result = List<List<_TaskItem>>.generate(7, (_) => []);

    // 1) 课表
    final schedule = provider.schedule;
    for (final raw in schedule) {
      final item = raw as Map<String, dynamic>;
      final weekday = (item['weekday'] as num?)?.toInt() ?? 0;
      if (weekday < 1 || weekday > 7) continue;
      final name = (item['course_name'] as String?) ??
          (item['name'] as String?) ??
          '课程';
      final location = (item['location'] as String?) ?? '';
      final startPeriod = (item['start_period'] as num?)?.toInt() ?? 1;
      final time = _periodToTime(startPeriod);
      result[weekday - 1].add(_TaskItem(time, name, location, _TaskKind.course));
    }

    // 2) 校历事件（按日期匹配到本周）
    final cal = provider.calendarData;
    final today = DateTime.now();
    final midnight = DateTime(today.year, today.month, today.day);
    final monday = midnight.subtract(Duration(days: midnight.weekday - 1));
    final sunday = monday.add(const Duration(days: 6));
    final events = (cal?['events'] as List?) ??
        (cal?['upcoming_events'] as List?) ??
        const [];
    for (final raw in events) {
      final e = raw as Map<String, dynamic>;
      final name = (e['event_name'] as String?) ??
          (e['name'] as String?) ??
          '事件';
      final startStr = (e['start_date'] as String?) ?? '';
      final start = _parseDate(startStr);
      if (start == null) continue;
      if (start.isBefore(monday) || start.isAfter(sunday)) continue;
      final idx = start.difference(monday).inDays;
      if (idx < 0 || idx >= 7) continue;
      final type = (e['event_type'] as String?) ?? '';
      final kind = switch (type) {
        'exam' => _TaskKind.exam,
        'activity' => _TaskKind.activity,
        _ => _TaskKind.study,
      };
      result[idx].add(_TaskItem('全天', name, '', kind));
    }

    // 按时间字符串排序（"全天" 排到最后）
    for (final list in result) {
      list.sort((a, b) {
        if (a.time == '全天') return 1;
        if (b.time == '全天') return -1;
        return a.time.compareTo(b.time);
      });
    }
    return result;
  }

  /// 节次到开始时间字符串的粗略映射（与学校常规作息对齐）
  String _periodToTime(int period) {
    const map = {
      1: '08:00',
      2: '08:55',
      3: '10:00',
      4: '10:55',
      5: '14:00',
      6: '14:55',
      7: '16:00',
      8: '16:55',
      9: '19:00',
      10: '19:55',
      11: '20:50',
      12: '21:45',
    };
    return map[period] ?? '全天';
  }

  DateTime? _parseDate(String s) {
    if (s.isEmpty) return null;
    try {
      final seg = s.length >= 10 ? s.substring(0, 10) : s;
      final parts = seg.split('-');
      if (parts.length != 3) return null;
      final y = int.tryParse(parts[0]);
      final m = int.tryParse(parts[1]);
      final d = int.tryParse(parts[2]);
      if (y == null || m == null || d == null) return null;
      return DateTime(y, m, d);
    } catch (_) {
      return null;
    }
  }
}

/// 顶部状态条 —— 加载/错误/空状态提示
class _StatusBanner extends StatelessWidget {
  final Widget icon;
  final String text;
  final String? actionText;
  final VoidCallback? onAction;
  const _StatusBanner({
    required this.icon,
    required this.text,
    this.actionText,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          icon,
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              text,
              style: theme.textTheme.bodySmall,
            ),
          ),
          if (actionText != null && onAction != null)
            TextButton(
              onPressed: onAction,
              style: TextButton.styleFrom(
                visualDensity: VisualDensity.compact,
                padding: const EdgeInsets.symmetric(horizontal: 8),
              ),
              child: Text(actionText!,
                  style: const TextStyle(fontSize: 12)),
            ),
        ],
      ),
    );
  }
}

class _DayCard extends StatelessWidget {
  final DateTime day;
  final bool isToday;
  final List<_TaskItem> tasks;
  const _DayCard({required this.day, required this.isToday, required this.tasks});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: isToday
            ? theme.colorScheme.primaryContainer
            : theme.colorScheme.surfaceContainerHighest.withOpacity( 0.45),
        borderRadius: BorderRadius.circular(12),
        border: isToday
            ? Border.all(color: theme.colorScheme.primary, width: 1.5)
            : null,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
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
                  TimeFormatter.weekdayShort(day.weekday),
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
                        _TaskRow(task: tasks[i]),
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
  const _TaskRow({required this.task});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = task.kind.color;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
          decoration: BoxDecoration(
            color: color.withOpacity( 0.15),
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
              if (task.location.isNotEmpty) ...[
                const SizedBox(height: 1),
                Text(task.location,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    )),
              ],
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
