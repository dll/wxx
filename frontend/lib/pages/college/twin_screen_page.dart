import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 学院管理员 - 数字孪生大屏
class TwinScreenPage extends StatefulWidget {
  const TwinScreenPage({super.key});
  @override
  State<TwinScreenPage> createState() => _TwinScreenPageState();
}

class _TwinScreenPageState extends State<TwinScreenPage> {
  final ApiService _api = ApiService();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  Future<void> _fetch() async {
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.get(ApiConfig.collegeTwinScreen);
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
      appBar: AppBar(title: const Text('数字孪生大屏')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(child: Text(_error, style: TextStyle(color: theme.colorScheme.error)))
              : _buildContent(theme),
    );
  }

  Widget _buildContent(ThemeData theme) {
    if (_result == null) return const Center(child: Text('暂无数据'));
    final overview = (_result!['overview'] as Map?)?.cast<String, dynamic>() ?? {};
    final aiInsight = (_result!['ai_insight'] ?? '').toString();
    final metricCards = <Map<String, String>>[
      {'value': '${overview['total_students'] ?? 0}', 'label': '学生总数'},
      {'value': '${overview['health_score'] ?? 0}', 'label': '健康度'},
      {'value': '${overview['risk_students'] ?? 0}', 'label': '风险关注'},
      {'value': '${((overview['active_rate'] ?? 0) as num? ?? 0 * 100).toStringAsFixed(0)}%', 'label': '健康率'},
    ];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              gradient: LinearGradient(colors: [theme.colorScheme.primary.withOpacity(0.15), theme.colorScheme.tertiary.withOpacity(0.05)]),
            ),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(Icons.dashboard, color: theme.colorScheme.primary, size: 28),
                const SizedBox(width: 8),
                Text(_result!['college'] ?? '学院', style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 4),
              Text('更新时间：${_result!['updated_at'] ?? ''}', style: theme.textTheme.bodySmall),
            ]),
          ),
        ),
        const SizedBox(height: 16),
        GridView.count(
          crossAxisCount: 2, shrinkWrap: true, physics: const NeverScrollableScrollPhysics(),
          mainAxisSpacing: 12, crossAxisSpacing: 12, childAspectRatio: 1.6,
          children: metricCards.map<Widget>((m) => Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                Text(m['value']!, style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold, color: theme.colorScheme.primary)),
                const SizedBox(height: 4),
                Text(m['label']!, style: theme.textTheme.bodySmall),
              ]),
            ),
          )).toList(),
        ),
        if (aiInsight.isNotEmpty) ...[
          const SizedBox(height: 16),
          Card(
            color: theme.colorScheme.primaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Icon(Icons.psychology, color: theme.colorScheme.onPrimaryContainer, size: 18),
                  const SizedBox(width: 6),
                  Text('AI 解读', style: theme.textTheme.titleSmall),
                ]),
                const SizedBox(height: 8),
                Text(aiInsight, style: theme.textTheme.bodyMedium),
              ]),
            ),
          ),
        ],
      ],
    );
  }
}
