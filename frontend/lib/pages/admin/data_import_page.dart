import 'dart:convert';

import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 管理员 - 数据底座导入（成绩 / 课表 JSON）
class DataImportPage extends StatefulWidget {
  const DataImportPage({super.key});
  @override
  State<DataImportPage> createState() => _DataImportPageState();
}

class _DataImportPageState extends State<DataImportPage> {
  final ApiService _api = ApiService();
  final _gradeCtrl = TextEditingController();
  final _scheduleCtrl = TextEditingController();
  bool _gradeLoading = false;
  bool _scheduleLoading = false;
  String _gradeResult = '';
  String _scheduleResult = '';

  static const _gradeExample = '''
{
  "grades": [
    {
      "user_id": "1",
      "course_id": "CS103",
      "course_name": "数据结构",
      "semester": "2025-2026-2",
      "score": 88,
      "gpa": 3.5,
      "passed": true,
      "credits": 4
    }
  ]
}''';

  static const _scheduleExample = '''
{
  "schedules": [
    {
      "username": "120001",
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

  @override
  void dispose() {
    _gradeCtrl.dispose();
    _scheduleCtrl.dispose();
    super.dispose();
  }

  Future<void> _importGrades() async {
    if (_gradeCtrl.text.trim().isEmpty) return;
    setState(() { _gradeLoading = true; _gradeResult = ''; });
    try {
      final res = await _api.post(ApiConfig.adminGradesImport, data: _parseJson(_gradeCtrl.text));
      setState(() => _gradeResult = _formatResult(res.data));
    } catch (e) {
      setState(() => _gradeResult = '导入失败：$e');
    } finally {
      setState(() => _gradeLoading = false);
    }
  }

  Future<void> _importSchedules() async {
    if (_scheduleCtrl.text.trim().isEmpty) return;
    setState(() { _scheduleLoading = true; _scheduleResult = ''; });
    try {
      final res = await _api.post(ApiConfig.adminSchedulesImport, data: _parseJson(_scheduleCtrl.text));
      setState(() => _scheduleResult = _formatResult(res.data));
    } catch (e) {
      setState(() => _scheduleResult = '导入失败：$e');
    } finally {
      setState(() => _scheduleLoading = false);
    }
  }

  Map<String, dynamic> _parseJson(String text) {
    try {
      final decoded = jsonDecode(text);
      if (decoded is Map<String, dynamic>) return decoded;
      throw const FormatException('需要 JSON 对象');
    } catch (_) {
      throw const FormatException('JSON 格式错误');
    }
  }

  String _formatResult(dynamic data) {
    try {
      final d = data is Map ? (data['data'] ?? data) : data;
      final total = d is Map ? (d['total'] ?? 0) : 0;
      final created = d is Map ? (d['created'] ?? 0) : 0;
      final updated = d is Map ? (d['updated'] ?? 0) : 0;
      final errors = d is Map ? (d['errors'] as List? ?? []) : [];
      var s = '导入完成：共 $total 条，新增 $created，更新 $updated';
      if (errors.isNotEmpty) s += '\n失败 ${errors.length} 条：${errors.take(5).join('；')}';
      return s;
    } catch (_) {
      return data.toString();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('数据底座导入')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _section(
            theme,
            icon: Icons.grade,
            title: '成绩导入（JSON）',
            hint: '填入成绩 JSON（可先点「填入示例」）',
            controller: _gradeCtrl,
            example: _gradeExample,
            loading: _gradeLoading,
            result: _gradeResult,
            onImport: _importGrades,
          ),
          const SizedBox(height: 24),
          _section(
            theme,
            icon: Icons.calendar_view_week,
            title: '课表导入（JSON）',
            hint: '填入课表 JSON',
            controller: _scheduleCtrl,
            example: _scheduleExample,
            loading: _scheduleLoading,
            result: _scheduleResult,
            onImport: _importSchedules,
          ),
          const SizedBox(height: 8),
          _buildScheduleHelp(theme),
        ],
      ),
    );
  }

  /// 课表字段说明：帮助辅导员/教务员填对 JSON（准确性第一）
  Widget _buildScheduleHelp(ThemeData theme) {
    const rows = [
      ('user_id', '学生用户 ID（数字）'),
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
    return Card(
      child: ExpansionTile(
        leading: Icon(Icons.help_outline, color: theme.colorScheme.primary),
        title: Text('课表字段说明', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600)),
        childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('节次对应时间段：1=08:00, 2=08:55, 3=10:00, 4=10:55, 5=14:00, 6=14:55, 7=16:00, 8=16:55, 9=19:00, 10=19:55',
              style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          const SizedBox(height: 8),
          ...rows.map((r) => Padding(
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
    );
  }

  Widget _section(ThemeData theme, {required IconData icon, required String title, required String hint,
      required TextEditingController controller, required String example, required bool loading,
      required String result, required VoidCallback onImport}) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(icon, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text(title, style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          ]),
          const SizedBox(height: 12),
          TextField(
            controller: controller,
            maxLines: 10,
            decoration: InputDecoration(
              hintText: hint,
              border: const OutlineInputBorder(),
              isDense: true,
              helperText: example,
            ),
          ),
          const SizedBox(height: 8),
          Row(children: [
            OutlinedButton(onPressed: () => controller.text = example, child: const Text('填入示例')),
            const SizedBox(width: 8),
            FilledButton.icon(
              onPressed: loading ? null : onImport,
              icon: loading ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.upload),
              label: const Text('导入'),
            ),
          ]),
          if (result.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(result, style: theme.textTheme.bodySmall?.copyWith(color: result.startsWith('导入完成') ? Colors.green : Colors.red)),
          ],
        ]),
      ),
    );
  }
}
