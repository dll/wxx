import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

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
    final name = _result!['student_name'] ?? '示例学生';
    final total = (_result!['total_credits'] ?? 0).toDouble();
    final required = (_result!['required_credits'] ?? 0).toDouble();
    final passed = (_result!['can_graduate'] ?? false) == true;
    final passedItems = (_result!['passed_items'] as List?)?.cast<String>() ?? [];
    final pendingItems = (_result!['pending_items'] as List?)?.cast<String>() ?? [];
    final summary = _result!['summary'] ?? '';
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        DataSrcBadge(src: _result?['data_source']),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Icon(passed ? Icons.verified : Icons.error_outline,
                    color: passed ? Colors.green : Colors.red),
                const SizedBox(width: 8),
                Text('审核结果', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
              ]),
              const SizedBox(height: 12),
              Text('学生：$name', style: theme.textTheme.bodyMedium),
              const SizedBox(height: 8),
              // 学分进度条（办理进度透明）
              Row(
                children: [
                  Text('学分进度', style: theme.textTheme.bodySmall),
                  const Spacer(),
                  Text('${total.toStringAsFixed(0)}/${required.toStringAsFixed(0)}',
                      style: theme.textTheme.bodySmall?.copyWith(fontWeight: FontWeight.w700)),
                ],
              ),
              const SizedBox(height: 4),
              ClipRRect(
                borderRadius: BorderRadius.circular(5),
                child: LinearProgressIndicator(
                  value: required > 0 ? (total / required).clamp(0.0, 1.0) : 0,
                  minHeight: 8,
                  backgroundColor: theme.colorScheme.surfaceContainerHighest,
                  color: passed ? Colors.green : theme.colorScheme.primary,
                ),
              ),
              const SizedBox(height: 10),
              Text('已获学分：${total.toStringAsFixed(0)} / 要求 ${required.toStringAsFixed(0)}',
                  style: theme.textTheme.bodyMedium),
              const SizedBox(height: 4),
              Text('结论：${passed ? '✅ 符合毕业条件' : '❌ 未达到毕业条件'}',
                  style: theme.textTheme.bodyMedium?.copyWith(
                      color: passed ? Colors.green : Colors.red,
                      fontWeight: FontWeight.w600)),
              if ((summary ?? '').toString().isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(summary.toString(), style: theme.textTheme.bodySmall),
              ],
            ]),
          ),
        ),
        if (passedItems.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('已达标项', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...passedItems.map((p) => Card(
            margin: const EdgeInsets.only(bottom: 6),
            child: ListTile(
              dense: true,
              leading: const Icon(Icons.check_circle, color: Colors.green, size: 20),
              title: Text(p, style: const TextStyle(fontSize: 14)),
            ),
          )),
        ],
        if (pendingItems.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text('待补项', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ...pendingItems.map((p) => Card(
            margin: const EdgeInsets.only(bottom: 6),
            child: ListTile(
              dense: true,
              leading: const Icon(Icons.cancel, color: Colors.red, size: 20),
              title: Text(p, style: const TextStyle(fontSize: 14)),
            ),
          )),
        ],
      ],
    );
  }
}
