import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/student_feature_provider.dart';
import '../../config/api_config.dart';

class ProcessEnhancedPage extends StatefulWidget {
  const ProcessEnhancedPage({super.key});
  @override
  State<ProcessEnhancedPage> createState() => _ProcessEnhancedPageState();
}

class _ProcessEnhancedPageState extends State<ProcessEnhancedPage> {
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      await context.read<StudentFeatureProvider>().askAI(ApiConfig.processEnhanced);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('AI 办事流程')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Row(children: [
                        Icon(Icons.assignment, color: theme.colorScheme.primary),
                        const SizedBox(width: 8),
                        Text('缓考申请流程', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                        const Spacer(),
                        Chip(label: const Text('进行中'), backgroundColor: Colors.orange.withValues(alpha: 0.2)),
                      ]),
                      const SizedBox(height: 16),
                      _stepTile(theme, 1, '填写缓考申请表', '已完成', true, false),
                      _stepTile(theme, 2, '辅导员签字', '进行中', false, true),
                      _stepTile(theme, 3, '教务处审批', '待办', false, false),
                    ]),
                  ),
                ),
                const SizedBox(height: 16),
                Card(
                  color: Colors.amber.withValues(alpha: 0.1),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Row(children: [
                      const Icon(Icons.alarm, color: Colors.amber),
                      const SizedBox(width: 12),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('缓考申请截止', style: theme.textTheme.titleSmall),
                        Text('2026-05-20 (剩余5天)', style: theme.textTheme.bodySmall),
                      ])),
                    ]),
                  ),
                ),
              ],
            ),
    );
  }

  Widget _stepTile(ThemeData theme, int step, String title, String status, bool done, bool current) {
    final color = done ? Colors.green : (current ? theme.colorScheme.primary : Colors.grey);
    return Padding(
      padding: const EdgeInsets.only(left: 8, bottom: 12),
      child: Row(children: [
        Container(
          width: 28, height: 28,
          decoration: BoxDecoration(color: color.withValues(alpha: 0.2), shape: BoxShape.circle),
          child: Center(child: done ? Icon(Icons.check, size: 16, color: color) : Text('$step', style: TextStyle(color: color, fontWeight: FontWeight.bold))),
        ),
        const SizedBox(width: 12),
        Expanded(child: Text(title, style: theme.textTheme.bodyMedium)),
        Text(status, style: TextStyle(color: color, fontSize: 12)),
      ]),
    );
  }
}
