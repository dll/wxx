import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

/// 教辅 - 学生信息查询（真实学生账号数据）
class StudentInfoPage extends StatefulWidget {
  const StudentInfoPage({super.key});
  @override
  State<StudentInfoPage> createState() => _StudentInfoPageState();
}

class _StudentInfoPageState extends State<StudentInfoPage> {
  final ApiService _api = ApiService();
  final TextEditingController _queryCtrl = TextEditingController();
  bool _loading = false;
  List<Map<String, dynamic>> _results = [];
  String _src = '';
  String _error = '';

  Future<void> _search([String? q]) async {
    final query = (q ?? _queryCtrl.text).trim();
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final uri = query.isEmpty
          ? Uri.parse(ApiConfig.assistantStudentInfo)
          : Uri.parse('${ApiConfig.assistantStudentInfo}?q=${Uri.encodeQueryComponent(query)}');
      final res = await _api.get(uri.toString());
      if (res.statusCode == 200 && res.data != null) {
        final data = res.data is Map<String, dynamic> ? res.data : {};
        setState(() {
          _results = (data['result'] as List?)?.cast<Map<String, dynamic>>() ?? [];
          _src = data['data_source']?.toString() ?? '';
        });
      }
    } catch (e) {
      setState(() => _error = friendlyApiError(e));
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _search('').catchError((Object e) {
        if (mounted) setState(() => _error = friendlyApiError(e));
      });
    });
  }

  @override
  void dispose() {
    _queryCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('学生信息查询')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(children: [
              Expanded(
                child: TextField(
                  controller: _queryCtrl,
                  decoration: const InputDecoration(
                    hintText: '按姓名 / 学号 / 班级 / 专业搜索',
                    prefixIcon: Icon(Icons.search),
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  onSubmitted: _search,
                ),
              ),
              const SizedBox(width: 8),
              IconButton.filled(
                onPressed: _loading ? null : () => _search(),
                icon: const Icon(Icons.search),
              ),
            ]),
          ),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : _error.isNotEmpty
                    ? Center(
                        child: Text(_error,
                            style: TextStyle(color: theme.colorScheme.error)))
                    : _buildResults(theme),
          ),
        ],
      ),
    );
  }

  Widget _buildResults(ThemeData theme) {
    if (_results.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Icon(Icons.person_search, size: 48, color: theme.colorScheme.outline),
            const SizedBox(height: 12),
            const Text('未查询到匹配学生，或暂无学生账号数据'),
            const SizedBox(height: 4),
            Text('数据来自真实学生账号，非示例数据（不瞎编）',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.outline)),
          ]),
        ),
      );
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        DataSrcBadge(src: _src),
        const SizedBox(height: 8),
        ..._results.map((s) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: CircleAvatar(
                  child: Text((s['name']?.toString() ?? '?').characters.first),
                ),
                title: Text(s['name']?.toString() ?? ''),
                subtitle: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('学号：${s['student_id']}'),
                    Text('${s['major']} · ${s['class']}'),
                    if ((s['college'] ?? '').toString().isNotEmpty)
                      Text('${s['college']}'),
                  ],
                ),
                trailing: Chip(
                  label: Text(
                    s['status']?.toString() == 'active' ? '在读' : '${s['status']}',
                    style: const TextStyle(fontSize: 11),
                  ),
                  visualDensity: VisualDensity.compact,
                ),
              ),
            )),
      ],
    );
  }
}
