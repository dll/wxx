import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/capability_utils.dart';

/// 教师 - 作业信息发布 + 该课程成绩统计（P2 轻量版）
///
/// 设计要点（对齐 pm-check-homework-module.md §5 / §6 诚实红线）：
/// - 蔚小芯侧重教育非教学：作业仅「信息发布 + 成绩统计（只读，复用现有 student_grades）」，
///   不做学生提交/教师批改/内容/附件/流转；
/// - **不造数据**：作业仅教师真实发布，统计仅基于真实成绩；approved 唯一来源为教辅真实审核；
/// - 归属强约束：课程下拉只给「本人 approved 授课课程」（teacher/homework/courses 白名单），
///   杜绝发布无关课程作业；0 approved → 诚实提示「先申报并经教辅确认」；
/// - 门控 teacher.grade.write（无权限入口隐藏 + 页面自身拦截双保险）；
/// - 诚实空态：无 approved 课程 / 无作业 / 0 成绩 三种场景如实展示，不伪造。
class TeacherHomeworkPage extends StatefulWidget {
  const TeacherHomeworkPage({super.key});
  @override
  State<TeacherHomeworkPage> createState() => _TeacherHomeworkPageState();
}

class _TeacherHomeworkPageState extends State<TeacherHomeworkPage> {
  final ApiService _api = ApiService();

  // 发布表单
  String? _selCourseId; // 选中课程 course_id
  String _selCourseName = '';
  String _semester = '';
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  final _publishCtrl = TextEditingController();
  final _dueCtrl = TextEditingController();
  bool _publishing = false;
  String _publishMsg = '';

  // 数据
  List<dynamic> _courses = []; // approved 授课白名单（课程下拉）
  List<dynamic> _homework = []; // 我的作业清单
  bool _loadingCourses = false;
  bool _loadingHomework = false;

  // 成绩统计（当前查看的课程）
  String? _statsCourseId;
  Map<String, dynamic> _stats = {};
  bool _loadingStats = false;

  bool get _canWrite => CapabilityUtils.has(Capability.teacherGradeWrite);

  static const _semesters = ['2025-2026-1', '2025-2026-2', '2026-2027-1', '2026-2027-2'];

