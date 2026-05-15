import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/counselor_feature_provider.dart';

/// 辅导员 — 学生列表页面
class StudentListPage extends StatefulWidget {
  const StudentListPage({super.key});

  @override
  State<StudentListPage> createState() => _StudentListPageState();
}

class _StudentListPageState extends State<StudentListPage> {
  String _search = '';
  List<Map<String, dynamic>> _filteredCache = [];
  List<Map<String, dynamic>> _lastSource = [];
  String _lastSearch = '';

  @override
  void initState() {
    super.initState();
    context.read<CounselorFeatureProvider>().fetchStudentList();
  }

  List<Map<String, dynamic>> _getFiltered(List<Map<String, dynamic>> students) {
    if (identical(students, _lastSource) && _search == _lastSearch) {
      return _filteredCache;
    }
    _lastSource = students;
    _lastSearch = _search;
    if (_search.isEmpty) {
      _filteredCache = students;
    } else {
      final q = _search.toLowerCase();
      _filteredCache = students.where((s) {
        final name = (s['name'] ?? '').toString().toLowerCase();
        final id = (s['student_id'] ?? '').toString().toLowerCase();
        return name.contains(q) || id.contains(q);
      }).toList();
    }
    return _filteredCache;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final prov = context.watch<CounselorFeatureProvider>();
    final filtered = _getFiltered(prov.studentList);

    return Scaffold(
      appBar: AppBar(title: const Text('我的学生')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              decoration: InputDecoration(
                hintText: '搜索姓名或学号...',
                prefixIcon: const Icon(Icons.search),
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                filled: true,
                fillColor: theme.colorScheme.surfaceContainerHighest,
                contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              ),
              onChanged: (v) {
                if (v != _search) setState(() => _search = v);
              },
            ),
          ),
          if (!prov.loading)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
              child: Row(
                children: [
                  Icon(Icons.people, size: 18, color: theme.colorScheme.primary),
                  const SizedBox(width: 6),
                  Text('共 ${filtered.length} 名学生',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      )),
                ],
              ),
            ),
          if (prov.error.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Text(prov.error, style: TextStyle(color: theme.colorScheme.error)),
            ),
          const SizedBox(height: 4),
          Expanded(
            child: prov.loading
                ? const Center(child: CircularProgressIndicator())
                : filtered.isEmpty
                    ? Center(
                        child: Text('暂无学生数据',
                            style: TextStyle(color: theme.colorScheme.outline)))
                    : RefreshIndicator(
                        onRefresh: prov.fetchStudentList,
                        child: ListView.separated(
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                          itemCount: filtered.length,
                          separatorBuilder: (_, __) => const Divider(height: 1),
                          itemBuilder: (context, index) =>
                              _buildStudentTile(filtered[index], theme),
                        ),
                      ),
          ),
        ],
      ),
    );
  }

  Widget _buildStudentTile(Map<String, dynamic> student, ThemeData theme) {
    final name = student['name'] ?? '未知';
    final studentId = student['student_id'] ?? '';
    final className = student['class_name'] ?? '';
    final status = student['status'] ?? 'normal';
    final avatar = (name as String).isNotEmpty ? name[0] : '?';

    Color statusColor;
    String statusLabel;
    switch (status) {
      case 'warning':
        statusColor = Colors.orange;
        statusLabel = '关注';
        break;
      case 'alert':
        statusColor = Colors.red;
        statusLabel = '预警';
        break;
      default:
        statusColor = Colors.green;
        statusLabel = '正常';
    }

    return ListTile(
      leading: CircleAvatar(
        backgroundColor: theme.colorScheme.primaryContainer,
        child: Text(avatar, style: TextStyle(
          color: theme.colorScheme.onPrimaryContainer,
          fontWeight: FontWeight.bold,
        )),
      ),
      title: Text(name, style: const TextStyle(fontWeight: FontWeight.w500)),
      subtitle: Text('$studentId · $className'),
      trailing: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
        decoration: BoxDecoration(
          color: statusColor.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Text(statusLabel, style: TextStyle(
          fontSize: 12,
          color: statusColor,
          fontWeight: FontWeight.w500,
        )),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
    );
  }
}
