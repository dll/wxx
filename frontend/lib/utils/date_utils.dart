/// 日期时间格式化工具
class TimeFormatter {
  TimeFormatter._();

  static const _weekdayFull = [
    '星期一', '星期二', '星期三', '星期四', '星期五', '星期六', '星期日',
  ];
  static const _weekdayShort = ['一', '二', '三', '四', '五', '六', '日'];

  /// 同一日历日（忽略时分秒）
  static bool isSameDay(DateTime a, DateTime b) =>
      a.year == b.year && a.month == b.month && a.day == b.day;

  /// HH:mm 零填充
  static String hhmm(DateTime dt) =>
      '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';

  /// 中文长名星期（DateTime.weekday: 1=Mon..7=Sun）
  static String weekdayFull(int weekday) => _weekdayFull[weekday - 1];

  /// 中文单字星期
  static String weekdayShort(int weekday) => _weekdayShort[weekday - 1];

  /// 相对时间格式（刚刚、X分钟前、X小时前、X天前、M/D）
  static String relative(String iso) {
    if (iso.isEmpty) return '';
    try {
      final dt = DateTime.parse(iso);
      final now = DateTime.now();
      final diff = now.difference(dt);
      if (diff.inMinutes < 1) return '刚刚';
      if (diff.inHours < 1) return '${diff.inMinutes}分钟前';
      if (diff.inDays < 1) return '${diff.inHours}小时前';
      if (diff.inDays < 7) return '${diff.inDays}天前';
      return '${dt.month}/${dt.day}';
    } catch (_) {
      return iso;
    }
  }

  /// 日期时间格式（今天 HH:mm 或 M/D HH:mm）
  static String dateTime(String iso) {
    if (iso.isEmpty) return '';
    try {
      final dt = DateTime.parse(iso);
      final now = DateTime.now();
      final timePart = hhmm(dt);
      if (isSameDay(dt, now)) return '今天 $timePart';
      return '${dt.month}/${dt.day} $timePart';
    } catch (_) {
      return iso;
    }
  }
}

