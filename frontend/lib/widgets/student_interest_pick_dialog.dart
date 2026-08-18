import 'package:flutter/material.dart';

import '../utils/storage.dart';

/// 学生关注内容选项定义。
/// key 需与 home_page 学生专区 `_FeatureCard.interestKey` 保持一致，
/// 首页据此与关注内容做匹配定制排序。
class StudentInterestOption {
  final String label;
  final String key;
  final IconData icon;
  final Color color;
  const StudentInterestOption(this.label, this.key, this.icon, this.color);
}

const kStudentInterestOptions = [
  StudentInterestOption('学业成长', '学业成长', Icons.menu_book_outlined,
      Color(0xFF1565C0)),
  StudentInterestOption(
      '竞赛科研', '竞赛科研', Icons.emoji_events_outlined, Color(0xFFE65100)),
  StudentInterestOption('职业就业', '职业就业', Icons.work_outline,
      Color(0xFF2E7D32)),
  StudentInterestOption(
      '思想政治', '思想政治', Icons.flag_outlined, Color(0xFFC62828)),
  StudentInterestOption(
      '心理健康', '心理健康', Icons.favorite_outline, Color(0xFF7B1FA2)),
  StudentInterestOption('校园生活', '校园生活', Icons.groups_outlined,
      Color(0xFF00695C)),
];

/// 展示「我的关注」多选对话框，返回选中的兴趣 key 列表；取消/跳过返回 null。
Future<List<String>?> showStudentInterestPickDialog(BuildContext context,
    {List<String>? initial}) async {
  return showDialog<List<String>>(
    context: context,
    barrierDismissible: false,
    builder: (_) => _StudentInterestPickDialog(initial: initial),
  );
}

class _StudentInterestPickDialog extends StatefulWidget {
  final List<String>? initial;
  const _StudentInterestPickDialog({this.initial});

  @override
  State<_StudentInterestPickDialog> createState() =>
      _StudentInterestPickDialogState();
}

class _StudentInterestPickDialogState extends State<_StudentInterestPickDialog> {
  late final Set<String> _selected;

  @override
  void initState() {
    super.initState();
    _selected = {...?widget.initial};
  }

  void _toggle(String key) {
    setState(() {
      if (_selected.contains(key)) {
        _selected.remove(key);
      } else {
        _selected.add(key);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Dialog(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('你在关注什么？',
                style: theme.textTheme.titleLarge
                    ?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 6),
            Text('选好后，首页「学生专区」会按你关注的内容优先展示。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            const SizedBox(height: 16),
            Wrap(
              spacing: 10,
              runSpacing: 10,
              children: [
                for (final o in kStudentInterestOptions)
                  InkWell(
                    onTap: () => _toggle(o.key),
                    borderRadius: BorderRadius.circular(24),
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 14, vertical: 9),
                      decoration: BoxDecoration(
                        color: _selected.contains(o.key)
                            ? o.color.withOpacity(0.14)
                            : theme.colorScheme.surfaceContainerHighest
                                .withOpacity(0.5),
                        borderRadius: BorderRadius.circular(24),
                        border: Border.all(
                          color: _selected.contains(o.key)
                              ? o.color
                              : theme.colorScheme.outlineVariant,
                        ),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            _selected.contains(o.key)
                                ? Icons.check_circle
                                : o.icon,
                            size: 16,
                            color: _selected.contains(o.key)
                                ? o.color
                                : theme.colorScheme.onSurfaceVariant,
                          ),
                          const SizedBox(width: 6),
                          Text(o.label,
                              style: theme.textTheme.bodyMedium?.copyWith(
                                fontWeight: _selected.contains(o.key)
                                    ? FontWeight.w700
                                    : FontWeight.w400,
                                color: _selected.contains(o.key)
                                    ? o.color
                                    : null,
                              )),
                        ],
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 20),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('取消'),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: () =>
                      Navigator.of(context).pop(_selected.toList()),
                  child: const Text('保存'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 便捷：读取当前已保存的关注，交给对话框。供首页首次采集/个人页修改共用。
Future<List<String>?> pickupStudentInterests(BuildContext context) {
  return showStudentInterestPickDialog(
    context,
    initial: Storage.studentInterests,
  );
}
