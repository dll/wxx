import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';
import '../../utils/api_error.dart';
import '../../widgets/data_src_badge.dart';

/// 教辅 - 通知批量（AI 辅助生成通知草稿）
class NotificationDraftPage extends StatefulWidget {
  const NotificationDraftPage({super.key});
  @override
  State<NotificationDraftPage> createState() => _NotificationDraftPageState();
}

class _NotificationDraftPageState extends State<NotificationDraftPage> {
  final ApiService _api = ApiService();
  final TextEditingController _topicCtrl = TextEditingController();
  String _channel = '班级群';
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('通知批量')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Text('填写通知主题，AI 辅助生成草稿',
                          style: theme.textTheme.titleMedium
                              ?.copyWith(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 4),
                      Text('草稿生成后请人工核对后发布，AI 不负责最终发布',
                          style: theme.textTheme.bodySmall
                              ?.copyWith(color: theme.colorScheme.outline)),
                      const SizedBox(height: 16),
                      DropdownButtonFormField<String>(
                        value: _channel,
                        decoration: const InputDecoration(
                            labelText: '通知渠道', border: OutlineInputBorder()),
                        items: ['班级群', '学生干部群', '学院公告'].map((c) =>
                            DropdownMenuItem(value: c, child: Text(c))).toList(),
                        onChanged: (v) => setState(() => _channel = v ?? _channel),
                      ),
                      const SizedBox(height: 16),
                      TextField(
                        controller: _topicCtrl,
                        decoration: const InputDecoration(
                          labelText: '通知主题',
                          hintText: '例：期末考试安排 / 课程设计提交提醒',
                          border: OutlineInputBorder(),
                        ),
                        minLines: 2,
                        maxLines: 4,
                      ),
                      const SizedBox(height: 16),
                      SizedBox(
                        width: double.infinity,
                        child: FilledButton.icon(
                          onPressed: _loading ? null : _generate,
                          icon: const Icon(Icons.auto_awesome),
                          label: const Text('AI 生成通知草稿'),
                        ),
                      ),
                    ]),
                  ),
                ),
                if (_error.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.all(8),
                    child: Text(_error,
                        style: TextStyle(color: theme.colorScheme.error)),
                  ),
                if (_result != null) ...[
                  const SizedBox(height: 8),
                  DataSrcBadge(src: _result?['data_source']),
                  Card(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('通知草稿（${_result?['channel']}）',
                                style: theme.textTheme.titleSmall
                                    ?.copyWith(fontWeight: FontWeight.bold)),
                            const SizedBox(height: 12),
                            SelectableText(
                                _result?['draft']?.toString() ?? '',
                                style: const TextStyle(fontSize: 15, height: 1.5)),
                            const SizedBox(height: 12),
                            Row(children: [
                              Icon(Icons.info_outline,
                                  size: 14, color: theme.colorScheme.outline),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text('发布前请人工核对内容与接收对象。',
                                    style: theme.textTheme.bodySmall?.copyWith(
                                        color: theme.colorScheme.outline)),
                              ),
                            ]),
                          ]),
                    ),
                  ),
                ],
              ],
            ),
    );
  }

  Future<void> _generate() async {
    final topic = _topicCtrl.text.trim();
    if (topic.isEmpty) {
      setState(() => _error = '请输入通知主题');
      return;
    }
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final res = await _api.post(ApiConfig.assistantNotification,
          data: {'channel': _channel, 'topic': topic});
      if (res.statusCode == 200 && res.data != null) {
        setState(() => _result = res.data is Map<String, dynamic> ? res.data : {});
      }
    } catch (e) {
      setState(() => _error = friendlyApiError(e));
    } finally {
      setState(() => _loading = false);
    }
  }
}
