/// 日期时间格式化工具
class TimeFormatter {
  TimeFormatter._();

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
      final hhmm = '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
      if (dt.year == now.year && dt.month == now.month && dt.day == now.day) {
        return '今天 $hhmm';
      }
      return '${dt.month}/${dt.day} $hhmm';
    } catch (_) {
      return iso;
    }
  }
}
