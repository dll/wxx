import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/capability_utils.dart';

/// 教师 - 作业信息发布 + 课程成绩统计（只读）（P2 轻量版）
///
/// 蔚小芯侧重教育非教学：作业仅做信息发布+成绩统计，不做学生提交/批改/内容/附件/流转。
/// 设计要点（对齐 pm-check-homework-module.md §5 / §6 诚实红线）：
/// - 归属强约束：课程下拉=本人 approved 授课课程（复用 /teacher/homework/courses，杜绝无关课程）；
/// - 0 条诚实空：无 approved 课程 / 无作业 / 成绩 0 行，三种场景如实展示，不造数、不伪均分；
/// - 门控 teacher.grade.write：无权限入口在 home 隐藏，页面自身也拦截（双重保险）。
/// - 后端复用 TeacherGradeWrite 门控；发布/编辑/下架经 GetTeacherCourseStatus 强校验 approved。
class TeacherHomeworkPage extends StatefulWidget {
  const TeacherHomeworkPage({super.key});
  @override
  State<TeacherHomeworkPage> createState() => _TeacherHomeworkPageState();
}

class _TeacherHomeworkPageState extends State<TeacherHomeworkPage> {
  final ApiService _api = ApiService();

  bool _canWrite = false;
  bool _coursesLoading = false;
  bool _listLoading = false;
  bool _statsLoading = false;
  bool _submitting = false;

  String _formMsg = '';
  String _listMsg = '';
  String _statsMsg = '';

  List<dynamic> _approvedCourses = []; // 仅本人 approved 授课课程
  List<dynamic> _homework = [];        // 我的作业
  Map<String, dynamic>? _stats;        // 课程成绩统计（只读）
  bool _statsOpen = false;             // 是否打开统计抽屉

  // 表单控制器
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  final _publishAtCtrl = TextEditingController();
  final _dueAtCtrl = TextEditingController();
  String? _courseId;
  String? _courseName;
  String? _semester;
  int? _editingId;

  static const _semesters = ['2025-2026-1', '2025-2026-2', '2026-2027-1', '2026-2027-2'];

  @override
  void initState() {
    super.initState();
    _canWrite = CapabilityUtils.has(Capability.teacherGradeWrite);
    if (_canWrite) {
      _loadApprovedCourses();
      _loadMyHomework();
    }
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    _publishAtCtrl.dispose();
    _dueAtCtrl.dispose();
    super.dispose();
  }

  bool get _gateOpen => _canWrite;

  Future<void> _loadApprovedCourses() async {
    setState(() => _coursesLoading = true);
    try {
      final res = await _api.get(ApiConfig.teacherHomeworkCourses);
      final data = res.data is Map ? (res.data['list'] ?? []) : [];
      setState(() => _approvedCourses = data is List ? data : []);
    } catch (_) {
      setState(() => _approvedCourses = []);
    } finally {
      setState(() => _coursesLoading = false);
    }
  }

  Future<void> _loadMyHomework() async {
    setState(() => _listLoading = true);
    try {
      final res = await _api.get(ApiConfig.teacherHomeworkMine);
      final data = res.data is Map ? (res.data['list'] ?? []) : [];
      setState(() {
        _homework = data is List ? data : [];
        _listMsg = _homework.isEmpty ? '暂未发布作业。' : '';
      });
    } catch (e) {
      setState(() => _listMsg = '查询我的作业失败：$e');
    } finally {
      setState(() => _listLoading = false);
    }
  }

  void _resetForm() {
    _courseId = null;
    _courseName = null;
    _semester = null;
    _editingId = null;
    _titleCtrl.clear();
    _descCtrl.clear();
    _publishAtCtrl.clear();
    _dueAtCtrl.clear();
    _formMsg = '';
    _stats = null;
    _statsMsg = '';
    _statsOpen = false;
  }

