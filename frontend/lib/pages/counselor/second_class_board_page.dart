import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';
import '../../widgets/data_src_badge.dart';

/// 辅导员 — 第二课堂班级看板
///
/// 按辅导员名下学生，真实聚合其第二课堂活动参与（报名/到场）与积分。
/// 数据源：health_activities + health_activity_signups + student_points（真实表聚合）。
/// 无真实记录时诚实显示「暂无记录/待接入」，不造假数字。
class SecondClassBoardPage extends StatefulWidget {
  const SecondClassBoardPage({super.key});

  @override
  State<SecondClassBoardPage> createState() => _SecondClassBoardPageState();
}

class _SecondClassBoardPageState extends State<SecondClassBoardPage> {
  String _search = '';

  @override
  void initState() {
    super.initState();
    context.read<CounselorFeatureProvider>().fetchSecondClassBoard();
  }

  List<dynamic> _filtered(List<dynamic> students) {
    if (_search.isEmpty) return students;
    final q = _search.toLowerCase();
    return students.where((s) {
      final name = (s['name'] ?? '').toString().toLowerCase();
      final id = (s['student_id'] ?? '').toString().toLowerCase();
      return name.contains(q) || id.contains(q);
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final prov = context.watch<CounselorFeatureProvider>();
    final board = prov.secondClassBoard;
    final dataSource = (board['data_source'] ?? 'not_available').toString();
    final students = List<dynamic>.from(board['students'] ?? []);
    final filtered = _filtered(students);

    final studentTotal = (board['student_total'] ?? students.length) as num;
    final activityTotal = (board['activity_total'] ?? 0) as num;
    final attendTotal = (board['attend_total'] ?? 0) as num;
    final pointTotal = (board['point_total'] ?? 0) as num;
    final note = (board['note'] ?? '').toString();

    return Scaffold(
      appBar: AppBar(
        title: const Text('第二课堂班级看板'),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: Center(child: DataSrcBadge(src: dataSource)),
          ),
        ],
      ),
      body: Column(
        children: [
          _summaryRow(theme, studentTotal, activityTotal, attendTotal,
              pointTotal),
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              decoration: InputDecoration(
                hintText: '搜索姓名或学号...',
                prefixIcon: const Icon(Icons.search),
                border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12)),
                filled: true,
                fillColor: theme.colorScheme.surfaceContainerHighest,
                contentPadding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              ),
              onChanged: (v) {
                if (v != _search) setState(() => _search = v);
              },
            ),
          ),
          if (dataSource != 'real')
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(
                children: [
                  Icon(Icons.info_outline,
                      size: 18, color: theme.colorScheme.outline),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      note.isEmpty ? '暂无真实第二课堂数据' : note,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          const SizedBox(height: 4),
          Expanded(
            child: prov.secondClassLoading
                ? const Center(child: CircularProgressIndicator())
                : filtered.isEmpty
                    ? Center(
                        child: Text(
                          dataSource == 'real'
                              ? '名下学生暂未参与第二课堂活动'
                              : '暂无记录（数据待接入）',
                          style: TextStyle(color: theme.colorScheme.outline),
                        ),
                      )
                    : RefreshIndicator(
                        onRefresh: prov.fetchSecondClassBoard,
                        child: ListView.separated(
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                          itemCount: filtered.length,
                          separatorBuilder: (_, __) => const Divider(height: 1),
                          itemBuilder: (context, index) =>
                              _buildStudentTile(theme, filtered[index]),
                        ),
                      ),
          ),
        ],
      ),
    );
  }

  Widget _summaryRow(ThemeData theme, num studentTotal, num activityTotal,
      num attendTotal, num pointTotal) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: Row(
        children: [
          _sumBox(theme, '${studentTotal}', '学生'),
          const SizedBox(width: 8),
          _sumBox(theme, '${activityTotal}', '活动'),
          const SizedBox(width: 8),
          _sumBox(theme, '${attendTotal}', '到场'),
          const SizedBox(width: 8),
          _sumBox(theme, '${pointTotal}', '积分'),
        ],
      ),
    );
  }

  Widget _sumBox(ThemeData theme, String value, String label) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          children: [
            Text(value,
                style: theme.textTheme.titleLarge?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.bold,
                )),
            const SizedBox(height: 2),
            Text(label,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                )),
          ],
        ),
      ),
    );
  }

  Widget _buildStudentTile(ThemeData theme, dynamic student) {
    final name = student['name'] ?? '未知';
    // 名称脱敏：保留首字，余用 * 遮盖
    final masked = _mask(name.toString());
    final studentId = student['student_id'] ?? '';
    final className = student['class_name'] ?? '';
    final actCount = (student['activity_count'] ?? 0) as num;
    final attCount = (student['attend_count'] ?? 0) as num;
    final pt = (student['point_total'] ?? 0) as num;

    return ListTile(
      leading: CircleAvatar(
        backgroundColor: theme.colorScheme.primaryContainer,
        child: Text(
          masked.isNotEmpty ? masked[0] : '?',
          style: TextStyle(
            color: theme.colorScheme.onPrimaryContainer,
            fontWeight: FontWeight.bold,
          ),
        ),
      ),
      title: Text(masked, style: const TextStyle(fontWeight: FontWeight.w500)),
      subtitle: Text('$studentId · $className'),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _chip(theme, '$actCount 活动'),
          const SizedBox(width: 4),
          _chip(theme, '$attCount 到场'),
          const SizedBox(width: 4),
          _chip(theme, '$pt 分', emphasize: true),
        ],
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
    );
  }

  Widget _chip(ThemeData theme, String text, {bool emphasize = false}) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
      decoration: BoxDecoration(
        color: emphasize
            ? theme.colorScheme.primaryContainer
            : theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(text, style: TextStyle(
        fontSize: 11,
        fontWeight: emphasize ? FontWeight.w600 : FontWeight.w400,
        color: emphasize
            ? theme.colorScheme.onPrimaryContainer
            : theme.colorScheme.onSurfaceVariant,
      )),
    );
  }

  String _mask(String name) {
    if (name.isEmpty) return name;
    if (name.length <= 2) return name;
    return name[0] + '*' * (name.length - 1);
  }
}
