import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';

/// 学院管理员 - 数据分析报告
class DataAnalysisPage extends StatefulWidget {
  const DataAnalysisPage({super.key});
  @override
  State<DataAnalysisPage> createState() => _DataAnalysisPageState();
}

class _DataAnalysisPageState extends State<DataAnalysisPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  Future<void> _fetch() async {
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.get(ApiConfig.collegeDataAnalysis);
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
      appBar: AppBar(title: const Text('数据分析报告')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(child: Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final query = (_result!['query'] ?? '').toString();
    final content = (_result!['content'] ?? '暂无分析').toString();
    final dataSource = (_result!['data_source'] ?? '').toString();
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        if (query.isNotEmpty) ...[
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(children: [
                Icon(Icons.search, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(child: Text('分析主题：$query', style: theme.textTheme.bodyMedium)),
              ]),
            ),
          ),
          const SizedBox(height: 12),
        ],
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.analytics, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text('分析结论', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text(content, style: theme.textTheme.bodyMedium),
              if (dataSource.isNotEmpty) ...[
                const SizedBox(height: 12),
                Text('数据来源：${dataSource == 'fallback' ? '暂无足够数据，展示占位结论' : dataSource}',
                    style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.outline)),
              ],
            ]),
          ),
        ),
      ],
    );
  }
}
