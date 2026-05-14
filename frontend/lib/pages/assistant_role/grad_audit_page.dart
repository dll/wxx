import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 教辅 - 毕业资格审核
class GradAuditPage extends StatefulWidget {
  const GradAuditPage({super.key});
  @override
  State<GradAuditPage> createState() => _GradAuditPageState();
}

class _GradAuditPageState extends State<GradAuditPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  Future<void> _fetch() async {
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.get(ApiConfig.assistantGradAudit);
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
      appBar: AppBar(title: const Text('毕业资格审核')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(child: Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final students = _result!['students'] as List? ?? [];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.school, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('审核概览', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text(_result!['summary'] ?? '暂无概览', style: theme.textTheme.bodyMedium),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        ...students.map((s) => Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: Icon(s['passed'] == true ? Icons.check_circle : Icons.cancel, color: s['passed'] == true ? Colors.green : Colors.red),
            title: Text(s['name'] ?? ''),
            subtitle: Text(s['reason'] ?? ''),
          ),
        )),
      ],
    );
  }
}
