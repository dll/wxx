import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

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
      setState(() => _error = e.toString());
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _fetch());
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
    final exams = _result!['exams'] as List? ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
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
              Text(_result!['summary'] ?? '暂无安排', style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        ...exams.map((e) => Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: Icon(Icons.assignment, color: theme.colorScheme.secondary),
            title: Text(e['course'] ?? ''),
            subtitle: Text('${e['time'] ?? ''} | ${e['room'] ?? ''}'),
            trailing: Text(e['status'] ?? ''),
          ),
        )),
      ],
    );
  }
}