  Future<void> _submit() async {
    if (!_gateOpen) {
      setState(() => _formMsg = '无教师权限（teacher.grade.write）');
      return;
    }
    final courseId = _courseId;
    final semester = _semester;
    final title = _titleCtrl.text.trim();
    if (courseId == null || semester == null || title.isEmpty) {
      setState(() => _formMsg = '请选择课程/学期并填写标题');
      return;
    }
    setState(() {
      _submitting = true;
      _formMsg = '';
    });
    try {
      final path = _editingId == null
          ? ApiConfig.teacherHomework
          : ApiConfig.teacherHomeworkItem(_editingId!);
      final payload = {
        'title': title,
        'description': _descCtrl.text.trim(),
        'publish_at': _publishAtCtrl.text.trim(),
        'due_at': _dueAtCtrl.text.trim(),
        if (_editingId == null) ...{
          'course_id': courseId,
          'course_name': _courseName ?? '',
          'semester': semester,
        },
      };
      final res = _editingId == null
          ? await _api.post(path, data: payload)
          : await _api.put(path, data: payload);
      final msg = res.data is Map
          ? (res.data['message']?.toString() ?? '保存成功')
          : res.data.toString();
      setState(() => _formMsg = msg);
      _resetForm();
      await _loadMyHomework();
    } catch (e) {
      setState(() => _formMsg = '保存失败：$e');
    } finally {
      setState(() => _submitting = false);
    }
  }

  Future<void> _edit(dynamic raw) async {
    if (raw is! Map) return;
    setState(() {
      _editingId = (raw['id'] as num?)?.toInt();
      _courseId = raw['course_id']?.toString();
      _courseName = raw['course_name']?.toString();
      _semester = raw['semester']?.toString();
      _titleCtrl.text = raw['title']?.toString() ?? '';
      _descCtrl.text = raw['description']?.toString() ?? '';
      _publishAtCtrl.text = raw['publish_at']?.toString() ?? '';
      _dueAtCtrl.text = raw['due_at']?.toString() ?? '';
      _formMsg = '';
    });
  }

