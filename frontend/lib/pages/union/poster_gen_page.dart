import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../config/api_config.dart';

/// 学生会 - AI 海报文案生成
class PosterGenPage extends StatefulWidget {
  const PosterGenPage({super.key});
  @override
  State<PosterGenPage> createState() => _PosterGenPageState();
}

class _PosterGenPageState extends State<PosterGenPage> {
  final ApiService _api = ApiService();
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  bool _loading = false;
  Map<String, dynamic>? _result;
  String _error = '';

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    super.dispose();
  }

  Future<void> _generate() async {
    if (_titleCtrl.text.trim().isEmpty) return;
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await _api.post(ApiConfig.unionPosterGen, data: {'title': _titleCtrl.text.trim(), 'description': _descCtrl.text.trim()});
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
      appBar: AppBar(title: const Text('AI 海报文案')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('海报信息', style: theme.textTheme.titleMedium),
                const SizedBox(height: 12),
                TextField(controller: _titleCtrl, decoration: const InputDecoration(labelText: '活动标题', border: OutlineInputBorder())),
                const SizedBox(height: 12),
                TextField(controller: _descCtrl, decoration: const InputDecoration(labelText: '活动简介', border: OutlineInputBorder()), maxLines: 3),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: _loading ? null : _generate,
                    icon: const Icon(Icons.brush),
                    label: const Text('生成文案'),
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
                    Icon(Icons.image, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('生成文案', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  ]),
                  const SizedBox(height: 12),
                  Text(_result!['slogan'] ?? '', style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  Text(_result!['body'] ?? _result!['content'] ?? '', style: theme.textTheme.bodyMedium),
                ]),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
