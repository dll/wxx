import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/capability_utils.dart';

/// 教辅/教务 - 教师授课申报审核（R3 越权边界升级 补 H）
///
/// 设计要点（对齐 pm-check-teacher-course-review.md §6.2 / §7 诚实红线）：
/// - 教师申报授课关系后挂 pending，教辅/教务在此 approve/reject，approved 才是成绩强校验的唯一放行判据；
/// - **不授 teacher 自审**：门控 teacher.course.review（无权限入口隐藏 + 页面自身拦截双保险）；
/// - 审核即真实操作留痕：通过/驳回落 reviewed_by/reviewed_name/reviewed_at/review_note；
/// - 诚实空态：无可审申报 → 「暂无待审授课申报」，绝不显示伪造的「已确认」。
class TeacherCourseReviewPage extends StatefulWidget {
  const TeacherCourseReviewPage({super.key});
  @override
  State<TeacherCourseReviewPage> createState() => _TeacherCourseReviewPageState();
}

class _TeacherCourseReviewPageState extends State<TeacherCourseReviewPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  bool _reviewing = false;
  String _error = '';
  int _pendingCount = 0;
  List<dynamic> _pending = [];

  bool get _canReview => CapabilityUtils.has(Capability.teacherCourseReview);

  @override
  void initState() {
    super.initState();
    if (_canReview) {
      _load();
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final pendingRes = await _api.get(ApiConfig.teacherCoursesPending);
      final countRes = await _api.get(ApiConfig.teacherCoursesPendingCount);
      final list = pendingRes.data is Map ? (pendingRes.data['list'] ?? []) : [];
      final count = countRes.data is Map ? (countRes.data['pending'] ?? 0) : 0;
      if (!mounted) return;
      setState(() {
        _pending = list is List ? list : [];
        _pendingCount = (count is num) ? count.toInt() : _pending.length;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = '加载待审申报失败：$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  /// 审核：approve/reject + 可选备注。仅教辅真实操作；成功后刷新列表与角标。
  Future<void> _review(int id, String status) async {
    if (!_canReview) {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('无审核权限（teacher.course.review）')));
      return;
    }
    final note = await _askNote(status);
    if (note == null || !mounted) return; // 取消
    setState(() => _reviewing = true);
    try {
      final res = await _api.put(
        '${ApiConfig.teacherCoursesReview}/$id',
        data: {'status': status, 'note': note},
      );
      final msg = res.data is Map
          ? (res.data['message']?.toString() ?? '已审核')
          : res.data.toString();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(msg)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('审核失败：$e')));
    } finally {
      if (mounted) setState(() => _reviewing = false);
    }
  }

  /// 弹出审核备注输入框；返回 null 表示取消。
  Future<String?> _askNote(String status) async {
    final ctrl = TextEditingController();
    final isApprove = status == 'approved';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(isApprove ? '通过授课申报' : '驳回授课申报'),
        content: TextField(
          controller: ctrl,
          maxLines: 3,
          decoration: const InputDecoration(
            labelText: '审核意见（可选，驳回时建议说明原因）',
            border: OutlineInputBorder(),
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(isApprove ? '通过' : '驳回'),
          ),
        ],
      ),
    );
    final value = ctrl.text.trim();
    if (ok != true) return null;
    return value;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('授课申报审核（R3）')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (!_canReview)
            Card(
              color: theme.colorScheme.errorContainer,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  '您没有授课申报审核权限。若您是教辅/教务，请联系管理员为该角色授予 teacher.course.review 能力。',
                  style: TextStyle(color: theme.colorScheme.onErrorContainer),
                ),
              ),
            )
          else ...[
            _buildPendingBar(theme),
            const SizedBox(height: 16),
            _buildList(theme),
          ],
        ],
      ),
    );
  }

  /// 待审角标条：pending-count 角标 + 刷新
  Widget _buildPendingBar(ThemeData theme) {
    return Card(
      color: Colors.orange.shade50,
      child: ListTile(
        leading: Icon(Icons.pending_actions, color: Colors.orange.shade800),
        title: Text('待审授课申报：$_pendingCount 条',
            style: const TextStyle(fontWeight: FontWeight.bold)),
        subtitle: const Text('通过（approved）后方可录入该课程成绩；驳回（rejected）教师可重新申报'),
        trailing: TextButton(
          onPressed: _loading ? null : _load,
          child: const Text('刷新'),
        ),
      ),
    );
  }

  Widget _buildList(ThemeData theme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('待审列表',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            if (_loading)
              const Center(child: Padding(
                padding: EdgeInsets.all(12),
                child: CircularProgressIndicator(),
              ))
            else if (_error.isNotEmpty)
              Text(_error,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.error))
            else if (_pending.isEmpty)
              // 诚实空态：无可审申报时不伪造成已有待审
              const Padding(
                padding: EdgeInsets.all(12),
                child: Text('暂无待审授课申报',
                    style: TextStyle(color: Colors.grey)),
              )
            else
              ..._pending.map((rec) => _PendingTile(
                    record: rec,
                    reviewing: _reviewing,
                    onApprove: () => _review(_idOf(rec), 'approved'),
                    onReject: () => _review(_idOf(rec), 'rejected'),
                  )),
          ],
        ),
      ),
    );
  }

  int _idOf(dynamic raw) {
    if (raw is Map) {
      final v = raw['id'];
      return v == null ? 0 : (v is num ? v.toInt() : int.tryParse('$v') ?? 0);
    }
    return 0;
  }
}

/// 单条待审申报：教师/课程/学期/申报时间/备注 + 批准/驳回按钮
class _PendingTile extends StatelessWidget {
  final dynamic record;
  final bool reviewing;
  final VoidCallback onApprove;
  final VoidCallback onReject;
  const _PendingTile({
    required this.record,
    required this.reviewing,
    required this.onApprove,
    required this.onReject,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final r = record is Map ? record : <String, dynamic>{};
    final teacher = r['teacher_name']?.toString() ?? '';
    final courseId = r['course_id']?.toString() ?? '';
    final courseName = r['course_name']?.toString() ?? '';
    final semester = r['semester']?.toString() ?? '';
    final createdAt = r['created_at']?.toString() ?? '';
    final note = r['review_note']?.toString() ?? '';
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              CircleAvatar(
                backgroundColor: Colors.orange.withOpacity(0.15),
                child: Icon(Icons.menu_book, color: Colors.orange.shade800),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      teacher.isEmpty ? '教师 #${r['teacher_id']}' : teacher,
                      style: const TextStyle(fontWeight: FontWeight.bold),
                    ),
                    Text(
                      '$courseId${courseName.isEmpty ? '' : ' · $courseName'}',
                      style: theme.textTheme.bodyMedium,
                    ),
                  ],
                ),
              ),
              if (reviewing)
                const SizedBox(
                    width: 16, height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2)),
            ]),
            const SizedBox(height: 8),
            Text(
              [semester, if (createdAt.isNotEmpty) '申报 $createdAt'].join(' · '),
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            ),
            if (note.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text('备注：$note', style: theme.textTheme.bodySmall),
            ],
            const SizedBox(height: 8),
            Row(children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: reviewing ? null : onReject,
                  icon: const Icon(Icons.close, color: Colors.red),
                  label: const Text('驳回', style: TextStyle(color: Colors.red)),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: FilledButton.icon(
                  onPressed: reviewing ? null : onApprove,
                  icon: const Icon(Icons.check_circle_outline, size: 18),
                  label: const Text('通过'),
                ),
              ),
            ]),
          ],
        ),
      ),
    );
  }
}
