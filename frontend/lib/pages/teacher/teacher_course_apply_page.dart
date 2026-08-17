import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/capability_utils.dart';

/// 教师 - 授课关系申报（R3 越权边界升级）
///
/// 设计要点（对齐 pm-check-teacher-course-review.md §6.1 / §7 诚实红线）：
/// - 教师申报「所授课程」，经教辅/教务审核 approved 后，才能录入该课程成绩；
/// - **不造权威关系**：approved 唯一来源为教辅真实审核，教师只能看到 pending/approved/rejected 状态；
/// - 门控 teacher.grade.write（申报复用）；无权限页面自身拦截 + 入口隐藏双保险；
/// - 诚实空态：0 申报 → 「暂无授课申报，请先申报并由教辅确认」。
class TeacherCourseApplyPage extends StatefulWidget {
  const TeacherCourseApplyPage({super.key});
  @override
  State<TeacherCourseApplyPage> createState() => _TeacherCourseApplyPageState();
}

class _TeacherCourseApplyPageState extends State<TeacherCourseApplyPage> {
  final ApiService _api = ApiService();
  final _courseIdCtrl = TextEditingController();
  final _courseNameCtrl = TextEditingController();
  final _semesterCtrl = TextEditingController();
  bool _submitting = false;
  bool _loading = false;
  String _submitMsg = '';
  List<dynamic> _mine = [];

  bool get _canWrite => CapabilityUtils.has(Capability.teacherGradeWrite);

  // 常见学期下拉（仅作提示，仍允许手输）
  static const _semesters = ['2025-2026-1', '2025-2026-2', '2026-2027-1', '2026-2027-2'];

  @override
  void initState() {
    super.initState();
    _loadMine();
  }

