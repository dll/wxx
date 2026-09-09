import 'package:flutter/material.dart';

import '../config/api_config.dart';
import '../services/api_service.dart';

/// A page-independent AI action sheet. It intentionally uses ApiService
/// directly so it can be opened from the global shell without provider wiring.
Future<void> showStudentAIAssistDialog(BuildContext context,
    {String initialFeature = 'general'}) async {
  await showDialog<void>(
    context: context,
    builder: (_) => _StudentAIAssistDialog(initialFeature: initialFeature),
  );
}

class _StudentAIAssistDialog extends StatefulWidget {
  final String initialFeature;
  const _StudentAIAssistDialog({required this.initialFeature});

  @override
  State<_StudentAIAssistDialog> createState() => _StudentAIAssistDialogState();
}

class _StudentAIAssistDialogState extends State<_StudentAIAssistDialog> {
  final _input = TextEditingController();
  final _api = ApiService();
  late String _feature;
  bool _loading = false;
  String? _response;
  String _source = '';

  static const _features = <String, String>{
    'general': '通用助手',
    'study': '学习与课程',
    'career': '职业与简历',
    'mental': '心理支持',
    'competition': '竞赛与项目',
    'campus': '校园生活',
    'process': '办事流程',
    'profile': '成长档案',
    'vopc': 'vOPC 项目',
  };

  @override
  void initState() {
    super.initState();
    _feature = _features.containsKey(widget.initialFeature)
        ? widget.initialFeature
        : 'general';
  }

  @override
  void dispose() {
    _input.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('AI 学生助手'),
        content: SizedBox(
          width: 520,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                DropdownButtonFormField<String>(
                  value: _feature,
                  decoration: const InputDecoration(labelText: '协助领域'),
                  items: _features.entries
                      .map((e) => DropdownMenuItem(
                          value: e.key, child: Text(e.value)))
                      .toList(),
                  onChanged: _loading
                      ? null
                      : (v) => setState(() => _feature = v ?? 'general'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _input,
                  minLines: 3,
                  maxLines: 6,
                  decoration: const InputDecoration(
                    labelText: '你希望 AI 帮你完成什么？',
                    hintText: '例如：把我的想法整理成 vOPC 项目目标和验证计划',
                    alignLabelWithHint: true,
                  ),
                ),
                if (_response != null) ...[
                  const SizedBox(height: 16),
                  const Divider(),
                  Text(_response!, style: Theme.of(context).textTheme.bodyMedium),
                  const SizedBox(height: 8),
                  Text('来源：${_source == 'ai' ? 'AI 生成' : '本地规则示例'} · 请核对后再使用',
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: Theme.of(context).colorScheme.onSurfaceVariant)),
                ],
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context), child: const Text('关闭')),
          FilledButton.icon(
            onPressed: _loading ? null : _ask,
            icon: _loading
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.auto_awesome),
            label: Text(_loading ? '生成中…' : '生成建议'),
          ),
        ],
      );

  Future<void> _ask() async {
    final input = _input.text.trim();
    if (input.isEmpty) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请先描述需求')));
      return;
    }
    setState(() => _loading = true);
    try {
      final res = await _api.post(ApiConfig.studentAIAssist, data: {
        'feature': _feature,
        'input': input,
      });
      final data = res.data is Map ? res.data['data'] : null;
      if (!mounted) return;
      setState(() {
        _response = data is Map ? data['response']?.toString() : null;
        _source = data is Map ? data['data_source']?.toString() ?? '' : '';
      });
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('AI 助手暂时不可用，请稍后重试')));
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}
