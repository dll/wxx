import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../providers/study_plan_provider.dart';
import '../../../widgets/error_view.dart';

/// 课表页面
///
/// 周课表网格：周一到周日 × 第 1-12 节。
/// 顶部显示当前教学周，可切换周次；若某周不在 weeks_pattern 内则课程灰显。
class TimetablePage extends StatefulWidget {
  const TimetablePage({super.key});

  @override
  State<TimetablePage> createState() => _TimetablePageState();
}

/// 每天最多显示的节次数
const int _kTotalPeriods = 12;

class _TimetablePageState extends State<TimetablePage> {
  /// 当前选中的周次（1-based）。null 表示尚未从校历获取
  int? _selectedWeek;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final p = context.read<StudyPlanProvider>();
      p.fetchTimetable();
      p.fetchCalendar();
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<StudyPlanProvider>();
    final cal = provider.calendarData;
    final currentWeek = cal?['current_week'] as int?;
    final totalWeeks =
        ((cal?['semester'] as Map<String, dynamic>?)?['total_weeks'] as int?) ??
            20;

    // 首次获取到当前周时同步选中
    if (_selectedWeek == null && currentWeek != null) {
      _selectedWeek = currentWeek;
    }
    final selected = _selectedWeek ?? currentWeek ?? 1;

    return Scaffold(
      appBar: AppBar(title: const Text('我的课表')),
      body: provider.loading && provider.timetable == null
          ? const Center(child: CircularProgressIndicator())
          : provider.error.isNotEmpty && provider.timetable == null
              ? ErrorView.error(
                  message: provider.error,
                  onRetry: () => provider.fetchTimetable(),
                )
              : provider.schedule.isEmpty
                  ? ErrorView.empty(
                      message: '暂无课表',
                      subtitle: '稍后再来看看吧',
                      icon: Icons.table_view_outlined,
                    )
                  : Column(
                      children: [
                        _WeekSelector(
                          currentWeek: currentWeek,
                          selectedWeek: selected,
                          totalWeeks: totalWeeks,
                          onChanged: (w) =>
                              setState(() => _selectedWeek = w),
                        ),
                        Expanded(
                          child: _TimetableGrid(
                            schedule: provider.schedule,
                            selectedWeek: selected,
                          ),
                        ),
                      ],
                    ),
    );
  }
}

