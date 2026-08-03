import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 学生会 - AI 活动策划
class EventPlanPage extends StatefulWidget {
  const EventPlanPage({super.key});
  @override
  State<EventPlanPage> createState() => _EventPlanPageState();
}

class _EventPlanPageState extends State<EventPlanPage> {
  final ApiService _api = ApiService();
  final _themeCtrl = TextEditingController();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  @override
  void dispose() {
    _themeCtrl.dispose();
    super.dispose();
  }

  Future<void> _generate() async {
    if (_themeCtrl.text.trim().isEmpty) return;
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.post(ApiConfig.unionEventPlan, data: {'theme': _themeCtrl.text.trim()});
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
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 活动策划')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('活动主题', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(controller: _themeCtrl, decoration: const InputDecoration(labelText: '输入活动主题', border: OutlineInputBorder(), hintText: '例如：迎新晚会、运动会、志愿服务')),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: _loading ? null : _generate,
                    icon: const Icon(Icons.auto_awesome),
                    label: const Text('生成策划方案'),
                  ),
                ),
              ]),
            ),
          ),
          if (_loading) const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
          if (_error.isNotEmpty) Padding(padding: const EdgeInsets.all(16), child: Text(_error, style: TextStyle(color: theme.colorScheme.error))),
          if (_result != null && !_loading) ...[
            const SizedBox(height: 16),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Icon(Icons.event, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text(_result!['title'] ?? '策划方案', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  ]),
                  if ((_result!['goal'] ?? '').toString().isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _section(theme, Icons.flag, '目标', _result!['goal']),
                  ],
                  if ((_result!['budget'] ?? '').toString().isNotEmpty) ...[
                    const SizedBox(height: 8),
                    _section(theme, Icons.payments, '预算', _result!['budget']),
                  ],
                  if (_result!['timeline'] != null) ...[
                    const SizedBox(height: 8),
                    Text('时间线', style: theme.textTheme.titleSmall),
                    const SizedBox(height: 4),
                    ...((_result!['timeline'] as List?) ?? []).map((t) {
                      final m = t as Map<String, dynamic>;
                      return ListTile(
                        dense: true,
                        contentPadding: EdgeInsets.zero,
                        leading: Icon(Icons.schedule, size: 18, color: theme.colorScheme.primary),
                        title: Text(m['phase'] ?? '', style: const TextStyle(fontSize: 14)),
                        subtitle: Text(m['tasks'] ?? '', style: const TextStyle(fontSize: 12)),
                      );
                    }),
                  ],
                  if ((_result!['promotion'] ?? '').toString().isNotEmpty) ...[
                    const SizedBox(height: 8),
                    _section(theme, Icons.campaign, '宣传方案', _result!['promotion']),
                  ],
                  if (_result!['risk_assessment'] != null) ...[
                    const SizedBox(height: 8),
                    Text('风险评估', style: theme.textTheme.titleSmall),
                    const SizedBox(height: 4),
                    ...((_result!['risk_assessment'] as List?) ?? []).map((r) => Text('•  $r',
                        style: theme.textTheme.bodySmall)),
                  ],
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _section(ThemeData theme, IconData icon, String title, dynamic value) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Icon(icon, size: 16, color: theme.colorScheme.primary),
        const SizedBox(width: 6),
        Text(title, style: theme.textTheme.titleSmall),
      ]),
      const SizedBox(height: 4),
      Text(value?.toString() ?? '', style: theme.textTheme.bodyMedium),
    ]);
  }
}