  @override
  void dispose() {
    _courseIdCtrl.dispose();
    _courseNameCtrl.dispose();
    _semesterCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_canWrite) {
      setState(() => _submitMsg = '无申报权限（teacher.grade.write）');
      return;
    }
    final courseId = _courseIdCtrl.text.trim();
    final semester = _semesterCtrl.text.trim();
    if (courseId.isEmpty || semester.isEmpty) {
      setState(() => _submitMsg = '请填写课程编号和学期');
      return;
    }
    setState(() {
      _submitting = true;
      _submitMsg = '';
    });
    try {
      final res = await _api.post(ApiConfig.teacherCourseApply, data: {
        'course_id': courseId,
        'course_name': _courseNameCtrl.text.trim(),
        'semester': semester,
      });
      final msg = res.data is Map
          ? (res.data['message']?.toString() ?? '申报成功')
          : res.data.toString();
      setState(() => _submitMsg = msg);
      _courseIdCtrl.clear();
      _courseNameCtrl.clear();
      await _loadMine();
    } catch (e) {
      setState(() => _submitMsg = '申报失败：$e');
    } finally {
      setState(() => _submitting = false);
    }
  }

  Future<void> _loadMine() async {
    if (!_canWrite) {
      return;
    }
    setState(() => _loading = true);
    try {
      final res = await _api.get(ApiConfig.teacherCoursesMine);
      final data = res.data is Map ? (res.data['list'] ?? []) : [];
      setState(() => _mine = data is List ? data : []);
    } catch (_) {
      // 静默：列表中已有诚实空态
    } finally {
      setState(() => _loading = false);
    }
  }

  String _statusLabel(String s) {
    switch (s) {
      case 'approved':
        return '已确认（可录成绩）';
      case 'rejected':
        return '已驳回';
      default:
        return '待审核';
    }
  }

  Color _statusColor(String s) {
    switch (s) {
      case 'approved':
        return Colors.green;
      case 'rejected':
        return Colors.red;
      default:
        return Colors.orange;
    }
  }

  Widget _statusChip(ThemeData theme, String s) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: _statusColor(s).withOpacity(0.12),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Text(
        _statusLabel(s),
        style: TextStyle(fontSize: 12, color: _statusColor(s), fontWeight: FontWeight.w600),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('我的授课申报（R3）')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (!_canWrite)
            Card(
              color: theme.colorScheme.errorContainer,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  '您没有教师申报权限。若您是教师，请联系管理员为该角色授予 teacher.grade.write 能力。',
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
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text('申报授课关系',
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 6),
                  Text(
                    '申报后需教辅/教务审核确认（approved），确认后才能录入该课程成绩。',
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _courseIdCtrl,
                    decoration: const InputDecoration(
                      labelText: '课程编号（course_id）',
                      hintText: '如 CS103',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                  const SizedBox(height: 10),
                  TextField(
                    controller: _courseNameCtrl,
                    decoration: const InputDecoration(
                      labelText: '课程名称（可选）',
                      hintText: '如 数据结构',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                  ),
                  const SizedBox(height: 10),
                  DropdownButtonFormField<String>(
                    value: _semesterCtrl.text.isEmpty ? null : _semesterCtrl.text,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: '学期',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    hint: const Text('选择学期'),
                    items: _semesters
                        .map((s) => DropdownMenuItem(value: s, child: Text(s)))
                        .toList(),
                    onChanged: (v) {
                      if (v != null) _semesterCtrl.text = v;
                    },
                  ),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: _submitting ? null : _submit,
                    icon: _submitting
                        ? const SizedBox(
                            width: 16, height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.send),
                    label: const Text('提交申报'),
                  ),
                  if (_submitMsg.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Text(
                      _submitMsg,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: _submitMsg.contains('成功') || _submitMsg.contains('已确认')
                            ? Colors.green
                            : Colors.red,
                      ),
                    ),
                  ],
                ]),
              ),
            ),
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Icon(Icons.list_alt, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('我的申报列表',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold)),
                    const Spacer(),
                    OutlinedButton.icon(
                      onPressed: _loading ? null : _loadMine,
                      icon: const Icon(Icons.refresh, size: 18),
                      label: const Text('刷新'),
                    ),
                  ]),
                  const SizedBox(height: 8),
                  if (_loading)
                    const Center(
                        child: Padding(
                      padding: EdgeInsets.all(8),
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ))
                  else if (_mine.isEmpty)
                    // 诚实空态：0 申报不伪造成已确认关系
                    Text(
                      '暂无授课申报。请先申报所授课程，并由教辅/教务审核确认通过后才能录入成绩。',
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                    )
                  else
                    ..._mine.map((tc) => _tile(theme, tc)),
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
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(Icons.info_outline, color: theme.colorScheme.primary),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              '越权边界升级（R3）：只有通过教辅/教务审核确认（approved）的课程才能录入成绩。\n'
              '系统不会自动生成或伪造授课关系 —— approved 只能来自教辅的真实审核操作。',
              style: theme.textTheme.bodySmall,
            ),
          ),
        ]),
      ),
    );
  }

  Widget _tile(ThemeData theme, dynamic raw) {
    if (raw is! Map) return const SizedBox.shrink();
    final courseId = raw['course_id']?.toString() ?? '';
    final courseName = raw['course_name']?.toString() ?? '';
    final semester = raw['semester']?.toString() ?? '';
    final status = raw['status']?.toString() ?? 'pending';
    final note = raw['review_note']?.toString() ?? '';
    final reviewer = raw['reviewed_name']?.toString() ?? '';
    return ListTile(
      dense: true,
      leading: Icon(Icons.menu_book,
          color: status == 'approved'
              ? Colors.green
              : status == 'rejected'
                  ? Colors.red
                  : Colors.orange),
      title: Text('$courseId · ${courseName.isEmpty ? '（未填课程名）' : courseName}'),
      subtitle: Text([
        semester,
        if (note.isNotEmpty) '备注：$note',
        if (reviewer.isNotEmpty) '审核人：$reviewer',
      ].join(' · ')),
      trailing: _statusChip(theme, status),
    );
  }
}
