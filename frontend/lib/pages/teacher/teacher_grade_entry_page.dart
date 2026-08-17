import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/capability_utils.dart';

/// 教师 - 成绩录入（方案A升级：教师申报授课关系→教辅审核【approved】后才可录入真实成绩）
///
/// 设计要点（对齐 pm 清单 P0-1 / P1 / R3）：
/// - 教师先通过“授课申报”声明课程（teacher_courses，R3）；教辅/教务审核 approved 后，
///   后端在写库前强校验该教师-课程-学期授课关系为 approved，未确认课程拒绝写入（停用旧的“声明即授权”）；
/// - 后端强制 target 为 student 角色 + created_by/updated_by=当前教师，审计可追溯；
/// - **绝不造数**：生产 student_grades=0 时显示诚实空态「成绩待教师录入真实数据」；
/// - 门控 teacher.grade.write（无权限入口在 home 隐藏，页面自身也拦截）。
class TeacherGradeEntryPage extends StatefulWidget {
  const TeacherGradeEntryPage({super.key});
  @override
  State<TeacherGradeEntryPage> createState() => _TeacherGradeEntryPageState();
}

class _TeacherGradeEntryPageState extends State<TeacherGradeEntryPage> {
  final ApiService _api = ApiService();
  final _gradeCtrl = TextEditingController();
  bool _importLoading = false;
  bool _listLoading = false;
  String _importResult = '';
  String _listResult = '';
  List<dynamic> _mineGrades = [];

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
    },
    {
      "user_id": "2",
      "course_id": "CS103",
      "course_name": "数据结构",
      "semester": "2025-2026-2",
      "score": 92,
      "gpa": 3.8,
      "passed": true,
      "credits": 4
    }
  ]
}''';

  @override
  void dispose() {
    _gradeCtrl.dispose();
    super.dispose();
  }

  /// 门控 1：本页面为教师成绩录入专属，无权限直接提示（配合入口隐藏形成双重保险）
  bool get _canWrite => CapabilityUtils.has(Capability.teacherGradeWrite);

  Future<void> _importGrades() async {
    if (!_canWrite) {
      setState(() => _importResult = '无教师成绩录入权限（teacher.grade.write）');
      return;
    }
    if (_gradeCtrl.text.trim().isEmpty) {
      setState(() => _importResult = '请先填入成绩 JSON');
      return;
    }
    setState(() { _importLoading = true; _importResult = ''; });
    try {
      final map = _parseJson(_gradeCtrl.text);
      final res =
          await _api.post(ApiConfig.teacherGradesImport, data: map);
      setState(() => _importResult = _formatResult(res.data));
      // 导入成功后刷新「我已录入」列表
      await _loadMine();
    } catch (e) {
      setState(() => _importResult = '导入失败：$e');
    } finally {
      setState(() => _importLoading = false);
    }
  }

  Future<void> _loadMine() async {
    if (!_canWrite) {
      setState(() => _listResult = '无教师成绩读取权限');
      return;
    }
    setState(() {
      _listLoading = true;
    });
    try {
      final res = await _api.get(ApiConfig.teacherGradesMine);
      final data = res.data is Map ? (res.data['data'] ?? []) : [];
      setState(() {
        _mineGrades = data is List ? data : [];
        // 诚实空态：无数据时明确提示，不伪造成绩数字
        _listResult = _mineGrades.isEmpty ? '暂无已录入成绩。请在上方录入您所授班级学生的真实期末成绩。' : '';
      });
    } catch (e) {
      setState(() => _listResult = '查询已录入成绩失败：$e');
    } finally {
      setState(() => _listLoading = false);
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
      var s = '录入完成：共 $total 条，新增 $created，更新 $updated';
      if (errors.isNotEmpty) {
        s += '\n失败 ${errors.length} 条：${errors.take(5).join('；')}';
      }
      return s;
    } catch (_) {
      return data.toString();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('成绩录入（所授班级）')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (!_canWrite)
            Card(
              color: theme.colorScheme.errorContainer,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  '您没有教师成绩录入权限。若您是教师，请联系管理员为该角色授予 teacher.grade.write 能力。',
                  style: TextStyle(color: theme.colorScheme.onErrorContainer),
                ),
              ),
            )
          else ...[
            _buildBanner(theme),
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(children: [
                        Icon(Icons.grade_outlined,
                            color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Text(
                          '录入成绩（JSON）',
                          style: theme.textTheme.titleMedium
                              ?.copyWith(fontWeight: FontWeight.bold),
                        ),
                      ]),
                      const SizedBox(height: 6),
                      Text(
                        '这是您声明授课的班级成绩。填写您所授班级学生的真实期末成绩。',
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _gradeCtrl,
                        maxLines: 14,
                        decoration: const InputDecoration(
                          hintText: '粘贴成绩 JSON（可点「填入示例」）',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Row(children: [
                        OutlinedButton(
                          onPressed: () => _gradeCtrl.text = _gradeExample,
                          child: const Text('填入示例'),
                        ),
                        const SizedBox(width: 8),
                        FilledButton.icon(
                          onPressed:
                              _importLoading ? null : _importGrades,
                          icon: _importLoading
                              ? const SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2))
                              : const Icon(Icons.upload),
                          label: const Text('提交录入'),
                        ),
                      ]),
                      if (_importResult.isNotEmpty) ...[
                        const SizedBox(height: 8),
                        Text(
                          _importResult,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: _importResult.startsWith('录入完成')
                                ? Colors.green
                                : Colors.red,
                          ),
                        ),
                      ],
                    ]),
              ),
            ),
            const SizedBox(height: 16),
            _buildFieldHelp(theme),
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(children: [
                        Icon(Icons.list_alt, color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Text('我已录入的成绩',
                            style: theme.textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.bold)),
                        const Spacer(),
                        OutlinedButton.icon(
                          onPressed: _listLoading ? null : _loadMine,
                          icon: const Icon(Icons.refresh, size: 18),
                          label: const Text('刷新'),
                        ),
                      ]),
                      const SizedBox(height: 8),
                      if (_listLoading)
                        const Center(
                            child: Padding(
                          padding: EdgeInsets.all(8),
                          child:
                              CircularProgressIndicator(strokeWidth: 2),
                        ))
                      else if (_listResult.isNotEmpty)
                        Text(_listResult,
                            style: theme.textTheme.bodySmall?.copyWith(
                                color:
                                    theme.colorScheme.onSurfaceVariant))
                      else if (_mineGrades.isNotEmpty)
                        ..._mineGrades.map((g) => _gradeTile(theme, g))
                      else
                        Text('暂无已录入成绩。请在上方录入您所授班级学生的真实期末成绩。',
                            style: theme.textTheme.bodySmall?.copyWith(
                                color:
                                    theme.colorScheme.onSurfaceVariant)),
                    ]),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildBanner(ThemeData theme) {
    return Card(
      color: theme.colorScheme.primaryContainer.withOpacity(0.5),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Icon(Icons.info_outline, color: theme.colorScheme.primary),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                '您在此声明的授课关系与成绩将被记录（created_by=您本人，可审计追溯）。\n'
                '仅能录入本校 role=student 学生的真实期末成绩；教师/管理员等非学生对象会被后端拒绝。\n'
                '请勿录入未核实或编造的成绩 —— 学生端将如实展示您录入的数据。',
                style: theme.textTheme.bodySmall,
              ),
            ),
          ]),
          const SizedBox(height: 8),
          Row(children: [
            Icon(Icons.verified_outlined,
                size: 18, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                'R3 强校验：只有经教辅/教务审核确认（approved）的授课课程才能录入成绩。未确认课程会被后端拒绝。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.tertiary),
              ),
            ),
            TextButton(
              onPressed: () =>
                  context.push('/teacher/course-apply'),
              child: const Text('去申报/查授课'),
            ),
          ]),
        ]),
      ),
    );
  }

  Widget _buildFieldHelp(ThemeData theme) {
    const rows = [
      ('user_id', '学生学号 / 用户 ID（必须为学生）'),
      ('course_id', '课程编号（用于幂等去重，如 CS103）'),
      ('course_name', '课程名称，如 数据结构'),
      ('semester', '学期，如 2025-2026-2'),
      ('score', '期末成绩（真实分数）'),
      ('gpa', '绩点（可选）'),
      ('passed', '是否及格 true/false'),
      ('credits', '学分'),
    ];
    return Card(
      child: ExpansionTile(
        leading: Icon(Icons.help_outline, color: theme.colorScheme.primary),
        title:
            Text('字段说明', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600)),
        childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ...rows.map((r) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 3),
                child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Container(
                        width: 118,
                        padding: const EdgeInsets.symmetric(
                            horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color:
                              theme.colorScheme.primaryContainer.withOpacity(0.4),
                          borderRadius: BorderRadius.circular(6),
                        ),
                        child: Text(r.$1,
                            style: TextStyle(
                                fontSize: 12,
                                color: theme.colorScheme.primary,
                                fontWeight: FontWeight.w600)),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                          child:
                              Text(r.$2, style: theme.textTheme.bodySmall)),
                    ]),
              )),
        ],
      ),
    );
  }

  Widget _gradeTile(ThemeData theme, dynamic raw) {
    if (raw is! Map) return const SizedBox.shrink();
    final name = raw['name']?.toString() ?? '';
    final username = raw['username']?.toString() ?? '';
    final course = raw['course_name']?.toString() ?? '';
    final courseId = raw['course_id']?.toString() ?? '';
    final semester = raw['semester']?.toString() ?? '';
    final score = raw['score'];
    final passed = raw['passed'] == true;
    return ListTile(
      dense: true,
      leading: Icon(passed ? Icons.check_circle : Icons.cancel,
          color: passed ? Colors.green : Colors.red),
      title: Text('$name($username) · $course($courseId)'),
      subtitle: Text(semester),
      trailing: Text('$score',
          style: theme.textTheme.titleMedium
              ?.copyWith(fontWeight: FontWeight.bold)),
    );
  }
}