/// 顶部周次选择器
class _WeekSelector extends StatelessWidget {
  final int? currentWeek;
  final int selectedWeek;
  final int totalWeeks;
  final ValueChanged<int> onChanged;
  const _WeekSelector({
    required this.currentWeek,
    required this.selectedWeek,
    required this.totalWeeks,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          bottom: BorderSide(
            color: theme.colorScheme.outlineVariant.withOpacity(0.4),
          ),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.event_note, color: theme.colorScheme.primary, size: 18),
              const SizedBox(width: 6),
              Text(
                currentWeek != null
                    ? '当前第 $currentWeek 教学周'
                    : '未获取到当前周',
                style: theme.textTheme.titleSmall
                    ?.copyWith(fontWeight: FontWeight.w600),
              ),
              const Spacer(),
              if (currentWeek != null && currentWeek != selectedWeek)
                TextButton.icon(
                  onPressed: () => onChanged(currentWeek!),
                  icon: const Icon(Icons.my_location, size: 14),
                  label: const Text('回到本周', style: TextStyle(fontSize: 12)),
                ),
            ],
          ),
          const SizedBox(height: 8),
          SizedBox(
            height: 38,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: totalWeeks,
              separatorBuilder: (_, __) => const SizedBox(width: 6),
              itemBuilder: (_, i) {
                final w = i + 1;
                final isCurrent = w == currentWeek;
                final isSelected = w == selectedWeek;
                return GestureDetector(
                  onTap: () => onChanged(w),
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10),
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: isSelected
                          ? theme.colorScheme.primary
                          : theme.colorScheme.surfaceContainerHighest
                              .withOpacity(0.5),
                      borderRadius: BorderRadius.circular(8),
                      border: isCurrent && !isSelected
                          ? Border.all(
                              color: theme.colorScheme.primary, width: 1.2)
                          : null,
                    ),
                    child: Text(
                      '第$w周',
                      style: theme.textTheme.labelMedium?.copyWith(
                        color: isSelected
                            ? theme.colorScheme.onPrimary
                            : theme.colorScheme.onSurfaceVariant,
                        fontWeight:
                            isSelected ? FontWeight.bold : FontWeight.w500,
                      ),
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

/// 课表网格 —— 左侧节次 + 7 列星期
class _TimetableGrid extends StatelessWidget {
  final List<dynamic> schedule;
  final int selectedWeek;
  const _TimetableGrid({required this.schedule, required this.selectedWeek});

  static const double _periodHeight = 56;
  static const double _periodColWidth = 36;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    // 按 weekday 分组（1=周一…7=周日）
    final byDay = <int, List<Map<String, dynamic>>>{};
    for (final raw in schedule) {
      final item = raw as Map<String, dynamic>;
      final weekday = (item['weekday'] as num?)?.toInt() ?? 0;
      if (weekday < 1 || weekday > 7) continue;
      byDay.putIfAbsent(weekday, () => []).add(item);
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 左侧节次列
          _buildPeriodColumn(theme),
          const SizedBox(width: 4),
          // 7 个星期列
          Expanded(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                for (int d = 1; d <= 7; d++)
                  Expanded(
                    child: _buildDayColumn(theme, d, byDay[d] ?? const []),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPeriodColumn(ThemeData theme) {
    return SizedBox(
      width: _periodColWidth,
      child: Column(
        children: [
          // 顶部留白与星期行对齐
          SizedBox(
            height: 28,
            child: Center(
              child: Text(
                '节',
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
            ),
          ),
          for (int p = 1; p <= _kTotalPeriods; p++)
            SizedBox(
              height: _periodHeight,
              child: Center(
                child: Text(
                  '$p',
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildDayColumn(
    ThemeData theme,
    int weekday,
    List<Map<String, dynamic>> courses,
  ) {
    final label = _weekdayLabel(weekday);
    final isWeekend = weekday >= 6;

    // 占位数组：每节指向占用的课程（同一课程的多节指向同一对象）
    final slots = List<Map<String, dynamic>?>.filled(_kTotalPeriods + 1, null);
    for (final c in courses) {
      final start = (c['start_period'] as num?)?.toInt() ?? 0;
      final end = (c['end_period'] as num?)?.toInt() ?? 0;
      if (start < 1 || end < start || start > _kTotalPeriods) continue;
      final clampedEnd = end > _kTotalPeriods ? _kTotalPeriods : end;
      for (int p = start; p <= clampedEnd; p++) {
        slots[p] = c;
      }
    }

    return Column(
      children: [
        // 星期标题
        Container(
          height: 28,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: isWeekend
                ? theme.colorScheme.errorContainer.withOpacity(0.3)
                : theme.colorScheme.primaryContainer.withOpacity(0.6),
            borderRadius: const BorderRadius.only(
              topLeft: Radius.circular(6),
              topRight: Radius.circular(6),
            ),
          ),
          child: Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w600,
              color: isWeekend
                  ? theme.colorScheme.onSurface
                  : theme.colorScheme.onPrimaryContainer,
            ),
          ),
        ),
        // 12 节格子
        for (int p = 1; p <= _kTotalPeriods; p++)
          _buildPeriodCell(theme, p, slots),
      ],
    );
  }

  Widget _buildPeriodCell(
    ThemeData theme,
    int period,
    List<Map<String, dynamic>?> slots,
  ) {
    final course = slots[period];
    // 该节不是任何课程的起始节 → 渲染空（已被上面的课程覆盖）
    if (course == null) {
      return SizedBox(
        height: _periodHeight,
        child: _dashedBorder(theme),
      );
    }
    final start = (course['start_period'] as num?)?.toInt() ?? period;
    if (period != start) {
      // 属于课程中间/末尾节，已被起始节渲染时占据，此处不渲染
      return const SizedBox.shrink();
    }
    final end = (course['end_period'] as num?)?.toInt() ?? start;
    final span = (end - start + 1).clamp(1, _kTotalPeriods);
    final height = _periodHeight * span;

    final inWeek = _weekInPattern(course['weeks_pattern'], selectedWeek);
    return _CourseBlock(
      course: course,
      height: height,
      dimmed: !inWeek,
    );
  }

  Widget _dashedBorder(ThemeData theme) {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 2, vertical: 1),
      decoration: BoxDecoration(
        border: Border(
          top: BorderSide(
            color: theme.colorScheme.outlineVariant.withOpacity(0.25),
            width: 0.5,
          ),
        ),
      ),
    );
  }

  String _weekdayLabel(int weekday) => switch (weekday) {
        1 => '周一',
        2 => '周二',
        3 => '周三',
        4 => '周四',
        5 => '周五',
        6 => '周六',
        7 => '周日',
        _ => '',
      };

  /// 解析 weeks_pattern（如 "1-18"、"1-16,18"、"1,3,5,7"），判断某周是否上课
  bool _weekInPattern(dynamic pattern, int week) {
    if (pattern == null) return true; // 缺省视为全周
    final s = pattern.toString().trim();
    if (s.isEmpty) return true;
    // 单数字或全量
    try {
      for (final part in s.split(',')) {
        final seg = part.trim();
        if (seg.isEmpty) continue;
        if (seg.contains('-')) {
          final tokens = seg.split('-');
          if (tokens.length != 2) continue;
          final a = int.tryParse(tokens[0].trim());
          final b = int.tryParse(tokens[1].trim());
          if (a == null || b == null) continue;
          final lo = a < b ? a : b;
          final hi = a < b ? b : a;
          if (week >= lo && week <= hi) return true;
        } else {
          final n = int.tryParse(seg);
          if (n == week) return true;
        }
      }
    } catch (_) {
      return true;
    }
    return false;
  }
}

/// 课程块（可跨多节）
class _CourseBlock extends StatelessWidget {
  final Map<String, dynamic> course;
  final double height;
  final bool dimmed;
  const _CourseBlock({
    required this.course,
    required this.height,
    required this.dimmed,
  });

  @override
  Widget build(BuildContext context) {
    final name = (course['course_name'] as String?) ??
        (course['name'] as String?) ??
        '课程';
    final location = (course['location'] as String?) ?? '';
    final teacher = (course['teacher'] as String?) ?? '';
    final color = _parseColor(course['color']);

    return Container(
      width: double.infinity,
      height: height,
      margin: const EdgeInsets.symmetric(horizontal: 2, vertical: 1),
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: dimmed ? color.withOpacity(0.12) : color.withOpacity(0.85),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(
          color: dimmed ? color.withOpacity(0.3) : color,
          width: dimmed ? 0.5 : 1.0,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            name,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.bold,
              color: dimmed ? color : Colors.white,
              height: 1.1,
            ),
          ),
          if (height >= 56 && location.isNotEmpty) ...[
            const SizedBox(height: 2),
            Text(
              location,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 9,
                color: dimmed ? color.withOpacity(0.7) : Colors.white70,
              ),
            ),
          ],
          if (height >= 80 && teacher.isNotEmpty) ...[
            const SizedBox(height: 1),
            Text(
              teacher,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 9,
                color: dimmed ? color.withOpacity(0.7) : Colors.white70,
              ),
            ),
          ],
        ],
      ),
    );
  }

  Color _parseColor(dynamic raw) {
    if (raw is String && raw.isNotEmpty) {
      var s = raw.trim();
      if (s.startsWith('#')) s = s.substring(1);
      if (s.length == 6) {
        final v = int.tryParse('FF$s', radix: 16);
        if (v != null) return Color(v);
      }
      if (s.length == 8) {
        final v = int.tryParse(s, radix: 16);
        if (v != null) return Color(v);
      }
    }
    return const Color(0xFF1565C0); // 默认：滁州学院蓝
  }
}
