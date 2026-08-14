import 'dart:convert';
import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 学生 - 导入我的课表（角色化导入：仅导入本人课表）
///
/// 流程：登录学校门户 → 教务系统 → 课表查询 → 将本人课表按下方格式填入，
/// 导入后即写入"我的课表"，首页「今日课表」与「学习计划」自动同步。
/// 后端强制 user_id = 当前登录学生，不会覆盖他人课表。
class MyScheduleImportPage extends StatefulWidget {
  const MyScheduleImportPage({super.key});
  @override
  State<MyScheduleImportPage> createState() => _MyScheduleImportPageState();
}

class _MyScheduleImportPageState extends State<MyScheduleImportPage> {
  final ApiService _api = ApiService();
  final _ctrl = TextEditingController();
  bool _loading = false;
  String _result = '';

  static const _example = '''
{
  "schedules": [
    {
      "course_id": "CS103",
      "course_name": "数据结构",
      "semester_code": "2025-2026-2",
      "weekday": 1,
      "start_period": 3,
      "end_period": 4,
      "weeks_pattern": "1-16",
      "location": "信息楼C301",
      "teacher": "王老师"
    }
  ]
}''';

  static const _fields = [
    ('course_id', '课程编号（用于去重，如 CS103）'),
    ('course_name', '课程名称，如 数据结构'),
    ('semester_code', '学期代码，如 2025-2026-2'),
    ('weekday', '星期：1=周一 … 7=周日'),
    ('start_period', '开始节次：1-10（1=08:00，3=10:00）'),
    ('end_period', '结束节次：≤10，且 ≥ start_period'),
    ('weeks_pattern', '周次，如 "1-16" 或 "1-8,10-16"'),
    ('location', '上课地点，如 信息楼C301'),
    ('teacher', '任课教师'),
  ];

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _import() async {
    if (_ctrl.text.trim().isEmpty) return;
    setState(() { _loading = true; _result = ''; });
    try {
      final res = await _api.post(ApiConfig.studentScheduleImport,
          data: _parseJson(_ctrl.text));
      final data = res.data;
      final created = data['data']?['created'] ?? 0;
      final updated = data['data']?['updated'] ?? 0;
      final total = data['data']?['total'] ?? 0;
      setState(() =>
          _result = '导入完成：我的课表共 $total 条（新增 $created，更新 $updated）');
    } catch (e) {
      setState(() => _result = '导入失败：$e');
    } finally {
      setState(() => _loading = false);
    }
  }

  Map<String, dynamic> _parseJson(String raw) {
    final obj = jsonDecode(raw);
    if (obj is Map<String, dynamic>) return obj;
    if (obj is List) return {'schedules': obj};
    return {};
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('导入我的课表')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Icon(Icons.person_outline, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text('导入我的课表', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                ]),
                const SizedBox(height: 8),
                Text(
                  '在学校的门户登录 → 教务系统 → 课表查询，把【你本人】的课表按下方 JSON 格式填入后导入。\n'
                  '只会写入你本人的课表，不会影响其他同学。',
                  style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                ),
              ]),
            ),
          ),
          const SizedBox(height: 12),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Icon(Icons.event_note, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text('课表 JSON', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600)),
                  const Spacer(),
                  OutlinedButton(
                    onPressed: () { _ctrl.text = _example; setState(() {}); },
                    child: const Text('填入示例'),
                  ),
                ]),
                const SizedBox(height: 8),
                TextField(
                  controller: _ctrl,
                  maxLines: 12,
                  style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                  decoration: const InputDecoration(
                    hintText: '粘贴课表 JSON',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: _loading ? null : _import,
                    icon: _loading
                        ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.upload),
                    label: Text(_loading ? '导入中…' : '导入我的课表'),
                  ),
                ),
                if (_result.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(_result,
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: _result.startsWith('导入完成') ? Colors.green : theme.colorScheme.error)),
                ],
              ]),
            ),
          ),
          const SizedBox(height: 8),
          Card(
            child: ExpansionTile(
              leading: Icon(Icons.help_outline, color: theme.colorScheme.primary),
              title: Text('字段说明', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600)),
              childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
              expandedCrossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('节次对应：1=08:00, 2=08:55, 3=10:00, 4=10:55, 5=14:00, 6=14:55, 7=16:00, 8=16:55, 9=19:00, 10=19:55',
                    style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                const SizedBox(height: 8),
                ..._fields.map((r) => Padding(
                      padding: const EdgeInsets.symmetric(vertical: 3),
                      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Container(
                          width: 118,
                          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: theme.colorScheme.primaryContainer.withOpacity(0.4),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Text(r.$1, style: TextStyle(fontSize: 12, color: theme.colorScheme.primary, fontWeight: FontWeight.w600)),
                        ),
                        const SizedBox(width: 8),
                        Expanded(child: Text(r.$2, style: theme.textTheme.bodySmall)),
                      ]),
                    )),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
