import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

/// 教辅 - 考试编排
class ExamArrangePage extends StatefulWidget {
  const ExamArrangePage({super.key});
  @override
  State<ExamArrangePage> createState() => _ExamArrangePageState();
}

class _ExamArrangePageState extends State<ExamArrangePage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  Future<void> _fetch() async {
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.get(ApiConfig.assistantExamArrange);
      if (res.statusCode == 200 && res.data != null) {
        setState(() => _result = res.data is Map<String, dynamic> ? res.data : {});
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
      _fetch().catchError((Object e) {
        if (mounted) setState(() => _error = friendlyApiError(e));
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('考试编排')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(child: Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final schedule = (_result!['schedule'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final conflicts = (_result!['conflicts'] as List?)?.cast<String>() ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        DataSrcBadge(src: _result?['data_source']),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.event_note, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('考试安排', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text('${_result!['total_exams'] ?? 0} 场考试 · ${_result!['total_rooms'] ?? 0} 个考场 · ${_result!['total_invigilators'] ?? 0} 名监考',
                  style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        Text('编排明细', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        if (schedule.isEmpty)
          const Card(child: Padding(padding: EdgeInsets.all(16), child: Text('暂无考试安排')))
        else
          ...schedule.map((e) => Card(
            margin: const EdgeInsets.only(bottom: 8),
            child: ListTile(
              leading: Icon(Icons.assignment, color: theme.colorScheme.secondary),
              title: Text(e['course'] ?? ''),
              subtitle: Text(
                '${e['date'] ?? ''} ${e['time'] ?? ''} | ${e['room'] ?? ''}',
                style: theme.textTheme.bodySmall,
              ),
              trailing: Text('${e['students'] ?? 0}人', style: theme.textTheme.bodySmall),
            ),
          )),
        if (conflicts.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('冲突提示', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...conflicts.map((c) => Card(
            color: theme.colorScheme.errorContainer.withOpacity(0.4),
            child: ListTile(
              leading: const Icon(Icons.warning_amber, color: Colors.red, size: 20),
              title: Text(c, style: const TextStyle(fontSize: 14)),
            ),
          )),
        ],
      ],
    );
  }
}