  @override
  void initState() {
    super.initState();
    // 取最近学期作默认发布学期
    _semester = _semesters.isNotEmpty ? _semesters.last : '';
    _loadCourses();
    _loadHomework();
  }

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    _publishCtrl.dispose();
    _dueCtrl.dispose();
    super.dispose();
  }

  /// 课程下拉数据源：仅本人 approved 授课课程（真实白名单，0 条诚实提示）
  Future<void> _loadCourses() async {
    if (!_canWrite) return;
    setState(() => _loadingCourses = true);
    try {
      final res = await _api.get(ApiConfig.teacherHomeworkCourses);
      final data = res.data is Map ? (res.data['list'] ?? []) : [];
      setState(() => _courses = data is List ? data : []);
      if (_selCourseId == null && _courses.isNotEmpty) {
        final first = _courses.first;
        _selCourseId = first['course_id']?.toString();
        _selCourseName = first['course_name']?.toString() ?? '';
      }
    } catch (_) {
      // 静默：诚实空态兜底
    } finally {
      setState(() => _loadingCourses = false);
    }
  }

  Future<void> _loadHomework() async {
    if (!_canWrite) return;
    setState(() => _loadingHomework = true);
    try {
      final res = await _api.get(ApiConfig.teacherHomeworkMine);
      final data = res.data is Map ? (res.data['list'] ?? []) : [];
      setState(() => _homework = data is List ? data : []);
    } catch (_) {
      // 静默
    } finally {
      setState(() => _loadingHomework = false);
    }
  }

  Future<void> _publish() async {
    if (!_canWrite) {
      setState(() => _publishMsg = '无教师作业权限（teacher.grade.write）');
      return;
    }
    if (_selCourseId == null || _selCourseId!.isEmpty) {
      setState(() => _publishMsg = '请选择课程（需已确认授课）');
      return;
    }
    if (_semester.trim().isEmpty || _titleCtrl.text.trim().isEmpty) {
      setState(() => _publishMsg = '请填写学期和作业标题');
      return;
    }
    setState(() {
      _publishing = true;
      _publishMsg = '';
    });
    try {
      final res = await _api.post(ApiConfig.teacherHomeworkCreate, data: {
        'course_id': _selCourseId,
        'course_name': _selCourseName,
        'semester': _semester.trim(),
        'title': _titleCtrl.text.trim(),
        'description': _descCtrl.text.trim(),
        'publish_at': _publishCtrl.text.trim(),
        'due_at': _dueCtrl.text.trim(),
      });
      final msg = res.data is Map
          ? (res.data['message']?.toString() ?? '作业发布成功')
          : res.data.toString();
      setState(() => _publishMsg = msg);
      _titleCtrl.clear();
      _descCtrl.clear();
      _publishCtrl.clear();
      _dueCtrl.clear();
      await _loadHomework();
    } catch (e) {
      setState(() => _publishMsg = '发布失败：$e');
    } finally {
      setState(() => _publishing = false);
    }
  }

  Future<void> _archive(int id) async {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('下架作业'),
        content: const Text('下架为软删除（置 archived，可追踪），作业不再于清单展示。确定下架？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(
            onPressed: () {
              Navigator.pop(ctx);
              _doArchive(id);
            },
            child: const Text('确定下架'),
          ),
        ],
      ),
    );
  }

  Future<void> _doArchive(int id) async {
    try {
      await _api.delete('${ApiConfig.teacherHomeworkCreate}/$id');
      await _loadHomework();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('下架失败：$e')));
      }
    }
  }

  Future<void> _edit(int id, String curTitle, String curDesc) async {
    final titleCtrl = TextEditingController(text: curTitle);
    final descCtrl = TextEditingController(text: curDesc);
    final msg = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('编辑作业'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: titleCtrl,
              decoration: const InputDecoration(labelText: '标题', isDense: true),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: descCtrl,
              maxLines: 3,
              decoration: const InputDecoration(labelText: '说明/要求', isDense: true),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, 'cancel'), child: const Text('取消')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, 'save'),
            child: const Text('保存'),
          ),
        ],
      ),
    );
    if (msg != 'save' || titleCtrl.text.trim().isEmpty) return;
    try {
      await _api.put('${ApiConfig.teacherHomeworkCreate}/$id', data: {
        'title': titleCtrl.text.trim(),
        'description': descCtrl.text.trim(),
        'publish_at': '',
        'due_at': '',
      });
      await _loadHomework();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('编辑失败：$e')));
      }
    }
  }

  /// 查看某课程成绩统计（只读，仅 approved 课程可由后端放行；0 行诚实空）
  Future<void> _loadStats(String courseId, String semester) async {
    setState(() {
      _statsCourseId = courseId;
      _loadingStats = true;
    });
    try {
      // GET /teacher/homework/:courseId/grade-stats?semester=
      final url = '${ApiConfig.teacherHomeworkGradeStats}$courseId/grade-stats?semester=${Uri.encodeQueryComponent(semester)}';
      final res = await _api.get(url);
      final s = res.data is Map ? (res.data['stats'] ?? {}) : {};
      setState(() => _stats = s is Map ? s : {});
    } catch (_) {
      setState(() => _stats = {});
    } finally {
      setState(() => _loadingStats = false);
    }
  }

  String _statusLabel(String s) {
    switch (s) {
      case 'published':
        return '已发布';
      case 'archived':
        return '已下架';
      default:
        return '进行中';
    }
  }

  Color _statusColor(String s) {
    switch (s) {
      case 'published':
        return Colors.green;
      case 'archived':
        return Colors.grey;
      default:
        return Colors.orange;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('我的作业与成绩统计')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (!_canWrite)
            Card(
              color: theme.colorScheme.errorContainer,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  '您没有教师作业权限。若您是教师，请联系管理员为该角色授予 teacher.grade.write 能力。',
                  style: TextStyle(color: theme.colorScheme.onErrorContainer),
                ),
              ),
            )
          else ...[
            _buildBanner(theme),
            const SizedBox(height: 16),
            _buildPublishCard(theme),
            const SizedBox(height: 16),
            _buildStatsCard(theme),
            const SizedBox(height: 16),
            _buildHomeworkCard(theme),
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
        child: Text(
          '作业仅作信息发布，不做学生提交/教师批改/内容流转（蔚小芯是教育平台非教学平台）。\n'
          '只能对「已确认授课（approved）」的课程发布作业；成绩统计基于真实 student_grades，不造数据。',
          style: theme.textTheme.bodySmall,
        ),
      ),
    );
  }

  Widget _buildPublishCard(ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('发布作业',
              style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 6),
          Text(
            '仅能对已确认授课的课程发布；若无私认课程请先申报并经教辅确认。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          if (_loadingCourses)
            const LinearProgressIndicator()
          else if (_courses.isEmpty)
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.errorContainer.withOpacity(0.4),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                '暂无已确认授课的课程，请先申报并由教辅确认（approved）后才能发布作业。',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onErrorContainer),
              ),
            )
          else ...[
            DropdownButtonFormField<String>(
              value: _selCourseId,
              isExpanded: true,
              decoration: const InputDecoration(
                  labelText: '课程（仅已确认授课）', border: OutlineInputBorder(), isDense: true),
              items: _courses.map((c) {
                final id = c['course_id']?.toString() ?? '';
                final name = c['course_name']?.toString() ?? '';
                final sem = c['semester']?.toString() ?? '';
                return DropdownMenuItem(
                  value: id,
                  child: Text('$id · ${name.isEmpty ? '（未填课程名）' : name} · $sem'),
                );
              }).toList(),
              onChanged: (v) {
                setState(() {
                  _selCourseId = v;
                  _selCourseName = _courses
                      .where((c) => c['course_id']?.toString() == v)
                      .map((c) => c['course_name']?.toString() ?? '')
                      .firstWhere((s) => true, orElse: () => '');
                });
              },
            ),
            const SizedBox(height: 10),
            DropdownButtonFormField<String>(
              value: _semesters.contains(_semester) ? _semester : null,
              isExpanded: true,
              decoration: const InputDecoration(
                  labelText: '学期', border: OutlineInputBorder(), isDense: true),
              items: _semesters
                  .map((s) => DropdownMenuItem(value: s, child: Text(s)))
                  .toList(),
              onChanged: (v) => setState(() => _semester = v ?? ''),
            ),
            const SizedBox(height: 10),
            TextField(
              controller: _titleCtrl,
              decoration: const InputDecoration(
                labelText: '作业标题 *',
                hintText: '如：第 3 章课后习题',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 10),
            TextField(
              controller: _descCtrl,
              maxLines: 3,
              decoration: const InputDecoration(
                labelText: '说明/要求（信息发布）',
                hintText: '纯文本说明，不做提交/批改',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 10),
            Row(children: [
              Expanded(
                child: TextField(
                  controller: _publishCtrl,
                  decoration: const InputDecoration(
                    labelText: '发布日期（可选）',
                    hintText: '如 2026-09-01',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: TextField(
                  controller: _dueCtrl,
                  decoration: const InputDecoration(
                    labelText: '截止日期（可选）',
                    hintText: '如 2026-09-10',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ),
            ]),
            const SizedBox(height: 12),
            FilledButton.icon(
              onPressed: _publishing ? null : _publish,
              icon: _publishing
                  ? const SizedBox(
                      width: 16, height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2))
                  : const Icon(Icons.add_circle_outline),
              label: const Text('发布作业'),
            ),
            if (_publishMsg.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                _publishMsg,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: _publishMsg.contains('成功') ? Colors.green : Colors.red,
                ),
              ),
            ],
          ],
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
            Text('该课程成绩统计（只读）',
                style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          ]),
          const SizedBox(height: 6),
          Text(
            '基于真实 student_grades（grade_type=final）聚合；仅 approved 授课课程可查。',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          if (_courses.isEmpty)
            Text(
              '暂无已确认授课课程，无法查看成绩统计。',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            )
          else ...[
            // 快速选择课程查看统计
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: _courses.map((c) {
                final id = c['course_id']?.toString() ?? '';
                final name = c['course_name']?.toString() ?? '';
                return ActionChip(
                  label: Text('$id${name.isEmpty ? '' : ' $name'}'),
                  avatar: _statsCourseId == id
                      ? const Icon(Icons.check, size: 18)
                      : null,
                  onPressed: () => _loadStats(id, c['semester']?.toString() ?? _semester),
                );
              }).toList(),
            ),
            const SizedBox(height: 12),
            if (_loadingStats)
              const Center(
                  child: Padding(
                padding: EdgeInsets.all(8),
                child: CircularProgressIndicator(strokeWidth: 2),
              ))
            else if (_statsCourseId != null)
              _renderStats(theme),
          ],
        ]),
      ),
    );
  }

  Widget _renderStats(ThemeData theme) {
    final total = (_stats['total'] ?? 0) as num;
    final notAvail = _stats['not_available'] == true;
    if (notAvail || total == 0) {
      // 诚实空态：0 成绩不补造分布/均分
      return Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest.withOpacity(0.5),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          '暂无成绩记录（total=0）。需教师先在「成绩录入」中录入真实学生成绩后，此处才会如实展示人数/均分/及格率/分档。',
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
        ),
      );
    }
    final avg = (_stats['avg_score'] ?? 0) as num;
    final passRate = ((_stats['pass_rate'] ?? 0) as num) * 100;
    final passedCount = (_stats['passed_count'] ?? 0) as num;
    final levels = _stats['levels'] is Map ? _stats['levels'] as Map : <dynamic, dynamic>{};
    int lv(String k) => (levels[k] ?? 0) as int;
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const SizedBox(height: 4),
      Text('人数：$total 人', style: theme.textTheme.bodyMedium),
      const SizedBox(height: 4),
      Text('均分：${avg.toStringAsFixed(1)}',
          style: theme.textTheme.bodyMedium),
      const SizedBox(height: 4),
      Text('及格：$passedCount 人（${passRate.toStringAsFixed(1)}%）',
          style: theme.textTheme.bodyMedium),
      const SizedBox(height: 8),
      Text('分档分布：',
          style: theme.textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600)),
      const SizedBox(height: 4),
      _distBar('优秀', lv('优秀'), Colors.green),
      _distBar('良好', lv('良好'), Colors.lightGreen),
      _distBar('及格', lv('及格'), Colors.orange),
      _distBar('不及格', lv('不及格'), Colors.red),
    ]);
  }

  Widget _distBar(String label, int count, Color color) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(children: [
        SizedBox(width: 56, child: Text(label, style: const TextStyle(fontSize: 12))),
        Expanded(
          child: LinearProgressIndicator(
            value: count > 0 ? 0.5 : 0,
            minHeight: 10,
            backgroundColor: color.withOpacity(0.15),
            valueColor: AlwaysStoppedAnimation(color),
          ),
        ),
        const SizedBox(width: 8),
        Text('$count 人', style: const TextStyle(fontSize: 12)),
      ]),
    );
  }

  Widget _buildHomeworkCard(ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Icon(Icons.home_work_outlined, color: theme.colorScheme.primary),
            const SizedBox(width: 8),
            Text('我的作业',
                style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            const Spacer(),
            OutlinedButton.icon(
              onPressed: _loadingHomework ? null : _loadHomework,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('刷新'),
            ),
          ]),
          const SizedBox(height: 8),
          if (_loadingHomework)
            const Center(
                child: Padding(
              padding: EdgeInsets.all(8),
              child: CircularProgressIndicator(strokeWidth: 2),
            ))
          else if (_homework.isEmpty)
            Text(
              '暂未发布作业。请在上方选择已确认授课的课程发布第一条作业。',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            )
          else
            ..._homework.map((h) => _hwTile(theme, h)),
        ]),
      ),
    );
  }

  Widget _hwTile(ThemeData theme, dynamic raw) {
    if (raw is! Map) return const SizedBox.shrink();
    final id = (raw['id'] ?? 0) as num;
    final courseId = raw['course_id']?.toString() ?? '';
    final courseName = raw['course_name']?.toString() ?? '';
    final semester = raw['semester']?.toString() ?? '';
    final title = raw['title']?.toString() ?? '';
    final status = raw['status']?.toString() ?? 'active';
    final publishAt = raw['publish_at']?.toString() ?? '';
    final dueAt = raw['due_at']?.toString() ?? '';
    return Card(
      key: ValueKey('hw-$id'),
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: ListTile(
        dense: true,
        leading: Icon(Icons.assignment,
            color: status == 'archived' ? Colors.grey : theme.colorScheme.primary),
        title: Text(
          title.isEmpty ? '（无标题）' : title,
          style: TextStyle(
            fontWeight: FontWeight.w600,
            color: status == 'archived' ? Colors.grey : null,
          ),
        ),
        subtitle: Text([
          '$courseId${courseName.isEmpty ? '' : '·$courseName'}',
          semester,
          if (publishAt.isNotEmpty) '发布 $publishAt',
          if (dueAt.isNotEmpty) '截止 $dueAt',
        ].join(' · ')),
        trailing: Row(mainAxisSize: MainAxisSize.min, children: [
          _statusChip(theme, status),
          if (status != 'archived') ...[
            IconButton(
              tooltip: '编辑',
              icon: const Icon(Icons.edit, size: 20),
              onPressed: () => _edit(id.toInt(), title, raw['description']?.toString() ?? ''),
            ),
            IconButton(
              tooltip: '下架',
              icon: const Icon(Icons.delete_outline, size: 20),
              onPressed: () => _archive(id.toInt()),
            ),
          ],
        ]),
      ),
    );
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
}
