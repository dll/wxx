import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../providers/personal_detail_provider.dart';
import '../../utils/portal_html_view.dart';

/// 学校门户浏览页 — 通过后端代理访问校内各级网站页面
class PortalBrowsePage extends StatefulWidget {
  const PortalBrowsePage({super.key});

  @override
  State<PortalBrowsePage> createState() => _PortalBrowsePageState();
}

class _PortalBrowsePageState extends State<PortalBrowsePage> {
  final TextEditingController _pathCtrl = TextEditingController(text: '/');
  String _path = '/';
  bool _loading = false;
  String _error = '';
  String _body = '';
  String _contentType = 'text/html';

  @override
  void initState() {
    super.initState();
    _load('/');
  }

  @override
  void dispose() {
    _pathCtrl.dispose();
    super.dispose();
  }

  Future<void> _load(String path) async {
    setState(() {
      _loading = true;
      _error = '';
    });
    final p = context.read<PersonalDetailProvider>();
    final result = await p.proxyPortal(path);
    if (!mounted) return;
    setState(() {
      _loading = false;
      if (result == null) {
        _error = p.error.isNotEmpty ? p.error : '门户访问失败';
      } else {
        _body = result.body;
        _contentType = result.contentType;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('学校门户'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(52),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _pathCtrl,
                    decoration: const InputDecoration(
                      hintText: '路径：/home  /edu/course  /library',
                      isDense: true,
                      prefixIcon: Icon(Icons.link, size: 18),
                      border: OutlineInputBorder(),
                    ),
                    onSubmitted: (v) {
                      final p = v.trim().startsWith('/') ? v.trim() : '/${v.trim()}';
                      _pathCtrl.text = p;
                      setState(() => _path = p);
                      _load(p);
                    },
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  tooltip: '刷新',
                  onPressed: () => _load(_path),
                  icon: const Icon(Icons.refresh),
                ),
              ],
            ),
          ),
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? _buildError(theme)
              : _body.isEmpty
                  ? const Center(child: Text('（空白页面）'))
                  : _contentType.contains('text/html')
                      ? portalHtmlView(_body)
                      : SingleChildScrollView(
                          padding: const EdgeInsets.all(16),
                          child: Text(_body, style: theme.textTheme.bodySmall),
                        ),
    );
  }

  Widget _buildError(ThemeData theme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.cloud_off_outlined,
                size: 48, color: theme.colorScheme.outline),
            const SizedBox(height: 12),
            Text(_error, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton.tonal(
              onPressed: () => _load(_path),
              child: const Text('重试'),
            ),
          ],
        ),
      ),
    );
  }
}
