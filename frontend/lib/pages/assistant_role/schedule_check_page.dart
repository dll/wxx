import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 教辅 - 排课冲突检测
class ScheduleCheckPage extends StatefulWidget {
  const ScheduleCheckPage({super.key});
  @override
  State<ScheduleCheckPage> createState() => _ScheduleCheckPageState();
}

class _ScheduleCheckPageState extends State<ScheduleCheckPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  Future<void> _check() async {
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.get(ApiConfig.assistantScheduleCheck);
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
    WidgetsBinding.instance.addPostFrameCallback((_) => _check());
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('排课冲突检测')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(child: Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final conflicts = _result!['conflicts'] as List? ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(children: [
              Icon(conflicts.isEmpty ? Icons.check_circle : Icons.warning, color: conflicts.isEmpty ? Colors.green : Colors.orange, size: 32),
              const SizedBox(width: 12),
              Text(conflicts.isEmpty ? '未检测到排课冲突' : '发现 ${conflicts.length} 个冲突', style: theme.textTheme.titleMedium),
            ]),
          ),
        ),
        ...conflicts.map((c) => Card(
          margin: const EdgeInsets.only(top: 12),
          child: ListTile(
            leading: const Icon(Icons.error_outline, color: Colors.orange),
            title: Text(c['description'] ?? ''),
            subtitle: Text(c['detail'] ?? ''),
          ),
        )),
      ],
    );
  }
}