  Future<void> _archive(int id) async {
    if (!_gateOpen) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (c) => AlertDialog(
        title: const Text('下架作业'),
        content: const Text('下架后该作业将不再展示（软删，可审计追溯）。确认下架？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(c, false), child: const Text('取消')),
          FilledButton(
            onPressed: () => Navigator.pop(c, true),
            child: const Text('确认下架'),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.delete(ApiConfig.teacherHomeworkItem(id));
      setState(() => _formMsg = '');
      await _loadMyHomework();
    } catch (e) {
      setState(() => _listMsg = '下架失败：$e');
    }
  }

  Future<void> _showStats(dynamic raw) async {
    if (raw is! Map) return;
    final courseId = raw['course_id']?.toString() ?? '';
    final semester = raw['semester']?.toString() ?? '';
    if (courseId.isEmpty || semester.isEmpty) return;
    setState(() {
      _statsLoading = true;
      _statsMsg = '';
      _stats = null;
    });
    try {
      final res = await _api.get(
        ApiConfig.teacherHomeworkGradeStats(courseId),
        params: {'semester': semester},
      );
      final data = res.data is Map ? res.data['stats'] : null;
      setState(() {
        if (data is Map) {
          _stats = Map<String, dynamic>.from(data);
        } else {
          _stats = null;
        }
        _statsOpen = true;
      });
    } catch (e) {
      setState(() => _statsMsg = '统计失败：$e');
    } finally {
      setState(() => _statsLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('作业信息发布')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (!_gateOpen)
            Card(
              color: theme.colorScheme.errorContainer,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  '您没有教师权限。若您是教师，请联系管理员为该角色授予 teacher.grade.write 能力。',
                  style: TextStyle(color: theme.colorScheme.onErrorContainer),
                ),
              ),
            )
          else ...[
            _buildBanner(theme),
            const SizedBox(height: 16),
            _buildFormCard(theme),
            const SizedBox(height: 16),
            _buildMyHomeworkCard(theme),
            const SizedBox(height: 16),
            if (_statsOpen) _buildStatsCard(theme),
            const SizedBox(height: 16),
            _buildHintCard(theme),
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
              '蔚小芯是教育辅助平台，不做教学管理：作业仅发布信息（标题/说明/时间），'
              '不做学生提交、教师批改、内容/附件流转。\n'
              '归属强约束：只能对已确认授课(approved)的课程发布作业（后端校验）。',
              style: theme.textTheme.bodySmall,
            ),
          ),
        ]),
      ),
    );
  }

  Widget _buildFormCard(ThemeData theme) {
    final editing = _editingId != null;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(editing ? Icons.edit : Icons.assignment_add,
                color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text(editing ? '编辑作业' : '发布作业信息',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
          ]),
          const SizedBox(height: 6),
          Text(
            '作业仅做信息发布（标题/说明/时间），归属为您已确认授课(approved)的课程；不做学生提交/批改/内容流转。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          if (_coursesLoading)
            const LinearProgressIndicator()
          else if (_approvedCourses.isEmpty && !editing)
            // 诚实空态：无 approved 课程，不填充虚构课程
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.secondaryContainer.withOpacity(0.4),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                '暂无已确认授课的课程，请先在「授课申报」申报所授课程，并由教辅/教务审核确认(approved)后，才能发布该课程作业。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSecondaryContainer),
              ),
            )
          else if (editing)
            ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: const Icon(Icons.menu_book),
              title: Text('$_courseId · $_courseName'),
              subtitle: Text('学期：$_semester（编辑不改归属）'),
            )
          else
            DropdownButtonFormField<String>(
              isExpanded: true,
              decoration: const InputDecoration(
                labelText: '课程（仅已确认授课）',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              hint: const Text('选择课程'),
              value: _courseId,
              items: _approvedCourses.map((c) {
                if (c is! Map) return const DropdownMenuItem<String>(value: null, child: SizedBox.shrink());
                final cid = c['course_id']?.toString() ?? '';
                final cname = c['course_name']?.toString() ?? '';
                final sem = c['semester']?.toString() ?? '';
                return DropdownMenuItem(
                  value: cid,
                  child: Text('$cid · ${cname.isEmpty ? '（未填课程名）' : cname}（$sem）',
                      overflow: TextOverflow.ellipsis),
                );
              }).toList(),
              onChanged: (v) {
                if (v == null) return;
                String? cname;
                String? sem;
                for (final e in _approvedCourses) {
                  if (e is Map && e['course_id']?.toString() == v) {
                    cname = e['course_name']?.toString() ?? '';
                    sem = e['semester']?.toString();
                    break;
                  }
                }
                setState(() {
                  _courseId = v;
                  _courseName = cname;
                  _semester = sem;
                });
              },
            ),
          if (!editing) ...[
            const SizedBox(height: 10),
            DropdownButtonFormField<String>(
              isExpanded: true,
              decoration: const InputDecoration(
                labelText: '学期',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              hint: const Text('选择学期'),
              value: _semester,
              items: [
                ..._semesters.map((s) => DropdownMenuItem(value: s, child: Text(s))),
                if (_semester != null && !_semesters.contains(_semester))
                  DropdownMenuItem(value: _semester, child: Text(_semester!)),
              ],
              onChanged: (v) => setState(() => _semester = v),
            ),
          ],
          const SizedBox(height: 10),
          TextField(
            controller: _titleCtrl,
            decoration: const InputDecoration(
              labelText: '作业标题 *',
              hintText: '如 第三章课后作业',
              border: OutlineInputBorder(),
              isDense: true,
            ),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: _descCtrl,
            maxLines: 3,
            decoration: const InputDecoration(
              labelText: '作业说明（可选）',
              hintText: '纯文本信息说明，不做内容/附件流转',
              border: OutlineInputBorder(),
              isDense: true,
            ),
          ),
          const SizedBox(height: 10),
          Row(children: [
            Expanded(
              child: TextField(
                controller: _publishAtCtrl,
                decoration: const InputDecoration(
                  labelText: '发布日期（可选）',
                  hintText: '如 2026-09-01',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: TextField(
                controller: _dueAtCtrl,
                decoration: const InputDecoration(
                  labelText: '截止日期（可选）',
                  hintText: '如 2026-09-15',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
              ),
            ),
          ]),
          const SizedBox(height: 12),
          Row(children: [
            FilledButton.icon(
              onPressed: _submitting ? null : _submit,
              icon: _submitting
                  ? const SizedBox(
                      width: 16, height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Icon(_editingId == null ? Icons.send : Icons.save),
              label: Text(_editingId == null ? '发布作业' : '保存修改'),
            ),
            if (editing) ...[
              const SizedBox(width: 8),
              OutlinedButton(
                onPressed: _resetForm,
                child: const Text('取消编辑'),
              ),
            ],
          ]),
          if (_formMsg.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              _formMsg,
              style: theme.textTheme.bodySmall?.copyWith(
                color: _formMsg.contains('成功') || _formMsg.contains('已')
                    ? Colors.green
                    : Colors.red,
              ),
            ),
          ],
        ]),
      ),
    );
  }

  Widget _buildMyHomeworkCard(ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(Icons.list_alt, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text('我的作业',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
            const Spacer(),
            OutlinedButton.icon(
              onPressed: _listLoading ? null : _loadMyHomework,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('刷新'),
            ),
          ]),
          const SizedBox(height: 8),
          if (_listLoading)
            const Center(
                child: Padding(
              padding: EdgeInsets.all(8),
              child: CircularProgressIndicator(strokeWidth: 2),
            ))
          else if (_listMsg.isNotEmpty)
            Text(_listMsg,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
          else if (_homework.isEmpty)
            Text('暂未发布作业。请在上方发布您所授班级的真实作业信息。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
          else
            ..._homework.map((hw) => _hwTile(theme, hw)),
        ]),
      ),
    );
  }

  Widget _buildHintCard(ThemeData theme) {
    return Card(
      color: theme.colorScheme.primaryContainer.withOpacity(0.35),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(Icons.info_outline, color: theme.colorScheme.primary),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              '点作业右侧「成绩」可查看该课程成绩统计（人数/均分/及格率/分档）。'
              '统计基于您已录入的真实 student_grades 只读聚合；无成绩时如实显示「暂无成绩记录」，不做伪造均分。',
              style: theme.textTheme.bodySmall,
            ),
          ),
        ]),
      ),
    );
  }

  Widget _buildStatsCard(ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(Icons.bar_chart, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text('课程成绩统计',
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
            const Spacer(),
            IconButton(
              onPressed: () => setState(() => _statsOpen = false),
              icon: const Icon(Icons.close, size: 20),
              tooltip: '关闭',
            ),
          ]),
          const SizedBox(height: 8),
          if (_statsLoading)
            const Center(
                child: Padding(
              padding: EdgeInsets.all(8),
              child: CircularProgressIndicator(strokeWidth: 2),
            ))
          else if (_statsMsg.isNotEmpty)
            Text(_statsMsg,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.error))
          else if (_stats == null)
            Text('暂无数据。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant))
          else if ((_stats?['total'] as num?) == 0 || _stats?['not_available'] == true)
            // 诚实空态：0 行成绩不补造分布/伪均分
            Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              const _StatRow(label: '人数', value: '0'),
              const SizedBox(height: 12),
              Text(
                '暂无成绩记录（total=0）。请在「成绩录入」录入该课程学生的真实期末成绩后，再查看统计。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
              ),
            ])
          else ...[
            _StatRow(label: '人数', value: '${_stats?['total'] ?? 0}'),
            _StatRow(
                label: '均分',
                value: ((_stats!['avg_score'] as num?) ?? 0).toStringAsFixed(1)),
            _StatRow(
                label: '及格率',
                value:
                    '${((((_stats!['pass_rate'] as num?) ?? 0)) * 100).toStringAsFixed(1)}%'),
            _StatRow(label: '及格人数', value: '${_stats?['passed_count'] ?? 0}'),
            const Divider(height: 20),
            Text('分档分布', style: theme.textTheme.titleSmall),
            const SizedBox(height: 8),
            _levelBar(theme, '优秀', (_stats!['levels'] as Map?)?['优秀'] ?? 0),
            _levelBar(theme, '良好', (_stats!['levels'] as Map?)?['良好'] ?? 0),
            _levelBar(theme, '及格', (_stats!['levels'] as Map?)?['及格'] ?? 0),
            _levelBar(theme, '不及格', (_stats!['levels'] as Map?)?['不及格'] ?? 0),
          ],
        ]),
      ),
    );
  }

  Widget _levelBar(ThemeData theme, String label, num count) {
    final total = (_stats?['total'] as num?)?.toDouble() ?? 0;
    final fraction = total <= 0 ? 0.0 : (count.toDouble() / total);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(children: [
        SizedBox(width: 76, child: Text(label, style: theme.textTheme.bodySmall)),
        Expanded(
          child: LinearProgressIndicator(
            value: fraction,
            minHeight: 6,
            borderRadius: BorderRadius.circular(4),
          ),
        ),
        const SizedBox(width: 12),
        SizedBox(
            width: 36,
            child: Text('$count', textAlign: TextAlign.right)),
      ]),
    );
  }

  String _statusLabel(String s) {
    switch (s) {
      case 'archived':
        return '已下架';
      case 'published':
        return '已发布';
      default:
        return '进行中';
    }
  }

  Widget _hwTile(ThemeData theme, dynamic raw) {
    if (raw is! Map) return const SizedBox.shrink();
    final id = (raw['id'] as num?)?.toInt() ?? 0;
    final title = raw['title']?.toString() ?? '';
    final courseId = raw['course_id']?.toString() ?? '';
    final courseName = raw['course_name']?.toString() ?? '';
    final semester = raw['semester']?.toString() ?? '';
    final desc = raw['description']?.toString() ?? '';
    final dueAt = raw['due_at']?.toString() ?? '';
    final status = raw['status']?.toString() ?? 'active';
    final archived = status == 'archived';
    return Column(children: [
      ListTile(
        dense: true,
        leading: Icon(Icons.assignment,
            color: archived
                ? theme.colorScheme.outline
                : theme.colorScheme.primary),
        title: Text(title,
            maxLines: 1, overflow: TextOverflow.ellipsis,
            style: archived
                ? TextStyle(color: theme.colorScheme.outline)
                : null),
        subtitle: Text([
          '$courseId · ${courseName.isEmpty ? '（未填课程名）' : courseName}',
          semester,
          if (desc.isNotEmpty) '说明：$desc',
          if (dueAt.isNotEmpty) '截止：$dueAt',
          _statusLabel(status),
        ].join(' · '), maxLines: 2, overflow: TextOverflow.ellipsis),
        trailing: Row(mainAxisSize: MainAxisSize.min, children: [
          if (!archived) ...[
            TextButton(
              onPressed: () => _showStats(raw),
              child: const Text('成绩'),
            ),
            IconButton(
              onPressed: () => _edit(raw),
              icon: const Icon(Icons.edit_outlined, size: 20),
              tooltip: '编辑',
            ),
            IconButton(
              onPressed: () => _archive(id),
              icon: const Icon(Icons.archive_outlined, size: 20),
              tooltip: '下架',
            ),
          ],
        ]),
      ),
      const Divider(height: 1),
    ]);
  }
}

/// 统计行：任意标签 + 值
class _StatRow extends StatelessWidget {
  const _StatRow({required this.label, required this.value});
  final String label;
  final String value;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(children: [
        SizedBox(
            width: 90,
            child: Text(label, style: theme.textTheme.bodyMedium)),
        Expanded(
          child: Text(value,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.bold)),
        ),
      ]),
    );
  }
}
